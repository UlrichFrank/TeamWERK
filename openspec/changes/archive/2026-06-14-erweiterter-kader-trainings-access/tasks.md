## 1. Backend: Erweiterter Kader in Trainings-Queries

- [x] 1.1 `teamMembersAndParents()` (~Zeile 32) um UNION auf `kader_extended_members` erweitern (nur user_id IS NOT NULL)
- [x] 1.2 `ListTrainingSessions` Spieler-Condition (~Zeile 701) um UNION auf `kader_extended_members` erweitern
- [x] 1.3 `GetAttendances` (~Zeile 1081) von `JOIN player_memberships` auf OR-EXISTS mit `kader_extended_members` umstellen

## 2. Tests

- [x] 2.1 Test: Erw.-Kader-Spieler sieht Training in `GET /api/training-sessions` (Happy Path)
- [x] 2.2 Test: Erw.-Kader-Spieler erscheint in `GET /api/training-sessions/{id}/attendances`
- [x] 2.3 Test: Spieler ohne Kader-Zugehörigkeit sieht Training nicht (Negativfall)
