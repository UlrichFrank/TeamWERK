## ADDED Requirements

### Requirement: Auto-Confirm gilt nur für reguläre Kader-Mitglieder

Das System SHALL bei opt-out-Spielen (`rsvp_opt_out = 1`) die automatische Zusage (`my_rsvp: "confirmed"`) nur für Mitglieder setzen, die im regulären Kader (`kader_members`) eines der am Spiel beteiligten Teams eingetragen sind. Mitglieder, die ausschließlich über `kader_extended_members` an einem Spiel beteiligt sind, erhalten keinen Auto-Confirm und müssen explizit zusagen.

#### Scenario: Opt-out greift nicht bei extended-only Mitglied

- **WHEN** ein Spieler nur über `kader_extended_members` dem Team eines Spiels zugeordnet ist
- **WHEN** das Spiel hat `rsvp_opt_out = 1`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: null` zurück

#### Scenario: Opt-out greift weiterhin für reguläres Mitglied

- **WHEN** ein Spieler über `kader_members` dem Team eines Spiels zugeordnet ist
- **WHEN** das Spiel hat `rsvp_opt_out = 1`
- **WHEN** der Spieler hat keinen `game_responses`-Eintrag
- **THEN** gibt `GET /api/games/my` für dieses Spiel `my_rsvp: "confirmed"` zurück
