# spiel-teilnahme Specification

## Purpose
TBD - created by archiving change spiel-teilnahme. Update Purpose after archive.
## Requirements
### Requirement: Spieldetail-Seite zeigt alle Kader-Mitglieder

Die Spieldetail-Seite (`/termine/spiel/{id}` bzw. `/termine/ereignis/{id}`) SHALL alle Mitglieder aus `kader_members` und `kader_extended_members` der zugehörigen Teams anzeigen, gefiltert nach Cross-Team-Opt-In (siehe `profile-cross-team-visibility`).

`GET /api/games/{id}/participants` SHALL `404 Not Found` zurückgeben, wenn der Caller das Game gemäß `auth.UserCanSeeGame` nicht sehen darf — auch dann, wenn das Game existiert. Funktionsträger (admin/trainer/sportliche_leitung/vorstand) sehen weiterhin 200.

#### Scenario: Caller ohne Team-Bezug erhält 404

- **WHEN** ein Standard-Nutzer ohne Mitgliedschaft in einem der Event-Teams (auch nicht über Kinder) `GET /api/games/{id}/participants` aufruft
- **THEN** antwortet der Server mit 404 (nicht 200 + leere Liste)

#### Scenario: Caller mit Team-Bezug erhält gefilterte Liste

- **WHEN** ein Spieler aus Team A `GET /api/games/{id}/participants` für ein Multi-Team-Event aufruft
- **THEN** antwortet der Server mit 200 und der gemäß `profile-cross-team-visibility` gefilterten Liste

#### Scenario: Funktionsträger erhält volle Liste

- **WHEN** ein Trainer `GET /api/games/{id}/participants` für ein beliebiges Event aufruft
- **THEN** antwortet der Server mit 200 und allen Mitgliedern aller Teams

