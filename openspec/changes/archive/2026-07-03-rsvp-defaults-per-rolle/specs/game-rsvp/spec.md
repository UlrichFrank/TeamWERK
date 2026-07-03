## ADDED Requirements

### Requirement: RSVP-Voreinstellung pro Rolle (Spiele)

Jedes Spiel SHALL für Stammkader-Spieler und den Erweiterten Kader **unabhängig** eine Voreinstellung tragen: `confirmed` („standardmäßig zugesagt"), `declined` („standardmäßig abgesagt") oder `none` („keine automatische Rückmeldung"). Die Spalten heißen `rsvp_default_players` und `rsvp_default_extended` (TEXT NOT NULL DEFAULT `'none'` mit `CHECK` auf die drei Werte). Trainer haben KEINE Voreinstellungs-Spalte und werden weiterhin hart als `confirmed` behandelt.

Die Voreinstellung wird virtuell angewendet: fehlt zu einem Mitglied eine `game_responses`-Row, liefert die API den passenden Default-Status. Es werden dabei KEINE Rows in `game_responses` erzeugt.

#### Scenario: `my_rsvp` reflektiert Default für Stammkader-Spieler
- **WHEN** ein Spiel `rsvp_default_players='confirmed'` hat und ein Spieler ist im Stammkader eines beteiligten Teams ohne `game_responses`-Eintrag
- **THEN** liefert `GET /api/games/my` für dieses Spiel `my_rsvp='confirmed'`

#### Scenario: `my_rsvp` bleibt null bei `extended='none'` für Extended-only-Mitglied
- **WHEN** ein Spiel `rsvp_default_players='confirmed'` und `rsvp_default_extended='none'` hat
- **AND** ein Mitglied ist nur über `kader_extended_members` beteiligt und hat keine Response
- **THEN** liefert `GET /api/games/my` `my_rsvp=null`

#### Scenario: `my_rsvp` = `'declined'` bei `extended='declined'` für Extended-only-Mitglied
- **WHEN** ein Spiel `rsvp_default_extended='declined'` hat und ein Erweitertes-Kader-Mitglied hat keine Response
- **THEN** liefert `GET /api/games/my` `my_rsvp='declined'`

#### Scenario: Aktive Response überschreibt Default
- **WHEN** dasselbe Spiel `rsvp_default_players='confirmed'` hat und ein Stammkader-Spieler hat `game_responses.status='maybe'`
- **THEN** liefert `GET /api/games/my` `my_rsvp='maybe'`

---

### Requirement: Konflikt-Sperre „standardmäßig abgesagt" plus „Grund erforderlich" (Spiele)

Das System SHALL `POST /api/games` und `PUT /api/games/{id}` mit HTTP 400 (`{"error":"invalid_rsvp_settings"}`) ablehnen, wenn der Payload gleichzeitig `rsvp_require_reason=1` UND (`rsvp_default_players='declined'` ODER `rsvp_default_extended='declined'`) enthält.

#### Scenario: Spiel-Anlage mit widersprüchlicher Kombination
- **WHEN** `POST /api/games` mit `{"rsvp_default_players":"declined","rsvp_require_reason":1, …}` gerufen wird
- **THEN** antwortet der Server mit HTTP 400 und dem Body enthält `"invalid_rsvp_settings"`
- **THEN** wird KEIN Spiel angelegt

#### Scenario: Spiel-Update mit widersprüchlicher Kombination auf Erweitertem Kader
- **WHEN** `PUT /api/games/{id}` mit `{"rsvp_default_extended":"declined","rsvp_require_reason":1}` gerufen wird
- **THEN** antwortet der Server mit HTTP 400 und das Spiel bleibt unverändert

---

### Requirement: Header-Zähler bezieht Voreinstellungen ein (Spiele)

`GET /api/games/{id}`, `GET /api/games` und `GET /api/games/my` SHALL in `confirmed_count`, `declined_count` und `maybe_count` Mitglieder mit virtuellem Default-Status ihrer Rolle mitzählen — nach der Formel `COALESCE(game_responses.status, game.rsvp_default_<role>)`, wobei `'none'` nirgends mitzählt. Trainer bleiben (unverändert) aus allen drei Zählern ausgeschlossen.

#### Scenario: Zähler bei `players='confirmed'` ohne Responses
- **WHEN** ein Spiel `rsvp_default_players='confirmed'` hat und 3 Stammkader-Spieler ohne Response existieren
- **THEN** enthält der Spiel-Response `confirmed_count=3`, `declined_count=0`

#### Scenario: Zähler bei `extended='declined'` ohne Responses
- **WHEN** ein Spiel `rsvp_default_extended='declined'` hat und 2 Erweiterte-Kader-Mitglieder ohne Response existieren
- **THEN** enthält der Spiel-Response `declined_count=2`

## MODIFIED Requirements

### Requirement: Auto-Confirm gilt nur für reguläre Kader-Mitglieder

Das System SHALL die automatische Zusage nur dann setzen, wenn die Voreinstellung des Spiels für die jeweilige Rolle des Mitglieds `'confirmed'` ist:

- `rsvp_default_players='confirmed'` greift ausschließlich für Mitglieder, die im regulären Kader (`kader_members`) eines der am Spiel beteiligten Teams eingetragen sind.
- `rsvp_default_extended='confirmed'` greift ausschließlich für Mitglieder, die nur über `kader_extended_members` beteiligt sind (nicht bereits im Stammkader).

Analog gilt `'declined'` als virtuelle Absage und `'none'` bedeutet keine Voreinstellung (Mitglied muss selbst antworten).

#### Scenario: Auto-Confirm greift nicht bei extended-only Mitglied, wenn `extended='none'`

- **WHEN** ein Spieler nur über `kader_extended_members` dem Team eines Spiels zugeordnet ist
- **WHEN** das Spiel hat `rsvp_default_players='confirmed'` und `rsvp_default_extended='none'`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: null` zurück

#### Scenario: Auto-Confirm greift für reguläres Mitglied bei `players='confirmed'`

- **WHEN** ein Spieler über `kader_members` dem Team eines Spiels zugeordnet ist
- **WHEN** das Spiel hat `rsvp_default_players='confirmed'`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: "confirmed"` zurück

#### Scenario: Extended-only-Mitglied wird bei `extended='confirmed'` autoconfirmed

- **WHEN** ein Spieler nur über `kader_extended_members` beteiligt ist
- **WHEN** das Spiel hat `rsvp_default_extended='confirmed'`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: "confirmed"` zurück
