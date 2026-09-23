## 1. Migration

- [x] 1.1 `internal/db/migrations/065_duty_reminder_offsets.up.sql`: `duty_reminder_log`
  per Tabellen-Rebuild (Muster wie Migration `018`) um `days_before INTEGER NOT NULL DEFAULT 2`
  erweitern, neuer PK `(user_id, event_date, days_before)`, Backfill bestehender Zeilen mit
  `days_before = 2`.
- [x] 1.2 Im selben Up-File: neue Tabelle `duty_board_reminder_log(user_id INTEGER NOT NULL
  REFERENCES users(id) ON DELETE CASCADE, event_date DATE NOT NULL, days_before INTEGER NOT
  NULL, sent_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (user_id, event_date,
  days_before))`.
- [x] 1.3 `065_duty_reminder_offsets.down.sql`: `duty_board_reminder_log` droppen;
  `duty_reminder_log` zurück auf PK `(user_id, event_date)` rebuilden (nächste freie
  Migrationsnummer vor dem Anlegen mit `ls internal/db/migrations/ | tail` verifizieren).

## 2. Scheduler — Mitglieder-Reminder auf vier Offsets erweitern

- [x] 2.1 In `internal/scheduler/scheduler.go`: Konstante `dutyReminderOffsets = []int{7, 3, 2,
  1}` einführen.
- [x] 2.2 `sendDutyReminders` auf eine Schleife über `dutyReminderOffsets` umstellen; pro
  Iteration `targetDate := time.Now().AddDate(0, 0, offset)`, bestehende Slot-Query und
  `eligibleUsers`-Logik unverändert wiederverwenden.
- [x] 2.3 E-Mail-Idempotenz: `duty_reminder_log`-Query/-Insert um `days_before = offset`
  erweitern (`SELECT 1 FROM duty_reminder_log WHERE user_id=? AND event_date=? AND
  days_before=?`, `INSERT OR IGNORE … (user_id, event_date, days_before)`).
- [x] 2.4 Push-Idempotenz: `ref_type` von `"duty_reminder"` auf
  `fmt.Sprintf("duty_reminder_%dd", offset)` ändern (kein Schema-Zusatz, `ref_id` bleibt
  `hashDate(targetDate)`).
- [x] 2.5 Neue Helper-Funktion `formatOffsetLabel(days int) string` (`1` → `"morgen"`, sonst
  `"noch N Tage"`); Mail-Betreff und Push-Body in `buildReminderMail`/`sendDutyReminders`
  entsprechend um den Zeithorizont ergänzen.

## 3. Scheduler — Vorstands-Übersicht

- [x] 3.1 Konstante `boardOverviewOffsets = []int{3, 1}`.
- [x] 3.2 Im selben Offset-Loop aus Task 2.2: wenn `offset` in `boardOverviewOffsets` und
  `slots` (bereits geladene offene Slots dieses Tages) nicht leer ist, Vorstands-Empfänger
  laden (`member_club_functions.function = 'vorstand'`, kein Team-Filter, kein `notAssigned`-
  Filter — Vorstand bekommt die Übersicht unabhängig von eigener Zuständigkeit/Eintragung).
- [x] 3.3 Neue Funktion `buildBoardOverviewMail(name, date string, slots []openSlot, offset int,
  baseURL string) string`, analog zu `buildReminderMail`, mit Eingreifen-Framing statt
  Eintragen-Framing.
- [x] 3.4 E-Mail-Versand über `duty_board_reminder_log` (Opt-in wie bisher via
  `push.HasEmailEnabled(s.db, uid, "duty_reminders")`).
- [x] 3.5 Push-Versand über `notify.Send(..., "duty_reminders", ...)` mit `ref_type =
  fmt.Sprintf("duty_board_vorstand_%dd", offset)` in `notification_log` für die Idempotenz.

## 4. Tests

- [x] 4.1 Test: bei offenen Slots an `today+7` erhält ein eligible User genau eine
  Erinnerung; bei vollständig belegten Slots keine (pro Offset wiederholen oder parametrisiert
  über `dutyReminderOffsets` testen).
- [x] 4.2 Test: ein Slot bleibt über mehrere Offsets hinweg offen → derselbe eligible User
  erhält bei jedem Offset (7/3/2/1) eine eigene Erinnerung (nicht nur einmal insgesamt).
- [x] 4.3 Test: wiederholter Scheduler-Lauf für denselben `(user, event_date, offset)` sendet
  keine zweite Mail/Push (Idempotenz pro Kombination, nicht nur pro Datum).
- [x] 4.4 Test: ein User mit bereits verschicktem 7-Tage-Reminder für `event_date = X` bekommt
  trotzdem den 3-Tage-Reminder für dasselbe `event_date`, wenn der Slot weiterhin offen ist
  (Regressionsschutz gegen den alten reinen Datums-Key).
- [x] 4.5 Test: Vorstands-Übersicht wird nur bei Offset 3 und 1 verschickt, nicht bei 7 oder 2.
- [x] 4.6 Test: Vorstands-Übersicht geht an alle Nutzer mit Funktion `vorstand`, auch für
  Slots eines Teams, dem der Vorstands-Nutzer nicht angehört; Nutzer mit `trainer`/
  `vorstand_beisitzer`/`kassierer` ohne `vorstand` erhalten sie nicht.
- [x] 4.7 Test: Vorstands-Nutzer mit deaktivierter Kategorie `duty_reminders` erhält weder die
  persönliche Erinnerung (falls selbst eligible) noch die Übersicht.
- [x] 4.8 Test: ein Vorstands-Nutzer, der gleichzeitig eligible User für einen offenen Slot
  ist, erhält an einem Offset-Tag sowohl die persönliche Erinnerung als auch die Übersicht
  (beide unabhängig idempotent, keine gegenseitige Unterdrückung).
- [x] 4.9 Test: Migration `065` — Up erzeugt `days_before=2` für Bestandszeilen und die neue
  Tabelle `duty_board_reminder_log`; Down entfernt beides wieder ohne Fehler auf einer DB mit
  Testdaten.

## 5. Doku & Verifikation

- [x] 5.1 `openspec/specs/duty-reminder-emails/spec.md` bleibt unverändert (wird erst beim
  Archivieren der Change gemergt) — keine manuelle Bearbeitung nötig.
- [x] 5.2 `make test` (inkl. `internal/scheduler`-Tests), `golangci-lint`, `openspec validate
  --strict` vor Abschluss grün.
