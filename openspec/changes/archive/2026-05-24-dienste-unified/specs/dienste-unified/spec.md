## ADDED Requirements

### Requirement: Kader-basierter Slot-Filter
Das System SHALL Dienst-Slots nach Kader-Zugehörigkeit filtern statt nach `team_memberships`. Spieler und Elternteile sehen Slots ihrer Kader-Teams. Trainer sehen Slots der Teams, für die sie als Trainer eingetragen sind. Admins sehen alle Slots der aktiven Saison.

#### Scenario: Spieler sieht Slots seines Kader-Teams
- **WHEN** ein Nutzer mit Rolle `spieler` `GET /api/duty-board` aufruft
- **THEN** werden Slots aller Teams zurückgegeben, in denen der Nutzer via `kader_members` in der aktiven Saison eingetragen ist

#### Scenario: Elternteil sieht Slots der Kinder-Teams
- **WHEN** ein Nutzer mit Rolle `elternteil` `GET /api/duty-board` aufruft
- **THEN** werden Slots aller Teams zurückgegeben, in denen die verknüpften Kinder (`family_links`) via `kader_members` eingetragen sind

#### Scenario: Trainer sieht Slots seiner verwalteten Teams
- **WHEN** ein Nutzer mit Rolle `trainer` `GET /api/duty-board` aufruft
- **THEN** werden Slots aller Teams zurückgegeben, für die der Nutzer via `kader_trainers` als Trainer eingetragen ist

#### Scenario: Admin sieht alle Slots
- **WHEN** ein Nutzer mit Rolle `admin` `GET /api/duty-board` aufruft
- **THEN** werden alle Slots der aktiven Saison zurückgegeben, ungefiltert

#### Scenario: Trainer ohne Kader-Eintrag sieht leere Liste
- **WHEN** ein Trainer für kein Team als `kader_trainers`-Eintrag existiert
- **THEN** gibt `GET /api/duty-board` eine leere Liste zurück (kein Fehler)

### Requirement: Meine-Ansicht für Organisatoren
Das System SHALL für Nutzer mit Rolle `admin` oder `trainer` eine gefilterte Ansicht bereitstellen, die nur Slots zeigt, für die der anfragende Nutzer selbst eine aktive Zuteilung hat.

#### Scenario: Meine-Ansicht zeigt nur eigene Zuteilungen
- **WHEN** `GET /api/duty-board?view=mine` von einem Admin oder Trainer aufgerufen wird
- **THEN** werden nur Slots zurückgegeben, bei denen `duty_assignments.user_id = current_user` und `status != 'cancelled'`

#### Scenario: Meine-Ansicht ohne eigene Zuteilungen
- **WHEN** der Nutzer keine Zuteilungen hat und `GET /api/duty-board?view=mine` aufruft
- **THEN** wird eine leere Liste zurückgegeben (kein Fehler)

### Requirement: Vereinheitlichte Dienste-Seite
Das System SHALL eine einzelne Seite `/dienste` bereitstellen, die für alle authentifizierten Rollen zugänglich ist und rollenabhängige Aktionen zeigt.

#### Scenario: Mitglied kann sich eintragen
- **WHEN** ein Nutzer mit Rolle `spieler` oder `elternteil` auf `/dienste` navigiert und ein Slot mit freien Plätzen sichtbar ist
- **THEN** wird ein „Eintragen"-Button angezeigt und der Nutzer kann sich für den Slot beanspruchen

#### Scenario: Mitglied kann sich austragen
- **WHEN** ein Nutzer sich für einen nicht vergangenen Slot eingetragen hat
- **THEN** wird ein „Austragen"-Button angezeigt

#### Scenario: Admin/Trainer sieht Management-Aktionen
- **WHEN** ein Nutzer mit Rolle `admin` oder `trainer` auf `/dienste` navigiert und Zuteilungen eines Slots aufklappt
- **THEN** sind für Zuteilungen mit Status `pending` die Aktionen „Erfüllt" und „Geldersatz" verfügbar

#### Scenario: Admin/Trainer sieht Toggle
- **WHEN** ein Nutzer mit Rolle `admin` oder `trainer` auf `/dienste` navigiert
- **THEN** sind zwei Toggle-Optionen sichtbar: „Meine Dienste" und „Alle Dienste"

#### Scenario: Spieler/Elternteil sieht keinen Toggle
- **WHEN** ein Nutzer mit Rolle `spieler` oder `elternteil` auf `/dienste` navigiert
- **THEN** ist kein Meine/Alle-Toggle sichtbar

### Requirement: Slot-Löschung durch Organisatoren
Das System SHALL Admins und Trainern erlauben, einzelne Dienst-Slots zu löschen. Bei belegten Slots SHALL eine Bestätigung eingeholt werden.

#### Scenario: Löschen eines leeren Slots
- **WHEN** ein Admin oder Trainer auf das Löschen-Symbol eines Slots mit `slots_filled = 0` klickt
- **THEN** wird der Slot ohne Bestätigungsdialog gelöscht und die Liste aktualisiert

#### Scenario: Löschen eines belegten Slots mit Bestätigung
- **WHEN** ein Admin oder Trainer auf das Löschen-Symbol eines Slots mit `slots_filled > 0` klickt
- **THEN** erscheint ein Bestätigungsdialog mit Hinweis auf bestehende Zuteilungen
- **THEN** wird der Slot nur bei Bestätigung gelöscht

#### Scenario: Mitglied sieht keinen Löschen-Button
- **WHEN** ein Nutzer mit Rolle `spieler` oder `elternteil` auf `/dienste` navigiert
- **THEN** ist kein Löschen-Symbol sichtbar

### Requirement: Entfernung des team-assignment-Endpoints
Das System SHALL `POST /api/members/{id}/team-assignment` nicht mehr bereitstellen.

#### Scenario: Endpoint ist nicht mehr erreichbar
- **WHEN** `POST /api/members/{id}/team-assignment` aufgerufen wird
- **THEN** antwortet der Server mit `404 Not Found`
