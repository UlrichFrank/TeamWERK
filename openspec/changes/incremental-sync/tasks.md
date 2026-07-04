# Tasks — incremental-sync

> Cursor-basierte Delta-Sync. Chat-Phase zuerst (ohne Schema). Backend vor Frontend. DB-Backup vor Migrationen.

## 1. Chat inkrementell (ohne Schema)

- [x] 1.1 `internal/chat/handler.go` (`ListMessages`): `?after=<msgId>` (nur `id > after`) und `?before=<msgId>` (ältere Seite). Ohne Param unverändert.
- [x] 1.2 Tests: `TestMessagesAfter_ReturnsOnlyNewer`, `TestMessagesBefore_ReturnsOlderPage`.
- [ ] 1.3 `web/src/pages/ChatPage.tsx`: bei `chat:new-message:<id>` per `?after=` anhängen statt `loadMessages()`; Verlaufs-Scroll per `?before=`.

  _Commit:_ `feat(chat): inkrementelles Nachladen (after/before) statt Voll-Reload`

## 2. `updated_at` nachrüsten

- [ ] 2.1 Migration (nächste freie Nummer): `updated_at` auf `games`, `duty_slots`, `training_sessions` (+ `.down.sql`), Backfill = `created_at`/jetzt.
- [ ] 2.2 Schreibpfade in `internal/games`, `internal/duties`, `internal/trainings` setzen `updated_at` bei INSERT/UPDATE.
- [ ] 2.3 Tests: `updated_at` ändert sich bei Mutation je Tabelle.

  _Commit:_ `feat(db): updated_at auf games/duty_slots/training_sessions`

## 3. Tombstone-Log

- [ ] 3.1 Migration: `sync_tombstones(entity TEXT, entity_id INTEGER, deleted_at)` (append-only) + Pruning-Kriterium.
- [ ] 3.2 Delete-Handler (games/duty_slots/training_sessions/kader) schreiben Tombstone.
- [ ] 3.3 Pruning-Job (Scheduler, idempotent) beschneidet Tombstones älter als Frist.
- [ ] 3.4 Tests: Delete erzeugt Tombstone; Pruning entfernt abgelaufene.

  _Commit:_ `feat(db): sync_tombstones für Lösch-Erkennung + Pruning`

## 4. `?since=`-Cursor auf schweren Listen

- [ ] 4.1 `GET /api/games`/`/api/duty-slots`/`/api/training-sessions`/`/api/kader`: `?since=<cursor>` → `{items, deleted_ids, cursor, full}`; zu alter Cursor → `full:true`. Koexistenz mit `limit`/`offset`.
- [ ] 4.2 Tests: `TestGamesSince_ReturnsOnlyChanged`, `TestGamesSince_DeletedReportedAsTombstone`, `TestGamesSince_StaleCursorFallsBackToFull`, `TestGames_NoSinceUnchanged`.
- [ ] 4.3 Frontend: Cursor halten, Bestand kombinieren, `full:true` → neu aufbauen.

  _Commit:_ `feat(games,duties,trainings): ?since=-Delta-Sync mit Tombstones`

## 5. Messung

- [ ] 5.1 `payload-measurement-harness` um Delta-Szenario erweitern (eine Mutation → `?since=`-Payload messen).
- [ ] 5.2 Baseline-Zeilen (`metrics/payload-baseline.md`) mit Delta-Zahlen fortschreiben.

  _Commit:_ `test(measure): Delta-Payload-Szenario für incremental-sync`

## 6. Abschluss

- [ ] 6.1 `make test` + `/verify-change`.
- [ ] 6.2 `openspec validate incremental-sync --strict`.
- [ ] 6.3 Proposal archivieren.

  _Commit:_ `chore(api): archiviere incremental-sync`
