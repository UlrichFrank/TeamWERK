# Spec Delta

## MODIFIED Requirements

### Requirement: Diensterfüllung und Geldersatz
Das System SHALL einen Trainer in die Lage versetzen, eine Dienstzuweisung als erfüllt zu markieren oder einen Geldersatz zu vermerken.

#### Scenario: Dienst als erfüllt markieren
- **WHEN** Trainer POST /api/duty-assignments/{id}/fulfill
- **THEN** HTTP 204, `duty_assignments.status='fulfilled'`, `fulfilled_at` gesetzt

#### Scenario: Geldersatz vermerken
- **WHEN** Trainer POST /api/duty-assignments/{id}/cash-substitute mit `{ amount: 15.0 }`
- **THEN** HTTP 204, `duty_assignments.status='cash_substitute'`, `cash_amount=15.0`
