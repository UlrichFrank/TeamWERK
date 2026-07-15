## MODIFIED Requirements

### Requirement: Phantom club functions are rejected

The system SHALL reject any club function value not listed in the canonical set (`spieler`, `trainer`, `vorstand`, `vorstand_beisitzer`, `kassierer`, `sportliche_leitung`, `medien`). In particular, the historically-referenced but never-defined function `sportvorstand` is invalid and MUST NOT be checked anywhere in code (backend `HasFunction`, frontend `hasFunction`).

The function `medien` is added in this change to gate Spielbericht-Freigabe (siehe `match-reports`-Capability). Träger dürfen eingereichte Berichte lesen, editieren und veröffentlichen.

#### Scenario: Database rejects sportvorstand insert
- **WHEN** any code attempts `INSERT INTO member_club_functions (member_id, function) VALUES (?, 'sportvorstand')`
- **THEN** the insert fails with a CHECK constraint error

#### Scenario: Database accepts medien insert
- **WHEN** any code attempts `INSERT INTO member_club_functions (member_id, function) VALUES (?, 'medien')`
- **THEN** the insert succeeds

#### Scenario: Backend never queries for sportvorstand
- **WHEN** the codebase is grepped for `HasFunction("sportvorstand")` or equivalent string literal
- **THEN** zero matches are found

#### Scenario: Frontend never queries for sportvorstand
- **WHEN** the frontend codebase is grepped for `hasFunction(user, 'sportvorstand')` or equivalent
- **THEN** zero matches are found
