# test-duties-gaps Specification

## Purpose

Diese Spezifikation beschreibt die Capability `test-duties-gaps`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Diensterfüllung und Geldersatz
Das System SHALL einen Trainer in die Lage versetzen, eine Dienstzuweisung als erfüllt zu markieren oder einen Geldersatz zu vermerken. `Fulfill` aktualisiert dabei NICHT `duty_accounts.ist` (Invariante: ist wird separat berechnet).

#### Scenario: Dienst als erfüllt markieren
- **WHEN** Trainer POST /api/duty-assignments/{id}/fulfill
- **THEN** HTTP 204, `duty_assignments.status='fulfilled'`, `fulfilled_at` gesetzt, `duty_accounts.ist` unverändert

#### Scenario: Geldersatz vermerken
- **WHEN** Trainer POST /api/duty-assignments/{id}/cash-substitute mit `{ amount: 15.0 }`
- **THEN** HTTP 204, `duty_assignments.status='cash_substitute'`, `cash_amount=15.0`

### Requirement: Zuweisungen eines Slots auflisten
Das System SHALL einem Trainer die Möglichkeit geben, alle Zuweisungen eines Dienst-Slots einzusehen.

#### Scenario: Assignments eines Slots lesen
- **WHEN** GET /api/duty-slots/{id}/assignments für Slot mit 2 Assignments
- **THEN** HTTP 200, Liste mit 2 Einträgen, jeder mit user_name, status, cash_amount
