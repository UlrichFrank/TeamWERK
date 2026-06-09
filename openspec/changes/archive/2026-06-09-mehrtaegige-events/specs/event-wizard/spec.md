## ADDED Requirements

### Requirement: Optionales Enddatum für generische Events im Wizard

Im Event-Wizard SHALL für Events vom Typ „Sonstiges Event" (generisch) ein optionales Enddatum-Feld angezeigt werden. Es ermöglicht das Anlegen mehrtägiger Events (z.B. Trainingslager).

#### Scenario: Enddatum-Feld bei generischem Event sichtbar
- **WHEN** der User im Wizard Typ „Sonstiges Event" gewählt hat
- **THEN** erscheint im Detailformular ein optionales Feld „Enddatum" unterhalb des Startdatums

#### Scenario: Enddatum-Feld bei Heimspiel/Auswärtsspiel nicht sichtbar
- **WHEN** der User im Wizard Typ Heimspiel oder Auswärtsspiel gewählt hat
- **THEN** ist kein Enddatum-Feld sichtbar

#### Scenario: Enddatum leer lassen erzeugt eintägiges Event
- **WHEN** der User das Enddatum-Feld leer lässt und das Formular abschickt
- **THEN** wird das Event ohne `end_date` angelegt (eintägig)

#### Scenario: Enddatum vor Startdatum wird abgelehnt
- **WHEN** der User ein Enddatum eingibt, das vor dem Startdatum liegt
- **THEN** zeigt das Formular eine Validierungsfehlermeldung und verhindert das Absenden
