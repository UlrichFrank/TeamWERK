# Tasks

## 1. Foundation — Textbaustein

- [x] 1.1 `internal/notify/change.go`: `FormatTimeHM(s string) string` — `"18:00:00"` und `"18:00"` ergeben `"18:00"`, leerer/kurzer Wert ergibt den leeren String
- [x] 1.2 `internal/notify/change.go`: `EventWhen(date, clock string) string` → `"am 20.08.2026 um 18:00 Uhr"` (ohne Uhrzeit: `"am 20.08.2026"`; ohne Datum: leerer String) — trägt wie `CancellationBody`s `when` seine eigene Präposition
- [x] 1.3 `internal/notify/change.go`: `EventMoment(date, clock string) string` → `"20.08.2026, 18:00 Uhr"` — präpositionslose Form für die Vorher-Klammer
- [x] 1.4 `internal/notify/change.go`: `ChangeBody(subject, when, previously, actor string) string` → `"{subject} {when} wurde geändert[ (vorher {previously})]. Geändert von {actor}."`; leeres `previously` lässt die Klammer weg, leerer `actor` fällt auf `fallbackActor` zurück, leeres `subject` auf `fallbackSubject`
- [x] 1.5 Tests `internal/notify/change_test.go`: mit/ohne Vorher-Klammer, ohne Aktor, ohne Subjekt, ohne Uhrzeit, Zeitraum-Phrase (`"ab … bis …"`), ISO-Timestamp als Datum, und: der Aktor-Name fällt **nie** auf die E-Mail zurück

## 2. Spiele und generische Termine

- [x] 2.1 `internal/games/handler.go` — `UpdateGame`: Pre-Update-SELECT um `time` erweitern (`oldTime`), damit die Verschiebung erkennbar ist
- [x] 2.2 `internal/games/handler.go`: `changeTitleFor(requested, stored string) string` — `"Termin geändert"` bei `generisch`, sonst `"Spielinfo geändert"`; effektiver Typ = `req.EventType` falls gültig, sonst `storedEventType`
- [x] 2.3 `internal/games/handler.go` — `UpdateGame`: Body über `notify.ChangeBody(eventName, notify.EventWhen(req.Date, req.Time), previously, notify.ActorName(h.db, claims.UserID))`; `previously` nur bei abweichendem Datum **oder** abweichender Uhrzeit; Terminname `req.Opponent`, leer → `"Termin"`
- [x] 2.4 Titel und Body identisch an das **entfernte** Team schicken (zweiter `notify.Send` bei Team-Umhängung)
- [x] 2.5 Test: PUT ohne Zeitänderung → genau eine Meldung „Spielinfo geändert", Body enthält Gegner, `14.09.2026`, `18:00`, `Tim Meier`, **keine** Vorher-Klammer, `url` = `/termine?focus=game-<id>`
- [x] 2.6 Test: PUT mit neuem Datum **und** neuer Uhrzeit → Body enthält beide Zeitpunkte (neu und alt)
- [x] 2.7 Test: PUT auf `event_type='generisch'` → Titel „Termin geändert", nicht „Spielinfo geändert"
- [x] 2.8 Test: Aktor ohne hinterlegten Namen → generische Formulierung, keine E-Mail im Body
- [x] 2.9 Test (Fehlerfall): 403 bei fremdem Team und 404 bei unbekannter ID senden **keine** Meldung

## 3. Trainingseinheiten

- [x] 3.1 `internal/trainings/handler.go` — `UpdateSession`: Pre-Read um `date` und `start_time` erweitern (`prevDate`, `prevStart`), Datum als `date(date)` scannen (SQLite-DATE-Gotcha)
- [x] 3.2 `internal/trainings/handler.go` — `UpdateSession`: Änderungs-Body über `notify.ChangeBody(sessionSubject(req.Title), notify.EventWhen(req.Date, req.StartTime), previously, actor)`; Absage-Zweig (`prevStatus != status && status == "cancelled"`) unverändert
- [x] 3.3 Test: PUT ohne Zeitänderung → „Training geändert", Body enthält Titel, Datum, Startzeit, Aktor, keine Vorher-Klammer, Link `/termine?focus=training-<id>`
- [x] 3.4 Test: PUT mit neuem Datum/neuer Startzeit → Vorher-Klammer mit dem alten Zeitpunkt
- [x] 3.5 Regression zu `absage-benachrichtigung`: `TestUpdateSession_ActiveZuCancelled_SendetAbsage` sowie die beiden Nicht-Wechsel-Fälle bleiben grün

## 4. Trainingsserien

- [x] 4.1 `internal/trainings/handler.go` — `UpdateSeries`: Pre-Read um `day_of_week` und `start_time` erweitern
- [x] 4.2 `internal/trainings/handler.go`: `weekdayAdverb(dayOfWeek int) string` → `"montags"`…`"sonntags"` (Schema 0=Montag wie `generateSessionDates`), außerhalb 0–6 leerer String
- [x] 4.3 `internal/trainings/handler.go` — `UpdateSeries`: nach `tx.Commit()` und `broadcastTeam` eine Meldung „Trainingsserie geändert" an `teamMembersAndParents(teamID)`, Kategorie `trainings`, `when` = `seriesPeriod(genFrom, req.ValidUntil)`, `previously` = alter Rhythmus nur bei geändertem Wochentag/Startzeit, `url` = `/termine`
- [x] 4.4 Test: PUT auf eine Serie → genau eine Meldung „Trainingsserie geändert" mit Seriennamen, Zeitraum, Aktor und `url` = `/termine`
- [x] 4.5 Test: geänderter Wochentag → Body enthält den alten Rhythmus („montags 18:00 Uhr")
- [x] 4.6 Test (Fehlerfall): 403 bei fremdem Team sendet keine Meldung

## 5. Neuanlage eines Spiels/Termins

- [x] 5.1 `internal/notify/change.go`: `CreationBody(subject, when, actor string) string` → `"{subject} {when}. Angelegt von {actor}."` — kein `previously`, ein neuer Termin hat keine Vergangenheit
- [x] 5.2 `internal/games/handler.go`: `creationTitle(eventType string) string` — `"Neuer Termin"` bei `generisch`, sonst `"Neues Spiel"`
- [x] 5.3 `internal/games/handler.go` — `CreateGame`: Body über `notify.CreationBody(eventName, notify.EventWhen(req.Date, req.Time), notify.ActorName(h.db, claims.UserID))` statt `eventName+" am "+req.Date` (rohes ISO-Datum)
- [x] 5.4 Test: POST → Body enthält Gegner, `05.10.2026`, `17:15`, Aktor und **kein** `2026-10-05`
- [x] 5.5 Test: POST mit `event_type='generisch'` → Titel „Neuer Termin"
- [x] 5.6 Test: Aktor ohne hinterlegten Namen → keine E-Mail im Body

## 6. Abschluss

- [x] 6.1 `make test` (inkl. Architektur- und Broadcast-Gate) und `make lint` grün
- [x] 6.2 `openspec validate aenderungs-benachrichtigung --strict`
- [ ] 6.3 Change archivieren
