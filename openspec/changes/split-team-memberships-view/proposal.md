## Why

Der `team_memberships`-View vereint in einer UNION sowohl `kader_members` (Spieler) als auch `kader_trainers` (Trainer). Dadurch erscheinen Trainer in der Teilnahmeliste von Trainingseinheiten, obwohl dort nur Spieler angezeigt werden sollen (Bug sichtbar unter `/termine/training/{id}`).

## What Changes

- Neue View `player_memberships` — enthält ausschließlich Einträge aus `kader_members`
- Neue View `trainer_memberships` — enthält ausschließlich Einträge aus `kader_trainers`
- Alle Queries in Go-Code, die `team_memberships` nutzen, werden auf die passende neue View umgestellt
- `team_memberships`-View bleibt vorerst bestehen (wird von `user_accessible_teams` genutzt und an anderen Stellen referenziert) — **kein Breaking Change für bestehende Nutzungen**
- `GetAttendances`-Query nutzt explizit `player_memberships` statt `team_memberships`

## Capabilities

### New Capabilities

- `split-membership-views`: Zwei semantisch klare DB-Views (`player_memberships`, `trainer_memberships`) als Ersatz der gemischten `team_memberships`-View, mit konsistenter Nutzung im Go-Backend

### Modified Capabilities

*(keine Verhaltensänderung aus Nutzersicht — der Bug-Fix ist internes Verhalten)*

## Impact

- **DB-Migration**: neue `.up.sql` zum Anlegen der zwei Views; kein Schema-Bruch, keine Datenmigration
- **Go-Backend**: `internal/trainings/handler.go` (`GetAttendances`), ggf. weitere Stellen die `team_memberships` für Spieler-spezifische Logik nutzen
- **Kein Frontend-Änderungsbedarf**
- **Kein API-Änderungsbedarf**
