## ADDED Requirements

### Requirement: Trainer can read duty types
The system SHALL allow users with `trainer` or `sportliche_leitung` club function to retrieve the list of all duty types via `GET /api/duty-types`.

#### Scenario: Trainer reads duty types
- **WHEN** a user with club_function `trainer` calls `GET /api/duty-types`
- **THEN** the system returns HTTP 200 with the full list of duty types

#### Scenario: Sportliche Leitung reads duty types
- **WHEN** a user with club_function `sportliche_leitung` calls `GET /api/duty-types`
- **THEN** the system returns HTTP 200 with the full list of duty types

#### Scenario: Spieler without trainer function cannot read duty types
- **WHEN** a user with no trainer-related club_function calls `GET /api/duty-types`
- **THEN** the system returns HTTP 403

### Requirement: Trainer can read duty templates
The system SHALL allow users with `trainer` or `sportliche_leitung` club function to retrieve duty templates and their previews.

#### Scenario: Trainer lists duty templates
- **WHEN** a user with club_function `trainer` calls `GET /api/duty-templates`
- **THEN** the system returns HTTP 200 with the list of templates

#### Scenario: Trainer fetches single template
- **WHEN** a user with club_function `trainer` calls `GET /api/duty-templates/{id}`
- **THEN** the system returns HTTP 200 with the template details

#### Scenario: Trainer fetches template preview
- **WHEN** a user with club_function `trainer` calls `GET /api/duty-templates/{id}/preview`
- **THEN** the system returns HTTP 200 with the preview slots

#### Scenario: Trainer cannot create or modify templates
- **WHEN** a user with only club_function `trainer` calls `POST /api/duty-templates`
- **THEN** the system returns HTTP 403

### Requirement: SpieltagDetailPage uses club_functions for edit permission check
The system SHALL gate the "Dienst hinzufügen" UI on `SpieltagDetailPage` via the JWT `club_functions` array, not the JWT `role` field.

#### Scenario: User with role=spieler and club_function=trainer sees edit UI
- **WHEN** a user with `role=spieler` and `club_functions=["trainer"]` opens a Spieltag detail page
- **THEN** the duty-type dropdown and "Dienst hinzufügen" button are visible

#### Scenario: User with role=spieler and no trainer function sees no edit UI
- **WHEN** a user with `role=spieler` and `club_functions=[]` opens a Spieltag detail page
- **THEN** no duty-management UI is shown
