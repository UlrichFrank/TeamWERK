## ADDED Requirements

### Requirement: Spiel-Tile zeigt genau ein Warn-Icon

Im Kalender-Grid SHALL jedes Spiel-Tile höchstens ein AlertTriangle-Icon anzeigen, das alle Warn-Gründe vereint. Das Icon erscheint wenn `note.trim() !== ''` ODER `filled_count < total_count` (bei `slot_count > 0`). Der farbige Duty-Dot (Slot-Füllgrad) entfällt ersatzlos.

#### Scenario: Spiel mit offenen Slots, ohne Note

- **WHEN** ein Spiel `slot_count > 0`, `filled_count < total_count` und `note.trim() === ''` hat
- **THEN** erscheint genau ein AlertTriangle oben rechts im Tile
- **AND** das `title`-Attribut enthält „X offene Dienst-Slots" (X = `total_count - filled_count`)
- **AND** kein Duty-Dot ist sichtbar

#### Scenario: Spiel mit Note, ohne offene Slots

- **WHEN** ein Spiel `note.trim() !== ''` und `filled_count >= total_count` hat
- **THEN** erscheint genau ein AlertTriangle oben rechts im Tile
- **AND** das `title`-Attribut enthält den Hinweistext

#### Scenario: Spiel mit Note und offenen Slots

- **WHEN** ein Spiel `note.trim() !== ''` UND `filled_count < total_count` hat
- **THEN** erscheint genau ein AlertTriangle oben rechts im Tile
- **AND** das `title`-Attribut enthält sowohl den Hinweistext als auch „X offene Dienst-Slots", getrennt durch Zeilenumbruch

#### Scenario: Spiel ohne Warns

- **WHEN** ein Spiel `note.trim() === ''` und `filled_count >= total_count` hat (oder `slot_count === 0`)
- **THEN** ist kein AlertTriangle und kein Duty-Dot sichtbar

#### Scenario: Training-Tile unverändert

- **WHEN** ein Training-Tile gerendert wird
- **THEN** bleibt der `EventNoteIndicator` in der unteren Zeile erhalten
- **AND** kein zusätzliches AlertTriangle aus Slot-Daten erscheint (Trainings haben keine Slots)

---

### Requirement: EventInfoModal zeigt generierte Slot-Info

Im EventInfoModal SHALL unterhalb der manuellen Hinweis-Sektion eine generierte Zeile „X offene Dienst-Slots" erscheinen, wenn das angezeigte Spiel offene Slots hat (`filled_count < total_count` bei `slot_count > 0`). Die Zeile hat keinen eigenen Icon und ist visuell durch Abstand von der Note abgesetzt.

#### Scenario: Modal mit offenen Slots und vorhandener Note

- **WHEN** das Modal ein Spiel mit `note.trim() !== ''` und offenen Slots anzeigt
- **THEN** erscheint zuerst der `EventNoteIndicator` (inline) mit dem Hinweistext
- **AND** darunter abgesetzt die Zeile „X offene Dienst-Slots" in gedimmter Farbe ohne Icon

#### Scenario: Modal mit offenen Slots, ohne Note

- **WHEN** das Modal ein Spiel ohne Note aber mit offenen Slots anzeigt
- **THEN** erscheint die Zeile „X offene Dienst-Slots" in gedimmter Farbe ohne Icon
- **AND** kein `EventNoteIndicator` (Note ist leer)

#### Scenario: Modal ohne offene Slots

- **WHEN** das Modal ein Spiel mit `filled_count >= total_count` (oder `slot_count === 0`) anzeigt
- **THEN** erscheint keine generierte Slot-Zeile

#### Scenario: Graceful Degradation bei fehlenden Slot-Feldern

- **WHEN** das Modal ohne `slot_count`/`filled_count`/`total_count` aufgerufen wird
- **THEN** erscheint keine generierte Slot-Zeile (kein Fehler)
