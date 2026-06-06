## MODIFIED Requirements

### Requirement: Member status lifecycle
The system SHALL track the member status of each player. Valid statuses: `aktiv`, `verletzt`, `pausiert`, `ausgetreten`, `honorar`.

The `honorar` status is used for external or honorary members (e.g., paid coaches, Honorartrainer) who are tracked in the system but are not full club members. They have no duty obligations, no RSVP requirements, and are excluded from active member counts.

#### Scenario: Status change recorded
- **WHEN** an admin or trainer updates a player's status
- **THEN** the system persists the new status and records the change timestamp

#### Scenario: Ausgetretene Mitglieder excluded from active lists
- **WHEN** any module queries active members
- **THEN** members with status `ausgetreten` are excluded from results unless explicitly requested

#### Scenario: Honorar-Mitglieder excluded from active obligations
- **WHEN** any module queries members for duty obligations, RSVP invitations, or Soll-calculations
- **THEN** members with status `honorar` are excluded from those results

#### Scenario: Honorar-Mitglieder visible in trainer and team contexts
- **WHEN** an admin views trainer assignments or team staff
- **THEN** members with status `honorar` are included and visually marked as honorary

## ADDED Requirements

### Requirement: Honorar-Mitglied creation
The system SHALL allow admins to create a member profile with status `honorar`. An honorary member MAY have a linked `users` account and MAY be assigned the `trainer` role to manage a team.

#### Scenario: Admin creates honorary member
- **WHEN** an admin submits a new member profile and selects status `honorar`
- **THEN** the system creates the profile with that status without assigning default active obligations

#### Scenario: Honorary member assigned trainer club function
- **WHEN** an admin assigns the club function `trainer` to a `honorar` member and links them to a `users` account with `role=standard`
- **THEN** the trainer can log in and access all trainer functionality identically to a regular member trainer

#### Scenario: Honorary member excluded from member count
- **WHEN** the system displays or exports total active member counts
- **THEN** members with status `honorar` are not included in the count
