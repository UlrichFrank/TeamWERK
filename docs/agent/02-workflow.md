# Entwicklungsworkflow

```bash
# Lokaler Start (zwei Prozesse)
go run ./cmd/teamwerk        # Backend :8080  (braucht cmd/teamwerk/web/dist/ wegen //go:embed — sonst cmd/teamwerk/web/dist/.gitkeep anlegen)
cd web && pnpm dev           # Vite :5173, proxyt /api → :8080

make build                   # pnpm build + go build → bin/teamwerk
make deploy                  # build + rsync auf VPS + systemctl restart (führt automatisch migrate up aus)
make migrate-up / migrate-down
make test / lint / coverage
make test-e2e                # Playwright (echter Chromium gegen Prod-Binary + Seed-DB) — ~2–4 min, NICHT Teil von make test/pre-push; für UI-riskante Änderungen (Scroll/Layout/Focus)
make metrics                 # Code-Metriken (Größe/Komplexität/Coverage/Lint-Dichte/Duplikation) → stdout + metrics/REPORT.md (Exit 0)
make metrics-gate            # Wie metrics + Schwellwert-Prüfung gegen metrics/thresholds.yml (Exit 1 bei Regression)
```

**Neue Migration:** `internal/db/migrations/00N_beschreibung.up.sql` + `.down.sql` mit der **nächsten freien Nummer**. Nie eine Nummer ≤ aktueller DB-Version — golang-migrate überspringt sie lautlos.
