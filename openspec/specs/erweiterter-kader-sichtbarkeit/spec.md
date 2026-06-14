### Requirement: user_accessible_teams enthält erweiterte Kader-Mitglieder

Das System SHALL die View `user_accessible_teams` so erweitern, dass Mitglieder, die über `kader_extended_members` einem Kader zugeordnet sind, als zugehörig zum Team und zur Saison gelten.

#### Scenario: Abgesetzter Spieler kann Teamseite öffnen

- **WHEN** ein Spieler nur über `kader_extended_members` einem Kader zugeordnet ist (nicht über `kader_members`)
- **THEN** gibt `GET /api/teams` dieses Team in der Teamliste zurück
- **THEN** gibt `GET /api/teams/{id}/roster` HTTP 200 zurück (kein 403)

#### Scenario: Regulärer Kader-Zugang bleibt unverändert

- **WHEN** ein Spieler nur über `kader_members` einem Kader zugeordnet ist
- **THEN** ändert sich sein Zugang zu `GET /api/teams/{id}/roster` nicht

### Requirement: Roster-API liefert erweiterte Spieler separat

Das System SHALL `GET /api/teams/{id}/roster` um das Feld `extended_players` erweitern. Das Feld enthält alle Mitglieder aus `kader_extended_members` des Teams in der aktiven Saison mit den gleichen Feldern wie `players` (`userId`, `name`, `jerseyNumber`).

#### Scenario: Team mit abgesetzten Spielern

- **WHEN** ein Team mindestens einen abgesetzten Spieler hat
- **THEN** enthält die Antwort das Feld `extended_players` als Array mit diesem Spieler
- **THEN** erscheint der Spieler nicht im Feld `players`

#### Scenario: Team ohne abgesetzte Spieler

- **WHEN** ein Team keine abgesetzten Spieler hat
- **THEN** enthält die Antwort das Feld `extended_players` als leeres Array `[]`

### Requirement: Abgesetzte Spieler sehen Spiele ihres erweiterten Teams

Das System SHALL `GET /api/games/my` so anpassen, dass Spiele für Teams, in denen der Spieler nur über `kader_extended_members` geführt ist, ebenfalls zurückgegeben werden.

#### Scenario: Abgesetzter Spieler sieht Teamspiel

- **WHEN** ein Spieler nur im erweiterten Kader von Team B ist
- **WHEN** Team B hat ein Spiel
- **THEN** enthält `GET /api/games/my` dieses Spiel

#### Scenario: Abgesetzter Spieler kann Spieldetail öffnen

- **WHEN** ein Spieler im erweiterten Kader von Team B ist
- **THEN** gibt `GET /api/games/{id}/participants` für ein Spiel von Team B HTTP 200 zurück
- **THEN** erscheint der Spieler in der Teilnehmerliste mit `is_extended: true`

### Requirement: Kein Auto-Confirm für erweiterte Kader-Mitglieder

Das System SHALL die opt-out-Auto-Zusage (`rsvp_opt_out = 1`) ausschließlich auf Mitglieder anwenden, die im **regulären** Kader (`kader_members`) eines der beteiligten Teams des Spiels eingetragen sind. Mitglieder, die für dieses Spiel ausschließlich über `kader_extended_members` beteiligt sind, erhalten keinen automatischen `my_rsvp: "confirmed"`.

#### Scenario: Reguläres Mitglied wird bei opt-out auto-confirmed

- **WHEN** ein Spieler im regulären Kader von Team A ist
- **WHEN** ein Spiel von Team A hat `rsvp_opt_out = 1`
- **WHEN** der Spieler hat noch keine `game_responses`-Eintrag für dieses Spiel
- **THEN** gibt `GET /api/games/my` `my_rsvp: "confirmed"` für dieses Spiel zurück

#### Scenario: Abgesetzter Spieler erhält KEINEN Auto-Confirm

- **WHEN** ein Spieler nur im erweiterten Kader von Team B ist (nicht im regulären Kader)
- **WHEN** ein Spiel von Team B hat `rsvp_opt_out = 1`
- **WHEN** der Spieler hat noch keine `game_responses`-Eintrag für dieses Spiel
- **THEN** gibt `GET /api/games/my` `my_rsvp: null` für dieses Spiel zurück

#### Scenario: Mischfall — regular + extended über verschiedene Teams

- **WHEN** ein Spieler im regulären Kader von Team A und im erweiterten Kader von Team B ist
- **WHEN** ein Spiel hat **beide** Teams (Team A und Team B) und `rsvp_opt_out = 1`
- **THEN** gibt `GET /api/games/my` `my_rsvp: "confirmed"` zurück (reguläre Mitgliedschaft in Team A greift)
