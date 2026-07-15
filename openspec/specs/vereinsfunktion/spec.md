# vereinsfunktion Specification

## Purpose

Diese Spezifikation beschreibt die Capability `vereinsfunktion`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)
## Requirements
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

### Requirement: Parent status is not a club function

The system SHALL keep parent status (`is_parent`) as a JWT claim that is derived at login time from the existence of `family_links` rows, and SHALL NOT model `elternteil` as a value in `member_club_functions.function`. Code that needs to check „is this user a parent" MUST read `claims.IsParent`, not call `claims.HasFunction("elternteil")`.

#### Scenario: Parent user has empty club_functions and is_parent true
- **WHEN** a user has `family_links` rows but no linked member with club functions
- **THEN** the JWT contains `club_functions: []` and `is_parent: true`

#### Scenario: elternteil is never stored as a club function
- **WHEN** any code attempts `INSERT INTO member_club_functions (member_id, function) VALUES (?, 'elternteil')`
- **THEN** the insert fails with a CHECK constraint error

#### Scenario: Backend never checks HasFunction("elternteil")
- **WHEN** the codebase is grepped for `HasFunction("elternteil")` or equivalent
- **THEN** zero matches are found; parent gates use `claims.IsParent` instead

