# members Specification

## Purpose
Verwaltung von Mitglieder-/Spielerprofilen: Stammdaten, Status-Lebenszyklus, Team-Zuordnung, Eltern-Kind-Verknüpfung, CSV-Import sowie die Stammverein-Referenz (`home_club_id`).

## Requirements

### Requirement: Player profile management
The system SHALL allow admins and users with the `trainer` club function to create and maintain player profiles. A player profile contains: first name, last name, date of birth, pass number, jersey number, position, and member status. The membership number (`member_number`) is NOT part of the editable base fields: it is system-assigned on creation and read-only for non-admins (see capability `mitgliedsnummer-verwaltung`). Club functions are stored as a set (zero or more) in `member_club_functions` and are managed separately from the profile's base fields.

#### Scenario: Admin creates player profile
- **WHEN** an admin submits a new player profile with required fields (first name, last name, date of birth)
- **THEN** the system creates the profile with status `aktiv` by default, an empty club function set, and an automatically assigned membership number (highest numeric + 1)

#### Scenario: Admin assigns club functions to member
- **WHEN** an admin submits a set of club functions (e.g., `["spieler", "trainer"]`) for an existing member
- **THEN** the system replaces the member's current function set with the submitted set

#### Scenario: Teamleiter creates player in own team
- **WHEN** a user with `trainer` club function creates a player profile
- **THEN** the player is automatically assigned to the trainer's team

#### Scenario: Duplicate pass number rejected
- **WHEN** a profile is saved with a pass number that already exists in the system
- **THEN** the system returns a validation error identifying the conflict

#### Scenario: Membership number is not taken from the create request
- **WHEN** a create request includes an explicit `member_number`
- **THEN** the system ignores it and assigns the next free number automatically

### Requirement: Member status lifecycle
The system SHALL track the member status of each player. Valid statuses: `aktiv`, `verletzt`, `pausiert`, `ausgetreten`.

#### Scenario: Status change recorded
- **WHEN** an admin or trainer updates a player's status
- **THEN** the system persists the new status and records the change timestamp

#### Scenario: Ausgetretene Mitglieder excluded from active lists
- **WHEN** any module queries active members
- **THEN** members with status `ausgetreten` are excluded from results unless explicitly requested

### Requirement: Team membership assignment
The system SHALL allow assigning a player to one or more teams, with a primary team designation.

#### Scenario: Assign player to team
- **WHEN** an admin assigns a player to a team for the active season
- **THEN** the player appears in that team's member list

#### Scenario: Multiple team membership
- **WHEN** a player is assigned to more than one team
- **THEN** the system stores all assignments and marks one as primary

#### Scenario: Teamleiter sees only own team members
- **WHEN** a `trainer` views the member list
- **THEN** only members assigned to their team(s) are shown

### Requirement: Parent/child linking
The system SHALL allow linking standard user accounts (acting as parents/guardians) to player profiles. A parent user can be linked to one or more player profiles via `family_links`. Parent users have no linked member record of their own. The API MUST return the correct parent user data when queried for a member's parents.

#### Scenario: Admin links parent to player
- **WHEN** an admin links a standard user account to a player profile via family_links
- **THEN** the parent can view that player's data and act on their behalf (Zu-/Absagen, Dienste)

#### Scenario: Parent linked to member — data returned correctly
- **WHEN** `GET /api/members/{id}/parents` is called for a member with linked parent users
- **THEN** the response contains each parent's `id`, full name (`first_name || ' ' || last_name`), and `email`
- **THEN** the response MUST NOT be empty due to a non-existent `name` column

#### Scenario: Member with no linked parents
- **WHEN** `GET /api/members/{id}/parents` is called for a member with no family links
- **THEN** the response contains an empty array

#### Scenario: Parent sees only linked children
- **WHEN** a user with `is_parent: true` views the member area
- **THEN** only their linked children's profiles are visible

#### Scenario: Player account linked to own profile
- **WHEN** a standard user (age ≥ 14) with the `spieler` club function is assigned a user account
- **THEN** they can view and partially edit their own profile

### Requirement: Vehicle information for transport planning
The system SHALL allow parents and players to store vehicle information (seats available) for use in future transport planning.

#### Scenario: Parent stores vehicle data
- **WHEN** an `elternteil` user submits vehicle type and available seats
- **THEN** the system stores the data against their user account for use in transport modules

### Requirement: Member list export
The system SHALL allow admins to export the full member list as CSV.

#### Scenario: CSV export
- **WHEN** an admin triggers the CSV export
- **THEN** the system returns a downloadable CSV file with all active member profiles and their team assignments

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

### Requirement: Abwesenheits-Sichtbarkeit am Member-Profil
Jeder Member SHALL ein Feld `absences_public` (Boolean, default `false`) besitzen, das steuert, ob seine Abwesenheiten für Trainer im Kalender sichtbar sind. Das Feld MUSS über das eigene Profil gesetzt werden können.

#### Scenario: Standard-Sichtbarkeit ist privat
- **WHEN** ein neuer Member angelegt wird
- **THEN** ist `absences_public` standardmäßig `false`

#### Scenario: Spieler aktiviert Sichtbarkeit
- **WHEN** ein Spieler `PUT /api/profile/absence-visibility` mit `{"public": true}` aufruft
- **THEN** wird `absences_public` auf `true` gesetzt und ist sofort wirksam für Kalenderabfragen

### Requirement: CSV-Import für Mitglieder
Das System SHALL einen CSV-Import für Mitgliedsdaten unterstützen. Geburtsdaten in zweistelligem Jahresformat (DD.MM.YY) MÜSSEN auf das Jahrhundert abgebildet werden, das **nicht in der Zukunft** liegt: `20YY`, sofern dieses Jahr nicht nach dem aktuellen Jahr läge, sonst `19YY`. So werden auch ältere Mitglieder korrekt erkannt (Geburts- und Beitrittsdaten sind nie zukünftig).

#### Scenario: Importierte Geburtstage — aktuelle Jugendliche
- **WHEN** ein CSV-Import ein Geburtsdatum mit zweistelligem Jahr enthält, dessen `20YY`-Form nicht in der Zukunft liegt (z.B. `01.03.25` bei aktuellem Jahr 2026)
- **THEN** wird das Geburtsjahr als `2025` gespeichert

#### Scenario: Importierte Geburtstage — ältere Mitglieder
- **WHEN** ein CSV-Import ein Geburtsdatum mit zweistelligem Jahr enthält, dessen `20YY`-Form in der Zukunft läge (z.B. `06.12.67` oder `15.07.72`)
- **THEN** wird das Geburtsjahr als `1967` bzw. `1972` gespeichert (nicht `2067`/`2072`)

#### Scenario: Vierstelliges Jahr bleibt unverändert
- **WHEN** ein CSV-Import ein Geburtsdatum mit vierstelligem Jahr enthält (z.B. `10.05.2030`)
- **THEN** wird das Geburtsjahr unverändert als `2030` gespeichert

### Requirement: Feld `beitragsfrei_grund`

Das System SHALL ein optionales Textfeld `beitragsfrei_grund` (`TEXT NULL`) auf der Tabelle `members` führen. Es speichert die Begründung, weshalb das Mitglied beitragsfrei gestellt ist (z. B. `kein aktiver Sportler mehr`, `Zweitspielrecht`). Das Feld wird via `GET /api/members/{id}` ausgeliefert, via `PUT /api/members/{id}` (Vorstand/Admin) und via `PUT /api/members/{id}/bank-details` (Kassierer, siehe Capability `kassierer-member-zugriff`) geschrieben.

#### Scenario: Feld in der Detail-Response

- **WHEN** ein Vorstand `GET /api/members/{id}` aufruft, dessen Mitglied `beitragsfrei_grund = "kein aktiver Sportler mehr"` hat
- **THEN** enthält die Response das Feld `beitragsfrei_grund` mit diesem Wert

#### Scenario: Feld setzbar durch Vorstand

- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `beitragsfrei: true, beitragsfrei_grund: "Zweitspielrecht"` aufruft
- **THEN** persistiert das System beide Felder und `GET /api/members/{id}` liefert sie zurück

### Requirement: Kopplung `beitragsfrei` und `beitragsfrei_grund`

Das System SHALL die Invariante durchsetzen: wenn `members.beitragsfrei = 0`, dann `members.beitragsfrei_grund IS NULL`. Diese Kopplung MUST auf Applikationsebene in jedem schreibenden Pfad (`PUT /api/members/{id}`, `PUT /api/members/{id}/bank-details`) erzwungen werden. Ein DB-CHECK-Constraint wird BEWUSST nicht eingeführt, damit fehlerhafte Eingaben mit HTTP 204 bereinigt werden statt 500 zu liefern.

#### Scenario: Deaktivieren räumt den Grund auf

- **GIVEN** ein Mitglied mit `beitragsfrei=true, beitragsfrei_grund="Zweitspielrecht"`
- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `beitragsfrei: false` aufruft (Grund-Feld nicht oder beliebig mitgesendet)
- **THEN** speichert das System `beitragsfrei=false` und `beitragsfrei_grund=NULL`
- **AND** `GET /api/members/{id}` liefert `beitragsfrei_grund` als leer/null

#### Scenario: Aktivieren ohne Grund ist erlaubt

- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `beitragsfrei: true` und leerem `beitragsfrei_grund` aufruft
- **THEN** speichert das System `beitragsfrei=true` und `beitragsfrei_grund=NULL`
- **AND** liefert HTTP 204

### Requirement: UI-Darstellung im Bankdaten-Block

Das Frontend SHALL im `MemberKontaktTab` unterhalb der Checkbox „Beitragsfrei" ein Textinput „Grund für Beitragsfreiheit" anzeigen, das NUR sichtbar ist, wenn die Checkbox aktiv ist. Beim Deselektieren der Checkbox MUST das Form-Feld lokal geleert werden, damit der Server beim Speichern `NULL` schreibt.

#### Scenario: Grund-Feld sichtbar, wenn beitragsfrei aktiv

- **WHEN** die Bankdaten-Sektion mit `form.beitragsfrei === true` gerendert wird
- **THEN** ist das Textinput „Grund" sichtbar und editierbar

#### Scenario: Grund-Feld verschwindet beim Toggle

- **GIVEN** `form.beitragsfrei === true` und `form.beitragsfrei_grund === "kein aktiver Sportler mehr"`
- **WHEN** der Nutzer die Checkbox abwählt
- **THEN** verschwindet das Grund-Input und der Form-State setzt `beitragsfrei: false, beitragsfrei_grund: ''`

### Requirement: Stammverein eines Mitglieds als Referenz
Ein Mitglied MUST einen Stammverein über `members.home_club_id` (FK auf `stammvereine`) zugeordnet bekommen können. `NULL` bedeutet „kein Stammverein". Beim Aktualisieren eines Mitglieds (`PUT /api/members/{id}`) MUST das Feld `home_club_id` (nullable Integer) akzeptiert und persistiert werden; ein gesetzter Wert MUST auf einen existierenden `stammvereine`-Eintrag verweisen, sonst HTTP 400.

#### Scenario: Stammverein zuweisen
- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `home_club_id` eines existierenden Vereins sendet
- **THEN** wird die Zuordnung gespeichert und in `GET /api/members/{id}` zurückgegeben

#### Scenario: Stammverein entfernen
- **WHEN** ein Vorstand `PUT /api/members/{id}` mit `home_club_id: null` sendet
- **THEN** wird die Zuordnung entfernt (Mitglied gilt im Beitragslauf als `aktiv_ohne`)

#### Scenario: Ungültiger Verein
- **WHEN** ein `PUT /api/members/{id}` mit einer `home_club_id` ohne passenden `stammvereine`-Eintrag erfolgt
- **THEN** antwortet der Server mit HTTP 400
