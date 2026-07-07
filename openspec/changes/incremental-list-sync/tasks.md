# Tasks — incremental-list-sync

> Herausgelöst aus `incremental-sync` (Chat-Phase dort bereits erledigt). Cursor-basierte Delta-Sync für schwere Listen. Backend vor Frontend. DB-Backup vor Migrationen.

## 1. `updated_at` nachrüsten

- [ ] 1.1 Migration (nächste freie Nummer): `updated_at` auf `games`, `duty_slots`, `training_sessions` (+ `.down.sql`), Backfill = `created_at`/jetzt.
- [ ] 1.2 Schreibpfade in `internal/games`, `internal/duties`, `internal/trainings` setzen `updated_at` bei INSERT/UPDATE.
- [ ] 1.3 Tests: `updated_at` ändert sich bei Mutation je Tabelle.

  _Commit:_ `feat(db): updated_at auf games/duty_slots/training_sessions`

## 2. Tombstone-Log

- [ ] 2.1 Migration: `sync_tombstones(entity TEXT, entity_id INTEGER, deleted_at)` (append-only) + Pruning-Kriterium.
- [ ] 2.2 Delete-Handler (games/duty_slots/training_sessions/kader) schreiben Tombstone.
- [ ] 2.3 Pruning-Job (Scheduler, idempotent) beschneidet Tombstones älter als Frist.
- [ ] 2.4 Tests: Delete erzeugt Tombstone; Pruning entfernt abgelaufene.

  _Commit:_ `feat(db): sync_tombstones für Lösch-Erkennung + Pruning`

## 3. `?since=`-Cursor auf schweren Listen

- [ ] 3.1 `GET /api/games`/`/api/duty-slots`/`/api/training-sessions`/`/api/kader`: `?since=<cursor>` → `{items, deleted_ids, cursor, full}`; zu alter Cursor → `full:true`. Koexistenz mit `limit`/`offset`.
- [ ] 3.2 Tests: `TestGamesSince_ReturnsOnlyChanged`, `TestGamesSince_DeletedReportedAsTombstone`, `TestGamesSince_StaleCursorFallsBackToFull`, `TestGames_NoSinceUnchanged`.
- [ ] 3.3 Frontend: Cursor halten, Bestand kombinieren, `full:true` → neu aufbauen.

  _Commit:_ `feat(games,duties,trainings): ?since=-Delta-Sync mit Tombstones`

## 4. Messung

- [ ] 4.1 `payload-measurement-harness` um Delta-Szenario erweitern (eine Mutation → `?since=`-Payload messen).
- [ ] 4.2 Baseline-Zeilen (`metrics/payload-baseline.md`) mit Delta-Zahlen fortschreiben.

  _Commit:_ `test(measure): Delta-Payload-Szenario für incremental-list-sync`

## 5. Abschluss

- [ ] 5.1 `make test` + `/verify-change`.
- [ ] 5.2 `openspec validate incremental-list-sync --strict`.
- [ ] 5.3 Proposal archivieren.

  _Commit:_ `chore(api): archiviere incremental-list-sync`
