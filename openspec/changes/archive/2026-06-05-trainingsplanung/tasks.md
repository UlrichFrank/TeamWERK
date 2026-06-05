## 1. Datenbank-Migration

- [x] 1.1 `internal/db/migrations/009_trainings.up.sql` anlegen: Tabellen `training_series`, `training_sessions`, `training_responses`, `training_attendances` mit allen FK-Constraints, CHECK-Constraints und Indizes
- [x] 1.2 `internal/db/migrations/009_trainings.down.sql` anlegen: alle 4 Tabellen in umgekehrter Reihenfolge droppen
- [x] 1.3 Migration lokal testen: `make migrate-up` und `make migrate-down`

## 2. Backend — Package-Grundstruktur

- [x] 2.1 Package `internal/trainings/` anlegen mit `handler.go` (Handler-Struct mit `db *sql.DB`) und `NewHandler`-Funktion
- [x] 2.2 Routes in `cmd/teamwerk/main.go` registrieren: Public-freie Middleware-Gruppen für authenticated, trainer+admin

## 3. Backend — Trainingsserien

- [x] 3.1 `POST /api/training-series`: Serie anlegen + alle Sessions für passende Wochentage zwischen `valid_from` und `valid_until` als Bulk-Insert generieren (SQLite-Transaktion)
- [x] 3.2 `PUT /api/training-series/{id}`: Serie bearbeiten mit `scope`-Parameter (`this_and_following` mit `from_date`, oder `all`): betroffene Sessions löschen und neu generieren
- [x] 3.3 `DELETE /api/training-series/{id}`: Serie löschen, nur Sessions mit `date >= today` entfernen; `training_series`-Row bleibt für historische Referenz

## 4. Backend — Einzelne Sessions

- [x] 4.1 `POST /api/training-sessions`: Einzeltermin ohne Serie anlegen
- [x] 4.2 `PUT /api/training-sessions/{id}`: Session-Felder aktualisieren (Ort, Zeit, Note, Status/Cancel-Reason); Trainer-Autorisierung prüfen (eigenes Team)
- [x] 4.3 `GET /api/training-sessions`: Liste aller Sessions für den anfragenden User (Spieler → eigene Teams, Elternteil → Teams der Kinder via family_links, Trainer → eigene Teams, Admin → alles); Filter `team_id`, `from`, `to`; inklusive Response-Summary (confirmed/declined/pending Count + eigener RSVP-Status)

## 5. Backend — Session-Detail und RSVP

- [x] 5.1 `GET /api/training-sessions/{id}`: Session-Detail inkl. vollständiger Response-Liste; Privacy-Enforcement: `reason` nur sichtbar für Trainer/Admin, eigenen Spieler (`member.user_id == claims.UserID`), oder Elternteil (JOIN auf `family_links`)
- [x] 5.2 `POST /api/training-sessions/{id}/respond`: RSVP abgeben oder updaten (Upsert auf `UNIQUE(training_id, member_id)`); Spieler: `member_id` aus eigenem Account ableiten; Elternteil: `member_id` aus Request-Body, Autorisierung via `family_links` prüfen

## 6. Backend — Anwesenheit (Ebene 2)

- [x] 6.1 `POST /api/training-sessions/{id}/attendances`: Bulk-Upsert für Anwesenheiten; Session-Datum muss `<= today` sein (HTTP 422 sonst); nur Trainer/Admin des Teams
- [x] 6.2 `GET /api/training-sessions/{id}/attendances`: Anwesenheitsliste mit JOIN auf `training_responses` für kombinierte Ansicht (rsvp_status + present); nur Trainer/Admin

## 7. Frontend — Trainings-Seiten

- [x] 7.1 `web/src/pages/TrainingsPage.tsx`: Liste der kommenden Trainingssessions des eigenen Teams; pro Session: Datum, Zeit, Ort, Response-Summary-Badge (12 ✓ 4 ✗ 2 ?), eigener RSVP-Status mit Inline-Buttons (Zusagen / Absagen / Vielleicht); abgesagte Sessions grau/durchgestrichen
- [x] 7.2 `web/src/pages/TrainingsDetailPage.tsx`: Session-Detail; oben: Info-Block (Datum, Zeit, Ort, Note); Mitte: RSVP-Buttons für eigenen Status; unten: Teilnehmerliste (Name + Status, Begründung wo sichtbar); für Trainer zusätzlich: Anwesenheits-Erfassungs-Sektion (Checkbox pro Mitglied) — nur wenn `session.date <= heute`
- [x] 7.3 `web/src/pages/AdminTrainingsPage.tsx`: Trainer/Admin-Verwaltungsseite; Tab/Abschnitt „Serien": Liste aller Serien mit Bearbeiten/Löschen-Actions; Tab/Abschnitt „Einzeltermine": Standalone-Sessions; Formular zum Anlegen neuer Serie (Wochentag-Auswahl als Dropdown, Zeiten, Ort, Gültigkeitszeitraum); Formular für Einzeltermin

## 8. Frontend — Kalender-Integration

- [x] 8.1 `KalenderPage.tsx` erweitern: zweiten API-Call `GET /api/training-sessions?from=…&to=…` parallel zu `GET /api/games`; Trainings in den bestehenden Kalender-Render einmischen mit anderem Icon (Lucide `Dumbbell` o.ä.) und Response-Counter statt Dienste-Ampel
- [x] 8.2 Klick auf Training im Kalender navigiert zu `/trainings/{id}` (TrainingsDetailPage)

## 9. Frontend — Navigation und Routing

- [x] 9.1 `App.tsx`: Routen `/trainings`, `/trainings/:id`, `/admin/trainings` anlegen
- [x] 9.2 `AppShell.tsx`: Nav-Eintrag „Trainings" für Rollen spieler, elternteil, trainer, admin; Nav-Eintrag „Trainings verwalten" für trainer, admin
