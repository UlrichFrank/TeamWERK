## 1. Berlin-Zeit-Helper extrahieren

- [x] 1.1 `parseDT(date, timeStr string, loc *time.Location) time.Time` aus `internal/calendar/handler.go` in einen wiederverwendbaren Foundation-Helper ziehen (z. B. `internal/timez` o. ä.), inkl. der `len(date)>10`-Normalisierung; Verhalten unverändert lassen.
- [x] 1.2 `internal/calendar` auf den neuen Helper umstellen; bestehende `calendar`-Tests müssen unverändert grün bleiben.
- [x] 1.3 Neues Package in `internal/arch/arch_test.go` als Foundation klassifizieren (keine Domain-Imports); `make test` Architektur-Test grün.

## 2. Spiel-Reminder auf Berlin-Instant + 24h/3h-Slots umstellen

- [x] 2.1 `sendGameReminders` (`internal/scheduler/scheduler.go`) umbauen: pro Spiel `eventAt` via Helper in `Europe/Berlin` bilden, gegen `time.Now().In(berlin)` vergleichen; 24h-Slot (`ref_type=game_reminder_24h`) und 3h-Slot (`ref_type=game_reminder_3h`) je mit Insert-vor-Send-Idempotenz (`RowsAffected==1`).
- [x] 2.2 Sicherstellen: vergangene Spiele (`now > eventAt`) lösen keinen Reminder aus; Spiel <24h bei Anlage löst 24h-Slot beim nächsten Lauf aus.
- [x] 2.3 Push respektiert weiterhin die `games`-Push-Präferenz; Titel/Body/URL unverändert.

## 3. Training-Reminder auf Berlin-Instant + 24h/3h-Slots umstellen

- [x] 3.1 `sendTrainingReminders` analog umbauen (`training_reminder_24h`, `training_reminder_3h`), Berlin-Instant aus `date`+`start_time`, nur Status `active`.
- [x] 3.2 `trainings`-Push-Präferenz respektieren; Titel/Body/URL unverändert.

## 4. Mitfahr-Reminder auf exakt 3h + Berlin-Instant schärfen

- [x] 4.1 `sendCarpoolingReminders` von der String-Konkatenation/2–4h-Fenster auf Berlin-Instant (`games.date`+`time`) + „≤3h"-Schwelle umstellen; `ref_type=carpooling_reminder` (ein Slot, idempotent), nur `status='confirmed'`-Paarungen.
- [x] 4.2 `carpooling`-Push-Präferenz respektieren; Wandzeit-Korrektheit der Abfahrt verifizieren.

## 5. Dienst-Reminder unverändert lassen (Scope-Guard)

- [x] 5.1 `sendDutyReminders` NICHT verändern; durch Review/Test bestätigen, dass die 48h-„offene Dienste"-Logik und ihr `ref_type`/`duty_reminder_log` unangetastet bleiben.

## 6. Tests

- [x] 6.1 Spiel: 24h-Slot feuert genau einmal; erneuter Lauf feuert nicht erneut (Idempotenz über `game_reminder_24h`).
- [x] 6.2 Spiel: 3h-Slot feuert zusätzlich genau einmal (`game_reminder_3h`), unabhängig vom 24h-Slot.
- [x] 6.3 DST-Grenzfall: Test mit Scheduler-Now in UTC und Event 15:00 Berlin im Sommer (CEST +2) → 3h-Slot feuert bei Berlin-Wandzeit 12:00 (nicht +/- Offset). Analog Wintertest (CET +1).
- [x] 6.4 Spiel <24h bei Anlage → 24h-Slot feuert sofort beim nächsten Lauf.
- [x] 6.5 Training: 24h- und 3h-Slot je genau einmal; `cancelled` → kein Reminder.
- [x] 6.6 Mitfahrt: feuert genau einmal im ≤3h-Fenster, wandzeit-korrekt; ohne bestätigte Paarung kein Reminder.
- [x] 6.7 Vergangenes Event löst in keinem Slot einen Reminder aus.

## 7. Verifikation & Abschluss

- [x] 7.1 `make test` (inkl. `-race` + Architektur-Test), `make lint`, `pnpm -C web build/test/lint`, `openspec validate` grün; `/verify-change` durchlaufen.
- [x] 7.2 Deploy-Hinweis prüfen: KEINE DB-Migration nötig (nur neue `ref_type`-Strings in `notification_log`).
