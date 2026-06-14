## MODIFIED Requirements

### Requirement: Trainer kann Anwesenheitsliste einer Session abrufen
Ein Trainer oder Admin SHALL die Anwesenheitsliste einer Session abrufen können, die beide Dimensionen zeigt: RSVP-Status (was angesagt wurde) und tatsächliche Anwesenheit. Jedes Element der Liste SHALL ein `is_extended`-Feld enthalten, das anzeigt, ob das Mitglied zum primären Kader (`false`) oder zum erweiterten Kader (`true`) gehört. Mitglieder, die in beiden Kadern sind, SHALL nur einmal erscheinen und gelten als primäres Kader-Mitglied (`is_extended: false`).

#### Scenario: Anwesenheitsliste abrufen
- **WHEN** ein Trainer GET `/api/training-sessions/{id}/attendances` aufruft
- **THEN** erhält er eine Liste aller Teammitglieder mit jeweils `member_id`, `member_name`, `rsvp_status`, `reason`, `present` und `is_extended` (bool)

#### Scenario: Primärer Kader korrekt markiert
- **WHEN** ein Mitglied via `kader_members` zum Team gehört
- **THEN** hat es `is_extended: false` in der Response

#### Scenario: Erweiterter Kader korrekt markiert
- **WHEN** ein Mitglied nur via `kader_extended_members` zum Team gehört (nicht im primären Kader)
- **THEN** hat es `is_extended: true` in der Response

#### Scenario: Kein Duplikat bei Overlap
- **WHEN** ein Mitglied sowohl im primären als auch im erweiterten Kader ist
- **THEN** erscheint es genau einmal in der Liste mit `is_extended: false`

#### Scenario: Diskrepanz sichtbar
- **WHEN** ein Mitglied `rsvp_status='confirmed'` hat, aber `present=false`
- **THEN** sind beide Werte in der Liste sichtbar, sodass Trainer Zusagen ohne Erscheinen erkennen kann
