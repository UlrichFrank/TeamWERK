## Context

TeamWERK modelliert Trainingstermine als `training_sessions` (konkrete Datums-Instanz, `date`), die optional zu einer `training_series` gehören (`series_id` nullable, `ON DELETE SET NULL`). Absagen laufen heute über `training_responses` (`status IN ('confirmed','declined','maybe')`, UNIQUE `(training_id, member_id)`). Datumsbasierte Selbst-Abwesenheiten (`member_absences`, Typ `vacation|injury`) legt der Spieler/Elternteil an; sie erzeugen gesperrte `declined`-Responses (`training_responses.absence_id`) und zählen in der Anwesenheitsstatistik als **entschuldigt** (bleiben im Nenner).

Es fehlt eine **serien-gebundene Dauer-Abmeldung durch den Trainer** für Spieler, die eine wiederkehrende Serie strukturell nie besuchen (feste A-Jugend-Zeit, Berufsschule, Langzeitverletzung). Solche Spieler sollen als „nicht erwartet" gelten — komplett aus dem Nenner, nicht als entschuldigt.

Relevanter Bestand (verifiziert am Code):
- Serien-Tier: Sub-Router mit `auth.RequireClubFunction("trainer","sportliche_leitung")` in `internal/app/router.go` (`training-series`-CRUD + `SaveAttendances`).
- Team-genaue Prüfung: `hasTeamAccess` in `internal/trainings/handler.go:133` (admin/vorstand/sportliche_leitung → true; sonst `kader_trainers` JOIN `kader` auf `k.team_id`).
- RSVP: `Respond` (`internal/trainings/handler.go:1359`, Route `POST /api/training-sessions/{id}/respond`, Authenticated-Tier) mit bestehendem Absence-Lock.
- Attendance: `SaveAttendances` (`internal/trainings/handler.go:1665`, Route `POST /api/training-sessions/{id}/attendances`, Trainer-Tier) — Bulk-Upsert, überspringt bereits trainer-only-Members.
- Statistik: `loadCounts` (`internal/attendance/handler.go:117`) aggregiert pro Member drei Buckets (present / missed / excused) per bedingter `SUM`; `GetTeamStats`/`GetMemberStats`.

## Goals / Non-Goals

**Goals:**
- Trainer des eigenen Teams kann einen Spieler serien-gebunden dauerhaft (Enddatum optional) abmelden, mit optionalem Grund.
- Betroffene Session×Spieler fallen aus dem Statistik-Nenner (weder anwesend/fehlt/entschuldigt); Detail-Liste zeigt Kategorie `unavailable`.
- RSVP (Spieler/Eltern) und Trainer-Anwesenheitserfassung für betroffene Sessions werden serverseitig unterbunden.
- In `/termine` bleibt der Spieler sichtbar mit Badge „dauerhaft abgemeldet" + Grund; nur der Trainer sieht die Lösch-Aktion.

**Non-Goals:**
- Kein Wochentag-/Kategorie-Muster unabhängig von einer konkreten Serie (YAGNI). „Nur diesen Termin" = `start_date = end_date`.
- Keine Abmeldung für Einzeltermine (`series_id IS NULL`) — die handhabt der Trainer per normaler Attendance.
- Keine Abmeldung für Spiele (`games`) — Spiele haben eigene Semantik.
- Kein Recovery/Undo über das simple `DELETE` hinaus.

## Decisions

**D1 — Eigene Tabelle `member_series_unavailabilities`, nicht Erweiterung von `member_absences`.**
Die beiden Semantiken sind orthogonal: `member_absences` ist member-global, selbst-gepflegt, datumsbasiert, zählt als *entschuldigt*. Die neue ist team-/serien-gebunden, trainer-gepflegt, zählt *gar nicht*. Ein gemeinsamer Tisch zwänge jede Query, beide Bedeutungen auseinanderzuhalten. Getrennt bleibt jede Query eindeutig.

Schema:
```sql
CREATE TABLE member_series_unavailabilities (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    member_id           INTEGER NOT NULL REFERENCES members(id)         ON DELETE CASCADE,
    training_series_id  INTEGER NOT NULL REFERENCES training_series(id) ON DELETE CASCADE,
    start_date          DATE,           -- NULL = ab Serien-Beginn
    end_date            DATE,           -- NULL = permanent (bis Serien-Ende)
    reason              TEXT NOT NULL DEFAULT '',
    created_by          INTEGER NOT NULL REFERENCES users(id)           ON DELETE RESTRICT,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (member_id, training_series_id, start_date)
);
CREATE INDEX idx_msu_series  ON member_series_unavailabilities(training_series_id);
CREATE INDEX idx_msu_member  ON member_series_unavailabilities(member_id);
```
`team_id` wird **nicht** redundant gespeichert — via `training_series.team_id` ableitbar (keine Inkonsistenz möglich). Nächste freie Migrationsnummer (`.up.sql` + `.down.sql`).

**D2 — Ableitung „greift Abmeldung für Session Y?" als reiner Lookup, keine vorab angelegten Response-Rows (Weg b).**
Session Y (series_id=S, date=D) ist für Member X abgemeldet gdw. eine Zeile existiert mit `member_id=X AND training_series_id=S AND (start_date IS NULL OR start_date<=D) AND (end_date IS NULL OR end_date>=D)`. Ein Index-Lookup (`idx_msu_series`). Kein Doppel-Zustand, kein Cleanup beim Löschen der Abmeldung, keine Race mit Session-Generierung — im Gegensatz zum `member_absences`-Auto-decline, das genau dort gelegentlich bricht.

**D3 — RSVP hart ablehnen, Attendance-Bulk still überspringen.**
`Respond` (Einzel-Endpoint) → HTTP 403 („für diese Terminserie abgemeldet"), analog zum Absence-Lock. `SaveAttendances` ist ein **Bulk**-Upsert; ein 403 wegen eines einzelnen Members würde den ganzen Speichervorgang des Trainers scheitern lassen. Stattdessen wird ein abgemeldeter Member — exakt wie die bereits vorhandene trainer-only-Ausnahme — aus dem Persist **übersprungen** (keine `training_attendances`-Zeile). Effekt ist identisch: kein Attendance-Record → nicht im Nenner. Das Frontend rendert für abgemeldete Spieler ohnehin keinen An-/Abwesenheits-Toggle.

**D4 — Statistik: Ausschluss additiv in `loadCounts`.**
LEFT JOIN auf `member_series_unavailabilities` über `(member_id, ts.series_id)` mit Datumsfenster; die drei bedingten `SUM`-Buckets bekommen zusätzlich `AND msu.id IS NULL`. Damit fällt Session×Member vollständig aus present/missed/excused. In `GetMemberStats` erhält ein solcher Termin die neue Kategorie `unavailable` (zählt in keiner Spalte). Team-Aggregat: der Nenner ist ohnehin die Summe der drei Buckets *pro Member* — er wird pro Spieler unterschiedlich, die Team-Quote ist der Ø der Pro-Spieler-Quoten. Frontend ergänzt eine Fußnote.

**D5 — Autorisierung wie beim Serien-CRUD.**
Neue Routen im bestehenden `RequireClubFunction("trainer","sportliche_leitung")`-Sub-Router, plus inline `hasTeamAccess(series.team_id)` — Trainer nur für das eigene Team, sportliche_leitung/Admin überall. Spieler/Eltern haben keinen Schreibzugriff.

**D6 — Routen an die Serie gehängt.**
```
GET    /api/training-series/{id}/unavailabilities
POST   /api/training-series/{id}/unavailabilities   { member_id, start_date?, end_date?, reason? }
DELETE /api/training-series/{id}/unavailabilities/{uid}
```
Der Termin-Detail-Einstieg in `/termine` löst über die `series_id` der Session auf denselben POST auf (Prefill `start_date=heute`, `end_date` leer). Broadcast `training-unavailability-changed` (global; Frontend filtert im `useLiveUpdates`-Callback) — erfüllt das Broadcast-Gate.

**D7 — Sichtbarkeit für den Spieler.**
`GET /api/training-sessions` / `GET /api/training-sessions/{id}` liefern pro Mitglied `unavailable: { reason, permanent } | null`. Der Spieler sieht seinen eigenen Status (Badge „dauerhaft abgemeldet"), kann aber nichts ändern; nur Trainer sehen die Lösch-Aktion.

## Risks / Trade-offs

- **Unterschiedliche Nenner pro Spieler in der Team-Quote** → Team-„Quote" ist kein einheitlicher Bruch mehr. Mitigation: bewusste Entscheidung (Ø der Pro-Spieler-Quoten) + Frontend-Fußnote „* dauerhaft abgemeldete Spieler zählen für ihre Termine nicht mit".
- **Überlappende Abmelde-Zeiträume derselben Serie/Member** möglich (UNIQUE greift nur bei gleichem `start_date`). Mitigation: bewusst harmlos — greift eine, ist die Ableitung korrekt; kein Extra-Check.
- **Serie gelöscht → CASCADE entfernt Abmeldungen**, während vergangene Sessions per `ON DELETE SET NULL` erhalten bleiben (`series_id`=NULL). Danach würde ein solcher vergangener Termin den Spieler wieder in den Nenner nehmen. Mitigation: akzeptiert — Serien-Löschung ist selten, und ohne `series_id` ist die Abmeldung ohnehin nicht mehr ableitbar.
- **Kollision mit `member-absences`** (beide greifen für dieselbe Session) → Mitigation: Serien-Abmeldung gewinnt für die Statistik (Ausschluss > entschuldigt); der `msu.id IS NULL`-Filter in `loadCounts` hat Vorrang vor dem excused-Bucket.
- **Trainer scheidet aus dem Kader aus** → Abmeldung bleibt bestehen (Team-State, kein Trainer-State); `created_by` ist nur Audit.

## Migration Plan

1. Neue Migration `internal/db/migrations/0NN_member_series_unavailabilities.{up,down}.sql` (nächste freie Nummer). `down` droppt Tabelle + Indizes.
2. Deploy zieht `migrate up` automatisch mit (`make deploy`). Rein additive Tabelle, keine Datenrückführung, kein Backfill — vor dem Feature existiert keine solche Abmeldung.
3. Rollback: `migrate down` einer Stufe entfernt die Tabelle; abhängige Statistik-/RSVP-Logik degradiert sauber (keine Abmeldungen ⇒ Verhalten wie zuvor).

## Open Questions

Keine offen — Statistik-Semantik (Ø Pro-Spieler-Quoten + Fußnote), Überlappung (harmlos erlaubt), CASCADE, Trainer-Ausscheiden und Broadcast-Granularität (global) sind entschieden.
