## ADDED Requirements

### Requirement: Slot-Marker `is_custom` für manuelle Edits

Das System SHALL für jeden `duty_slot` ein boolesches Flag `is_custom` führen. Slots mit `is_custom=1` SHALL vom Auto-Regen (Drei-Tage-Fenster nach Game-Mutation) niemals gelöscht oder verändert werden.

#### Scenario: Manueller Slot-Anlage setzt `is_custom=1`

- **WHEN** ein Vorstand/Trainer `POST /api/duty-slots` mit Slot-Daten aufruft
- **THEN** wird der Slot mit `is_custom=1` in `duty_slots` persistiert

#### Scenario: Manueller Slot-Edit setzt `is_custom=1`

- **WHEN** ein Vorstand/Trainer `PUT /api/duty-slots/{id}` mit aktualisierten Slot-Daten aufruft
- **THEN** wird `is_custom=1` gesetzt (auch wenn der Slot zuvor `is_custom=0` war)

#### Scenario: Auto-Regen schont `is_custom=1`-Slot

- **GIVEN** ein Slot mit `is_custom=1` existiert für Event-Datum D, duty_type T, event_time E
- **WHEN** durch eine Game-Mutation am Datum D-1, D oder D+1 ein Auto-Regen für D ausgelöst wird
- **THEN** bleibt der `is_custom=1`-Slot unverändert (kein Delete, kein Update, kein Re-Insert)
- **AND** falls die Auto-Regen-Logik einen template-basierten Slot mit identischem (T, E) erzeugen würde, taucht er nicht in `duty_slots` auf, sondern in `regen_summary.conflicts`

#### Scenario: Auto-Regen darf `is_custom=0`-Slot löschen, auch wenn befüllt

- **GIVEN** ein Slot mit `is_custom=0` und `slots_filled > 0` existiert
- **WHEN** Auto-Regen den Slot wegen `same_day_behavior=skip` oder `adjacent_day_behavior=skip` entfernen muss
- **THEN** werden alle zugehörigen `duty_assignments` per FK-Cascade entfernt
- **AND** die `user_id`-Liste der ehemals zugewiesenen Helfer wird in `regen_summary.notified_users` gesammelt
- **AND** jeder Helfer erhält nach Commit eine Notification der Kategorie `duties` mit Titel „Dienst angepasst"

### Requirement: Helfer-Benachrichtigung bei Auto-Regen-Slot-Änderung

Das System SHALL Helfer benachrichtigen, deren `duty_assignment` durch Auto-Regen entfernt oder zu einer anderen `duty_type`-Variante migriert wurde. Die Notification SHALL über `notify.Send(..., "duties", ...)` versendet werden und kategoriegebunden den Push/Email-Präferenzen des Helfers folgen.

#### Scenario: Slot entfällt durch skip-Regel

- **WHEN** ein Helfer auf einen Slot eingetragen war, der durch Auto-Regen aufgrund `same_day_behavior=skip` oder `adjacent_day_behavior=skip` gelöscht wird
- **THEN** erhält der Helfer eine Notification mit Titel „Dienst angepasst" und Body „Dein Dienst zum {Event-Name} am {Datum} wurde aufgrund einer Spielplanänderung entfernt."
- **AND** Link auf `/dienste`

#### Scenario: Slot-Variante wechselt durch reduce-Regel

- **WHEN** ein Helfer auf einen Slot des duty_type T eingetragen war und Auto-Regen die `same_day_variant_id` oder `adjacent_day_variant_id` aktiv setzt (neuer duty_type T')
- **THEN** wird der alte Slot mit der `duty_assignment` des Helfers gelöscht
- **AND** ein neuer Slot mit duty_type T' wird angelegt (ohne automatische Übernahme des Helfers)
- **AND** der Helfer erhält eine Notification mit Titel „Dienst angepasst" und Body „Dein Dienst zum {Event-Name} am {Datum} wurde zur Variante {T'-Name} geändert. Bitte überprüfe deinen Dienstplan."
