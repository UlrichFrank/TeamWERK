## ADDED Requirements

### Requirement: Kontaktdaten-Endpoint
Das System SHALL einen Endpoint `GET /api/users/:id/contact` bereitstellen, der für einen authentifizierten Nutzer die öffentlich freigegebenen Kontaktdaten einer Person zurückgibt.

#### Scenario: Nutzer mit freigegebenen Daten
- **GIVEN** ein authentifizierter Nutzer
- **WHEN** `GET /api/users/42/contact` aufgerufen wird und Nutzer 42 hat `phones_visible=true` und `address_visible=true` gesetzt
- **THEN** gibt der Endpoint `{ name, photo_url, phones: [...], address: "..." }` zurück

#### Scenario: Nutzer ohne Freigaben
- **GIVEN** ein authentifizierter Nutzer
- **WHEN** `GET /api/users/42/contact` aufgerufen wird und Nutzer 42 hat keine Sichtbarkeiten freigegeben
- **THEN** gibt der Endpoint `{ name }` zurück (nur Name, keine Kontaktfelder)

#### Scenario: Nutzer nicht gefunden
- **WHEN** `GET /api/users/99999/contact` aufgerufen wird und user_id 99999 existiert nicht
- **THEN** antwortet der Endpoint mit HTTP 404

#### Scenario: Nicht authentifiziert
- **WHEN** `GET /api/users/42/contact` ohne gültigen JWT aufgerufen wird
- **THEN** antwortet der Endpoint mit HTTP 401

### Requirement: PersonChip-Komponente
Das System SHALL eine `PersonChip`-Komponente bereitstellen, die auf Hover (Desktop) oder Tap (Mobile) Kontaktdaten anzeigt.

#### Scenario: Erster Hover — Daten werden geladen
- **WHEN** ein Nutzer mit der Maus über einen PersonChip fährt (userId vorhanden)
- **THEN** öffnet sich der Tooltip mit einem Lade-Indikator und `GET /api/users/:id/contact` wird gefetcht

#### Scenario: Wiederholter Hover — Cache trifft
- **WHEN** ein Nutzer erneut über denselben PersonChip fährt (Daten bereits gecacht)
- **THEN** öffnet sich der Tooltip sofort mit den gecachten Daten — kein neuer Request

#### Scenario: Tap auf Mobile
- **WHEN** ein Nutzer auf einen PersonChip tappt
- **THEN** öffnet sich der Tooltip; ein Tap außerhalb schließt ihn

#### Scenario: Person ohne userId
- **WHEN** ein PersonChip ohne `userId` gerendert wird (Member ohne Account)
- **THEN** zeigt er den Namen als Plain-Text, kein Tooltip, keine Hover-Interaktion

### Requirement: Cache-Invalidierung bei Logout
Das System SHALL den Kontaktdaten-Cache leeren, wenn ein Nutzer sich ausloggt.

#### Scenario: Logout löscht Cache
- **WHEN** ein Nutzer sich ausloggt
- **THEN** wird der PersonContactContext-Cache geleert, sodass bei erneutem Login (anderer Nutzer) keine alten Daten angezeigt werden

### Requirement: Rollout auf alle Personen-Darstellungen
Das System SHALL `PersonChip` an allen Stellen verwenden, an denen Personen aktuell angezeigt werden.

#### Scenario: Duty-Board
- **WHEN** ein Slot Assignees hat
- **THEN** wird jeder Assignee als PersonChip dargestellt (mit userId aus dem Board-Response)

#### Scenario: Kader-Trainer
- **WHEN** ein Kader-Eintrag Trainer hat
- **THEN** wird jeder Trainer als PersonChip dargestellt; Trainer ohne Account als Plain-Text

#### Scenario: Mitglieder-Liste
- **WHEN** ein Mitglied in der Liste angezeigt wird und hat eine user_id
- **THEN** wird der Name als PersonChip dargestellt; Mitglieder ohne Account als Plain-Text
