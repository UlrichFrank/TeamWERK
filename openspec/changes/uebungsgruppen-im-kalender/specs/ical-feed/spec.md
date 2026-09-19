## ADDED Requirements

### Requirement: Übungsgruppen-Termine im Feed

Das System SHALL Trainingstermine von Übungsgruppen (`kader.kind='practice'`, erkennbar an
`training_sessions.team_id IS NULL`) im iCal-Feed ausgeben, wenn der Token
`include_practice_groups=true` trägt.

Die Zuordnung zum Feed-Nutzer erfolgt über dieselbe Kader-Zugehörigkeit wie bei
Mannschaftsterminen (`kader_members`, `kader_extended_members`, `kader_trainers`), aufgelöst
über `training_sessions.kader_id`. Der Funktionsträger-Bypass bleibt auch hier ausgeschlossen.

`include_practice_groups` und `include_training` sind voneinander unabhängig:
`include_training` steuert ausschließlich Termine mit `team_id IS NOT NULL`,
`include_practice_groups` ausschließlich Termine mit `team_id IS NULL`.

Beim Anlegen eines neuen Tokens ist `include_practice_groups` wie die übrigen Toggles
standardmäßig aktiv; bestehende Tokens erhalten den Wert `true` per Migration.

#### Scenario: Übungsgruppen-Termin erscheint im Feed

- **WHEN** ein User über `kader_members` an einer Übungsgruppe hängt, für die ein
  `training_sessions`-Eintrag mit `status='active'` existiert, und sein Token
  `include_practice_groups=true` trägt
- **THEN** enthält der Feed ein VEVENT mit `UID:training-<id>@teamwerk`

#### Scenario: Beschriftung eines Übungsgruppen-Termins

- **WHEN** ein Übungsgruppen-Termin im Feed ausgegeben wird und die Gruppe
  `kader.name = "Förderkinder 2016"` heißt
- **THEN** lautet das `SUMMARY` des VEVENT `Training: Förderkinder 2016`
- **AND** der Name stammt aus `kader.name`, nicht aus `training_sessions.title`

#### Scenario: include_practice_groups=false filtert Übungsgruppen heraus

- **WHEN** ein Token `include_practice_groups=false` und `include_training=true` trägt und
  der User sowohl an einer Mannschaft als auch an einer Übungsgruppe hängt, für die je ein
  Trainingstermin existiert
- **THEN** enthält der Feed den Mannschaftstermin
- **AND** enthält der Feed den Übungsgruppen-Termin nicht

#### Scenario: include_training=false lässt Übungsgruppen unberührt

- **WHEN** ein Token `include_training=false` und `include_practice_groups=true` trägt und
  der User sowohl an einer Mannschaft als auch an einer Übungsgruppe hängt, für die je ein
  Trainingstermin existiert
- **THEN** enthält der Feed den Übungsgruppen-Termin
- **AND** enthält der Feed den Mannschaftstermin nicht

## MODIFIED Requirements

### Requirement: Token-Verwaltung

Das System SHALL pro User genau ein Calendar-Token verwalten. Ein Token ist ein UUID-v4-String, der in `calendar_tokens` zusammen mit 6 Boolean-Toggles gespeichert wird (`include_heim`, `include_auswaerts`, `include_training`, `include_generisch`, `include_duty`, `include_practice_groups`). Jeder authentifizierte Nutzer kann sein Token anlegen, seine Einstellungen ändern oder das Token löschen.

`POST /api/calendar/token` ist idempotent: existiert bereits ein Token für den User, werden nur die Einstellungen aktualisiert; der Token-Wert bleibt unverändert. Bei Neuanlage wird ein UUID v4 via `crypto/rand` generiert.

#### Scenario: Erstes Token anlegen

- **WHEN** ein authentifizierter Nutzer `POST /api/calendar/token` mit `{"include_heim": true, "include_auswaerts": true, "include_training": true, "include_generisch": true, "include_duty": true, "include_practice_groups": true}` aufruft
- **THEN** antwortet das System mit HTTP 200 und `{"token": "<uuid>", "include_heim": true, ...}`
- **AND** ein Eintrag in `calendar_tokens` mit dem generierten UUID und den Toggles existiert

#### Scenario: Einstellungen ändern (Token bleibt gleich)

- **WHEN** ein Nutzer mit bestehendem Token `POST /api/calendar/token` mit geänderten Toggles aufruft
- **THEN** antwortet das System mit HTTP 200 und den aktualisierten Einstellungen
- **AND** der `token`-Wert in der Response ist identisch zum bisherigen Token

#### Scenario: Token abrufen (existiert)

- **WHEN** ein Nutzer `GET /api/calendar/token` aufruft und ein Token besitzt
- **THEN** antwortet das System mit HTTP 200 und `{"token": "<uuid>", "include_heim": ..., "include_practice_groups": ...}`

#### Scenario: Token abrufen (existiert nicht)

- **WHEN** ein Nutzer `GET /api/calendar/token` aufruft und kein Token besitzt
- **THEN** antwortet das System mit HTTP 404

#### Scenario: Token löschen

- **WHEN** ein Nutzer `DELETE /api/calendar/token` aufruft
- **THEN** antwortet das System mit HTTP 204
- **AND** der Feed-Endpunkt unter dem bisherigen Token-Pfad liefert danach HTTP 404

#### Scenario: Nicht-authentifizierter Zugriff auf Token-Management

- **WHEN** ein nicht eingeloggter Client `GET /api/calendar/token` aufruft
- **THEN** antwortet das System mit HTTP 401

#### Scenario: Bestandstoken nach der Migration

- **WHEN** ein vor der Migration angelegtes Token abgerufen wird
- **THEN** trägt es `include_practice_groups: true`

### Requirement: Konfigurierbare Feed-Inhalte

Das System SHALL die im Token gespeicherten Toggles beim Feed-Abruf auswerten. Ein
deaktivierter Toggle bewirkt, dass die entsprechenden Events nicht im iCal ausgegeben werden.

Spiele und Trainings werden dem User über seine Kader-Zugehörigkeit zugeordnet. Als
Zugehörigkeit zählen `kader_members` (regulär), `kader_extended_members` (erweiterter Kader)
und `kader_trainers` (Trainer) — dieselbe Menge, die `auth.GameVisibilityClause` für den
Spielplan auflöst. Für Spiele muss das Team via `game_teams` am Spiel hängen und die Saison
des Kaders mit der des Spiels übereinstimmen; für Trainings muss der Kader des Nutzers der
`kader_id` einer `training_sessions`-Row mit `status='active'` entsprechen. Die Auflösung
läuft für Trainings über `training_sessions.kader_id` und **nicht** über `team_id` — sonst
fielen Übungsgruppen strukturell heraus, deren `team_id` NULL ist.

Den Funktionsträger-Bypass aus `auth` (admin/trainer/sportliche_leitung/vorstand sehen alle
Events der Saison) übernimmt der Feed bewusst NICHT — ein Vorstand hätte sonst den gesamten
Vereinsspielplan im privaten Kalender. Die Auflösung über `family_links` entfällt ebenfalls,
weil Eltern für jedes Kind einen eigenen Kind-Token bekommen.

Hängt ein User über mehrere Kader an demselben Termin, erscheint er trotzdem nur einmal im
Feed (die UID ist pro Spiel bzw. Training eindeutig); für die Beschriftung schlägt dabei eine
reguläre Zugehörigkeit die erweiterte. Damit entfällt für diesen Nutzer auch der
Aufstellungsstatus — er hängt regulär am Termin.

Dienste werden dem User zugeordnet wenn ein Eintrag in `duty_assignments` mit
`user_id = user_id_des_tokens` und `status IN ('assigned', 'fulfilled')` existiert.

#### Scenario: Termine aus dem erweiterten Kader sind gekennzeichnet

- **WHEN** ein User über `kader_extended_members` am Kader eines Teams hängt und für dieses
  Team ein Spiel und ein Training existieren
- **THEN** enthält der Feed beide Events
- **AND** die Mannschaft trägt im `SUMMARY` den Zusatz `<team_name> · erw. Kader` (z. B.
  `SUMMARY:Training: mB1 · erw. Kader`)
- **AND** das Spiel-Event trägt zusätzlich den Aufstellungsstatus (z. B.
  `SUMMARY:Heim: Team (mB1 · erw. Kader · aufgestellt) – <Gegner>`)

#### Scenario: Vereinsfunktion allein zieht keine fremden Termine in den Feed

- **WHEN** ein User die Vereinsfunktion `trainer` hat, aber an keinem Kader des Teams hängt,
  dem ein Spiel zugeordnet ist
- **THEN** enthält der Feed dieses Spiel nicht

#### Scenario: include_training=false filtert training_sessions heraus

- **WHEN** ein Token mit `include_training=false` existiert und der User Mitglied eines Teams
  ist, für das eine `training_sessions`-Row existiert
- **THEN** enthält der Feed kein VEVENT mit `UID:training-*` für diesen Mannschaftstermin
- **AND** alle anderen aktivierten Event-Typen sind weiterhin enthalten

#### Scenario: include_duty=false filtert Dienste heraus

- **WHEN** ein Token mit `include_duty=false` existiert und der User einen zugewiesenen Dienst
  hat
- **THEN** enthält der Feed kein VEVENT mit `UID:duty-*`

#### Scenario: Alle Toggles deaktiviert — leerer Feed

- **WHEN** alle 6 Toggles auf false gesetzt sind
- **THEN** enthält der Feed einen validen VCALENDAR-Rahmen aber keine VEVENTs
