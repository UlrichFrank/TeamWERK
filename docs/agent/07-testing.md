# Test-Standard

Jede neue HTTP-Route **muss** mindestens **Happy-Path** (Erfolg) und **Fehlerfall** (401/403/400/404/409) abdecken. Tests prüfen fachliche Invarianten — keine Dummy-Assertions zur Coverage-Erhöhung.

Jeder OpenSpec-Proposal mit neuen Routen / geänderter Geschäftslogik braucht einen Abschnitt `## Test-Anforderungen` (Route → Testname + erwarteter Status, plus die garantierte Invariante).

**Fixtures** in `internal/testutil/`: `NewDB`, `NewServer`, `CreateUser`, `CreateMember`, `CreateSeason`, `CreateTeam`, `CreateGame`, `CreateKader`, `CreateDutyType`, `CreateDutySlot`, `CreateInvitationToken`, `CreatePasswordResetToken`, `CreateRefreshToken`.

`make coverage` → stdout + HTML nach `/tmp/teamwerk-coverage.html`. Coverage ist Indikator, kein Gate.

`make metrics` → erhebt Größe/Komplexität/Coverage/Lint-Dichte/Duplikation, schreibt `metrics/REPORT.md` (gitignored). Komplexität nutzt **separate** `.golangci.metrics.yml` (`gocyclo`, `gocognit`, `funlen`, `dupl`) — die Haupt-`.golangci.yml` (Gate) bleibt unangetastet. Tools sind im jeweiligen Manifest gepinnt: `scc` als `go.mod`-`tool`-Direktive (`go tool scc`), `jscpd` als pnpm-devDependency (`pnpm -C web exec jscpd`). `make metrics-gate` vergleicht zusätzlich gegen `metrics/thresholds.yml` (Ratchet-Prinzip; Exit 1 bei Regression).
