## Datenmodell — Vorher / Nachher

```
VORHER
────────────────────────────────────────────────────────────
users                            members
──────                           ──────
id                               id
photo_path (own upload)          user_id  ────► users.id (nullable)
                                 photo_path (parent/admin upload)
                                 photo_visible

user_visibility
──────────────
user_id  ────► users.id
photo_visible

Ergebnis:  Zwei Slots, zwei Sichtbarkeits-Felder,
           drift zwischen Kind-Selbstupload und Eltern-Upload.

NACHHER
────────────────────────────────────────────────────────────
users                            members
──────                           ──────
id                               id
photo_path (SINGLE SOURCE)       user_id  ────► users.id (nullable)
                                 (photo_path entfernt)
                                 (photo_visible entfernt)

user_visibility
──────────────
user_id  ────► users.id
photo_visible (SINGLE SOURCE)

Ergebnis:  Ein Slot, eine Sichtbarkeit, kein Drift möglich.
           Member ohne user_id → kein Foto (bewusst).
```

## Endpoint-Verhalten

Alle drei Upload-Routen produzieren dasselbe Resultat auf **denselben** Spalten:

```
Route                                    Ziel-Spalte              Precondition
────────────────────────────────────────────────────────────────────────────────
POST /api/upload/user-photo              users.photo_path         eingeloggt
                                         (claims.UserID)
POST /api/profile/kind/{id}/photo        users.photo_path         family_link
                                         (members[id].user_id)    + user_id NOT NULL
POST /api/upload/member-photo/{id}       users.photo_path         admin/vorstand
                                         (members[id].user_id)    + user_id NOT NULL
```

Precondition-Fehlschlag (Member ohne User): HTTP 409 mit Body
`{"error": "member_has_no_user_account"}` — kein File-Write, kein Storage-Leak.

## Migrationsstrategie (029_photo_user_only)

Migration in einer Transaktion:

```sql
-- 029_photo_user_only.up.sql

-- 1) User hat noch kein Foto, Member hat eins → übernehmen
UPDATE users
SET photo_path = (SELECT m.photo_path FROM members m WHERE m.user_id = users.id)
WHERE users.photo_path IS NULL
  AND EXISTS (
    SELECT 1 FROM members m
    WHERE m.user_id = users.id AND m.photo_path IS NOT NULL
  );

-- 2) photo_visible aus members auf user_visibility spiegeln, wo user_visibility
--    noch keinen Eintrag hat (photo_visible-only Fallback stirbt).
INSERT INTO user_visibility (user_id, photo_visible)
SELECT m.user_id, m.photo_visible
FROM members m
WHERE m.user_id IS NOT NULL
  AND m.photo_visible = 1
  AND NOT EXISTS (
    SELECT 1 FROM user_visibility uv WHERE uv.user_id = m.user_id
  );

UPDATE user_visibility
SET photo_visible = 1
FROM members m
WHERE user_visibility.user_id = m.user_id
  AND m.photo_visible = 1
  AND user_visibility.photo_visible = 0;

-- 3) Spalten entfernen (SQLite: table recreate — analog zu Migration 019)
--    Restliche Member-Spalten bleiben unverändert.
--    photo_path von Members ohne user_id werden mit dem Column-Drop obsolet;
--    die zugehörigen Dateien im uploadDir werden in Schritt 4 (Go-Code)
--    aufgeräumt.

-- (Table-Recreate-Boilerplate hier ausgelassen — Muster: 019_match_reports.up.sql)
```

**Schritt 4 (Go, nicht SQL):** Der Migrations-Runner ist ein SQL-only-Werkzeug.
Das Löschen verwaister Dateien läuft als **einmaliger idempotenter Backfill**
beim Server-Start (Muster: `internal/videos/backfill.go`) — Query „Dateien in
`uploadDir/member-photos/*`, deren Name in **keiner** `users.photo_path` mehr
vorkommt", `os.Remove` je Treffer, Log-Zeile mit Zusammenfassung. Der Backfill
läuft als Goroutine und blockt `serve()` nicht.

**Rollback:** `029_photo_user_only.down.sql` legt `members.photo_path` und
`members.photo_visible` wieder an (nullable, DEFAULT 0). **Datenverlust
akzeptiert** — Down-Migration kopiert nicht zurück. Betriebsseitige
Absicherung: DB-Backup vor `make migrate-remote-up` (steht schon so für
Zero-Knowledge-Prod-Migrationen im Runbook).

## Warum kein Foto für Members-ohne-User?

Diskutierte Alternativen:

- **Shadow-User anlegen:** würde jedes Member-ohne-Account unfreiwillig zu
  einem Login-fähigen Konto machen (mit Passwort-Recovery-Angriffsfläche,
  Konsent-Fragen). Scope-Explosion.
- **`members.photo_path` als Fallback behalten:** reintroduziert exakt das
  Zwei-Slot-Problem, das dieser Change auflöst. Kein Kompromiss.
- **Foto-Slot auf `family_links` oder `member_metadata`:** eine dritte
  Datenstelle, die nichts löst und alle Lesepfade verkompliziert.

Entscheidung: Ohne User kein Foto. Sobald das Mitglied einen Account bekommt
(Einladung, Selbst-Registrierung), kann das Foto gesetzt werden. Bis dahin
zeigt die UI das übliche „Kein Bild"-Placeholder.

## Auswirkung auf laufende Änderungen

- `profile-datenschutz-dsgvo-editable` (in progress): berührt
  `foto_veroeffentlichung` (Presse-Konsens), nicht `photo_path`/`photo_visible`.
  Keine Kollision.
- `kind-profil-user-strang` (archiviert): dieser Change modifiziert dessen
  Requirement zur Visibility (Fallback-Klausel für `photo_visible` entfällt)
  und ergänzt ein neues Requirement für die Foto-Route.

## Test-Strategie

- **Unit/Handler-Tests** (`internal/upload/*_test.go`,
  `internal/members/*_test.go`): siehe Test-Anforderungen im Proposal.
- **Migration-Test:** neuer Test in `internal/db/migrations_test.go` (Pattern
  existiert bereits): setzt drei Members auf (mit User+beide Fotos /
  mit User+nur member-Foto / ohne User+member-Foto), fährt `023` hoch, prüft
  User-Precedence, Kopie, `photo_visible`-Übernahme, Spalten-Drop.
- **Backfill-Test:** verwaiste Datei im temporären `uploadDir` anlegen,
  Backfill triggern, prüfen dass Datei weg und referenzierte Dateien
  unangetastet sind.
- **Frontend:** vitest für `MembersPage`/`MemberDetailPage`/`ChildProfilePage`
  auf konsolidiertes `photo_url`-Feld anpassen (bestehende Tests, kein
  Neu-Aufbau).
