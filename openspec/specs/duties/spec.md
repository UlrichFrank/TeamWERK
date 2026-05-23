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
The system SHALL present a duty board showing all open duty slots, sorted by event date. Authenticated users can claim an open slot.

#### Scenario: View open duties
- **WHEN** any authenticated user opens the duty board
- **THEN** all open slots (unfilled, future event date) are shown with event name, date, duty type, and remaining vacancies

#### Scenario: Claim a duty slot
- **WHEN** a user (or a parent on behalf of their family) claims an open slot
- **THEN** the system records the assignment, decrements the vacancy count, and updates the claimant's duty account

#### Scenario: Slot fully filled
- **WHEN** the last vacancy of a slot is claimed
- **THEN** the slot no longer appears as open in the duty board

#### Scenario: Cannot claim already-assigned slot
- **WHEN** a user attempts to claim a slot they or their family already hold
- **THEN** the system returns a validation error

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
