# Design

## Kontext

`folder_permissions` kennt heute vier Principal-Typen und wird von zwei wortgleichen Funktionen
ausgewertet:

```
internal/files/handler.go:91    resolveAccess()        ← 14 Aufrufstellen
internal/policy/folders.go:64   policy.FolderAccess()  ← 1 Aufrufstelle (FolderContents:294)
```

Beide laufen den Pfad `[Ordner, Elternteil, …, Wurzel]` hoch und stoppen beim ersten Ordner mit
`hasAny == true`, also mit **irgendeiner** ACL-Zeile. Genau dieses `hasAny` — nicht „hat eine
passende Zeile" — erzeugt den gemeldeten Fehler.

## Entscheidung 1: Eigentümer-Vorrang vor dem Pfad-Walk

Der Check läuft **vor** der Schleife und kurzschließt sie, parallel zum bestehenden
Admin-Kurzschluss:

```go
func FolderAccess(db *sql.DB, p *Principal, folderID int) (canRead, canWrite bool, err error) {
    if p.Role == "admin" { return true, true, nil }

    path, err := folderPath(db, folderID)   // [folder, parent, …, root]
    if err != nil { return false, false, err }

    owns, err := ownsAnyOf(db, p.UserID, path)
    if err != nil { return false, false, err }
    if owns { return true, true, nil }      // ← neu

    // … unveränderter Nearest-Ancestor-Walk
}
```

`ownsAnyOf` ist **eine** Query über den bereits berechneten Pfad, keine Schleife:

```sql
SELECT 1 FROM file_folders
 WHERE created_by = ? AND id IN (?, ?, …)
 LIMIT 1
```

**Warum vor dem Walk und nicht als zusätzlicher Match im Walk?** Ein Match innerhalb der Schleife
würde nur auf dem Ordner greifen, auf dem der Walk ohnehin stoppt. Genau der Fall, den wir
reparieren müssen — Vererbung vom Elternordner, gekappt durch eine fremde ACL-Zeile — läge dann
weiterhin außerhalb. Der Vorrang muss dem Walk vorgelagert sein, damit er unabhängig davon greift,
wo der Walk endet.

**Warum unterbaumweit (`path` statt nur `folderID`)?** Legt jemand mit Schreibrecht einen
Unterordner in einem fremden Ordner an und trägt dort nur sich selbst ein, reproduziert die
Ordner-lokale Variante exakt den gemeldeten Bug eine Ebene tiefer — der Eigentümer hätte ein
schwarzes Loch im eigenen Baum. Root-Ordner sind admin-erstellt (`CreateFolder:255` verlangt
`role == admin` für `parent_id == NULL`), das obere Ende jeder Kette ist also folgenlos, weil Admin
ohnehin kurzschließt.

**Bewusst in Kauf genommen:** Das Recht ist absolut und unbefristet. Es überlebt den Verlust der
Vereinsfunktion und ist nur über ein `UPDATE file_folders SET created_by = …` entziehbar. Die
Alternative — den Walk beim Eigentümer nicht stoppen zu lassen und die geerbten Rechte
hinzuzuunionen — hätte kein Dauerrecht erzeugt, wäre aber in keiner Oberfläche erklärbar gewesen.
Gegen die Unsichtbarkeit hilft hier stattdessen der Pseudo-Eintrag (Entscheidung 5).

**Wechselwirkung mit `checkAntiEscalation`:** Die Funktion ruft `FolderAccess` auf und sieht damit
automatisch `(true, true)` für den Eigentümer. Er darf auf seinem Ordner also alles vergeben — was
konsistent ist, denn genau das gilt heute schon für jeden mit `can_write`.

## Entscheidung 2: `team` und `team_parents` als getrennte Principal-Typen

Alternative wäre ein Typ `team` mit zusammengesetztem `principal_ref` (`"7:spieler"`) oder eine
zusätzliche Scope-Spalte gewesen. Zwei Typen gewinnen, weil:

- die bestehenden Spalten `can_read`/`can_write` unverändert reichen,
- jede Zeile für sich löschbar ist („Eltern von mA1 wieder rausnehmen"),
- `principal_ref` ein sauberer Fremdschlüsselwert bleibt (`teams.id` als Text, wie `user` heute
  schon die `users.id` als Text hält),
- das Frontend ohne Sonderfall auskommt: zwei Einträge im Typ-Dropdown, ein gemeinsames zweites
  Dropdown.

**`teams.id`, nicht `kader.id`.** Ein Kader ist saisongebunden; eine Berechtigung auf `kader.id`
wäre am nächsten Saisonwechsel tot. Über `teams.id` + Filter auf die aktive Saison folgt die
Berechtigung automatisch dem neuen Kader derselben Mannschaft.

**Nicht über `user_accessible_teams`.** Der View unioniert Spieler, erweiterten Kader, Trainer und
Eltern in eine Menge und verliert dabei genau die Unterscheidung, die `team` von `team_parents`
trennt. Deshalb zwei eigene Queries:

```sql
-- team: Spieler + erweiterter Kader + Trainer der Mannschaft, aktive Saison
SELECT DISTINCT k.team_id
  FROM kader k
  JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
 WHERE k.team_id IS NOT NULL
   AND EXISTS (
     SELECT 1 FROM members m WHERE m.user_id = ?1 AND (
          EXISTS (SELECT 1 FROM kader_members          km  WHERE km.kader_id  = k.id AND km.member_id  = m.id)
       OR EXISTS (SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id = k.id AND kem.member_id = m.id)
       OR EXISTS (SELECT 1 FROM kader_trainers         kt  WHERE kt.kader_id  = k.id AND kt.member_id  = m.id)
     ))

-- team_parents: Elternteile derselben Kader-Menge (ohne kader_trainers)
SELECT DISTINCT k.team_id
  FROM family_links fl
  JOIN kader k ON k.team_id IS NOT NULL
  JOIN seasons s ON s.id = k.season_id AND s.is_active = 1
 WHERE fl.parent_user_id = ?1
   AND (   EXISTS (SELECT 1 FROM kader_members          km  WHERE km.kader_id  = k.id AND km.member_id  = fl.member_id)
        OR EXISTS (SELECT 1 FROM kader_extended_members kem WHERE kem.kader_id = k.id AND kem.member_id = fl.member_id))
```

`team_parents` deckt bewusst nur Eltern von Spielern und Förderkader-Mitgliedern ab, nicht „Eltern
von Trainern" — das ist keine sinnvolle Personengruppe.

**Keine aktive Saison vorhanden** → beide Mengen sind leer → Team-Zeilen matchen niemanden. Das ist
das gewünschte Fail-Closed-Verhalten und deckt sich mit dem Gotcha „Aktive Saison".

## Entscheidung 3: Lazy Principal-Kontext statt zwei weiterer Queries pro Ordner

`FolderAccess` wird in `ListRootFolders` und `FolderContents` **pro Ordner in einer Schleife**
aufgerufen. Schon heute läuft `fetchFamilyContext` dabei unbedingt mit — auch für Ordner, die gar
keine `club_function`- oder `user`-Zeile haben. Zwei weitere unbedingte Queries für die Team-Mengen
würden das auf vier Zusatz-Queries je Ordner treiben.

Deshalb wird der Kontext bedarfsgesteuert aufgebaut:

```go
type principalCtx struct {
    db     *sql.DB
    userID int

    familyLoaded    bool
    linkedUserIDs   []int
    linkedFunctions []string

    teamsLoaded  bool
    playerTeams  []int   // principal_type = 'team'
    parentTeams  []int   // principal_type = 'team_parents'
}
```

Die Getter (`family()`, `teams()`) laden beim ersten Zugriff und cachen danach. Eine ACL-Zeile vom
Typ `everyone` oder `role` löst gar keine Query mehr aus. Netto ist das **weniger** DB-Last als
heute, obwohl zwei Auflösungsarten hinzukommen.

Der Kontext lebt pro `FolderAccess`-Aufruf. Ihn über eine ganze Ordnerliste zu teilen wäre der
nächste Optimierungsschritt, erfordert aber eine Signaturänderung an allen 15 Aufrufstellen —
bewusst nicht Teil dieses Changes.

## Entscheidung 4: Anzeigename — Langname vom Server, Kurzname im Client

`ListPermissions` liefert für Team-Zeilen `display_name = teams.name` (z. B. „mA-Jugend"), analog
zum bestehenden Fallback für gelöschte User auf `principal_ref`.

Den **Kurznamen** („mA1") berechnet weiterhin ausschließlich das Frontend über
`buildTeamShortNames` (`web/src/lib/teamName.ts:24`). Der Kurzname braucht `group_count` — die
Anzahl Kader derselben Altersklasse/Geschlecht in der Saison —, und diese Logik in Go zu
duplizieren würde eine zweite Wahrheit über Teamnamen schaffen. Das Modal lädt die Teamliste
ohnehin für das Dropdown; es nutzt sie auch für die Liste und fällt auf `display_name` zurück,
solange sie nicht geladen ist. Das entspricht exakt der Fallback-Kette in `formatTeamList`
(`display_short ?? name ?? ''`).

## Entscheidung 5: Eigentümer als Pseudo-Eintrag

`GET /api/folders/{id}/permissions` stellt der Liste einen synthetischen Eintrag voran:

```json
{ "id": 0, "principal_type": "owner", "principal_ref": "12",
  "display_name": "Florian Steinle", "can_read": true, "can_write": true }
```

`id: 0` existiert in `folder_permissions` nicht; das Frontend rendert die Zeile ohne
Löschen-Button. Ohne diesen Eintrag wäre das stärkste Recht am Ordner das einzige, das nirgends
sichtbar ist — ein Vorstand, der die Berechtigungen prüft, würde Florians Schreibrecht nicht sehen.
Es ist ausdrücklich **keine** ACL-Zeile: `owner` taucht nicht im CHECK-Constraint auf und wird von
`AddPermission` mit HTTP 400 abgelehnt.

Fallback wie beim `user`-Typ: ist der Ersteller nicht auflösbar, steht die ID in `display_name`.

## Entscheidung 6: Konsolidierung auf `policy.FolderAccess`

`files.resolveAccess` entfällt ersatzlos; alle Aufrufstellen bauen sich einen `policy.Principal`
aus den Claims — das Muster steht schon in `FolderContents:287`. Ein kleiner Helfer im
`files`-Package kapselt das:

```go
func (h *Handler) access(r *http.Request, folderID int) (canRead, canWrite bool, err error) {
    c := auth.ClaimsFromCtx(r.Context())
    p := &policy.Principal{UserID: c.UserID, Role: c.Role,
                           ClubFunctions: c.ClubFunctions, IsParent: c.IsParent}
    return policy.FolderAccess(h.db, p, folderID)
}
```

Der Architektur-Test erlaubt `files → policy` bereits (die Kante existiert über
`FolderContents`). Nach dem Umbau ist die Rechtelogik an genau einer Stelle änderbar.

## Entscheidung 7: Migration 038 — Table-Rebuild

SQLite kann einen `CHECK`-Constraint nicht per `ALTER TABLE` ändern. `folder_permissions` hat einen
ausgehenden Fremdschlüssel auf `file_folders` und **keine eingehenden**, der Rebuild ist damit
unkritisch:

```sql
-- up
CREATE TABLE folder_permissions_new (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id      INTEGER NOT NULL REFERENCES file_folders(id) ON DELETE CASCADE,
    principal_type TEXT    NOT NULL CHECK (principal_type IN
                     ('everyone','role','club_function','user','team','team_parents')),
    principal_ref  TEXT,
    can_read       INTEGER NOT NULL DEFAULT 0,
    can_write      INTEGER NOT NULL DEFAULT 0
);
INSERT INTO folder_permissions_new SELECT * FROM folder_permissions;
DROP TABLE folder_permissions;
ALTER TABLE folder_permissions_new RENAME TO folder_permissions;
```

Der `down`-Pfad **verliert Daten** und muss das explizit tun, sonst scheitert das Rückspielen am
alten CHECK:

```sql
-- down
DELETE FROM folder_permissions WHERE principal_type IN ('team','team_parents');
-- … Rebuild mit dem alten Vier-Werte-CHECK
```

Das ist als Kommentar in beiden Dateien zu vermerken. Ein Down-Migrate nach produktiver Nutzung
der Team-Berechtigungen löscht diese ersatzlos.

## Was dieser Change nicht tut

- **Kein explizites Vererbungs-Flag.** Die Ursache — „erste ACL-Zeile kappt implizit die
  Vererbung" — bleibt bestehen. Wer über `club_function` geerbt hatte und durch eine fremde Zeile
  herausfällt, wird nicht geheilt; nur der Eigentümer ist geschützt. Ein Modell mit
  `inherit_permissions`-Flag pro Ordner (NTFS/SharePoint-Muster) wäre die vollständige Lösung und
  bleibt als möglicher Folge-Change offen.
- **Keine Eigentümer-Übertragung.** `created_by` ist über die UI nicht änderbar.
- **Kein Backfill.** Bestehende Ordner brauchen keine Datenreparatur, weil der Eigentümer-Vorrang
  aus dem bereits gefüllten `created_by` liest.
