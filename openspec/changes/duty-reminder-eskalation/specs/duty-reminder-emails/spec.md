## MODIFIED Requirements

### Requirement: Automatische Erinnerungsmail für offene Duty-Slots

Das System SHALL an vier wiederkehrenden Zeitpunkten prüfen — **7, 3, 2 und 1 Tag(e)** vor dem
Event-Datum —, ob an `today + N Tagen` Duty-Slots existieren, die noch nicht vollständig belegt
sind (`slots_filled < slots_total`). Für jeden berechtigten User, der noch keinen Eintrag in
einem dieser Slots hat und Erinnerungen aktiviert hat, SHALL das System für jeden der vier
Zeitpunkte unabhängig eine aggregierte Mail versenden, sofern zu diesem Zeitpunkt noch offene
Slots für den User existieren.

#### Scenario: Mail wird versendet wenn offene Slots 2 Tage vor Event existieren
- **WHEN** der Scheduler läuft und `target_date = today + 2` hat offene Duty-Slots
- **THEN** erhalten alle eligible User (Rolle + Team-Match, nicht eingetragen, Reminder
  aktiviert) genau eine aggregierte Mail mit allen offenen Slots dieses Tages

#### Scenario: Mail wird zusätzlich 7, 3 und 1 Tag(e) vor Event versendet
- **WHEN** der Scheduler läuft und `target_date = today + N` (N ∈ {7, 3, 1}) hat offene
  Duty-Slots
- **THEN** erhalten alle eligible User (Rolle + Team-Match, nicht eingetragen, Reminder
  aktiviert) genau eine aggregierte Mail mit allen offenen Slots dieses Tages

#### Scenario: Derselbe offene Slot löst mehrere Erinnerungen über die Woche aus
- **WHEN** ein Duty-Slot an `target_date` sowohl 7 Tage als auch 3, 2 und 1 Tag(e) vorher noch
  `slots_filled < slots_total` hat
- **THEN** erhält ein weiterhin nicht eingetragener eligible User an jedem dieser vier
  Zeitpunkte erneut eine Erinnerung (Eskalation, kein "einmal erinnert, nie wieder")

#### Scenario: Keine Mail wenn alle Slots belegt sind
- **WHEN** alle Duty-Slots an `target_date` vollständig belegt sind (`slots_filled =
  slots_total`)
- **THEN** werden für diesen Zeitpunkt keine Erinnerungsmails versendet

#### Scenario: Kein Reminder für Zwischenzeitpunkte außerhalb der vier Offsets
- **WHEN** an `today + 5 Tagen` (kein Wert aus {7, 3, 2, 1}) offene Duty-Slots existieren
- **THEN** wird an diesem Tag kein Duty-Reminder für diese Slots verschickt

#### Scenario: Keine Mail wenn kein Event an target_date
- **WHEN** an `target_date` keine Duty-Slots existieren
- **THEN** werden keine Mails versendet

### Requirement: Deduplizierung verhindert Mehrfachversand

Das System SHALL sicherstellen, dass pro User, `event_date` **und Zeit-Offset** maximal eine
Erinnerungsmail versendet wird, auch wenn der Scheduler mehrfach täglich läuft. Ein bereits für
einen Offset versendeter Reminder SHALL den Versand für einen anderen Offset desselben
`event_date` nicht blockieren.

#### Scenario: Kein Mehrfachversand bei wiederholtem Scheduler-Lauf
- **WHEN** der Scheduler für denselben `target_date` und denselben Offset ein zweites Mal läuft
- **THEN** wird für User, die für diese Kombination aus `event_date` und Offset bereits eine
  Mail erhalten haben (Eintrag in `duty_reminder_log`), keine weitere Mail versendet

#### Scenario: Ein anderer Offset für dasselbe event_date wird nicht deduplizierend blockiert
- **WHEN** ein User für `event_date = X` bereits den 7-Tage-Reminder erhalten hat und derselbe
  Slot bei `today + 3` weiterhin offen ist
- **THEN** erhält der User für `event_date = X` auch den 3-Tage-Reminder, weil dieser als
  eigenständige (User, event_date, Offset)-Kombination gilt

#### Scenario: Log-Eintrag wird beim Mailversand erstellt
- **WHEN** eine Erinnerungsmail erfolgreich versendet wurde
- **THEN** wird ein Eintrag in `duty_reminder_log(user_id, event_date, days_before, sent_at)`
  angelegt
