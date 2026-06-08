## MODIFIED Requirements

### Requirement: Duty board (Dienstbörse)
Das System SHALL eine Dienstbörse mit allen Duty-Slots anzeigen. Jeder Slot enthält neben den bisherigen Informationen (event name, date, duty type, vacancies) auch die Liste der eingetragenen Personen mit privacy-gefiltertem Kontaktdaten-Payload. Beim Beanspruchen eines Slots MUSS für Elternteile mit verknüpften Kindern mit Proxy-Account ein „Für wen?"-Selektor erscheinen.

#### Scenario: View open duties
- **WHEN** any authenticated user opens the duty board
- **THEN** all open slots (unfilled, future event date) are shown with event name, date, duty type, remaining vacancies, and the list of assignees (name + conditionally photo URL, phones, address)

#### Scenario: Claim a duty slot — kein Familienmitglied vorhanden
- **WHEN** a user without linked children with proxy accounts claims an open slot
- **THEN** the system records the assignment directly for that user, decrements the vacancy count, updates the claimant's duty account, and the claimant's name appears in the assignee list

#### Scenario: Claim a duty slot — Elternteil mit Kind-Auswahl
- **WHEN** ein Elternteil mit mindestens einem verknüpften Kind mit Proxy-Account auf „Eintragen" klickt
- **THEN** erscheint ein „Für wen?"-Dialog mit dem eigenen Namen als Default und je einem Eintrag pro Kind mit Proxy-Account
- **WHEN** das Elternteil sich selbst auswählt und bestätigt
- **THEN** wird der Dienst dem Elternteil zugebucht (Verhalten wie bisher)
- **WHEN** das Elternteil ein Kind auswählt und bestätigt
- **THEN** wird der Dienst der `user_id` des Kindes zugebucht und das Dienstkonto des Kindes aktualisiert

#### Scenario: Slot fully filled
- **WHEN** the last vacancy of a slot is claimed
- **THEN** the slot no longer shows vacancies but the assignee names remain visible

#### Scenario: Cannot claim already-assigned slot
- **WHEN** a user attempts to claim a slot they or their family already hold
- **THEN** the system returns a validation error

#### Scenario: Privacy-gefilterte Assignee-Daten im API-Response
- **WHEN** der `/duty-board`-Endpoint einen Slot mit Assignees zurückgibt
- **THEN** enthält jeder Assignee-Eintrag: `name` (immer), `photo_url` (nur wenn `photo_visible=1`), `phones` (nur wenn `phones_visible=1`, sonst leeres Array), `address` (nur wenn `address_visible=1`, sonst null)
- **THEN** haben Proxy-Account-Assignees keine `phones` und keine `address` (da Proxy-Accounts diese Daten nicht haben)
