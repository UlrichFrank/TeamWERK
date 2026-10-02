# Spec Delta

## ADDED Requirements

### Requirement: Ausgetretene Mitglieder gehören zu keiner Standard-Gruppe

Das System SHALL bei der Auflösung **jeder** Standard-Gruppe (Mannschaft `trainer`/`spieler`/`eltern`, „Alle Trainer", Übungsgruppe `trainer`/`spieler`/`eltern`) Mitglieder mit `members.status = 'ausgetreten'` ausschließen. Bei `eltern` entscheidet der Status des **Kindes**: Ein Elternteil gehört zur Gruppe, solange es mindestens ein nicht ausgetretenes Kind in der Gruppe hat. Der Filter MUSS `status <> 'ausgetreten'` lauten, nicht `status = 'aktiv'`. Sonst fielen Mitglieder mit den Status `verletzt`, `pausiert` oder `foerderkind` heraus.

Die Zahl `count` in `GET /api/chat/team-groups` SHALL dieselbe Menge zählen, die der zugehörige `…/members`-Endpoint liefert.

#### Scenario: Ausgetretener Spieler fehlt in der Spieler-Gruppe
- **WHEN** ein Spieler mit `members.status = 'ausgetreten'` noch in `kader_members` eines Kaders der aktiven Saison von T1 steht und `GET /api/chat/team-groups/T1/spieler/members` aufgerufen wird
- **THEN** fehlt er in der Antwort, und `count` der Kachel zählt ihn nicht

#### Scenario: Verletzter Spieler bleibt in der Spieler-Gruppe
- **WHEN** ein Spieler mit `members.status = 'verletzt'` im Kader von T1 steht
- **THEN** ist er in `GET /api/chat/team-groups/T1/spieler/members` enthalten

#### Scenario: Elternteil eines ausgetretenen Kindes fällt heraus
- **WHEN** ein Elternteil über `family_links` nur mit einem ausgetretenen Kind im Kader von T1 verbunden ist
- **THEN** fehlt es in `GET /api/chat/team-groups/T1/eltern/members`

#### Scenario: Ausgetretener Trainer fehlt in „Alle Trainer"
- **WHEN** ein Mitglied mit `status = 'ausgetreten'` noch in `kader_trainers` der aktiven Saison steht
- **THEN** fehlt es in `GET /api/chat/team-groups/0/alle_trainer/members`
