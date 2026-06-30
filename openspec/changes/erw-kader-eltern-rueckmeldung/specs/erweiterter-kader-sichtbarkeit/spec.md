## ADDED Requirements

### Requirement: Eltern abgesetzter Spieler sehen das erw.-Kader-Team im Teamfilter

`GET /api/teams` SHALL das Team eines abgesetzten Spielers auch dann zurückgeben, wenn der anfragende User **Elternteil** des Spielers ist (verknüpft via `family_links`) und das Kind nur über `kader_extended_members` im Team geführt ist. Damit erscheint das Team im Teamfilter auf `/termine`. Maßgeblich ist die aktive Saison.

#### Scenario: Elternteil eines erw.-Kader-Kindes erhält das Team

- **WHEN** ein Elternteil ein Kind via `family_links` verknüpft hat, das nur über `kader_extended_members` einem Team der aktiven Saison zugeordnet ist
- **WHEN** das Elternteil `GET /api/teams` aufruft
- **THEN** enthält die Teamliste dieses Team

#### Scenario: Elternteil ohne Kader-Bezug erhält das Team nicht

- **WHEN** ein Elternteil ein Kind verknüpft hat, das weder in `kader_members` noch in `kader_extended_members` eines Teams steht
- **WHEN** das Elternteil `GET /api/teams` aufruft
- **THEN** enthält die Teamliste dieses Team nicht

#### Scenario: Stammkader-Eltern-Zugang bleibt unverändert

- **WHEN** ein Elternteil ein Kind im Stammkader (`kader_members`) eines Teams hat
- **THEN** enthält `GET /api/teams` dieses Team weiterhin (Verhalten unverändert)
