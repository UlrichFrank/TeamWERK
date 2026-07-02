## ADDED Requirements

### Requirement: Board-Gruppe trägt die Teams ihres Termins

`GET /duty-board` SHALL für jede Board-Gruppe mit `game_id` die Menge der dem Termin zugeordneten Teams (`game_teams`) als `team_ids: number[]` und deren Kurznamen als `team_names: string[]` liefern — einheitlich für Event-Typen `heim`, `auswärts` und `generisch`. Die Reihenfolge von `team_ids` und `team_names` MUSS positionsgleich sein. Bestehende Zugriffskontrollen (Team-Sichtbarkeit, aktive Saison, Audience) bleiben unverändert.

#### Scenario: Mehr-Team-Spiel liefert alle Teams
- **WHEN** ein berechtigter Nutzer `GET /duty-board` aufruft und ein Termin ist Team A und Team B zugeordnet
- **THEN** enthält die zugehörige Gruppe `team_ids` mit beiden IDs (A und B) und `team_names` mit beiden Kurznamen in gleicher Reihenfolge

#### Scenario: Generisches Event trägt Termin-Teams
- **WHEN** ein berechtigter Nutzer `GET /duty-board` aufruft und ein generisches Event ist Team A zugeordnet
- **THEN** enthält die zugehörige Gruppe `team_ids` mit Team A, obwohl die einzelnen `duty_slots.team_id` `NULL` sind

### Requirement: Game-lose Handslots behalten ihr Slot-Team

`GET /duty-board` SHALL für Board-Gruppen ohne `game_id` (manuell angelegte Slots) die Team-Menge aus `duty_slots.team_id` ableiten: `[team_id]`, falls gesetzt, sonst ein leeres Array.

#### Scenario: Handslot mit Team
- **WHEN** ein Slot ohne `game_id` mit gesetztem `team_id = A` existiert
- **THEN** trägt die zugehörige Gruppe `team_ids = [A]`

#### Scenario: Handslot ohne Team
- **WHEN** ein Slot ohne `game_id` und ohne `team_id` existiert
- **THEN** trägt die zugehörige Gruppe ein leeres `team_ids`-Array

### Requirement: Team-Filter matcht per Zugehörigkeit

Das Board `/dienste` SHALL bei aktivem Team-Filter eine Gruppe genau dann anzeigen, wenn das gewählte Team in `team_ids` der Gruppe enthalten ist. Ohne aktiven Team-Filter werden alle berechtigten Gruppen angezeigt.

#### Scenario: Filter auf ein Team eines Mehr-Team-Spiels
- **WHEN** ein Nutzer auf `/dienste` nach Team B filtert und eine Gruppe ist Team A und Team B zugeordnet
- **THEN** bleibt die Gruppe sichtbar

#### Scenario: Filter auf nicht zugeordnetes Team
- **WHEN** ein Nutzer nach Team C filtert und eine Gruppe ist nur Team A und Team B zugeordnet
- **THEN** wird die Gruppe ausgeblendet

### Requirement: Board zeigt alle adressierten Teams

Der Gruppen-Header auf `/dienste` SHALL alle Teams der Gruppe (`team_names`) anzeigen, nicht nur ein einzelnes Team.

#### Scenario: Anzeige mehrerer Teams
- **WHEN** eine Gruppe zwei Teams adressiert
- **THEN** zeigt der Header beide Team-Kurznamen an
