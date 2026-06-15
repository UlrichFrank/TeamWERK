## ADDED Requirements

### Requirement: Duty type definition
The system SHALL allow admins to define duty types. A duty type has: name, required hours value (decimal), and optional cash substitute amount (€).

#### Scenario: Admin creates duty type
- **WHEN** an admin submits a duty type with name and hours value
- **THEN** the system stores the duty type and it becomes available for assignment

#### Scenario: Duty type with cash substitute
- **WHEN** a duty type is created with a cash substitute amount
- **THEN** families can optionally pay the amount instead of fulfilling the duty

### Requirement: Duty slot creation
The system SHALL allow admins and trainer to create duty slots attached to an event (e.g., a home game). A slot has: event name, event date, duty type, required role description, and number of persons needed.

#### Scenario: Create duty slot for event
- **WHEN** an admin creates a duty slot with event reference, duty type, and person count
- **THEN** the slot appears in the duty board as open

#### Scenario: Multiple slots per event
- **WHEN** multiple duty slots are created for the same event
- **THEN** each slot is listed independently in the duty board

### Requirement: Duty board (Dienstbörse)
Das System SHALL eine Dienstbörse mit allen Duty-Slots anzeigen. Jeder Slot enthält neben den bisherigen Informationen (event name, date, duty type, vacancies) auch die Liste der eingetragenen Personen mit privacy-gefiltertem Kontaktdaten-Payload. Beim Beanspruchen eines Slots MUSS für Elternteile mit verknüpften Kindern mit Proxy-Account ein „Für wen?"-Selektor erscheinen. Das Beanspruchen eines Slots MUSS race-frei implementiert sein: die Prüfung auf verfügbare Kapazität, das Eintragen des Nutzers und das Aktualisieren des Zählers MÜSSEN als eine atomare Operation erfolgen, die auch bei gleichzeitigen Anfragen korrekt funktioniert.

Die `GET /api/duty-board`-Response gruppiert Slots pro Spiel bzw. pro game-losem Termin. Jede Gruppe SHALL folgende Felder enthalten:

- `game_id` (Integer oder null)
- `team_id` (Integer oder null) — für den Frontend-Team-Filter
- `date`, `event_time`
- `opponent`, `event_type` — bei Spielen aus der `games`-Tabelle; bei game-losen Gruppen ist `opponent` leer und `event_type` SHALL `"generisch"` sein
- `team_name`, `label`, `past`
- `slots[]` — Liste der Slot-Objekte mit den bisherigen Feldern

Die Sichtbarkeit der Gruppen wird wie folgt gefiltert:

- System-Rolle `admin`: alle Gruppen aller Teams der aktiven Saison.
- Vereinsfunktion `vorstand`: alle Gruppen aller Teams der aktiven Saison.
- Alle anderen Rollen (Trainer, Sportliche Leitung, Spieler, Eltern): nur Gruppen, deren `team_id` einem Team entspricht, in dem der Nutzer oder ein verknüpftes Familienmitglied via Kader eingetragen ist; zusätzlich game-lose Gruppen, die zu einem Spiel ihrer Teams gehören.

Die Audience-Filterung auf Slot-Ebene (`audiences`-JSON-Array mit `eltern`/Vereinsfunktionen) bleibt unverändert und gilt orthogonal zur Team-Sichtbarkeit.

#### Scenario: View open duties
- **WHEN** any authenticated user opens the duty board
- **THEN** all open slots (unfilled, future event date) are shown with event name, date, duty type, remaining vacancies, and the list of assignees (name + conditionally photo URL, phones, address)

#### Scenario: Vorstand sieht Dienste fremder Teams
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` (System-Rolle `standard`) `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort auch Gruppen für Teams, in denen der Nutzer kein Mitglied ist

#### Scenario: Spieler sieht nur eigene Team-Dienste
- **WHEN** ein Spieler ohne privilegierte Vereinsfunktion `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort nur Gruppen für Teams, in denen der Spieler oder ein Familienmitglied über Kader eingetragen ist

#### Scenario: Game-lose Gruppe trägt event_type=generisch
- **WHEN** ein Dienst-Slot ohne `game_id` (z. B. Vereinsfest) existiert
- **THEN** enthält die zugehörige Gruppe `event_type: "generisch"` in der API-Response

#### Scenario: Gruppe enthält team_id
- **WHEN** eine team-spezifische Gruppe in der Response erscheint
- **THEN** enthält das Gruppen-Objekt ein numerisches `team_id`-Feld

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

#### Scenario: Concurrent claim — letzter freier Slot
- **WHEN** zwei Nutzer gleichzeitig den letzten freien Slot beanspruchen
- **THEN** gelingt genau einem der Claim (HTTP 204), der andere erhält HTTP 409
- **THEN** ist `slots_filled` danach exakt gleich `slots_total` (kein Überlauf)

#### Scenario: Slot fully filled
- **WHEN** the last vacancy of a slot is claimed
- **THEN** the slot no longer shows vacancies but the assignee names remain visible

#### Scenario: Cannot claim already-assigned slot
- **WHEN** a user attempts to claim a slot they or their family already hold
- **THEN** the system returns a validation error

#### Scenario: Unclaim — atomare Gegenbuchung
- **WHEN** ein Nutzer seinen Dienst-Claim aufhebt
- **THEN** wird die `duty_assignments`-Zeile gelöscht UND `slots_filled` dekrementiert in einer einzigen Transaktion
- **THEN** bei einem Datenbankfehler während der Transaktion bleibt der Zähler konsistent (kein partieller State)

#### Scenario: Privacy-gefilterte Assignee-Daten im API-Response
- **WHEN** der `/duty-board`-Endpoint einen Slot mit Assignees zurückgibt
- **THEN** enthält jeder Assignee-Eintrag: `name` (immer), `photo_url` (nur wenn `photo_visible=1`), `phones` (nur wenn `phones_visible=1`, sonst leeres Array), `address` (nur wenn `address_visible=1`, sonst null)
- **THEN** haben Proxy-Account-Assignees keine `phones` und keine `address` (da Proxy-Accounts diese Daten nicht haben)

### Requirement: Duty account per family
The system SHALL maintain a duty account per family (user/parent unit) per season, tracking target hours (Soll) and fulfilled hours (Ist).

#### Scenario: Soll configured per season
- **WHEN** an admin sets the seasonal duty target for a duty type
- **THEN** each family's Soll is updated to reflect the target

#### Scenario: Ist updated on duty fulfillment
- **WHEN** an admin or trainer marks a duty slot as fulfilled for a family
- **THEN** the family's Ist balance increases by the duty type's hours value

#### Scenario: Family views own duty account
- **WHEN** an `elternteil` or `spieler` views their duty account
- **THEN** they see Soll, Ist, and the balance (Soll − Ist) for the active season

#### Scenario: Admin views all duty accounts
- **WHEN** an admin views the duty overview
- **THEN** all families with their Soll, Ist, and balance are shown, sortable by balance

### Requirement: Cash substitute recording
The system SHALL allow recording a cash substitute payment as an alternative to fulfilling a duty.

#### Scenario: Record cash substitute
- **WHEN** an admin records a cash substitute payment for a family and duty type
- **THEN** the equivalent hours are credited to the family's Ist balance and the payment amount is logged

### Requirement: Duty account export
The system SHALL allow admins to export all duty accounts as CSV for the season treasurer report.

#### Scenario: Export duty accounts
- **WHEN** an admin triggers the duty account export for the active season
- **THEN** the system returns a CSV with: family name, Soll, Ist, balance, and any cash substitute amounts

### Requirement: Dienstbörse als chronologische Liste mit Pill-Filtern

Die Dienstbörse (`/dienste`) SHALL alle Dienst-Gruppen in einer einzigen chronologischen Liste anzeigen. Eine Gruppe entspricht entweder einem Spiel mit n Slot-Zeilen oder einer game-losen Sammelgruppe für einen Vereinsdienst. Die Gruppen-Reihenfolge SHALL primär aufsteigend nach Datum und Uhrzeit erfolgen (Backend liefert dies bereits).

Die heutige binäre Toggle-Leiste „Alle Dienste / Meine Dienste" und der Text-Link „Vergangene einblenden" SHALL NICHT mehr existieren — beide Funktionen werden durch Pill-Buttons im Header ersetzt.

#### Scenario: Liste statt Tabs/Toggles

- **WHEN** ein Nutzer die Dienste-Seite öffnet
- **THEN** sieht er eine einzige chronologisch sortierte Liste aller berechtigten Dienst-Gruppen und keine Tab- oder Segment-Toggles

#### Scenario: Gruppen-Karte bündelt Slots eines Spiels

- **WHEN** ein Spiel mehrere Slots hat (z. B. Schiedsrichter, Kasse, Hallendienst)
- **THEN** erscheinen diese Slots als Zeilen innerhalb einer einzigen Spiel-Karte und nicht als separate Karten

### Requirement: Event-Typ-Filter als Pill-Buttons

Der Header SHALL drei Event-Typ-Pills enthalten: „Heim" 🏠, „Auswärts" ✈ und „Sonstiges" 📅. Mehrere Pills SHALL gleichzeitig aktiv sein können. Eine Gruppe wird angezeigt, wenn ihr `event_type` in der Menge der aktiven Pills enthalten ist. Wenn keine Pill aktiv ist, ist die Liste leer. Ein Trainings-Filter SHALL NICHT existieren (es gibt keine Trainings-Dienste).

#### Scenario: Standardansicht — alle Pills aktiv

- **WHEN** ein Nutzer die Seite ohne URL-Filter öffnet
- **THEN** sind alle drei Event-Typ-Pills aktiv und alle Gruppen-Typen werden angezeigt

#### Scenario: Nur Heimspiele

- **WHEN** ein Nutzer „Auswärts" und „Sonstiges" deaktiviert
- **THEN** zeigt die Liste nur Gruppen mit `event_type=heim`

#### Scenario: Generischer Vereinsdienst unter „Sonstiges"

- **WHEN** ein game-loser Vereinsdienst (z. B. Vereinsfest-Aufbau) existiert
- **THEN** ist seine Gruppe nur sichtbar, wenn die „Sonstiges"-Pill aktiv ist

### Requirement: Team-Filter als Dropdown

Der Header SHALL als erstes Filter-Element ein Team-Dropdown enthalten. Die Default-Option ist „Teams". Die Optionen entsprechen dem Resultat von `GET /api/teams` (rollenabhängig gefiltert: Vorstand/Admin sehen alle Teams, andere Rollen ihre eigenen).

Bei aktivem Team-Filter SHALL nur Gruppen angezeigt werden, deren `team_id` mit dem Filter übereinstimmt. Gruppen ohne `team_id` (vereinsweite Dienste) sind nur bei nicht-aktivem Team-Filter sichtbar.

#### Scenario: Filter auf ein konkretes Team

- **WHEN** ein Nutzer im Team-Dropdown ein bestimmtes Team auswählt
- **THEN** zeigt die Liste nur Gruppen mit passender `team_id`

#### Scenario: Vereinsweite Dienste bei „Teams"

- **WHEN** ein Nutzer „Teams" (Default) ausgewählt lässt und vereinsweite Gruppen ohne `team_id` existieren
- **THEN** erscheinen diese vereinsweiten Gruppen in der Liste

#### Scenario: Vereinsweite Dienste bei aktivem Team-Filter

- **WHEN** ein Nutzer ein konkretes Team filtert und vereinsweite Gruppen existieren
- **THEN** sind die vereinsweiten Gruppen in der gefilterten Liste **nicht** sichtbar

### Requirement: Meine-Pill für alle Rollen

Der Header SHALL eine „Meine"-Pill (`UserCheck`-Icon) im gleichen Stil wie die Event-Typ-Pills enthalten. Die Pill SHALL für **alle** authentifizierten Rollen sichtbar sein — nicht mehr nur für Trainer und Admins.

Im aktiven Zustand SHALL die Seite den bestehenden Backend-Filter `GET /api/duty-board?view=mine` nutzen, der nur Slots zurückgibt, in denen der Nutzer selbst eingetragen ist.

#### Scenario: Spieler nutzt Meine-Pill

- **WHEN** ein Spieler die Meine-Pill aktiviert
- **THEN** zeigt die Liste nur Gruppen mit Slots, in denen der Spieler eingetragen ist

#### Scenario: Trainer nutzt Meine-Pill

- **WHEN** ein Trainer die Meine-Pill aktiviert
- **THEN** zeigt die Liste nur Slots, in denen der Trainer selbst eingetragen ist (nicht: alle Slots seiner Teams)

#### Scenario: Meine ohne Eintragungen

- **WHEN** ein Nutzer die Meine-Pill aktiviert und keinen einzigen Slot übernommen hat
- **THEN** ist die Liste leer und zeigt eine entsprechende Hinweismeldung

### Requirement: Vergangene-Pill statt Text-Link

Der heutige Text-Link „Vergangene einblenden" SHALL durch eine Pill (`History`-Icon) im gleichen visuellen Stil wie die anderen Pills ersetzt werden. Die Filterlogik bleibt identisch — vergangene Gruppen werden client-seitig ein- oder ausgeblendet.

#### Scenario: Vergangene-Pill aktiv

- **WHEN** ein Nutzer die Vergangene-Pill aktiviert
- **THEN** zeigt die Liste auch Gruppen mit `past=true` (vergangenes Datum)

#### Scenario: Vergangene-Pill inaktiv (Default)

- **WHEN** ein Nutzer die Seite ohne aktive Vergangene-Pill öffnet
- **THEN** sind nur Gruppen mit `past=false` sichtbar

### Requirement: Farbcodierung der Karten nach Event-Typ mit Past-Override

Jede Gruppen-Karte SHALL einen farblichen Border-Top und einen leichten Background-Tint erhalten, abgeleitet aus `getEventColors(event_type)`. Die Farbzuordnung SHALL konsistent mit `TerminePage` und `MitfahrgelegenheitenPage` sein:

| Event-Typ   | Border / Tint              |
|-------------|----------------------------|
| `heim`      | brand-yellow               |
| `auswärts`  | brand-text-muted / grau    |
| `generisch` | brand-blue                 |

Vergangene Gruppen (`past=true`) SHALL die Farbcodierung **überschreiben** und stattdessen den heutigen Past-Stil zeigen: `bg-brand-surface-card border-brand-border opacity-60`. Past schlägt Farbe.

#### Scenario: Zukünftiges Heimspiel gelb

- **WHEN** eine zukünftige Spiel-Gruppe mit `event_type=heim` angezeigt wird
- **THEN** zeigt die Karte einen gelben Border-Streifen und einen leichten gelben Background-Tint

#### Scenario: Zukünftiges Auswärtsspiel grau

- **WHEN** eine zukünftige Spiel-Gruppe mit `event_type=auswärts` angezeigt wird
- **THEN** zeigt die Karte einen grauen Border-Streifen und einen leichten grauen Background-Tint

#### Scenario: Zukünftiger generischer Vereinsdienst blau

- **WHEN** eine zukünftige game-lose Gruppe mit `event_type=generisch` angezeigt wird
- **THEN** zeigt die Karte einen blauen Border-Streifen und einen leichten blauen Background-Tint

#### Scenario: Vergangene Gruppe grau und opak

- **WHEN** eine Gruppe mit `past=true` angezeigt wird
- **THEN** zeigt die Karte den Past-Stil (grauer Border, neutraler Hintergrund, `opacity-60`) — unabhängig vom Event-Typ

### Requirement: Filter-State persistiert in URL-Search-Params

Die Auswahl von Team-Filter, aktiven Event-Typ-Pills, Meine-Pill und Vergangene-Pill SHALL als URL-Search-Params gespeichert werden — `?team=<id>`, `?types=<csv>`, `?mine=1`, `?past=1`. Der Default-State (alle Event-Typ-Pills aktiv, kein Team, Meine inaktiv, Vergangene inaktiv) SHALL keine Params in der URL erzeugen.

#### Scenario: Reload erhält Filter

- **WHEN** ein Nutzer „Heim" und „Sonstiges" deaktiviert (nur „Auswärts" aktiv) und die Seite neu lädt
- **THEN** zeigt die URL `?types=auswärts` und es werden nur Auswärtsspiele angezeigt

#### Scenario: Default-Zustand zeigt saubere URL

- **WHEN** ein Nutzer alle Event-Typ-Pills aktiviert lässt, keinen Team-Filter setzt, Meine und Vergangene inaktiv lässt
- **THEN** enthält die URL keine `team`-, `types`-, `mine`- oder `past`-Params

#### Scenario: Deep-Link mit Filter

- **WHEN** ein Nutzer eine URL `?team=3&types=heim&mine=1&past=1` öffnet
- **THEN** sind Team 3 ausgewählt, nur die Heim-Pill aktiv, Meine-Pill aktiv und Vergangene-Pill aktiv

### Requirement: Compact-Header bei schmalen Viewports

Bei Viewport-Breiten unter 950 px SHALL die Pill-Leiste nur die Icons anzeigen (Labels ausgeblendet). Die Schwelle entspricht der `TerminePage`-Konvention via `useCompactHeader(950)`.

#### Scenario: Compact-Modus aktiviert

- **WHEN** die Viewport-Breite < 950 px ist
- **THEN** zeigen alle Filter-Pills nur ihr Icon und keine Text-Labels; Padding ist auf `px-2` reduziert

#### Scenario: Vollformat-Modus

- **WHEN** die Viewport-Breite ≥ 950 px ist
- **THEN** zeigen die Filter-Pills sowohl Icon als auch Label

## MODIFIED Requirements

### Requirement: Duty account per family
Das Duty-Account-System bleibt unverändert — Ist-Wert, Claim-Logik und Export bleiben identisch. Geändert wird ausschließlich die Berechnung des `soll`-Werts für die Rolle `elternteil` im Dashboard-Endpoint.

**Vorher:** `soll = 5 × COUNT(family_links WHERE parent_user_id = user_id)`

**Nachher:** Dynamische Formel basierend auf Kader-Daten (siehe Capability `dienstkonto-dynamische-soll-formel`). Der in der `duty_accounts`-Tabelle gespeicherte Wert bleibt davon unberührt — der `/api/dashboard`-Endpoint berechnet den Wert live.

#### Scenario: Family views own duty account (updated)
- **WHEN** ein `elternteil` das Dashboard aufruft
- **THEN** sieht er `soll` basierend auf der dynamischen Formel (Kader-Spielanzahl, Templates, Spielerzahl, Elternanzahl)
- **AND** der Erklärtext lautet „Ziel: {soll} Dienste (Saison {name})" ohne Formel-Details
