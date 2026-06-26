# club-config Specification

## Purpose

Diese Spezifikation beschreibt die Capability `club-config`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Club master data management
The system SHALL allow an admin to maintain club master data (name, logo, contact address). Only one club record exists per installation.

#### Scenario: Admin updates club name
- **WHEN** an admin submits a new club name via the settings form
- **THEN** the system persists the change and reflects it in the page title and header

#### Scenario: Non-admin cannot access club settings
- **WHEN** a user without `admin` role navigates to club settings
- **THEN** the system returns HTTP 403

### Requirement: Season configuration
The system SHALL allow an admin to define seasons (name, start date, end date). Exactly one season is marked as active.

#### Scenario: Create new season
- **WHEN** an admin creates a season with a unique name, start date, and end date
- **THEN** the system stores the season and it becomes selectable in other modules

#### Scenario: Activate season
- **WHEN** an admin marks a season as active
- **THEN** all previously active seasons are deactivated and the new one becomes the single active season

#### Scenario: Active season used as default context
- **WHEN** any module loads data that is scoped to a season
- **THEN** the system defaults to the currently active season unless overridden by the user

### Requirement: Team and age-class management
The system SHALL allow an admin to create and manage teams with an associated age class and gender category.

#### Scenario: Create team
- **WHEN** an admin submits a team name, age class (e.g., „A-Jugend"), and gender category (m/f/mixed)
- **THEN** the system creates the team and it becomes available for member assignments and duty planning

#### Scenario: Assign trainer to team
- **WHEN** an admin assigns a user with role `trainer` to a team
- **THEN** that user gains full access to the team's member list and duty planning for that team

#### Scenario: Deactivate team
- **WHEN** an admin deactivates a team
- **THEN** the team no longer appears in active team selections but its historical data is preserved
