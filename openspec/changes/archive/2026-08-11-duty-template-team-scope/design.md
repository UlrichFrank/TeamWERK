## Context

`game_template_items` (`internal/db/migrations/001_initial.up.sql:183-192`) hat bereits eine
Item-Filter-Spalte mit exakt dem Leer-Muster, das dieser Change für Teams braucht:
`audiences TEXT` (JSON-Array, nachträglich per `ALTER TABLE` ergänzt), gelesen über
`audiencesFromDB`/`audiencesToDB` (`internal/games/handler.go:2929-2953`). Dort filtert
`audiences`, **wer den Slot übernehmen darf** — orthogonal zu diesem Change, der filtert,
**für welches Team der Slot überhaupt entsteht**.

Der Erzeugungsort ist `regenGameItems` (`internal/games/regen.go:377-462`). Für
`heim`/`auswärts`-Events läuft pro Item eine Schleife über `teamIDs` — die Teams, die über
`game_teams` am Spiel hängen (`loadGameTeamIDsTx`, `regen.go:572-587`):

```go
for _, tid := range teamIDs {
    if err := insertOne(sql.NullInt64{Int64: int64(tid), Valid: true}); err != nil { ... }
}
```

Ein Spiel kann laut Schema mehrere Teams tragen (`CreateGame` nimmt `team_ids []int`
entgegen, `handler.go:992`) — der Filter muss also **pro Item gegen die Teammenge des
konkreten Spiels** wirken, nicht auf Vorlagen-Ebene pauschal.

**Team-Identität ist stabil über Saisongrenzen.** `teams.id` wird über
`ensureTeam(ageClass, gender, teamNumber)` (`internal/kader/handler.go:110-126`) gefunden
oder angelegt, keyed by `(name, age_class, gender)` mit `name = teamLabel(...)`. Der
Saison-Kopiervorgang (`internal/kader/copy.go:81`) ruft dieselbe Funktion bei jedem
Saisonwechsel erneut auf und findet die bestehende Zeile wieder — eine Team-Scoping-Regel
auf `teams.id` bleibt also über Jahre gültig, ohne dass die Vorlage jährlich nachgepflegt
werden muss. Das ist die Grundlage dafür, dass eine Speicherung auf `teams.id` (statt z.B.
auf `age_class`+`gender`+`team_number` als Text-Tripel) die richtige Wahl ist.

Für die Team-Auswahl im Editor existiert bereits `GET /api/teams/names`
(`internal/games/handler.go:3037`, Route `router.go:353`) — liefert **alle** aktiven Teams
(`teams.is_active=1`, global, nicht saisongebunden) mit `id`, `age_class`, `gender`,
`team_number`, `group_count` für die clientseitige Kurznamen-Berechnung
(`buildTeamShortNames`, siehe `openspec/specs/team-names-endpoint/spec.md`). Das ist **nicht**
dieselbe Menge wie „Kaderteams der aktiven Saison" — letztere ergibt sich erst aus
`kader.team_id WHERE kader.season_id = aktive Saison` (bereits abrufbar über
`GET /api/kader?season_id=`, genutzt in `AdminKaderPage.tsx:107`).

## Goals / Non-Goals

**Goals:**
- Pro Vorlagen-Item eine optionale Team-Allowlist (`teams.id[]`), leer/NULL = alle Teams
  (heutiges Verhalten, keine Verhaltensänderung für Bestandsdaten).
- Der Editor zeigt nur Kaderteams der **aktiven** Saison zur Auswahl an.
- Regen-Filter greift pro Item, unabhängig von anderen Items derselben Vorlage.

**Non-Goals:**
- Keine Altersklassen-Gruppierung als Auswahlmodus (explizit vom Nutzer ausgeschlossen —
  siehe Explore-Diskussion, Entscheidung 3).
- Keine Änderung der `audiences`-Semantik oder ihres Speicherformats.
- Keine serverseitige Durchsetzung, dass gewählte Teams zur aktiven Saison gehören (siehe
  Entscheidung 3 unten) — das ist reine Editor-Bequemlichkeit, keine Datenintegritäts-Regel.
- Keine Koordination mit `duty-bulk-regen` in diesem Change — die Interaktion (§ Zählung) ist
  bereits in dessen `design.md` §11 dokumentiert und wird dort behandelt, wenn dieser Change
  vor jenem landet.
- Kein neuer Endpoint für die Team-Auswahl im Editor — Zusammensetzung aus zwei
  bestehenden Endpoints (siehe Entscheidung 4).

## Decisions

### 1. Speicherform: `team_ids TEXT` (JSON-Array), analog zu `audiences`

Neue Spalte `game_template_items.team_ids` (nullable TEXT, JSON-Array von `teams.id`).
Alternative wäre eine Verknüpfungstabelle `game_template_item_teams (item_id, team_id)`
gewesen — dagegen spricht, dass `audiences` bereits denselben Bedarf (kleine, item-lokale
Filterliste) als JSON-Spalte löst und dieses Muster im selben Handler
(`audiencesFromDB`/`audiencesToDB`) schon Lese-/Schreib-Infrastruktur hat, die sich für
`team_ids` fast unverändert wiederverwenden lässt (`teamIDsFromDB`/`teamIDsToDB` als
Geschwister-Funktionen, gleiche Signatur-Form). Eine Zwischentabelle wäre nur gerechtfertigt,
wenn Team-Zuordnungen eigenständig abgefragt werden müssten (z.B. „alle Items, die Team X
betreffen") — das ist aktuell kein Anwendungsfall.

### 2. Leer-Semantik: NULL **und** `[]` bedeuten „alle Teams"

Bewusst **kein** Tri-State wie bei `games.template_id` (fehlt/`null`/Zahl). `audiences` nutzt
dasselbe Zwei-Zustands-Muster (leer = ein Zustand, nicht drei) und ist im selben Formular
sichtbar — zwei verschiedene Leer-Konventionen nebeneinander in derselben Item-Zeile wären
eine unnötige Falle für Frontend wie API-Konsumenten. Die Umkehrung der Bedeutung gegenüber
`audiences` (dort: leer = **niemand** darf/keine Einschränkung sichtbar im Slot; hier: leer =
**alle** Teams) ist eine bewusste inhaltliche Differenz, die im UI-Hinweistext explizit
benannt werden muss (siehe Punkt 5).

### 3. Serverseitige Validierung prüft Existenz, nicht Saison-Zugehörigkeit

`PUT /api/duty-templates/{id}` validiert `team_ids` gegen `SELECT id FROM teams WHERE id IN
(...)` (400 `invalid_team` bei unbekannter ID) — **nicht** gegen die aktive Saison. Grund:
eine Vorlage überlebt Saisonwechsel (das ist der ganze Punkt der stabilen `teams.id`, siehe
Context). Würde der Server zusätzlich verlangen, dass jedes referenzierte Team im
`kader.team_id` der *aktuell* aktiven Saison auftaucht, bräche das Speichern in der Phase
zwischen Saisonende und Kader-Kopiervorgang (`copy.go`), obwohl die Vorlage inhaltlich
weiterhin korrekt ist. Die Einschränkung auf aktive Kaderteams ist ausschließlich eine
Editor-Anzeige-Entscheidung (Punkt 4), keine Datenregel.

### 4. Team-Auswahl im Editor: Komposition aus zwei bestehenden Endpoints, kein neuer

`AdminDutyTemplateDetailPage.tsx` lädt beim Öffnen zusätzlich `GET /api/kader?season_id=`
(aktive Saison, wie in `AdminKaderPage.tsx` bereits genutzt) für die Menge der
`team_id`-Werte der aktiven Saison, und `GET /api/teams/names` für die
Kurznamen-Berechnungsfelder — gefiltert clientseitig auf die Schnittmenge. Ein dedizierter
Endpoint (`GET /api/teams/active-kader` o.ä.) wäre eine zusätzliche Route für einen reinen
Anzeige-Join, den zwei bestehende, bereits authentifizierte, bereits getestete Endpoints ohne
neue Backend-Logik abdecken.

### 5. Regen-Filter: eine Zeile vor `insertOne`, kein neuer Query-Roundtrip

```go
for _, tid := range teamIDs {
    if len(it.TeamIDs) > 0 && !containsInt(it.TeamIDs, tid) {
        continue // Team nicht in der Item-Allowlist — kein Slot für dieses Team
    }
    if err := insertOne(sql.NullInt64{Int64: int64(tid), Valid: true}); err != nil { ... }
}
```

`it.TeamIDs` kommt aus `loadTemplateItemsTx` (`regen.go:626-648`), die die neue Spalte
zusätzlich zu `audiences` selektiert — **eine** zusätzliche Spalte in der bestehenden Query,
kein zusätzlicher Query. Die Zählung für `RegenSummary.Created[].Count`
(`regen.go:468/474`, aktuell `max(1, len(teamIDs)) * n`) muss auf die **gefilterte**
Teammenge umgestellt werden (Anzahl tatsächlich ausgeführter `insertOne`-Aufrufe je Item),
sonst meldet die Zusammenfassung mehr erzeugte Slots, als tatsächlich entstanden sind.

### 6. Vorschau bekommt die Teams als Parameter, nicht als zweiten Regen-Lauf

`GET /api/duty-templates/{id}/preview` (`handler.go`, einziger Aufrufer
`KalenderPage.tsx:581`) rendert Schritt 4 des Anlege-Wizards. Ohne Team-Wissen zeigt sie
team-eingeschränkte Einträge auch dann, wenn real kein Slot entstünde — der Wizard kennt die
Auswahl (`selectedTeamIds`), reichte sie bisher nur nicht weiter.

Gewählt: **`team_ids` als komma-separierter Query-Parameter**, ersatzweise aus `game_id` über
`game_teams` abgeleitet, sonst ungefiltert. Alternativen und warum nicht:

- *Preview durch `regenGameItems` laufen lassen (echter Dry-Run)*: bräuchte eine Transaktion,
  ein Spiel-Objekt und einen Rollback für einen reinen Lesepfad. Die Vorschau ist bewusst eine
  vereinfachte Projektion (ein Eintrag je Item, nicht je Item × Team) — ein echter Dry-Run
  würde diese Struktur sprengen, ohne dem Nutzer mehr zu sagen.
- *Teams serverseitig aus der aktiven Saison raten*: der Wizard legt ein Event für eine
  konkrete Auswahl an; jede Ableitung wäre eine Vermutung.

**Zwei Fallstricke, die die Filterbedingung bestimmen:**

1. **Ohne Team-Angabe nicht filtern.** Ein Aufruf ohne `team_ids` darf nicht plötzlich alle
   eingeschränkten Einträge verlieren — das wäre eine stille Verhaltensänderung für jeden
   Bestandsaufrufer und in der Wirkung schlimmer als die zu behebende Ungenauigkeit.
2. **`generisch` nie filtern.** Der Regen ignoriert `team_ids` bei generischen Events
   (Entscheidung aus Task 3.5, generische Slots tragen keine `team_id`). Würde die Vorschau
   dort filtern, verschwiege sie einen Slot, der real entsteht — die Lüge nur umgedreht.
   Deshalb lädt die Vorschau zusätzlich `template_type` und wendet den Filter nur auf
   `heim`/`auswärts` an.

Konsequenz fürs Frontend: die Kaderteams-Auswahl im Editor wird bei `generisch`-Vorlagen
**ausgeblendet** statt wirkungslos angeboten. Bereits gespeicherte `team_ids` einer solchen
Vorlage bleiben unangetastet in der DB (kein Aufräum-Schreibvorgang) — sie sind lediglich
folgenlos, in Editor wie Vorschau wie Regen konsistent.

## Risks / Trade-offs

- **[Risiko] Stiller Nulleffekt.** Ist ein Item auf Teams beschränkt, die nie gemeinsam an
  einem Spiel derselben Vorlage auftreten (z.B. Konfigurationsfehler: Team aus einer anderen
  Altersklasse gewählt), entstehen für dieses Item nie Slots — ohne Fehlermeldung, da das
  fachlich ein gültiger Zustand ist (Item soll ja selektiv gelten). → **Mitigation:** keine im
  Backend; der Editor zeigt nach dem Speichern weiterhin exakt die gewählten Teams an, sodass
  ein Konfigurationsfehler beim nächsten Öffnen sichtbar bleibt. Eine Live-Vorschau „wie viele
  Slots entstehen mit dieser Auswahl" wäre denkbar, ist aber eine Erweiterung des
  Vorlagen-Editors über den Rahmen dieses Change hinaus.
- **[Risiko] Team-Restrukturierung.** Spaltet der Verein eine Mannschaft (z.B. neue mA2 neben
  bestehender mA1), entsteht eine neue `teams.id` (`ensureTeam` legt sie nur an, wenn
  `team_number` differiert). Ein bestehendes Item mit `team_ids: [mA1]` schließt mA2 **nicht**
  automatisch ein. → **Mitigation:** keine Automatik vorgesehen (Non-Goal) — das ist dieselbe
  Art manueller Pflegeaufwand wie jede andere Vorlagen-Anpassung bei Vereinsstruktur-Änderung.
- **[Risiko] Cross-Change-Zählung.** `duty-bulk-regen` (paralleler, noch offener Change)
  rechnet ebenfalls mit `RegenSummary.Created[].Count`. → **Mitigation:** dort bereits in
  `design.md` §11 dokumentiert; keine Handlung in diesem Change nötig, nur Reihenfolge im
  Blick behalten.
- **[Trade-off] Keine Server-Enforcement der Saison-Zugehörigkeit** (Entscheidung 3) bedeutet,
  dass ein Admin theoretisch über die rohe API ein längst inaktives Team eintragen könnte. Das
  ist bewusst hingenommen — dieselbe Großzügigkeit gilt bereits für `games.template_id` und
  andere Fremdschlüssel-Felder in diesem Codebereich; UI-Führung (Punkt 4) verhindert den
  Normalfall, ist aber keine Sicherheitsgrenze.

## Migration Plan

Rein additiv, keine Datenmigration:

1. Migration `044_duty_template_item_team_scope` (nächste freie Nummer — vor Task-Umsetzung
   `ls internal/db/migrations/ | sort -V | tail -1` erneut prüfen, falls parallel andere
   Changes landen): `ALTER TABLE game_template_items ADD COLUMN team_ids TEXT;` /
   Down: `ALTER TABLE game_template_items DROP COLUMN team_ids;` (DROP COLUMN ist in diesem
   Codebase bereits belegtes Muster, siehe `025_match_report_title_und_typo3_cat_drop.up.sql`).
2. Bestehende Items haben `team_ids IS NULL` → Regen-Verhalten unverändert (Invariante 1 aus
   dem Proposal).
3. Kein Backfill, kein Datenumbau, kein Downtime-Fenster. Rollback ist die Standard-`migrate
   down` — nach Rollback verlieren evtl. bereits gepflegte Team-Zuordnungen ihre Wirkung
   (Spalte weg), Items gelten dann wieder für alle Teams — sicherer Fallback, kein
   Datenverlust an anderer Stelle.

## Open Questions

- Soll der Editor perspektivisch eine Live-Anzeige „N Slots entstehen mit dieser Auswahl"
  bekommen (siehe Risiko „Stiller Nulleffekt")? Nicht Teil dieses Change, aber ein
  naheliegender Folge-Change, falls sich Konfigurationsfehler in der Praxis häufen.
- Reicht `containsInt` als einfache lineare Suche über die (typischerweise <5 Einträge lange)
  `team_ids`-Liste, oder lohnt sich eine `map[int]bool`-Konvertierung in `regenGameItems`?
  Bei der erwarteten Größenordnung (wenige Teams pro Item, wenige Items pro Vorlage)
  vernachlässigbar — Entscheidung bei Implementierung, kein Design-Blocker.
