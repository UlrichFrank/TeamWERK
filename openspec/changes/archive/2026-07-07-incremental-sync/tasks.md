# Tasks — incremental-sync

> id-basiertes inkrementelles Chat-Nachladen (append-only, ohne Schema). Die Delta-Sync der schweren Listen wurde nach `incremental-list-sync` extrahiert.

## 1. Chat inkrementell (ohne Schema)

- [x] 1.1 `internal/chat/handler.go` (`ListMessages`): `?after=<msgId>` (nur `id > after`) und `?before=<msgId>` (ältere Seite). Ohne Param unverändert.
- [x] 1.2 Tests: `TestMessagesAfter_ReturnsOnlyNewer`, `TestMessagesBefore_ReturnsOlderPage`.
- [x] 1.3 `web/src/pages/ChatPage.tsx`: bei `chat:new-message:<id>` per `?after=` anhängen statt `loadMessages()`; Verlaufs-Scroll per `?before=`.

  _Commit:_ `feat(chat): inkrementelles Nachladen (after/before) statt Voll-Reload`

## 2. Abschluss

- [x] 2.1 `make test` + `/verify-change`.
- [x] 2.2 `openspec validate incremental-sync --strict`.
- [ ] 2.3 Proposal archivieren.

  _Commit:_ `chore(api): archiviere incremental-sync`
