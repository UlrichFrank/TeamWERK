## 1. Migration: calendar_tokens (Option A — keine games-Änderungen)

- [x] 1.1 Migration 045 up: Tabelle `calendar_tokens` anlegen (`id`, `user_id UNIQUE`, `token UNIQUE`, `include_heim`, `include_auswaerts`, `include_training`, `include_generisch`, `include_duty`, `created_at`)
- [x] 1.2 Migration 045 down: `calendar_tokens` droppen
- [x] 1.3 ENTFÄLLT: `games.event_type` bleibt unverändert. Trainings werden aus `training_sessions` gelesen (Option A statt der ursprünglichen CHECK-Erweiterung)
- [x] 1.4 ENTFÄLLT (siehe 1.3)

## 2. Backend: iCal-Package

- [x] 2.1 Package `internal/calendar` anlegen mit `Handler`-Struct (`db *sql.DB`)
- [x] 2.2 iCal-Hilfsfunktionen implementieren: `escapeText()`, `foldLine()`, `formatDT()` (Europe/Berlin via `time.LoadLocation`)
- [x] 2.3 `renderICal(events []calEvent) string` implementieren — VCALENDAR-Rahmen, VEVENT pro Event, CRLF-Zeilenenden, Line-Folding
- [x] 2.4 Query für Spiele des Users: JOIN `game_teams` → `kader_members` → `members` WHERE `user_id = ?` AND `event_type IN (aktivierte Typen)`, inkl. Venue-Join
- [x] 2.5 Query für Dienste des Users: JOIN `duty_assignments` WHERE `user_id = ?` AND `status IN ('assigned','fulfilled')`, inkl. `duty_types.name`
- [x] 2.5a Query für Trainings des Users: JOIN `training_sessions` → `kader_members` → `members` WHERE `user_id = ?` AND `status='active'` (Option A — neue Quelle für `include_training`)
- [x] 2.6 `GET /api/calendar/feed/{token}.ics` implementieren: Token aus DB lesen, User-ID + Toggles auslesen, Events abfragen, iCal rendern, Content-Type setzen
- [x] 2.7 `GET /api/calendar/token` implementieren: Token + Einstellungen für eingeloggten User zurückgeben (404 wenn keines)
- [x] 2.8 `POST /api/calendar/token` implementieren: Upsert — existiert Token → Einstellungen updaten; sonst UUID v4 via `crypto/rand` generieren + INSERT
- [x] 2.9 `DELETE /api/calendar/token` implementieren: Token des eingeloggten Users löschen, HTTP 204

## 3. Backend: Router + event_type-Erweiterung

- [x] 3.1 `internal/calendar.Handler` in `app.Handlers` eintragen und in `main.go` via `NewHandler(db)` initialisieren
- [x] 3.2 Public-Route `GET /api/calendar/feed/{token}.ics` in `router.go` registrieren (außerhalb des Auth-Middleware-Blocks)
- [x] 3.3 Auth-Routen `GET/POST/DELETE /api/calendar/token` in `router.go` im Authenticated-Block registrieren
- [x] 3.4 ENTFÄLLT (Option A): `games.event_type` bleibt heim/auswärts/generisch
- [x] 3.5 ENTFÄLLT (Option A): kein `training`-Branch in `runAutoRegen` nötig

## 4. Backend: Tests

- [x] 4.1 Test `GET /api/calendar/feed/{token}.ics` mit gültigem Token — HTTP 200, Content-Type, VCALENDAR-Rahmen
- [x] 4.2 Test Feed mit ungültigem Token — HTTP 404
- [x] 4.3 Test Feed-Filterung: include_training=false → kein training-VEVENT im Output (Quelle: training_sessions)
- [x] 4.3a Test Feed-Filterung: include_training=true → training-VEVENT mit Team-Name im Summary erscheint
- [x] 4.4 Test Feed-Filterung: include_duty=false → kein duty-VEVENT im Output
- [x] 4.5 Test `POST /api/calendar/token` (Neuanlage) — HTTP 200, Token in Response
- [x] 4.6 Test `POST /api/calendar/token` (Update) — Token bleibt gleich, Einstellungen ändern sich
- [x] 4.7 Test `DELETE /api/calendar/token` — HTTP 204, anschließend Feed-Endpunkt liefert 404
- [x] 4.8 Test `GET /api/calendar/token` ohne existierendes Token — HTTP 404
- [x] 4.9 ENTFÄLLT (Option A): kein `event_type=training` mehr in `games`
- [x] 4.10 Test `POST /api/games` mit ungültigem `event_type` — HTTP 400

## 5. Frontend: Spielplan-Formular

- [x] 5.1 ENTFÄLLT (Option A): Trainings werden weiterhin über den bestehenden Wizard-Button „Einzeltraining"/„Trainingsserie" angelegt (→ `training_sessions`)
- [x] 5.2 ENTFÄLLT (Option A): `event-type-colors` deckt Heim/Auswärts/Generisch bereits ab; Trainings haben eigene visuelle Repräsentation

## 6. Frontend: Kalender-Abo in Einstellungen

- [x] 6.1 Neuer Tab „Kalender-Abo" in `ProfilePage.tsx` (via `ProfileKalenderTab`-Komponente)
- [x] 6.2 `GET /api/calendar/token` beim Laden abfragen — Token und aktuelle Toggle-Werte anzeigen; bei 404 „Noch kein Kalender-Link" anzeigen
- [x] 6.3 5 Toggle-Buttons (statt Checkboxen, konsistent mit `ProfileMiscTab`-Stil): „Heim-Spiele", „Auswärts-Spiele", „Trainings", „Sonstige Events", „Dienste" (Standard: alle aktiviert)
- [x] 6.4 „Link aktivieren / Änderungen speichern"-Button ruft `POST /api/calendar/token` auf
- [x] 6.5 Feed-URL anzeigen und mit „Kopieren"-Button (`navigator.clipboard`) versehen sobald Token vorhanden
- [x] 6.6 „Link löschen"-Button ruft `DELETE /api/calendar/token` auf (mit confirm()-Dialog), UI kehrt zu „Noch kein Kalender-Link" zurück
- [x] 6.7 Hinweis-Text: „Abonniere diesen Link in Google Calendar, Apple Kalender oder Outlook. Der Link ist privat — teile ihn nicht."
