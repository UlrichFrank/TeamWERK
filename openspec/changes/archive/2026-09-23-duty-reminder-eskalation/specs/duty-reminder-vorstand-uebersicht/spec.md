## Purpose

Gibt dem Vorstand rechtzeitig vor einem Termin eine vereinsweite, aggregierte Übersicht über
weiterhin unbesetzte Duty-Slots, damit noch manuell eingegriffen werden kann (Person ansprechen,
selbst einspringen, Dienst streichen), bevor er am Termin unbesetzt bleibt.

## ADDED Requirements

### Requirement: Vorstands-Übersicht über offene Duty-Slots

Das System SHALL an zwei Zeitpunkten vor jedem Termin — **3 und 1 Tag(e)** vorher — prüfen, ob
an `today + N Tagen` (N ∈ {3, 1}) Duty-Slots existieren, die noch nicht vollständig belegt sind
(`slots_filled < slots_total`). Für jeden Nutzer mit Vereinsfunktion `vorstand` SHALL das System
eine aggregierte Übersichts-Meldung mit allen offenen Slots dieses Tages versenden, unabhängig
von dessen eigener Team-Zugehörigkeit.

#### Scenario: Vorstand erhält Übersicht bei offenen Slots 3 oder 1 Tag(e) vor Event
- **WHEN** der Scheduler läuft und `target_date = today + N` (N ∈ {3, 1}) hat offene Duty-Slots
- **THEN** erhält jeder Nutzer mit Vereinsfunktion `vorstand` eine aggregierte Meldung mit allen
  offenen Slots dieses Tages, auch wenn ein Slot ein Team betrifft, dem der Vorstands-Nutzer
  selbst nicht angehört

#### Scenario: Kein Übersichts-Versand außerhalb von 3 und 1 Tag(en)
- **WHEN** an `today + 7` oder `today + 2` offene Duty-Slots existieren
- **THEN** wird an diesen Tagen keine Vorstands-Übersicht verschickt (nur die reguläre
  Mitglieder-Erinnerung, siehe `duty-reminder-emails`)

#### Scenario: Keine Übersicht wenn alle Slots belegt sind
- **WHEN** an `target_date` (3 oder 1 Tag vorher) alle Duty-Slots vollständig belegt sind
- **THEN** wird keine Vorstands-Übersicht versendet

#### Scenario: Nur die Vereinsfunktion vorstand ist Empfänger
- **WHEN** ein Nutzer die Vereinsfunktion `trainer`, `sportliche_leitung`, `vorstand_beisitzer`
  oder `kassierer`, aber nicht `vorstand` trägt
- **THEN** erhält dieser Nutzer keine Vorstands-Übersicht über diesen Weg

### Requirement: Vorstands-Übersicht teilt sich die Kategorie duty_reminders

Die Vorstands-Übersicht SHALL über dieselbe Benachrichtigungs-Kategorie `duty_reminders`
laufen wie die persönliche Dienst-Erinnerung (kein eigenes Kategorie-Enum in
`notification_preferences`).

#### Scenario: Deaktivierte Kategorie unterdrückt auch die Übersicht
- **WHEN** ein Vorstands-Nutzer die Kategorie `duty_reminders` in seinen
  `notification_preferences` deaktiviert hat
- **THEN** erhält er weder die persönliche Dienst-Erinnerung (falls selbst eligible) noch die
  Vorstands-Übersicht

#### Scenario: Persönliche Erinnerung und Übersicht sind unabhängig voneinander zustellbar
- **WHEN** ein Vorstands-Nutzer gleichzeitig als `eligibleUser` für einen offenen Slot in
  Frage kommt (z. B. weil er selbst `spieler` ist) und Empfänger der Vorstands-Übersicht ist
- **THEN** kann er an einem Tag beide Meldungen erhalten — sie sind unabhängig voneinander
  idempotent und ersetzen sich nicht gegenseitig

### Requirement: Eigenständige Deduplizierung der Vorstands-Übersicht

Das System SHALL sicherstellen, dass pro Vorstands-Nutzer, `event_date` und Offset (3 oder 1
Tag) maximal eine Übersichts-Meldung versendet wird, auch wenn der Scheduler mehrfach täglich
läuft. Diese Deduplizierung SHALL unabhängig von der Deduplizierung der persönlichen
Dienst-Erinnerung (`duty_reminder_log`) erfolgen.

#### Scenario: Kein Mehrfachversand der Übersicht bei wiederholtem Scheduler-Lauf
- **WHEN** der Scheduler für denselben `target_date` und denselben Offset (3 oder 1 Tag) ein
  zweites Mal läuft
- **THEN** wird für Vorstands-Nutzer, die für diese Kombination aus `event_date` und Offset
  bereits eine Übersichts-Meldung erhalten haben, keine weitere Meldung versendet
