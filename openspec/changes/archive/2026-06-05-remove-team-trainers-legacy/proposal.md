## Why

Die `team_trainers`-Tabelle ist ein Legacy-Konstrukt, das parallel zum korrekten `kader_trainers`-System existiert. Der Training-Handler (`hasTeamAccess`, `ListSeries`, `ListSessions`) nutzt ausschließlich `team_trainers`, wodurch Trainer, die nur über das Kader-System eingetragen sind, weder ihre Trainings sehen noch neue anlegen können. Die Tabelle muss entfernt und alle Zugriffe auf das kader-basierte System umgestellt werden.

## What Changes

- **BREAKING** `POST /api/admin/teams/{id}/assign-trainer` wird entfernt — Trainer werden ausschließlich über das Kader zugewiesen
- `hasTeamAccess()` im Training-Handler prüft künftig `kader_trainers` statt `team_trainers`
- `ListSeries`-Filter verwendet `kader_trainers`
- `ListSessions`-Filter verwendet `kader_trainers`
- Migration droppt die `team_trainers`-Tabelle

## Capabilities

### New Capabilities

- `training-management-access`: Definiert, welche Nutzer Trainings eines Teams verwalten dürfen — ausschließlich auf Basis des kader-basierten Trainer-Eintrags (`kader_trainers`).

### Modified Capabilities

_(keine bestehenden Specs mit Anforderungsänderung)_

## Impact

- `internal/trainings/handler.go`: `hasTeamAccess`, `ListSeries`, `ListSessions`
- `internal/config/handler.go`: `AssignTrainer`-Handler und Route entfernen
- `cmd/teamwerk/main.go`: Route `POST /api/admin/teams/{id}/assign-trainer` entfernen
- `internal/db/migrations/`: neue Migration `team_trainers` DROP
- Frontend `AdminTrainingsPage.tsx`: keine Änderung nötig (nutzt `/api/teams` → bereits `kader_trainers`)
