## MODIFIED Requirements

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
- Alle anderen Rollen (Trainer, Sportliche Leitung, Spieler, Eltern): nur Gruppen, deren `team_id` einem Team entspricht, in dem der Nutzer als Spieler (`player_memberships`) ODER als Trainer (`trainer_memberships`) eingetragen ist oder ein verknüpftes Familienmitglied (`family_links`) als Spieler eingetragen ist; zusätzlich game-lose Gruppen, die zu einem Spiel eines dieser Teams gehören.

Die Audience-Filterung auf Slot-Ebene (`audiences`-JSON-Array mit `eltern`/Vereinsfunktionen) erfolgt nach folgender Regel:

- System-Rolle `admin`: kein Audience-Filter (Bypass), unabhängig vom Query-Parameter.
- Privilegierte Vereinsfunktionen `vorstand`, `vorstand_beisitzer`, `trainer`, `sportliche_leitung`: standardmäßig Audience-Filter aktiv (nur Slots mit NULL-Audience oder Audience-Match zur eigenen Funktion); per Query-Parameter `?audience=all` deaktivierbar.
- Alle anderen Rollen: Audience-Filter immer aktiv, nicht abschaltbar (Query-Parameter `?audience=all` wird ignoriert).

Der Audience-Match prüft pro Slot, ob das `audiences`-Array eines der folgenden Elemente enthält:
- eine der Vereinsfunktionen des Nutzers (`mcf.function`)
- den Wert `'eltern'`, falls der Nutzer mindestens ein verknüpftes Kind (`family_links`) hat, das **im Team des Slots** spielt (`player_memberships.team_id = ds.team_id`); bei game-losen Slots reicht es, wenn das Kind in einem der teilnehmenden Teams des Spiels spielt.

#### Scenario: View open duties
- **WHEN** any authenticated user opens the duty board
- **THEN** all open slots (unfilled, future event date) are shown with event name, date, duty type, remaining vacancies, and the list of assignees (name + conditionally photo URL, phones, address)

#### Scenario: Vorstand sieht Dienste fremder Teams
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` (System-Rolle `standard`) `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort auch Gruppen für Teams, in denen der Nutzer kein Mitglied ist

#### Scenario: Spieler sieht nur eigene Team-Dienste
- **WHEN** ein Spieler ohne privilegierte Vereinsfunktion `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort nur Gruppen für Teams, in denen der Spieler oder ein Familienmitglied über Kader eingetragen ist

#### Scenario: Trainer sieht Dienste seines trainierten Teams
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer`, der als Trainer (via `kader_trainers`) im Kader von Team A der aktiven Saison eingetragen ist, aber **nicht** als Spieler dort spielt, `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort die Gruppen für Team A
- **AND** enthält **nicht** Gruppen anderer Teams, in denen er weder als Spieler noch als Trainer eingetragen ist

#### Scenario: Sportliche Leitung sieht Dienste ihrer trainierten Teams
- **WHEN** ein Nutzer mit Vereinsfunktion `sportliche_leitung`, der als Trainer in mehreren Kadern eingetragen ist, `GET /api/duty-board` aufruft
- **THEN** enthält die Antwort Gruppen für alle Teams, in deren Kader der Nutzer als Trainer steht

#### Scenario: Game-lose Gruppe trägt event_type=generisch
- **WHEN** ein Dienst-Slot ohne `game_id` (z. B. Vereinsfest) existiert
- **THEN** enthält die zugehörige Gruppe `event_type: "generisch"` in der API-Response

#### Scenario: Gruppe enthält team_id
- **WHEN** eine team-spezifische Gruppe in der Response erscheint
- **THEN** enthält das Gruppen-Objekt ein numerisches `team_id`-Feld

#### Scenario: Trainer sieht standardmäßig nur Audience-Match
- **WHEN** ein Trainer ohne `?audience`-Query-Parameter `GET /api/duty-board` aufruft, und Team A enthält sowohl Slots mit `audiences=["trainer"]` als auch Slots mit `audiences=["spieler"]`
- **THEN** enthält die Antwort nur die Slots mit `audiences=["trainer"]` (und Slots mit NULL-Audience)
- **AND** enthält **nicht** die Slots mit `audiences=["spieler"]`

#### Scenario: Trainer deaktiviert Audience-Filter
- **WHEN** ein Trainer `GET /api/duty-board?audience=all` aufruft, und Team A enthält Slots mit verschiedenen Audiences
- **THEN** enthält die Antwort **alle** Slots der sichtbaren Gruppen, unabhängig von ihrem Audience-Array

#### Scenario: Vorstand deaktiviert Audience-Filter
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` `GET /api/duty-board?audience=all` aufruft
- **THEN** enthält die Antwort alle Slots aller Teams ohne Audience-Filterung

#### Scenario: Spieler kann Audience-Filter nicht deaktivieren
- **WHEN** ein Spieler ohne privilegierte Funktion `GET /api/duty-board?audience=all` aufruft
- **THEN** wird der Query-Parameter ignoriert und der Audience-Filter bleibt aktiv (nur Slots mit Match zur Spieler-Audience oder NULL)

#### Scenario: Eltern-Audience ist team-gescoped
- **WHEN** ein Trainer (gleichzeitig Elternteil eines Kindes in Team B) `GET /api/duty-board` ohne `?audience=all` aufruft, und Team A hat einen Slot mit `audiences=["eltern"]`
- **THEN** ist der Slot **nicht** in der Antwort enthalten — der Eltern-Match greift nur, wenn ein Kind im Slot-Team spielt
- **WHEN** der gleiche Nutzer `GET /api/duty-board?audience=all` aufruft
- **THEN** ist der Slot in der Antwort sichtbar (über die Trainer-Team-Quelle, Audience-Filter deaktiviert)

#### Scenario: Admin sieht immer alle Audiences
- **WHEN** ein Admin `GET /api/duty-board` (ohne Query-Param) aufruft
- **THEN** enthält die Antwort alle Slots ohne Audience-Filterung
- **AND** das Ergebnis ist identisch mit `GET /api/duty-board?audience=all`

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

## ADDED Requirements

### Requirement: Audience-Filter-Pille auf Dienste-Seite
Die Dienstbörse-UI (`/dienste`, `web/src/pages/DutyPage.tsx`) SHALL eine zusätzliche Filter-Pille „Nur meine Audience" mit `Filter`-Icon enthalten, die ausschließlich für Nutzer mit mindestens einer der Vereinsfunktionen `vorstand`, `vorstand_beisitzer`, `trainer`, `sportliche_leitung` sichtbar ist. Die Pille SHALL standardmäßig aktiv sein und ihren Zustand in den URL-Search-Params persistieren: aktiv = kein Param (Default), inaktiv = `?audience=all`. Beim Aufruf von `/api/duty-board` SHALL der Query-Parameter `?audience=all` exakt dann angehängt werden, wenn die Pille deaktiviert ist.

#### Scenario: Pille für Trainer sichtbar
- **WHEN** ein Nutzer mit Vereinsfunktion `trainer` die Seite `/dienste` öffnet
- **THEN** ist die „Nur meine Audience"-Pille in der Filter-Leiste sichtbar
- **AND** ist sie standardmäßig im aktiven Zustand (gelb hinterlegt)

#### Scenario: Pille für Vorstand sichtbar
- **WHEN** ein Nutzer mit Vereinsfunktion `vorstand` die Seite `/dienste` öffnet
- **THEN** ist die „Nur meine Audience"-Pille in der Filter-Leiste sichtbar

#### Scenario: Pille für Spieler nicht sichtbar
- **WHEN** ein Nutzer ohne privilegierte Vereinsfunktion (nur Spieler oder Elternteil) die Seite `/dienste` öffnet
- **THEN** ist die „Nur meine Audience"-Pille **nicht** sichtbar

#### Scenario: Default-Zustand erzeugt keine URL-Params
- **WHEN** ein Trainer die Seite ohne Filter-Änderung lädt
- **THEN** enthält die URL keinen `audience`-Parameter
- **AND** wird `GET /api/duty-board` ohne `audience`-Query aufgerufen

#### Scenario: Deaktivierte Pille schreibt audience=all in URL
- **WHEN** ein Trainer die „Nur meine Audience"-Pille deaktiviert
- **THEN** enthält die URL `?audience=all`
- **AND** wird `GET /api/duty-board?audience=all` aufgerufen

#### Scenario: Deep-Link mit audience=all
- **WHEN** ein Trainer eine URL `/dienste?audience=all` öffnet
- **THEN** ist die Audience-Pille im inaktiven Zustand dargestellt
- **AND** zeigt die Liste alle Slots seiner Teams unabhängig vom Audience-Array
