## ADDED Requirements

### Requirement: Duty type definition
The system SHALL allow admins to define duty types. A duty type has: name, required hours value (decimal), and optional cash substitute amount (€).

#### Scenario: Admin creates duty type
- **WHEN** an admin submits a duty type with name and hours value
- **THEN** the system stores the duty type and it becomes available for assignment

#### Scenario: Duty type with cash substitute
- **WHEN** a duty type is created with a cash substitute amount
- **THEN** families can optionally pay the amount instead of fulfilling the duty

### Requirement: Duty slot creation
The system SHALL allow admins and trainer to create duty slots attached to an event (e.g., a home game). A slot has: event name, event date, duty type, required role description, and number of persons needed.

#### Scenario: Create duty slot for event
- **WHEN** an admin creates a duty slot with event reference, duty type, and person count
- **THEN** the slot appears in the duty board as open

#### Scenario: Multiple slots per event
- **WHEN** multiple duty slots are created for the same event
- **THEN** each slot is listed independently in the duty board

### Requirement: Duty board (Dienstbörse)
Das System SHALL eine Dienstbörse mit allen Duty-Slots anzeigen. Jeder Slot enthält neben den bisherigen Informationen (event name, date, duty type, vacancies) auch die Liste der eingetragenen Personen mit privacy-gefiltertem Kontaktdaten-Payload.

#### Scenario: View open duties
- **WHEN** any authenticated user opens the duty board
- **THEN** all open slots (unfilled, future event date) are shown with event name, date, duty type, remaining vacancies, and the list of assignees (name + conditionally photo URL, phones, address)

#### Scenario: Claim a duty slot
- **WHEN** a user (or a parent on behalf of their family) claims an open slot
- **THEN** the system records the assignment, decrements the vacancy count, updates the claimant's duty account, and the claimant's name appears in the assignee list

#### Scenario: Slot fully filled
- **WHEN** the last vacancy of a slot is claimed
- **THEN** the slot no longer shows vacancies but the assignee names remain visible

#### Scenario: Cannot claim already-assigned slot
- **WHEN** a user attempts to claim a slot they or their family already hold
- **THEN** the system returns a validation error

#### Scenario: Privacy-gefilterte Assignee-Daten im API-Response
- **WHEN** der `/duty-board`-Endpoint einen Slot mit Assignees zurückgibt
- **THEN** enthält jeder Assignee-Eintrag: `name` (immer), `photo_url` (nur wenn `photo_visible=1`), `phones` (nur wenn `phones_visible=1`, sonst leeres Array), `address` (nur wenn `address_visible=1`, sonst null)

### Requirement: Duty account per family
The system SHALL maintain a duty account per family (user/parent unit) per season, tracking target hours (Soll) and fulfilled hours (Ist).

#### Scenario: Soll configured per season
- **WHEN** an admin sets the seasonal duty target for a duty type
- **THEN** each family's Soll is updated to reflect the target

#### Scenario: Ist updated on duty fulfillment
- **WHEN** an admin or trainer marks a duty slot as fulfilled for a family
- **THEN** the family's Ist balance increases by the duty type's hours value

#### Scenario: Family views own duty account
- **WHEN** an `elternteil` or `spieler` views their duty account
- **THEN** they see Soll, Ist, and the balance (Soll − Ist) for the active season

#### Scenario: Admin views all duty accounts
- **WHEN** an admin views the duty overview
- **THEN** all families with their Soll, Ist, and balance are shown, sortable by balance

### Requirement: Cash substitute recording
The system SHALL allow recording a cash substitute payment as an alternative to fulfilling a duty.

#### Scenario: Record cash substitute
- **WHEN** an admin records a cash substitute payment for a family and duty type
- **THEN** the equivalent hours are credited to the family's Ist balance and the payment amount is logged

### Requirement: Duty account export
The system SHALL allow admins to export all duty accounts as CSV for the season treasurer report.

#### Scenario: Export duty accounts
- **WHEN** an admin triggers the duty account export for the active season
- **THEN** the system returns a CSV with: family name, Soll, Ist, balance, and any cash substitute amounts

## MODIFIED Requirements

### Requirement: Duty account per family
Das Duty-Account-System bleibt unverändert — Ist-Wert, Claim-Logik und Export bleiben identisch. Geändert wird ausschließlich die Berechnung des `soll`-Werts für die Rolle `elternteil` im Dashboard-Endpoint.

**Vorher:** `soll = 5 × COUNT(family_links WHERE parent_user_id = user_id)`

**Nachher:** Dynamische Formel basierend auf Kader-Daten (siehe Capability `dienstkonto-dynamische-soll-formel`). Der in der `duty_accounts`-Tabelle gespeicherte Wert bleibt davon unberührt — der `/api/dashboard`-Endpoint berechnet den Wert live.

#### Scenario: Family views own duty account (updated)
- **WHEN** ein `elternteil` das Dashboard aufruft
- **THEN** sieht er `soll` basierend auf der dynamischen Formel (Kader-Spielanzahl, Templates, Spielerzahl, Elternanzahl)
- **AND** der Erklärtext lautet „Ziel: {soll} Dienste (Saison {name})" ohne Formel-Details
