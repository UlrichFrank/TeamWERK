## Why

Der Reminder-Scheduler vergleicht `time.Now()` in der **Server-Zeitzone** (VPS läuft UTC) gegen die als **Berlin-Wandzeit** gespeicherten Event-Daten (`games.date`+`time`, `training_sessions.date`+`start_time`). Im Sommer (CEST = UTC+2) feuern Reminder dadurch bis zu **2 Stunden daneben**; an Tagesgrenzen verschiebt sich der „24h"-Reminder sogar um einen ganzen Tag. Zusätzlich sind die heutigen Auslöse-Fenster grob (Spiel/Training „20–28h", Mitfahrt „2–4h") und es gibt nur **einen** Reminder pro Event, obwohl ein zweiter, kurzfristiger Reminder am Event-Tag gewünscht ist.

## What Changes

- **Zeitzonen-Korrektheit:** Der Scheduler bildet den Event-Zeitpunkt explizit in `Europe/Berlin` (`time.LoadLocation` + `time.ParseInLocation`) und vergleicht ihn gegen `time.Now()` im selben Standort. Das vorhandene Pattern `parseDT()` aus `internal/calendar/handler.go` wird dafür in einen wiederverwendbaren Helper extrahiert. Reminder feuern damit wandzeit-korrekt, unabhängig von der Server-Zeitzone und über DST-Wechsel hinweg.
- **Zwei feste Reminder-Slots statt grober Fenster** für Spiele und Trainings: ein **24h-Slot** (Planung) und ein **3h-Slot** (am Event-Tag). „Maximal 24h vorher" ist damit garantiert — nie früher als der 24h-Slot.
- **Fahrgemeinschaften:** das bisherige 2–4h-Fenster wird auf **exakt 3h** geschärft (ein Slot, wie bisher).
- **Idempotenz:** jeder neue Slot erhält einen eigenen `ref_type` in `notification_log` (`game_reminder_24h`, `game_reminder_3h`, `training_reminder_24h`, `training_reminder_3h`); Mitfahrt behält `carpooling_reminder`. Jeder Slot feuert pro Nutzer+Event genau einmal.
- **Bewusst unverändert (out of scope):** Die Dienst-Erinnerung (`sendDutyReminders`, „offene Dienste" 48h/2 Tage vorher) bleibt. Sie ist ein *Handlungs-Nudge* zum Eintragen und braucht lange Vorlaufzeit — eine „max 24h"-Regel würde ihren Zweck zerstören. Ebenso unverändert bleibt die ereignisgesteuerte „Neues Spiel angelegt"-Push (feuert sofort bei Anlage, ist Anlage-Info, kein Reminder).

## Capabilities

### New Capabilities

(keine — die betroffene Capability existiert bereits)

### Modified Capabilities

- `push-reminders`: Reminder werden wandzeit-korrekt in `Europe/Berlin` ausgelöst (neue Zeitzonen-Invariante). Spiele und Trainings erhalten **zwei** Reminder (24h **und** 3h) statt eines „~24h"-Reminders; Fahrgemeinschaften werden auf exakt 3h präzisiert. Die Dienst-Erinnerung (48h) bleibt unverändert.

## Impact

- **Code:** ausschließlich `internal/scheduler/scheduler.go` (Auslöse-Logik + neue `ref_type`-Werte) und ein extrahierter Berlin-Zeit-Helper (gemeinsam mit `internal/calendar`, ohne dessen Verhalten zu ändern). Tests in `internal/scheduler/scheduler_test.go`.
- **Datenbank:** **keine** Migration. Event-Spalten bleiben naive `DATE`+`TEXT` (Berlin-Wandzeit). `notification_log` erhält lediglich neue `ref_type`-String-Werte (keine Schema-Änderung).
- **Frontend:** **keine** Änderung.
- **Verhalten für Nutzer:** Spieler/Trainer/Eltern erhalten künftig zwei Erinnerungen pro Spiel/Training (24h + 3h) statt einer. Erwartete Mehr-Last ist gering (kleiner Verein, minütlicher Scheduler, idempotent).
- **RAM/VPS:** vernachlässigbar — kein zusätzlicher Dienst, nur Stdlib-`time`.
