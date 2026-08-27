## ADDED Requirements

### Requirement: Manuell angelegter Dienst-Slot an einem Mehr-Team-Termin trägt kein Team

Wird ein Dienst-Slot über das Spieltag-Detail-Modal (`POST /api/duty-slots` mit gesetztem
`game_id`) an einem Termin angelegt, dem **mehr als ein Team** zugeordnet ist
(`game_teams`), SHALL das Frontend `team_id: null` senden. Nur bei genau einem
zugeordneten Team SHALL dessen ID übertragen werden.

Begründung: `duty_slots.team_id` ist das Sichtbarkeits-Feld, nicht eine
Zuordnungs-Notiz. Ein gesetztes Team schränkt die Dienstbörse auf genau dieses Team ein
(`ds.team_id IN (…)`), während `team_id IS NULL` zusammen mit `game_id` über den
bestehenden Fallback auf **alle** Teams des Termins auflöst — inklusive deren Eltern über
den `eltern`-Audience-Zweig. Ein Slot ohne Team und ohne Spiel bleibt unverändert nur für
Vorstand/Admin sichtbar.

#### Scenario: Slot an Termin mit drei Teams ist für alle drei sichtbar
- **WHEN** ein Vorstand an einem Termin mit den Teams A, B und C einen Dienst-Slot anlegt
- **THEN** wird der Slot mit `team_id = NULL` und gesetztem `game_id` gespeichert
- **AND** erscheint er in `GET /api/duty-board` für Spieler, Trainer und Eltern aller drei Teams

#### Scenario: Slot an Termin mit genau einem Team bleibt team-gebunden
- **WHEN** ein Vorstand an einem Termin mit nur Team A einen Dienst-Slot anlegt
- **THEN** wird der Slot mit `team_id = A` gespeichert
- **AND** sehen Mitglieder anderer Teams ihn nicht
