## 1. Upload-Fix: kein Session-Hijack

- [x] 1.1 `web/src/pages/VideoUploadPage.tsx`: `handleSubmit`/`startTus` so ändern, dass ein frischer Upload nie `resumeFromPreviousUpload` aufruft (immer neue Session für die neue `video_id`); `resumable` nur noch für den Button „Upload fortsetzen" (`handleResume`) nutzen.
- [x] 1.2 Frontend-Test in `web/src/pages/__tests__/VideoUploadPage.test.tsx`: frischer „Hochladen"-Klick startet Upload mit der neuen `video_id` und resumt keine vorhandene Session (kein `resumeFromPreviousUpload`), auch wenn eine Resume-Session vorliegt.

## 2. Spiel-Zuordnung im Bearbeiten-Modal

- [x] 2.1 `web/src/pages/VideoDetailPage.tsx`: Spiele des Video-Teams laden (`GET /api/games`, nach `team_id` filtern), Spiel-Selector inkl. „Kein Spiel zuordnen" ins Edit-Modal, aktuellen `game_id`-Wert vorbelegen.
- [x] 2.2 `handleSave` sendet `game_id` als Tri-State im `PATCH /api/videos/{id}` (Zahl bei Auswahl, `null` bei „Kein Spiel zuordnen").
- [x] 2.3 Frontend-Test in `web/src/pages/__tests__/`: Edit-Modal zeigt Selector, Ändern der Auswahl sendet das erwartete `game_id` im PATCH; „Kein Spiel zuordnen" sendet `null`.
- [x] 2.4 Backend-Test in `internal/videos/crud_test.go` (`TestUpdate_GameID`): `game_id` setzen → 200 + Wert gesetzt; `game_id: null` → 200 + `NULL`; Feld weglassen → unverändert; ohne Verwaltungsrecht → 403 und `game_id` unverändert.

## 3. Cleanup hängender Uploads

- [x] 3.1 `internal/scheduler/scheduler.go`: Inline-Job `failStaleVideoUploads` (SQL `UPDATE videos SET status='failed', failure_reason='Upload abgebrochen' WHERE status='uploading' AND created_at < datetime('now','-24 hours')`), im Scheduler-Tick aufrufen; Erfolg loggen wie bei bestehenden Jobs.
- [x] 3.2 Scheduler-Test: alte `uploading`-Zeile (>24 h) wird `failed`; frische `uploading`-Zeile und `queued`/`ready` bleiben unverändert.

## 4. Verifikation & Abschluss

- [x] 4.1 `/verify-change` bzw. Gate: `go test ./...`, `golangci-lint`, `pnpm -C web build/test/lint`, `openspec validate` grün.
- [ ] 4.2 Change archivieren (Commit pro Task, abschließender Archiv-Commit).
