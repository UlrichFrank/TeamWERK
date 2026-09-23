# Tasks

## 1. Routen und Handler

- [x] 1.1 `GET /api/duty-accounts` und `GET /api/duty-accounts/export` aus `internal/app/router.go`, Handler `Accounts`/`ExportAccounts` aus `internal/duties/handler.go` entfernen; Einträge in `internal/permissions/matrix_test.go` und Tests in `internal/duties/handler_test.go` entfernen; `go test ./internal/duties/ ./internal/permissions/ ./internal/arch/` grün

## 2. Schreibpfade

- [x] 2.1 `Claim`: `INSERT OR IGNORE INTO duty_accounts` entfernen, Claim-Test ohne Konto-Assertion; Fulfill-Test ohne `ist`-Invariante; `go test ./internal/duties/` grün
- [x] 2.2 `DeleteGame`: Neuberechnung von `duty_accounts.ist` und nur dafür gesammelte `fulfilledUIDs` entfernen, Doc-Kommentar anpassen; `ist`-Assertions in `internal/games/handler_test.go` entfernen, Cascade-Aussagen (Slots/Zuweisungen weg) bleiben; `go test ./internal/games/` grün
- [x] 2.3 `DeleteUser`: `DELETE FROM duty_accounts` entfernen; `go test ./internal/auth/` grün

## 3. Doku und Abschluss

- [x] 3.1 `docs/berechtigungen.md`, `docs/agent/06-gotchas.md` (Bekannter Rest im Massenlauf-Absatz), Paketkommentar `internal/dutyfairness/fairness.go` nachziehen; `git grep -n duty_accounts -- internal web/src` liefert nur Migrationen
- [x] 3.2 Stub `openspec/changes/dienstkonto-ist-buchung` entfernen
- [x] 3.3 `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `openspec validate dienstkonto-entfernen --strict` grün
