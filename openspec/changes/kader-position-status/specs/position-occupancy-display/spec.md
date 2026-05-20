## ADDED Requirements

### Requirement: Display position occupancy status
The AdminKaderPage SHALL display a compact visual indicator for each of the 7 handball positions, showing how many members in the current Kader can play each position.

#### Scenario: Show all 7 positions with status
- **WHEN** an Admin views the AdminKaderPage and navigates to a Kader card
- **THEN** the system displays all 7 positions (TW, LA, RA, RL, RM, RR, KL) in a single row between the Jahrgänge-mode toggle and the Trainer search section

#### Scenario: Display red circle for empty position
- **WHEN** a position has 0 members with that position
- **THEN** the system displays 1 red circle (⭕) for that position

#### Scenario: Display yellow circle for single occupancy
- **WHEN** a position has exactly 1 member with that position
- **THEN** the system displays 1 yellow circle (🟡) for that position

#### Scenario: Display green circles for good occupancy
- **WHEN** a position has exactly 2 members with that position
- **THEN** the system displays 2 green circles stacked vertically for that position

#### Scenario: Display blue circles for excellent occupancy
- **WHEN** a position has 3 or more members with that position
- **THEN** the system displays 3 blue circles stacked vertically for that position

### Requirement: Aggregate position data from member objects
The system SHALL count members by position without API calls. The count is computed client-side from the Kader's member list.

#### Scenario: Count members with position
- **WHEN** a Kader has 5 members, and 2 of them have "Linksaußen" in their positions array
- **THEN** the system counts "Linksaußen" as 2 and displays 2 green circles for that position

#### Scenario: Handle members with multiple positions
- **WHEN** a member can play multiple positions (e.g., ["Linksaußen", "Rechtsaußen"])
- **THEN** the system counts that member once for each position they can play

### Requirement: Use position abbreviations
The system SHALL display position names using standard handball abbreviations (TW, LA, RA, RL, RM, RR, KL) to minimize space usage.

#### Scenario: Show position abbreviations
- **WHEN** a position row is rendered
- **THEN** each position is labeled with its 2-letter abbreviation in the same font size as the Jahrgänge badge

### Requirement: Minimize visual footprint
The position status display SHALL use very small circles (14px diameter) and tight spacing to avoid wasting layout space on the Kader card.

#### Scenario: Compact layout
- **WHEN** the position status component is rendered
- **THEN** all 7 positions fit in a single row without wrapping or significant height increase
