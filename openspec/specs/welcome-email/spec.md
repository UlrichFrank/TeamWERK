## ADDED Requirements

### Requirement: Manual welcome email dispatch
The system SHALL allow admins to manually send a personalized welcome email to a member, provided a user account is linked to that member. The email SHALL be sent exactly once; once sent, the action cannot be repeated.

#### Scenario: Button visible when user account linked
- **WHEN** an admin views the Admin-Tab of a member who has a linked user account and no welcome email has been sent yet
- **THEN** the system displays an active „Willkommensmail senden" button

#### Scenario: Button disabled when no user account linked
- **WHEN** an admin views the Admin-Tab of a member who has no linked user account
- **THEN** the button is either hidden or disabled with a hint that a user account must be linked first

#### Scenario: Button disabled after email sent
- **WHEN** a welcome email has already been sent for this member
- **THEN** the button is disabled and the dispatch timestamp is displayed

#### Scenario: Successful dispatch
- **WHEN** an admin clicks „Willkommensmail senden" and the SMTP server is reachable
- **THEN** the system sends the email, records the timestamp in `welcome_email_sent_at`, and confirms success in the UI

#### Scenario: SMTP failure
- **WHEN** an admin clicks „Willkommensmail senden" but the SMTP server is unreachable or returns an error
- **THEN** the system returns an error, does NOT set `welcome_email_sent_at`, and displays an error message in the UI

### Requirement: Welcome email content
The email sent by the system SHALL contain a personalized greeting, the member's admission details, and three fixed PDF attachments.

#### Scenario: Gender-correct salutation
- **WHEN** the member's `gender` field is `m`
- **THEN** the email begins with „Lieber <Vorname>,"

#### Scenario: Female salutation
- **WHEN** the member's `gender` field is `f`
- **THEN** the email begins with „Liebe <Vorname>,"

#### Scenario: Neutral salutation
- **WHEN** the member's `gender` field is neither `m` nor `f`
- **THEN** the email begins with „Liebe/r <Vorname>,"

#### Scenario: Admission date in body
- **WHEN** the member has a `join_date` set
- **THEN** the email body contains that date formatted as DD.MM.YYYY

#### Scenario: Admission date fallback
- **WHEN** the member has no `join_date`
- **THEN** the email body contains today's date formatted as DD.MM.YYYY

#### Scenario: Member number in body
- **WHEN** the member has a `member_number`
- **THEN** the email body contains that member number

#### Scenario: PDF attachments included
- **WHEN** the welcome email is sent
- **THEN** the email includes three PDF attachments: Vereinssatzung, Gebührenordnung, Leitbild

#### Scenario: Logo attached
- **WHEN** the welcome email is sent
- **THEN** the Team Stuttgart logo (PNG) is attached to the email

### Requirement: Dispatch tracking
The system SHALL record when a welcome email was sent for each member and expose this information via the member detail API.

#### Scenario: Timestamp stored on success
- **WHEN** the welcome email is successfully sent
- **THEN** `welcome_email_sent_at` is set to the current UTC timestamp on the member record

#### Scenario: Timestamp exposed in API
- **WHEN** an admin fetches the member detail via `GET /api/members/{id}`
- **THEN** the response includes `welcome_email_sent_at` (ISO string or null)
