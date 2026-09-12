## Context

Befunde: Review-Abschnitte Backend, DB/Deploy. Muster im Repo: Architektur-Gates in
`internal/arch`, Permission-Matrix in `internal/permissions/matrix_test.go`, Backfill-Goroutinen
in `cmd/teamwerk/main.go`, Scheduler-Jobs in `internal/scheduler/scheduler.go` (`Run()` ruft
idempotente Jobs, Minutentakt per Cron).

## Goals / Non-Goals

**Goals:** jeder 5xx hinterlässt eine Log-Zeile; kein Panic außerhalb des HTTP-Pfads beendet den
Prozess; Deploy verifiziert sich selbst; Objektrechte sind mechanisch geprüft.
**Non-Goals:** vollständige Migration aller ~1.500 `http.Error`-Aufrufe (nur `games`,
`trainings`, `kader`; der Rest folgt als Ratchet); Errcheck-Ausnahme aufheben (Welle 3);
Staging-Umgebung.

## Decisions

**1. `internal/httpx` als Foundation.** `WriteError(w, r, status, code string, err error)`:
Body `{"error": code}`; bei `status >= 500` `slog.Error(code, "path", "method", "user_id", "err")`,
bei 4xx `slog.Debug`. `err.Error()` erreicht den Client nie. `PathID(r, name) (int, bool)`,
`Paging(r, defaultLimit, maxLimit) (limit, offset int)`. Kein Wrapper um `http.Error`, damit die
Migration greifbar bleibt (grep zählt Restbestand). Arch-Test: `httpx` ist Foundation, importiert
nur stdlib + `auth` (für die User-ID aus den Claims).

**2. `internal/background.Go(name string, fn func())`** mit `defer recover()`, `slog.Error` und
Metrik `teamwerk_background_panics_total` (neben `teamwerk_panics_total`). `notify.SendAsync`
ist die dünne Fassade dafür. Arch-Gate `internal/arch/goroutine_test.go`: nackte `go func`/`go x()`
im Domänencode nur über Allowlist (Hub-Kanäle, Video-Worker mit eigenem Shutdown).

**3. Context-Bug:** `auth/handler.go` Reset-Token-INSERT bekommt `context.Background()` mit
5-s-Timeout statt `r.Context()`; Fehler geloggt statt `//nolint`.

**4. Indizes** als Migration 061, `CREATE INDEX IF NOT EXISTS`, down droppt sie.

**5. DATE:** `RegenerateSlots`/`PreviewSlots` truncaten direkt nach dem Scan auf `date[:10]`
(Muster `loadBulkRangeGames`). Regressionstest legt ein Spiel an, ruft Regenerate und erwartet
Slots; ohne Fix entstehen keine.

**6. Deploy:** nach `systemctl restart` bis 30 s auf `curl -fsS https://<host>/api/healthz`
warten, sonst Abbruch mit Fehlermeldung und Hinweis auf `make deploy-rollback`. Vor dem
Austausch wird das laufende Binary als `teamwerk.prev` gesichert; `deploy-rollback` tauscht
zurück und startet neu (keine Migration rückwärts — die DB bleibt vorwärtskompatibel, siehe
Welle-1-Erfahrung). Backup: `setup-vps.sh` richtet einen täglichen Cron (03:30) ein, der
`sqlite3 .backup` plus Storage-Tar nach `/var/backups/teamwerk/<datum>` schreibt und 14 Tage
behält; `make backup` bleibt der Pull-Weg. WAL-Checkpoint: Scheduler-Job `walCheckpoint`, läuft
sonntags 04:00 (Guard über `notification_log`-artige Marke oder Zeitfenster), `PRAGMA
wal_checkpoint(TRUNCATE)`, Dauer und Ergebnis geloggt.

**7. Fixture-Matrix** (`internal/permissions/object_matrix_test.go`): für jede Route mit `{id}`
(aus `chi.Walk`) ein Fixture-Erzeuger, der ein Objekt anlegt, das **User B gehört**; Aufruf als
User A (Standard-Rolle, keine Funktion) muss 403 oder 404 liefern, nie 2xx. Routen ohne Erzeuger
lassen den Test fehlschlagen („nicht klassifiziert“, Muster Drift-Schutz); bewusst offene Routen
(z. B. öffentliche Spiele) stehen in einer begründeten Allowlist.

**8. `EditModal`:** `role="dialog"`, `aria-modal="true"`, `aria-labelledby` auf den Titel,
Fokus beim Öffnen auf das erste fokussierbare Element, Fokus-Trap (Tab/Shift+Tab zyklisch),
Fokus-Rückgabe an den Auslöser beim Schließen. Umsetzung als Hook `useDialogA11y(ref, isOpen)`,
den die eigenständigen Modals ebenfalls nutzen (ChatSearchModal hat Rolle schon, bekommt Trap).

## Risks / Trade-offs

- Fehlerbody-Wechsel: `lib/errors.ts` muss `{error}` bevorzugen; Vitest-Mocks in `games`/
  `trainings`/`kader`-Seiten prüfen.
- `deploy-rollback` ohne Migration rückwärts: nur sicher, solange Migrationen additiv sind
  (bisher der Fall; Doku-Hinweis).
- Fixture-Matrix ist testseitig teuer (ein Fixture je Route); Erzeuger werden domänenweise
  ergänzt, die Allowlist macht den Rest sichtbar.

## Migration Plan

Migration 061 additiv. Deploy wie üblich; erster Lauf mit neuem `deploy`-Target verifiziert sich
selbst. Rollback per `make deploy-rollback`.
