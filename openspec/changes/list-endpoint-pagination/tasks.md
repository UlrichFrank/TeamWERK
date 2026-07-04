# Tasks — list-endpoint-pagination

> Vertrags-ändernd (Array → {items,total}). Backend + zugehöriges Frontend je Route im selben Commit-Block. Backend zuerst.

## 1. Kader-Paginierung

- [ ] 1.1 `internal/kader/handler.go` (`ListKader`): `?limit&offset` + `COUNT(*)`, Response `{items,total}`, Default-Limit 50.
- [ ] 1.2 Tests: `TestListKader_PaginationLimitOffset`, `TestListKader_DefaultLimitApplied`.
- [ ] 1.3 `web/src/pages/AdminKaderPage.tsx`: `{items,total}` + „Mehr laden".

  _Commit:_ `feat(kader): Paginierung für GET /api/kader`

## 2. Duty-Slots- & Board-Paginierung/Trim

- [ ] 2.1 `internal/duties/handler.go:306` (`ListSlots`): `?limit&offset` (+ optional `?season_id&date_from`), `{items,total}`, Default 100.
- [ ] 2.2 `internal/duties/handler.go:412` (`Board`): Assignees auf `{user_id,name}` trimmen; `photo_url`/Kontakt aus Inline-Payload entfernen; optional `?from&to`.
- [ ] 2.3 On-Demand-Pfad für Assignee-Kontakt/Avatar eines Slots (falls nicht vorhanden), Sichtbarkeitsregeln wie heute.
- [ ] 2.4 Tests: `TestListDutySlots_Paginated`, `TestDutyBoard_NamesWithoutHeavyFields`, Auth-Fehlerfall.
- [ ] 2.5 `web/src/pages/DutyPage.tsx`/`DutySlotList`: Lazy-Load Avatar/Kontakt bei Slot-Öffnung; „Mehr laden" für Slots.

  _Commit:_ `feat(duties): duty-slots paginieren + duty-board Assignee-Felder aufschieben`

## 3. Games- & Participants-Paginierung

- [ ] 3.1 `internal/games/handler.go:454` (`ListGames`): `?season_id&limit&offset`, `{items,total}`, Default 50 (kein unbeschränkter Fallback mehr).
- [ ] 3.2 `internal/games/handler.go:2268` (`GetParticipants`): `?limit&offset`, `{items,total}`, Default 200.
- [ ] 3.3 Tests: `TestListGames_PaginatedAndSeasonFilter`, `TestParticipants_Paginated`, Auth-Fehlerfall.
- [ ] 3.4 `TerminePage.tsx`, `KalenderPage.tsx`, Game-Detail: `{items,total}`-Handling.

  _Commit:_ `feat(games): Paginierung für games + participants`

## 4. Training-Sessions: Paginierung + serverseitiger Filter

- [ ] 4.1 `internal/trainings/handler.go:910` (`ListSessions`): `?limit&offset`, `{items,total}`, Default 100; `?exclude_series=1` (ersetzt Client-`filter(series_id===null)`).
- [ ] 4.2 Tests: `TestListSessions_Paginated`, `TestListSessions_ExcludeSeriesFilter`.
- [ ] 4.3 `web/src/pages/AdminTrainingsPage.tsx`: `?exclude_series=1` nutzen, Client-`filter()` entfernen; `{items,total}`.

  _Commit:_ `feat(trainings): training-sessions paginieren + exclude_series-Filter`

## 5. Chat: Body-Preview

- [ ] 5.1 `internal/chat/handler.go:426` (`ListMessages`): `preview` (≤280 Zeichen) + `truncated`; gelöschte Nachrichten ohne Body; Volltext nur im Einzel-Pfad.
- [ ] 5.2 Tests: `TestListMessages_BodyPreviewTruncated`, `TestListMessages_DeletedNoBody`.
- [ ] 5.3 Chat-Frontend: Preview rendern, Volltext bei Bedarf nachladen.

  _Commit:_ `feat(chat): Nachrichtenliste liefert Body-Preview statt Volltext`

## 6. Abschluss

- [ ] 6.1 `/verify-change` (insb. Route→Tests, brand-Tokens, lucide-Icons).
- [ ] 6.2 `openspec validate list-endpoint-pagination --strict`.
- [ ] 6.3 Proposal archivieren.

  _Commit:_ `chore(api): archiviere list-endpoint-pagination`
