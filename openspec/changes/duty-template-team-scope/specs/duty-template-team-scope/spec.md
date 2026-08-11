## ADDED Requirements

### Requirement: Vorlagen-Item kann auf einzelne Kaderteams eingeschränkt werden

Das System SHALL pro `game_template_items`-Eintrag eine optionale Liste von `teams.id`
(`team_ids`) speichern, die einschränkt, für welche Teams eines Spiels bei der
Auto-Regeneration ein Dienst-Slot aus diesem Item entsteht. Ist `team_ids` NULL oder ein
leeres Array, MUSS das Item weiterhin für **alle** Teams des Spiels gelten (unverändertes
Bestandsverhalten). Ist `team_ids` nicht leer, DARF ein Slot aus diesem Item **nur** für
Teams entstehen, deren `teams.id` in der Liste enthalten ist.

#### Scenario: Item ohne Team-Einschränkung (Default)
- **WHEN** ein Vorlagen-Item `team_ids: null` (bzw. fehlend) hat
- **AND** ein Spiel mit diesem Template und den Teams mA1 und mC1 regeneriert wird
- **THEN** entsteht der Slot aus diesem Item für **beide** Teams (mA1 und mC1)

#### Scenario: Item mit Team-Einschränkung bei Mehrteam-Spiel
- **WHEN** ein Vorlagen-Item `team_ids: [mA1.id]` hat
- **AND** ein Spiel mit diesem Template und den Teams mA1 und mC1 regeneriert wird
- **THEN** entsteht der Slot aus diesem Item **nur** für mA1
- **AND** für mC1 entsteht **kein** Slot aus diesem Item

#### Scenario: Andere Items derselben Vorlage bleiben unbeeinflusst
- **WHEN** eine Vorlage ein team-eingeschränktes Item ("Kamera", `team_ids: [mA1.id, mB1.id]`)
  und ein uneingeschränktes Item ("Kasse", `team_ids: null`) enthält
- **AND** ein Spiel mit den Teams mA1 und mC1 regeneriert wird
- **THEN** entsteht "Kamera" nur für mA1
- **AND** "Kasse" entsteht für mA1 **und** mC1

#### Scenario: Zählung berücksichtigt die Team-Einschränkung
- **WHEN** ein team-eingeschränktes Item bei einem Spiel mit 3 Teams regeneriert wird, von
  denen nur 1 Team in der Item-Liste steht
- **THEN** meldet `RegenSummary.Created[].Count` für dieses Item die Anzahl der tatsächlich
  getroffenen Teams (1), nicht die Gesamtzahl der Spiel-Teams (3)

### Requirement: Server validiert `team_ids` gegen existierende Teams

`PUT /api/duty-templates/{id}` SHALL jede in `team_ids` eines Items übergebene ID gegen
`teams.id` prüfen. Enthält die Liste eine unbekannte ID, MUSS der Request mit HTTP 400 und
Fehlercode `invalid_team` abgelehnt werden, ohne dass die Vorlage verändert wird. Die Prüfung
MUSS unabhängig davon greifen, ob das referenzierte Team der aktuell aktiven Saison
zugeordnet ist — ein Team, das nur in einer vergangenen oder noch nicht angelegten Saison im
Kader steht, bleibt ein gültiger Wert.

#### Scenario: Unbekannte Team-ID wird abgelehnt
- **WHEN** Vorstand `PUT /api/duty-templates/{id}` mit einem Item sendet, dessen `team_ids`
  eine nicht existierende `teams.id` enthält
- **THEN** antwortet das System mit HTTP 400 und `{"error": "invalid_team"}`
- **AND** die gespeicherte Vorlage bleibt unverändert

#### Scenario: Team aus vergangener Saison bleibt gültig
- **WHEN** Vorstand `PUT /api/duty-templates/{id}` mit einem Item sendet, dessen `team_ids`
  eine `teams.id` enthält, die in der aktiven Saison keinem `kader`-Eintrag zugeordnet ist,
  aber in `teams` existiert
- **THEN** antwortet das System mit HTTP 200 und speichert die Zuordnung

#### Scenario: Standard-Nutzer ohne Vorstand-Funktion
- **WHEN** ein Nutzer ohne Vereinsfunktion `vorstand` (und ohne Rolle `admin`)
  `PUT /api/duty-templates/{id}` aufruft
- **THEN** antwortet das System mit HTTP 403

### Requirement: Slot-Vorschau spiegelt die Team-Einschränkung

Die Vorschau (`GET /api/duty-templates/{id}/preview`) SHALL dieselbe Team-Einschränkung
anwenden wie die Auto-Regeneration, damit ein Eintrag, der für die gewählten Teams keinen
Slot erzeugen würde, in der Vorschau nicht erscheint. Dazu MUSS die Vorschau die Teams des
geplanten Events kennen: über den Query-Parameter `team_ids` (komma-separierte `teams.id`)
oder — falls dieser fehlt und `game_id` gesetzt ist — abgeleitet aus `game_teams`. Sind
weder `team_ids` noch `game_id` gesetzt, MUSS die Vorschau ungefiltert bleiben
(Bestandsverhalten).

Die Einschränkung DARF NUR für Vorlagen vom Typ `heim`/`auswärts` angewandt werden. Bei
`template_type='generisch'` ignoriert die Regeneration `team_ids` — die Vorschau MUSS das
ebenso tun, sonst verschwiege sie einen Slot, der real entsteht.

#### Scenario: Eingeschränkter Eintrag verschwindet aus der Vorschau
- **WHEN** ein Vorlagen-Item `team_ids: [mA1.id]` hat
- **AND** die Vorschau mit `team_ids=<mA2.id>` abgerufen wird
- **THEN** enthält die Antwort diesen Eintrag **nicht**

#### Scenario: Eingeschränkter Eintrag bleibt bei passendem Team sichtbar
- **WHEN** ein Vorlagen-Item `team_ids: [mA1.id, mB1.id]` hat
- **AND** die Vorschau mit `team_ids=<mA2.id>,<mB1.id>` abgerufen wird
- **THEN** enthält die Antwort diesen Eintrag (mB1 trifft die Allowlist)

#### Scenario: Uneingeschränkter Eintrag bleibt immer sichtbar
- **WHEN** ein Vorlagen-Item keine `team_ids` hat
- **AND** die Vorschau mit einem beliebigen `team_ids` abgerufen wird
- **THEN** enthält die Antwort diesen Eintrag

#### Scenario: Ohne Team-Angabe bleibt die Vorschau ungefiltert
- **WHEN** die Vorschau ohne `team_ids` und ohne `game_id` abgerufen wird
- **THEN** enthält die Antwort alle Einträge der Vorlage, auch team-eingeschränkte

#### Scenario: Generische Vorlage wird nicht gefiltert
- **WHEN** eine Vorlage mit `template_type='generisch'` ein Item mit `team_ids: [mA1.id]` hat
- **AND** die Vorschau mit `team_ids=<mA2.id>` abgerufen wird
- **THEN** enthält die Antwort diesen Eintrag, weil die Regeneration ihn ebenfalls erzeugt

### Requirement: Vorlagen-Editor bietet nur Kaderteams der aktiven Saison zur Auswahl an

Das Frontend SHALL im Vorlagen-Editor pro Item eine Team-Auswahl anbieten, deren Optionen auf
die Kaderteams der **aktiven** Saison beschränkt sind (Teams mit einem `kader`-Eintrag, dessen
`season_id` der aktiven Saison entspricht). Teams ohne aktuellen Kader-Eintrag SHALL NICHT in
der Auswahlliste erscheinen, auch wenn bereits gespeicherte `team_ids` eines Items auf sie
verweisen.

Die Auswahl SHALL NICHT angeboten werden, wenn die Vorlage vom Typ `generisch` ist — dort
ignoriert die Regeneration `team_ids`, ein Bedienelement ohne Wirkung wäre irreführend.

#### Scenario: Editor zeigt nur aktive Kaderteams
- **WHEN** ein Vorstand ein Vorlagen-Item öffnet
- **THEN** enthält die Team-Auswahlliste nur Teams, die einen `kader`-Eintrag der aktiven
  Saison haben, dargestellt mit ihrem berechneten Kurznamen (z.B. "mA1")

#### Scenario: Generische Vorlage bietet keine Team-Auswahl
- **WHEN** ein Vorstand eine Vorlage mit `template_type='generisch'` öffnet
- **THEN** erscheint die Kaderteams-Auswahl nicht

#### Scenario: Gespeichertes Team ist in der aktiven Saison nicht mehr im Kader
- **WHEN** ein Item bereits `team_ids` enthält, von denen eine `teams.id` in der aktiven
  Saison keinen `kader`-Eintrag mehr hat
- **THEN** bleibt die Zuordnung beim Speichern erhalten (siehe vorheriges Requirement)
- **AND** das Team erscheint nicht in der Auswahlliste zum Hinzufügen weiterer Team-Bindungen
