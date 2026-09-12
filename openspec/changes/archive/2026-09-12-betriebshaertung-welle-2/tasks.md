## 1. httpx und Fehlersichtbarkeit

- [x] 1.1 `internal/httpx`: `WriteJSON`, `WriteError` (JSON-Code, slog bei 5xx, nie err.Error()), `PathID`, `Paging(defaultLimit,maxLimit)`; Unit-Tests; Arch-Klassifikation Foundation.
- [x] 1.2 `internal/games/handler.go`, `internal/trainings/handler.go`, `internal/kader/handler.go` auf httpx migrieren (alle `http.Error` mit 5xx, alle `writeJSON`-Kopien, Paginierung mit Deckel 200).
- [x] 1.3 Tests der drei Packages grün; ein Test je Package, dass ein 5xx eine slog-Zeile erzeugt (Log-Buffer) und der Body keinen SQL-Text enthält.
- [x] 1.4 `web/src/lib/errors.ts`: `{error: code}` wird bevorzugt gelesen, Fallback Plain-Text; Vitest.

## 2. Hintergrund-Robustheit

- [x] 2.1 `internal/background.Go(name, fn)` mit recover, slog, Metrik `teamwerk_background_panics_total` (in `/api/metrics`); `notify.SendAsync` als Fassade.
- [x] 2.2 Alle `go`-Aufrufe im Domänencode auf `background.Go`/`notify.SendAsync`; Arch-Gate `goroutine_test.go` mit begründeter Allowlist (Hub, Video-Worker, Backfills in main).
- [x] 2.3 `auth/handler.go` Reset-Token: `context.Background()` + Timeout statt `r.Context()`, Fehler geloggt; Test, dass der Token nach der 204-Antwort in der DB steht.

## 3. Datenbank

- [x] 3.1 Migration 061: `idx_duty_slots_game`, `idx_duty_slots_date_season`, `idx_family_links_member`, `idx_duty_assignments_user` (up/down, Roundtrip-Test).
- [x] 3.2 DATE-Gotcha in `RegenerateSlots` und `PreviewSlots` (`date[:10]`), Regressionstest „Regenerate erzeugt Slots für ISO-Datum".
- [x] 3.3 Scheduler-Job `walCheckpoint` (wöchentlich, `PRAGMA wal_checkpoint(TRUNCATE)`, geloggt), Test.

## 4. Deploy

- [x] 4.1 `make deploy`: Binary vor dem Austausch als `teamwerk.prev` sichern; nach Restart Smoke-Test `/api/healthz` (30 s), bei Fehlschlag Abbruch mit Hinweis.
- [x] 4.2 `make deploy-rollback`: `teamwerk.prev` zurücktauschen, Restart, Smoke-Test.
- [x] 4.3 `deploy/setup-vps.sh`: täglicher Backup-Cron (`sqlite3 .backup` + Storage-Tar nach `/var/backups/teamwerk`, 14 Tage Retention), idempotent; Doku in `10-deployment.md`.

## 5. Objektrechte mechanisch

- [x] 5.1 `internal/permissions/object_matrix_test.go`: Routen mit `{id}` aus `chi.Walk`, Fixture-Erzeuger je Domäne (Objekt gehört User B), Aufruf als User A → 403/404; unklassifizierte Route = rot; Allowlist für bewusst offene Routen mit Begründung.
- [x] 5.2 Fixture-Erzeuger für alle heute existierenden `{id}`-Routen; Befunde (2xx auf fremdes Objekt) als Liste im Bericht, nicht still allowlisten.

## 6. Zugänglichkeit

- [x] 6.1 Hook `useDialogA11y` (Fokus setzen, Trap, Rückgabe, Escape bleibt bei useEscapeKey); `EditModal` mit `role="dialog"`, `aria-modal`, `aria-labelledby`; Vitest (Tab zyklisch, Fokus zurück).
- [x] 6.2 Eigenständige Modals (`ChatSearchModal`, `H4AImportModal`, `DutyBulkRegenModal`, `GameEditModal`, Poll-Modals, weitere per grep `fixed inset-0`) auf Rolle + Hook.

## 7. Verifikation

- [x] 7.1 `make test`, `make lint`, Frontend build/test/lint, `make test-e2e` grün; `openspec validate --strict`.
- [x] 7.2 Gotcha-Absätze (httpx-Pflicht für 5xx, background.Go-Pflicht, Fixture-Matrix) in `06-gotchas.md`; `08-verification.md` um die zwei neuen Gates ergänzen; `10-deployment.md` (Smoke, Rollback, Backup-Cron).
