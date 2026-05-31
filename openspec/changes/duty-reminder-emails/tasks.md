## 1. Datenbank-Migration

- [ ] 1.1 Migration `003_duty_reminder.up.sql` anlegen: `ALTER TABLE users ADD COLUMN duty_reminder_days INT NULL DEFAULT NULL`
- [ ] 1.2 Migration `003_duty_reminder.up.sql` ergänzen: `CREATE TABLE duty_reminder_log (user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, event_date DATE NOT NULL, sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (user_id, event_date))`
- [ ] 1.3 Migration `003_duty_reminder.down.sql` anlegen (DROP TABLE duty_reminder_log, ALTER TABLE users DROP COLUMN duty_reminder_days — SQLite: neue Tabelle ohne Spalte + INSERT)

## 2. Scheduler-Job: sendDutyReminders

- [ ] 2.1 In `internal/scheduler/scheduler.go` den Mailer als Dependency einführen (`type Scheduler struct{ db *sql.DB; mailer Mailer }`) und `New()` entsprechend anpassen
- [ ] 2.2 `sendDutyReminders()`-Methode anlegen: offene Slots für `today + 2 Tage` laden (`slots_filled < slots_total`)
- [ ] 2.3 Eligible-User-Query für `target_role = 'spieler'` implementieren (via `members → team_memberships`, aktive Saison, NOT EXISTS in duty_assignments, duty_reminder_days IS NOT NULL)
- [ ] 2.4 Eligible-User-Query für `target_role = 'elternteil'` implementieren (via `family_links → members → team_memberships`)
- [ ] 2.5 Eligible-User-Query für `target_role = 'trainer'` implementieren (via `team_trainers`)
- [ ] 2.6 Fallback: wenn `duty_slot.team_id IS NULL`, kein Team-Filter anwenden
- [ ] 2.7 User→Slots-Map aufbauen und Deduplizierung gegen `duty_reminder_log` prüfen (NOT EXISTS)
- [ ] 2.8 `sendDutyReminders()` in `Run()` aufrufen

## 3. Mail-Template

- [ ] 3.1 Plain-Text-Mail-Template als Funktion in `internal/scheduler` oder `internal/mailer` erstellen: Event-Name, Datum, Uhrzeit, Diensttyp, Rollenbeschreibung, offene Plätze, Link zu `/duty-board`
- [ ] 3.2 Mail-Versand im Scheduler-Job implementieren: pro User `mailer.Send()` aufrufen
- [ ] 3.3 Nach erfolgreichem Versand Eintrag in `duty_reminder_log` schreiben (INSERT OR IGNORE)

## 4. API-Endpoint: Reminder-Preference

- [ ] 4.1 `PUT /api/profile/reminder-preference` Handler in `internal/auth` oder `internal/config` anlegen: akzeptiert `{ "duty_reminder_days": 2 }` oder `{ "duty_reminder_days": null }`, Validierung (nur `2` oder `null`)
- [ ] 4.2 Route in `cmd/teamwerk/main.go` unter der Authenticated-Gruppe registrieren
- [ ] 4.3 `GET /api/profile/me` Response um `duty_reminder_days` ergänzen

## 5. Frontend: Profil-Toggle

- [ ] 5.1 In `web/src/pages/ProfilePage.tsx` (oder entsprechende Profil-Seite) Abschnitt "Benachrichtigungen" ergänzen
- [ ] 5.2 Toggle-Komponente anlegen: "Erinnerungsmail 2 Tage vor Event" / "Nie" — brand-Styling
- [ ] 5.3 Initialen Zustand aus `GET /api/profile/me` laden (`duty_reminder_days`)
- [ ] 5.4 Bei Toggle-Änderung `PUT /api/profile/reminder-preference` aufrufen und visuelles Feedback (Erfolg/Fehler) anzeigen

## 6. Scheduler-Binary anpassen

- [ ] 6.1 In `cmd/teamwerk/main.go` im `scheduler:run`-Zweig den Mailer initialisieren und an `scheduler.New()` übergeben
