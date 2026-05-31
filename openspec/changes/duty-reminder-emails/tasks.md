## 1. Datenbank-Migration

- [x] 1.1 Migration `003_duty_reminder.up.sql` anlegen: `ALTER TABLE users ADD COLUMN duty_reminder_days INT NULL DEFAULT NULL`
- [x] 1.2 Migration `003_duty_reminder.up.sql` ergänzen: `CREATE TABLE duty_reminder_log (user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, event_date DATE NOT NULL, sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (user_id, event_date))`
- [x] 1.3 Migration `003_duty_reminder.down.sql` anlegen (DROP TABLE duty_reminder_log, ALTER TABLE users DROP COLUMN duty_reminder_days — SQLite: neue Tabelle ohne Spalte + INSERT)

## 2. Scheduler-Job: sendDutyReminders

- [x] 2.1 In `internal/scheduler/scheduler.go` den Mailer als Dependency einführen (`type Scheduler struct{ db *sql.DB; mailer Mailer }`) und `New()` entsprechend anpassen
- [x] 2.2 `sendDutyReminders()`-Methode anlegen: offene Slots für `today + 2 Tage` laden (`slots_filled < slots_total`)
- [x] 2.3 Eligible-User-Query für `target_role = 'spieler'` implementieren (via `members → team_memberships`, aktive Saison, NOT EXISTS in duty_assignments, duty_reminder_days IS NOT NULL)
- [x] 2.4 Eligible-User-Query für `target_role = 'elternteil'` implementieren (via `family_links → members → team_memberships`)
- [x] 2.5 Eligible-User-Query für `target_role = 'trainer'` implementieren (via `team_trainers`)
- [x] 2.6 Fallback: wenn `duty_slot.team_id IS NULL`, kein Team-Filter anwenden
- [x] 2.7 User→Slots-Map aufbauen und Deduplizierung gegen `duty_reminder_log` prüfen (NOT EXISTS)
- [x] 2.8 `sendDutyReminders()` in `Run()` aufrufen

## 3. Mail-Template

- [x] 3.1 Plain-Text-Mail-Template als Funktion in `internal/scheduler` oder `internal/mailer` erstellen: Event-Name, Datum, Uhrzeit, Diensttyp, Rollenbeschreibung, offene Plätze, Link zu `/duty-board`
- [x] 3.2 Mail-Versand im Scheduler-Job implementieren: pro User `mailer.Send()` aufrufen
- [x] 3.3 Nach erfolgreichem Versand Eintrag in `duty_reminder_log` schreiben (INSERT OR IGNORE)

## 4. API-Endpoint: Reminder-Preference

- [x] 4.1 `PUT /api/profile/reminder-preference` Handler in `internal/auth` oder `internal/config` anlegen: akzeptiert `{ "duty_reminder_days": 2 }` oder `{ "duty_reminder_days": null }`, Validierung (nur `2` oder `null`)
- [x] 4.2 Route in `cmd/teamwerk/main.go` unter der Authenticated-Gruppe registrieren
- [x] 4.3 `GET /api/profile/me` Response um `duty_reminder_days` ergänzen

## 5. Frontend: Profil-Toggle

- [x] 5.1 In `web/src/pages/ProfilePage.tsx` (oder entsprechende Profil-Seite) Abschnitt "Benachrichtigungen" ergänzen
- [x] 5.2 Toggle-Komponente anlegen: "Erinnerungsmail 2 Tage vor Event" / "Nie" — brand-Styling
- [x] 5.3 Initialen Zustand aus `GET /api/profile/me` laden (`duty_reminder_days`)
- [x] 5.4 Bei Toggle-Änderung `PUT /api/profile/reminder-preference` aufrufen und visuelles Feedback (Erfolg/Fehler) anzeigen

## 6. Scheduler-Binary anpassen

- [x] 6.1 In `cmd/teamwerk/main.go` im `scheduler:run`-Zweig den Mailer initialisieren und an `scheduler.New()` übergeben
