## MODIFIED Requirements

### Requirement: Smart Copy from previous season
The system SHALL copy kader members from the previous season using correct age-class progression: the younger birth year stays in the same age class, the older birth year from the class below moves up. Only members whose birth year falls within the target kader's bracket SHALL be copied.

#### Scenario: Younger year stays in same class
- **WHEN** copying A-Jugend to the next season
- **THEN** members from the previous A-Jugend whose birth year is within the new A-Jugend bracket are copied

#### Scenario: Older year moves up from class below
- **WHEN** copying A-Jugend to the next season
- **THEN** members from the previous B-Jugend whose birth year is within the new A-Jugend bracket are also copied

#### Scenario: Aged-out members are excluded
- **WHEN** copying A-Jugend to the next season
- **THEN** members from the previous A-Jugend whose birth year is outside the new bracket (the oldest year) are NOT copied

#### Scenario: D-Jugend has no class below
- **WHEN** copying D-Jugend to the next season
- **THEN** only the remaining year from the previous D-Jugend bracket is copied; the new birth year cohort is not populated

#### Scenario: Empty copy option remains available
- **WHEN** a kader assignment has member_source "empty"
- **THEN** the kader structure is created with no members

## REMOVED Requirements

### Requirement: Auto-assign option in copy modal
**Reason**: Auto-assign is a separate workflow with a different mental model (populate by birth year without a previous season) and is now available as a dedicated action.
**Migration**: Use the new Auto-Assign action button on the Kader page.

### Requirement: Same-age-previous and age-before-previous as separate options
**Reason**: Replaced by unified smart-copy that combines both sources with birth year filtering.
**Migration**: The smart-copy default performs the correct combined operation automatically.
