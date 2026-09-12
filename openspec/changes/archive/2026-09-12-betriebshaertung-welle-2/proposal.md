## Why

Welle 2 des Architektur-Reviews vom 11.09.2026 (`docs/reviews/2026-09-11-architektur-review.md`):
Betriebsfähigkeit und Fehlersichtbarkeit. Heute enden 500er in drei Domänen ohne Log-Zeile,
ein Panic in einer Hintergrund-Goroutine reißt den Prozess mit, heiße Tabellen haben keinen
Index, ein Deploy gilt als erfolgreich, sobald `systemctl restart` durchläuft, und die
Permission-Matrix beweist Auth-Tiers, aber keine Objektrechte — genau die Fehlerklasse aus
Welle 1 kann deshalb unbemerkt zurückkehren.

## What Changes

1. **`internal/httpx`**: `WriteJSON`, `WriteError(w, r, status, code, err)` (JSON-Code nach außen,
   `slog.Error` mit Request-Kontext nach innen, nie `err.Error()` an den Client), `PathID`,
   `Paging` mit Deckel. Migration zuerst in `games`, `trainings`, `kader` (dort sitzen die
   spurlosen 500er und die ungedeckelte Paginierung).
2. **Hintergrund-Goroutinen mit `recover()`**: `notify.SendAsync` (und ein generischer
   `background.Go`) fangen Panics, zählen sie als Metrik und loggen sie; alle `go`-Aufrufe im
   Domänencode laufen darüber. Context-Bug im Passwort-Reset (`r.Context()` in Goroutine nach
   der Antwort) behoben.
3. **Migration 061**: Indizes `duty_slots(game_id)`, `duty_slots(event_date, season_id)`,
   `family_links(member_id)`, `duty_assignments(user_id)`.
4. **DATE-Gotcha** in `games.RegenerateSlots` und `PreviewSlots` behoben (`date[:10]`), mit
   Regressionstest, der den stillen No-op nachweist.
5. **Deploy absichern**: Smoke-Test gegen `/api/healthz` nach dem Restart, `deploy-rollback`
   (vorheriges Binary), serverseitiger Backup-Cron, wöchentlicher `wal_checkpoint(TRUNCATE)`
   im Scheduler.
6. **Fixture-Matrix für Objekt-Autorisierung**: zweite Matrix, die je `{id}`-Route ein fremdes
   Objekt anlegt und 403/404 erwartet.
7. **Modale zugänglich**: `EditModal` mit `role="dialog"`, `aria-modal`, `aria-labelledby`,
   Fokus-Trap und Fokus-Rückgabe; eigenständige Modals auf denselben Rumpf.

## Capabilities

### New Capabilities

- `api-error-responses`: einheitliche, maschinenlesbare Fehlerantworten ohne interne Details,
  mit Server-Log je 5xx.
- `background-task-resilience`: Panics in Hintergrundarbeit beenden nie den Prozess.
- `db-maintenance`: periodischer WAL-Checkpoint.
- `dialog-accessibility`: Modale sind per Tastatur und Screenreader bedienbar.

### Modified Capabilities

- `permissions`: Objektrechte werden mechanisch über eine Fixture-Matrix geprüft.
- `vps-deployment`: Deploy verifiziert den Prozessstart, hat einen Rückweg und ein
  serverseitiges Backup.

## Impact

Backend: neues Package `internal/httpx` (Foundation), `internal/background` (Foundation),
`internal/notify`, `internal/auth`, `internal/games`, `internal/trainings`, `internal/kader`,
`internal/scheduler`, Migration 061, `internal/permissions` (neue Testdatei), `internal/arch`
(Klassifikation). Frontend: `components/EditModal.tsx` und die eigenständigen Modals.
Betrieb: `Makefile`, `deploy/setup-vps.sh`. Fehlerbodies der migrierten Handler wechseln von
Plain-Text auf `{"error":"code"}`; das Frontend liest Fehler über `errorMessage()`, das beide
Formen versteht (prüfen, Task 1.4).
