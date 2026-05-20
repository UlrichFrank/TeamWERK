## ADDED Requirements

### Requirement: Auto-Assign action on Kader page
The system SHALL provide a dedicated "Auto-Assign" action button on the Kader page, separate from "Aus vorheriger Saison kopieren".

#### Scenario: Button visible on Kader page
- **WHEN** an admin views the Kader page with an active season
- **THEN** both "Aus vorheriger Saison kopieren" and "Auto-Assign" action buttons are visible in the page header area

### Requirement: Auto-Assign kader selection modal
The system SHALL show a modal with checkboxes for each kader in the active season, allowing the admin to select which kader to auto-assign.

#### Scenario: Modal shows all active-season kader with brackets
- **WHEN** admin opens the Auto-Assign modal
- **THEN** each kader is listed with its age class, gender, and birth year bracket (e.g. "A-Jugend männlich (Jg. 2008/2009)")

#### Scenario: All kader checked by default
- **WHEN** admin opens the Auto-Assign modal
- **THEN** all kader checkboxes are checked by default

#### Scenario: Admin deselects some kader
- **WHEN** admin unchecks one or more kader
- **THEN** only the checked kader are auto-assigned on confirm

#### Scenario: Confirm triggers auto-assign for selected kader
- **WHEN** admin confirms the selection
- **THEN** the system auto-assigns members matching birth year bracket and gender to each selected kader
- **THEN** existing members in those kader are not duplicated (INSERT OR IGNORE semantics)

#### Scenario: Success feedback and reload
- **WHEN** auto-assign completes successfully
- **THEN** the modal closes and the kader page reloads to show updated member counts

#### Scenario: No active season
- **WHEN** there is no active season
- **THEN** the Auto-Assign button is disabled or hidden

### Requirement: Auto-Assign does not replace existing members
The system SHALL only ADD members via auto-assign; it SHALL NOT remove existing kader members.

#### Scenario: Kader already has members
- **WHEN** auto-assign runs on a kader that already has members
- **THEN** only members not yet in the kader are added; existing assignments remain unchanged
