## ADDED Requirements

### Requirement: Duty type target_role uses club function vocabulary

The system SHALL constrain `duty_types.target_role` to values that describe a *target audience* for slot assignment. Valid values are: `spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`, `kassierer`, `sportliche_leitung` (each corresponding to a club function), plus the special audience marker `elternteil` (resolved at scheduler time via `family_links` to parents of members with the `spieler` club function).

The legacy value `admin` is no longer accepted, because System-Admins are not a duty audience. Existing rows with `target_role='admin'` SHALL be migrated to `vorstand`.

#### Scenario: Duty type accepts spieler target
- **WHEN** an admin creates a duty type with `target_role='spieler'`
- **THEN** the row is inserted successfully

#### Scenario: Duty type accepts elternteil target
- **WHEN** an admin creates a duty type with `target_role='elternteil'`
- **THEN** the row is inserted successfully

#### Scenario: Duty type accepts sportliche_leitung target
- **WHEN** an admin creates a duty type with `target_role='sportliche_leitung'`
- **THEN** the row is inserted successfully

#### Scenario: Duty type rejects admin target
- **WHEN** any code attempts `INSERT INTO duty_types (..., target_role='admin') ...`
- **THEN** the insert fails with a CHECK constraint error

#### Scenario: Migration backfills admin to vorstand
- **WHEN** migration 042 runs against a database with `duty_types` rows where `target_role='admin'`
- **THEN** those rows have `target_role='vorstand'` after the migration completes

---

### Requirement: Duty reminder scheduler resolves recipients via club functions

The system SHALL resolve duty-slot reminder recipients by matching `duty_slots.target_role` against club functions and family links — NOT against `users.role`. Specifically:

- `target_role='spieler'`: users via `members → member_club_functions(function='spieler')` (and, if the slot has a team, restricted to that team's active-season kader via `kader_members → kader`).
- `target_role='trainer'`: users via `members → member_club_functions(function='trainer')` (and, if the slot has a team, restricted to that team's `kader_trainers`).
- `target_role='elternteil'`: users via `family_links → members → member_club_functions(function='spieler')` (and, if the slot has a team, restricted to that team's active-season kader).
- Other club-function `target_role` values (`vorstand`, `sportliche_leitung`, …): users via `members → member_club_functions(function=<target_role>)`.

#### Scenario: Spieler reminder reaches user with spieler function
- **WHEN** a duty slot has `target_role='spieler'` and `team_id` set, and a user has a member with `club_functions=['spieler']` in that team's active-season kader
- **THEN** the scheduler's eligible-recipient list includes that user

#### Scenario: Spieler reminder does NOT match users with role='spieler' (legacy)
- **WHEN** a user has `users.role='standard'` and no `member_club_functions` entry, regardless of any historical `users.role` value
- **THEN** the scheduler's eligible-recipient list for a `target_role='spieler'` slot does NOT include that user

#### Scenario: Elternteil reminder reaches parent of kader member
- **WHEN** a duty slot has `target_role='elternteil'` and `team_id` set, and a user is in `family_links` for a member with `spieler` function in that team's active-season kader
- **THEN** the scheduler's eligible-recipient list includes that parent user

#### Scenario: Trainer reminder reaches trainer via kader_trainers
- **WHEN** a duty slot has `target_role='trainer'` and `team_id` set, and a user has a member that is registered in `kader_trainers` for a kader of that team with club function `trainer`
- **THEN** the scheduler's eligible-recipient list includes that user
