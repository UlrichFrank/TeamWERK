## ADDED Requirements

### Requirement: Dienstbörse zeigt Dienste gefiltert auf eigene Teams

`GET /api/duty-board` SHALL nur Dienste von Teams zurückgeben, in denen der eingeloggte User Mitglied ist — direkt (über `members.user_id`) oder indirekt als Elternteil (über `family_links` → `team_memberships`, aktive Saison).

#### Scenario: Spieler sieht nur seine Mannschaft
- **WHEN** ein User mit Rolle `spieler` die Dienstbörse aufruft und Mitglied in Team A ist
- **THEN** enthält die Antwort nur Gruppen mit `team_id` von Team A

#### Scenario: Elternteil sieht alle Kinder-Teams
- **WHEN** ein User mit Rolle `elternteil` zwei Kinder in verschiedenen Teams hat
- **THEN** enthält die Antwort Gruppen beider Teams in getrennten Kacheln

#### Scenario: User ohne Teamzuordnung
- **WHEN** ein User keiner Mannschaft zugeordnet ist
- **THEN** gibt die API eine leere Liste zurück (`[]`)

---

### Requirement: Dienste werden nach Spiel gruppiert

Die API-Antwort SHALL eine Liste von Gruppen sein. Jede Gruppe mit `game_id != null` repräsentiert ein Heimspiel. Gruppen ohne `game_id` tragen das Label „Sonstige Dienste" und sind pro Team zusammengefasst.

Jede Gruppe enthält: `game_id`, `date`, `event_time`, `opponent`, `team_name`, `past` (bool), `label` (optional), `slots` (Array).

#### Scenario: Slots mit game_id in Spielgruppe
- **WHEN** mehrere Slots denselben `game_id`-Wert haben
- **THEN** erscheinen sie in einer gemeinsamen Gruppe mit Spieldatum und Gegner

#### Scenario: Slots ohne game_id in Sonstige-Gruppe
- **WHEN** ein Slot keinen `game_id` hat, aber team_id = Team A
- **THEN** erscheint er in einer „Sonstige Dienste"-Gruppe für Team A

---

### Requirement: claimed_by_me-Flag pro Slot

Jeder Slot in der Antwort SHALL ein `claimed_by_me`-Feld (boolean) enthalten, das anzeigt ob der eingeloggte User bereits für diesen Slot eingetragen ist.

#### Scenario: Eingetragener Slot
- **WHEN** User X für Slot 12 eingetragen ist und die Dienstbörse aufruft
- **THEN** hat Slot 12 in der Antwort `claimed_by_me: true`

#### Scenario: Nicht eingetragener Slot
- **WHEN** User X nicht für Slot 7 eingetragen ist
- **THEN** hat Slot 7 in der Antwort `claimed_by_me: false`

---

### Requirement: Dienste austragen

`DELETE /api/duty-board/{slotId}/claim` SHALL die Eintragung des eingeloggten Users für den angegebenen Slot aufheben, `slots_filled` dekrementieren und das Dienstkonto aktualisieren.

Wenn der Dienst bereits als `fulfilled` markiert ist, SHALL die API HTTP 409 zurückgeben.

#### Scenario: Erfolgreiches Austragen
- **WHEN** User für Slot eingetragen ist (status=assigned) und DELETE aufruft
- **THEN** wird die Assignment gelöscht, slots_filled dekrementiert, HTTP 204 zurückgegeben

#### Scenario: Austragen eines erfüllten Dienstes
- **WHEN** User für Slot eingetragen ist mit status=fulfilled und DELETE aufruft
- **THEN** gibt die API HTTP 409 zurück

#### Scenario: Austragen ohne Eintragung
- **WHEN** User nicht für den Slot eingetragen ist und DELETE aufruft
- **THEN** gibt die API HTTP 404 zurück

---

### Requirement: Vergangene Spieltage steuerbar

Das Frontend SHALL vergangene Gruppen (`past: true`) standardmäßig ausblenden. Ein Button „Vergangene Spieltage einblenden" zeigt sie an; bei erneutem Klick werden sie wieder ausgeblendet.

Vergangene Slots mit `claimed_by_me: true` zeigen keinen aktiven Button mehr.

#### Scenario: Standardansicht ohne vergangene Spieltage
- **WHEN** User die Dienstbörse öffnet
- **THEN** sind nur Gruppen mit `past: false` sichtbar

#### Scenario: Vergangene Spieltage einblenden
- **WHEN** User auf „Vergangene Spieltage einblenden" klickt
- **THEN** werden alle Gruppen angezeigt, vergangene mit reduziertem Stil

#### Scenario: Kein Eintragen/Austragen für vergangene Dienste
- **WHEN** ein vergangener Slot `claimed_by_me: true` hat
- **THEN** ist der Austragen-Button deaktiviert oder nicht sichtbar
