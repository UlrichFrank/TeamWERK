## 1. Datenbank

- [ ] 1.1 Migration `011_notification_log.up.sql`: Tabelle `notification_log (id INTEGER PK, user_id INTEGER NOT NULL, ref_type TEXT NOT NULL, ref_id INTEGER NOT NULL, sent_at DATETIME DEFAULT CURRENT_TIMESTAMP, UNIQUE(user_id, ref_type, ref_id))`
- [ ] 1.2 Migration `011_notification_log.down.sql`: `DROP TABLE IF EXISTS notification_log`

## 2. Trainings-Notifications

- [ ] 2.1 `internal/trainings/handler.go` — `UpdateSession`-Handler: nach erfolgreichem DB-Write Team-Mitglieder-IDs ermitteln, `go notifications.SendToUsers(...)` mit Titel „Training geändert" und Text „[Datum] [Ort/Zeit geändert]"
- [ ] 2.2 `internal/trainings/handler.go` — Cancel-Handler: analog, Titel „Training abgesagt", Text „[Datum] wurde abgesagt"
- [ ] 2.3 `internal/trainings/handler.go` — `CreateSession`-Handler (Einzeltermin): Notification an Teammitglieder mit Titel „Neuer Trainingstermin", Text „[Datum] [Zeit] in [Ort]"
- [ ] 2.4 `internal/trainings/handler.go` — `CreateSeries`-Handler: Notification an Teammitglieder mit Titel „Neue Trainingsserie", Text „Ab [Datum]: jeden [Wochentag] [Zeit]"
- [ ] 2.5 Hilfsfunktion `getTeamUserIDs(db, teamID int) []int` in `internal/trainings/` (JOIN `team_memberships → members → users` + `users WHERE team_id = teamID`)

## 3. Admin-Notifications

- [ ] 3.1 `internal/auth/handler.go` — `RequestMembership`-Handler: nach DB-Insert alle User mit `role = 'admin'` ermitteln, `go notifications.SendToUsers(...)` mit Titel „Neue Mitgliedsanfrage", Text „[Name] möchte beitreten"
- [ ] 3.2 `internal/members/handler.go` — `CreateChangeDraft`-Handler: Trainer des Teams + alle Admins ermitteln, Notification mit Titel „Datenänderung beantragt", Text „[Name] hat eine Änderung eingereicht"

## 4. Scheduled Duty-Notifications

- [ ] 4.1 `internal/scheduler/` — neuer Job `sendDutyReminders`: Query `duty_assignments JOIN duty_slots WHERE date(event_date) = date('now', '+1 day') AND strftime('%H', 'now') = '18'`; für jede Assignment notification senden und in `notification_log` mit `ref_type = 'duty_reminder'` eintragen (INSERT OR IGNORE für Idempotenz)
- [ ] 4.2 `internal/scheduler/` — neuer Job `sendDutyGapAlerts`: Query `duty_slots WHERE date(event_date) = date('now', '+2 days') AND slots_filled < slots_total`; Trainer/Admin des Teams benachrichtigen mit `ref_type = 'duty_gap'`; INSERT OR IGNORE auf `notification_log`
- [ ] 4.3 Beide Jobs in `scheduler:run`-Subcommand einbinden
