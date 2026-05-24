## ADDED Requirements

### Requirement: Welcome email sent timestamp on member record
The system SHALL store a nullable `welcome_email_sent_at` timestamp on every member record, set when a welcome email is successfully dispatched.

#### Scenario: Field null by default
- **WHEN** a new member is created
- **THEN** `welcome_email_sent_at` is null

#### Scenario: Field set after dispatch
- **WHEN** a welcome email is successfully sent for a member
- **THEN** `welcome_email_sent_at` is set to the dispatch timestamp and cannot be reset to null via the API

#### Scenario: Field returned in member detail
- **WHEN** `GET /api/members/{id}` is called
- **THEN** the response JSON includes `welcome_email_sent_at` as an ISO-8601 string or null
