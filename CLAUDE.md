# CLAUDE.md

Guidance for Claude Code working in this repository.

## Hard Rules

- **`pnpm`, nie `npm`** für alle Frontend-/npm-Operationen.
- **Go 1.26+** für alle Go-Befehle (go.mod: `go 1.26.0`). `/usr/local/go/bin/go` (1.25 als Bootstrap) zieht via `GOTOOLCHAIN` automatisch die 1.26-Toolchain; ein global exportiertes `GOROOT=/usr/local/go` muss dafür ungesetzt sein (sonst „version does not match"-Fehler).
- **Nur `brand-*`-Tokens**, keine Raw-Tailwind-Farben (`bg-gray-50`, `text-red-600`, …).
- **Keine Unicode-Icons/Emojis in JSX** — `lucide-react`.
- **Jede Mutations-Route ruft `h.hub.Broadcast(...)`**, das Frontend abonniert mit `useLiveUpdates` (siehe Gotcha SSE).
- **Jede neue Route bekommt Tests** (Happy-Path + Fehlerfall).
- **Kein ORM** — direktes `database/sql`.

---

## Kapitel

Die folgenden Themen sind je in einer eigenen Datei unter `docs/agent/` abgelegt und
werden hier per `@`-Import vollständig in den Kontext geladen (eine Datei = ein Thema).

**Context — Was & Wie**

@docs/agent/01-overview.md
@docs/agent/02-workflow.md
@docs/agent/03-go.md
@docs/agent/04-api-db.md
@docs/agent/05-frontend.md
@docs/agent/06-gotchas.md

**Constraints — mechanisch erzwungen**

@docs/agent/07-testing.md
@docs/agent/08-verification.md

**Process — Spezifikation & Betrieb**

@docs/agent/09-openspec.md
@docs/agent/10-deployment.md
