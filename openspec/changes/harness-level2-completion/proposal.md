## Why

Der Agent-Harness erfüllt Level 1+2 (nach „Harness-Engineering") bereits zu ~85 %. Drei Lücken aus dem Level-1+2-Katalog bleiben: (a) die Git-Hooks sind gebaut, aber nicht aktiviert; (b) „Documentation-as-Code mit Linter-Validierung" fehlt — das ist ein deterministischer `make audit`, der Doku-gegen-Code-Drift fängt (wir hatten in einer Session drei solche Drifts); (c) die „agent-specific PR review checklist" fehlt ganz. Dieser Change schließt (b) und (c). (a) ist nur ein Befehl (`make hooks`) und wird als Voraussetzung dokumentiert, nicht als Change-Arbeit.

## What Changes

- **`make audit` (Documentation-as-Code-Validierung):** Neues Makefile-Target + `scripts/audit.sh`, das deterministisch Drift erkennt. **Harte** Checks (Exit ≠ 0): Go-Version `CLAUDE.md` vs `go.mod`; keine Raw-Tailwind-Farben in `web/src`; keine Unicode-Icons/Emojis in JSX; Architektur-Test (`go test ./internal/arch/`). **Warnung** (Exit 0): in `CLAUDE.md` dokumentierte `/api/`-Pfade, die nicht mehr in `internal/app/router.go` existieren.
- **Projektspezifischer Review-Subagent:** Neue `.claude/agents/teamwerk-reviewer.md` — eine unabhängige Review-Instanz mit eigenem Kontext, die einen Diff gegen die TeamWERK-Invarianten prüft (brand-Tokens, SSE `hub.Broadcast`/`useLiveUpdates`, Rollen/Vereinsfunktionen-Modell, Test-Standard, Migrationsregeln, Architektur-Layering). Ergänzt `/verify-change` (Selbstprüfung) um eine zweite Perspektive.
- **Doku:** `CLAUDE.md`-Harness-Abschnitt um `make audit` und den Reviewer-Subagent ergänzen; Hooks-Aktivierung (`make hooks`) als Voraussetzung kenntlich machen.

## Capabilities

### Modified Capabilities
- `agent-harness`: Zwei neue Garantien — eine deterministische Doku-Drift-Validierung (`make audit`) und eine unabhängige, projektkundige Review-Instanz.

## Impact

- **Neue Dateien:** `scripts/audit.sh`, `.claude/agents/teamwerk-reviewer.md`.
- **Geänderte Dateien:** `Makefile` (`audit`-Target), `CLAUDE.md` (Harness-Abschnitt), `openspec/specs/agent-harness/spec.md` (beim Archivieren).
- **Voraussetzung (kein Change-Code):** `make hooks` einmalig ausführen, um die bestehenden Git-Hooks zu aktivieren.
- **Kein** Produktionscode (Go/React) betroffen; keine DB-Migration; keine API-Route. Reine Entwickler-/Agenten-Infrastruktur.
- **Bewusst nicht enthalten (Level 3):** scheduled/autonomer Entropie-Agent, Observability, A/B-Testing, Dashboards.
