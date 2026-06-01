## ADDED Requirements

### Requirement: Automatische Erinnerungsmail für offene Duty-Slots

Das System SHALL täglich prüfen, ob an `today + 2 Tagen` Duty-Slots existieren, die noch nicht vollständig belegt sind (`slots_filled < slots_total`). Für jeden berechtigten User, der noch keinen Eintrag in einem dieser Slots hat und Erinnerungen aktiviert hat, SHALL das System eine aggregierte Mail versenden.

#### Scenario: Mail wird versendet wenn offene Slots 2 Tage vor Event existieren
- **WHEN** der Scheduler läuft und `target_date = today + 2` hat offene Duty-Slots
- **THEN** erhalten alle eligible User (Rolle + Team-Match, nicht eingetragen, Reminder aktiviert) genau eine aggregierte Mail mit allen offenen Slots dieses Tages

#### Scenario: Keine Mail wenn alle Slots belegt sind
- **WHEN** alle Duty-Slots an `target_date` vollständig belegt sind (`slots_filled = slots_total`)
- **THEN** werden keine Erinnerungsmails versendet

#### Scenario: Keine Mail wenn kein Event an target_date
- **WHEN** an `target_date` keine Duty-Slots existieren
- **THEN** werden keine Mails versendet

### Requirement: Rollenbasierte Empfängerbestimmung

Das System SHALL Empfänger anhand von `duty_type.target_role` und `duty_slot.team_id` bestimmen.

#### Scenario: Spieler-Empfänger via team_memberships
- **WHEN** `target_role = 'spieler'` und `duty_slot.team_id = X`
- **THEN** erhalten alle User mit `role = 'spieler'` die über `members → team_memberships` (aktive Saison) dem Team X angehören eine Erinnerung, sofern sie den Slot noch nicht belegt haben

#### Scenario: Elternteil-Empfänger via family_links (indirekt)
- **WHEN** `target_role = 'elternteil'` und `duty_slot.team_id = X`
- **THEN** erhalten alle User mit `role = 'elternteil'` deren Kind über `family_links → members → team_memberships` (aktive Saison) dem Team X angehört eine Erinnerung, sofern sie den Slot noch nicht belegt haben

#### Scenario: Trainer-Empfänger via team_trainers
- **WHEN** `target_role = 'trainer'` und `duty_slot.team_id = X`
- **THEN** erhalten alle User mit `role = 'trainer'` die über `team_trainers` dem Team X zugeordnet sind eine Erinnerung, sofern sie den Slot noch nicht belegt haben

#### Scenario: Vereinsweite Empfänger bei fehlendem team_id
- **WHEN** `duty_slot.team_id IS NULL`
- **THEN** werden alle User mit der passenden Rolle (unabhängig von Team-Zugehörigkeit) als Empfänger betrachtet

### Requirement: Aggregierte Mail pro User und Tag

Das System SHALL pro User und `target_date` genau eine Mail versenden, die alle infrage kommenden offenen Slots zusammenfasst.

#### Scenario: Mehrere offene Slots an einem Tag → eine Mail
- **WHEN** ein User für 3 verschiedene offene Slots an `target_date` berechtigt ist
- **THEN** erhält er genau eine Mail mit allen 3 Slots (nicht 3 separate Mails)

#### Scenario: Mail-Inhalt enthält alle relevanten Slot-Informationen
- **WHEN** eine Erinnerungsmail versendet wird
- **THEN** enthält die Mail für jeden Slot: Event-Name, Datum, Uhrzeit, Diensttyp, Rollenbeschreibung, Anzahl offener Plätze, sowie einen Link zur Duty-Board-Seite

### Requirement: Deduplizierung verhindert Mehrfachversand

Das System SHALL sicherstellen, dass pro User und `event_date` maximal eine Erinnerungsmail versendet wird, auch wenn der Scheduler mehrfach täglich läuft.

#### Scenario: Kein Mehrfachversand bei wiederholtem Scheduler-Lauf
- **WHEN** der Scheduler für denselben `target_date` ein zweites Mal läuft
- **THEN** wird für User, die bereits eine Mail erhalten haben (Eintrag in `duty_reminder_log`), keine weitere Mail versendet

#### Scenario: Log-Eintrag wird beim Mailversand erstellt
- **WHEN** eine Erinnerungsmail erfolgreich versendet wurde
- **THEN** wird ein Eintrag in `duty_reminder_log(user_id, event_date, sent_at)` angelegt
