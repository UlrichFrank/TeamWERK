## 1. Datenbank-Migration

- [x] 1.1 Migration `028_notification_preferences.up.sql` erstellen: `notification_preferences`-Tabelle (user_id, category, push_enabled, email_enabled) und `notification_log`-Tabelle (user_id, ref_type, ref_id, sent_at)
- [x] 1.2 Migration `028_notification_preferences.down.sql` erstellen

## 2. Backend: Notification-Präferenzen-Infrastruktur

- [x] 2.1 Hilfsfunktion `notifications.FilterByPushPref(db, userIDs []int, category string) []int` in `internal/notifications/prefs.go` — gibt nur User-IDs zurück, die Push für diese Kategorie nicht deaktiviert haben
- [x] 2.2 API-Handler `GetNotificationPreferences` und `UpdateNotificationPreferences` in `internal/notifications/handler.go` für `GET/PUT /api/profile/notification-preferences`
- [x] 2.3 Routen in `cmd/teamwerk/main.go` eintragen: `GET/PUT /api/profile/notification-preferences` (authenticated)

## 3. Backend: Push-Notifications bei Spiel-Ereignissen

- [x] 3.1 `games/handler.go` — `CreateGame`: Team-Mitglieder + Eltern ermitteln, `FilterByPushPref` anwenden, `go notifications.SendToUsers(...)` mit „Neues Spiel"
- [x] 3.2 `games/handler.go` — `UpdateGame`: Push „Spielinfo geändert" an Team-Mitglieder + Eltern (nach FilterByPushPref)
- [x] 3.3 `games/handler.go` — `DeleteGame`: Push „Spiel abgesagt" an Team-Mitglieder + Eltern (nach FilterByPushPref) — vor dem DELETE ermitteln

## 4. Backend: Push-Notifications bei Trainings-Ereignissen

- [x] 4.1 `trainings/handler.go` — `DeleteSession`: Push „Training abgesagt" an Kader-Mitglieder + Eltern (FilterByPushPref `trainings`)
- [x] 4.2 `trainings/handler.go` — `UpdateSession`: Push „Training geändert" bei Zeit-/Ort-Änderung (FilterByPushPref `trainings`)
- [x] 4.3 `trainings/handler.go` — `DeleteSeries`: Push „Trainingsserie gelöscht" (FilterByPushPref `trainings`)

## 5. Backend: Push-Notifications bei Dienst-Ereignissen

- [x] 5.1 `duties/handler.go` — `CreateDutySlot`: Push „Neuer Dienst verfügbar" an berechtigte User (FilterByPushPref `duties`)
- [x] 5.2 `duties/handler.go` — `DeleteDutySlot`: Push „Dienst abgesagt" an zugeteilte User (FilterByPushPref `duties`) — Assignments vor dem DELETE abfragen

## 6. Backend: Push-Notifications bei Fahrgemeinschaften

- [x] 6.1 `carpooling/handler.go` — Match accepted: Push „Fahrgemeinschaft bestätigt" an anfragenden User (FilterByPushPref `carpooling`)
- [x] 6.2 `carpooling/handler.go` — Match cancelled/rejected: Push „Fahrgemeinschaft abgesagt" an betroffenen User (FilterByPushPref `carpooling`)

## 7. Backend: Push-Notification bei Beitrittsanfrage

- [x] 7.1 `auth/handler.go` (RequestMembership): Push „Neue Beitrittsanfrage" an alle Admins (FilterByPushPref `membership`)

## 8. Backend: Scheduler-Reminders

- [x] 8.1 `scheduler/scheduler.go` — `sendDutyReminders` auf `notification_preferences`-Check umstellen: `duty_reminder_days IS NOT NULL`-Filter durch `notification_log`-Idempotenz + Präferenz-Check ersetzen; Push-Reminder hinzufügen (zusätzlich zur E-Mail wenn `email_enabled=1`)
- [x] 8.2 `scheduler/scheduler.go` — neuer Job `sendGameReminders`: Spiele in 24h ermitteln, Push-Reminder (FilterByPushPref `games`), Idempotenz via `notification_log (ref_type='game', ref_id=game_id)`
- [x] 8.3 `scheduler/scheduler.go` — neuer Job `sendTrainingReminders`: Trainings in 24h, Push-Reminder (FilterByPushPref `trainings`), Idempotenz via `notification_log`
- [x] 8.4 `scheduler/scheduler.go` — neuer Job `sendCarpoolingReminders`: Fahrgemeinschaften in 3h, Push-Reminder (FilterByPushPref `carpooling`), Idempotenz via `notification_log`
- [x] 8.5 `scheduler/scheduler.go` — `Run()` um neue Jobs erweitern

## 9. Frontend: Profil-UI für Notification-Präferenzen

- [x] 9.1 `ProfileMiscTab.tsx` — bestehenden „Dienst-Erinnerungsmail"-Toggle durch vollständige Präferenz-UI ersetzen: Toggle-Rows für Spiele (Push), Trainings (Push), Dienste (Push), Dienst-Erinnerung (Push + E-Mail), Fahrgemeinschaften (Push)
- [x] 9.2 `ProfileMiscTab.tsx` — Laden via `GET /api/profile/notification-preferences` und Speichern via `PUT /api/profile/notification-preferences`
- [x] 9.3 `ProfileMiscTab.tsx` — altes `PUT /api/profile/reminder-preference`-API-Call entfernen
