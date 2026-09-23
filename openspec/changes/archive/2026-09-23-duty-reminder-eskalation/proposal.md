## Why

Der bestehende Dienst-Reminder (`internal/scheduler/scheduler.go`, `sendDutyReminders`) erinnert
potenzielle Dienstleistende genau einmal, exakt 2 Tage vor dem Termin. Bleibt ein Dienst trotzdem
unbesetzt, erfährt niemand außer den (nicht adressierten) Betroffenen davon, bis der Dienst am
Termin selbst fehlt — der Vorstand hat keine Möglichkeit, rechtzeitig manuell nachzusteuern
(Person ansprechen, selbst einspringen, Dienst streichen). Eine einzelne, 2 Tage vor dem Termin
verschickte Erinnerung ist außerdem oft zu spät, um noch jemanden zu finden.

## What Changes

- `sendDutyReminders` prüft künftig an **vier** Zeitpunkten vor jedem Termin — **7, 3, 2 und 1
  Tag(e)** vorher — statt nur an einem. Jeder Zeitpunkt ist unabhängig: bleibt ein Slot über
  mehrere dieser Zeitpunkte hinweg offen, wird derselbe eligible User bei jedem Zeitpunkt erneut
  erinnert (Eskalation über die Woche), nicht nur einmalig.
- Die Idempotenz-Mechanismen (`duty_reminder_log` für E-Mail, `notification_log` für Push) werden
  um den Zeit-Offset erweitert, damit ein bereits verschickter 7-Tage-Reminder den 3-/2-/1-Tage-
  Reminder für denselben `event_date` nicht mehr fälschlich blockiert. **BREAKING** (intern):
  `duty_reminder_log` bekommt eine neue Pflichtspalte `days_before` als Teil des
  Eindeutigkeits-Schlüssels — bestehende Zeilen sind Alt-Reminder ohne Offset-Information.
- **Neu:** Eine zusätzliche, aggregierte Vorstands-Übersicht über weiterhin offene Duty-Slots wird
  an Nutzer mit Vereinsfunktion `vorstand` verschickt — an den zwei kritischeren Zeitpunkten
  **3 und 1 Tag(e)** vor dem Termin, vereinsweit (nicht auf eigene Teams beschränkt), damit noch
  manuell eingegriffen werden kann. Läuft über die bestehende Kategorie `duty_reminders` (kein
  neues Präferenz-Enum) — ein Vorstandsmitglied, das diese Kategorie deaktiviert hat, bekommt
  weder die persönliche Erinnerung (falls selbst eligible) noch die Übersicht.
- Mail-/Push-Texte benennen künftig den jeweiligen Zeithorizont (z. B. "noch 7 Tage" / "morgen").

## Capabilities

### New Capabilities
- `duty-reminder-vorstand-uebersicht`: aggregierte, vereinsweite Vorstands-Benachrichtigung über
  offene Duty-Slots an 3 und 1 Tag(en) vor dem Termin.

### Modified Capabilities
- `duty-reminder-emails`: von einem einzelnen 2-Tage-Zeitpunkt auf vier wiederkehrende Zeitpunkte
  (7/3/2/1 Tage) erweitert; Idempotenz-Schema um den Offset ergänzt.

## Impact

- `internal/scheduler/scheduler.go` (`sendDutyReminders`, `eligibleUsers`, `hashDate`,
  `buildReminderMail`, `openSlot`, `reminderUser`)
- Neue Migration: `duty_reminder_log` um `days_before` erweitern (Spalte + zusammengesetzter
  Unique-Index `(user_id, event_date, days_before)`); ggf. neue Tabelle
  `duty_board_reminder_log` (Name TBD, siehe design.md) für die Vorstands-Idempotenz.
- `internal/scheduler/*_test.go` (neue/angepasste Tests für Multi-Offset und Vorstands-Pfad)
- Keine neuen Routen, kein Frontend-Impact, keine neue `notification_preferences.category`.
