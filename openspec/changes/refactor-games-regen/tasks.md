## 1. Extract (Suite grün nach jedem Schritt)

- [ ] 1.1 `loadDayGames` extrahieren (Query + Scan der Tages-Spiele, ~119-146); `go test ./internal/games/` grün; Commit
- [ ] 1.2 `snapshotDeletedSlots` extrahieren (Loop ~192-222); grün; Commit
- [ ] 1.3 `snapshotCustomSlots` extrahieren (Loop ~225-255); grün; Commit
- [ ] 1.4 `regenGameItems` extrahieren (Item-Schleife ~270-362 inkl. Conflict-Branch/Team-Iteration; KRITISCH); grün; Commit
- [ ] 1.5 `buildNotificationIntents` extrahieren (~364-399); grün; Commit

## 2. Abschluss

- [ ] 2.1 `make metrics-gate`: `regenSingleDay` unter gocognit 20 / gocyclo 15; falls ein Helfer knapp darüber → Ratchet mit Begründung, nicht still
- [ ] 2.2 `go test ./...` + `go test -race ./internal/games/` grün; `git diff`-Check der `regen_summary`-/Conflict-/Notification-Literale (unverändert)
- [ ] 2.3 `openspec validate refactor-games-regen --strict` grün
- [ ] 2.4 Change archivieren (`openspec archive`); Roadmap 8.3-Folgeschritt als erledigt vermerken

## Test-Anforderungen

Reiner Struktur-Refactor — die Abnahme-Instanz sind die bereits auf `main` liegenden
HTTP-Charakterisierungstests der Auto-Duty-Regen (kein neuer Test):
- `internal/games/handler_test.go`: `TestCreateGame_AutoRegenSkipsAdjacentDay`, `TestUpdateGame_TimeChangeRegenSlots`, `TestRegenSummary_SkippedContent`, `TestRegen_TemplateIDNull_DeletesAutoKeepsCustom`, `TestRegen_ConflictWithCustomSlot`, `TestRegen_NotifiesRemovedAssignee`, `TestRegen_SameDayBehaviorReduced`, `TestDeleteGame_*`.
- `internal/games/scoped_events_test.go`: `TestRegenerateDaySlots_BroadcastsDuties`.
Alle SHALL nach jedem Extract-Schritt unverändert grün bleiben.
