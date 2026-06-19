## Context

Harness-Engineering kennt drei Säulen (Context Engineering, Architectural Constraints, Entropy Management) plus Feedback-Loops und Provider-Agnostik. TeamWERK hat einen starken Kontext-Layer (`CLAUDE.md`, OpenSpec) und seit dem letzten Commit auch mechanische Architektur-Durchsetzung (`internal/arch/arch_test.go`) und Git-Hooks. Dieser Change schließt die verbleibenden Lücken, die alle in der Entwickler-/Agenten-Infrastruktur liegen — kein Produktionscode. Constraint des Projekts: Solo/kleines Team, lokale Durchsetzung (keine Cloud-CI gewünscht), 1-GB-VPS (kein Einfluss hier, da nur Dev-Tooling).

## Goals / Non-Goals

**Goals:**
- Konventionen für *jeden* Coding-Agenten auffindbar machen (nicht nur Claude).
- Den Selbstkorrektur-Loop schließen: vom Agenten editierte Go-Dateien sind immer gofmt-konform, bevor der `pre-commit`-Hook greift.
- Routine-Befehle vorab freigeben, sodass Permission-Prompts sinken statt steigen.
- Eine wiederholbare Pre-Completion-Prüfung als Slash-Command bereitstellen.
- Doku-Drift beseitigen (Go-Version, Test-Policy).

**Non-Goals:**
- Keine Cloud-CI (bewusst gegen GitHub Actions entschieden — lokale Hooks decken das ab).
- Kein automatischer Doku-Drift-Checker für die 221 Routen (manuelle Pflege bleibt).
- Keine geplanten Entropie-Audit-Agenten, kein Custom-Review-Subagent, keine Observability (Level 3, als Overkill verworfen).
- Keine Änderung an `internal/arch/`, `.githooks/`, `scripts/claude-gofmt-hook.sh`, `Makefile` — bereits gemergt.

## Decisions

- **`AGENTS.md` als dünner Zeiger, `CLAUDE.md` bleibt kanonisch.** Statt Inhalte zu duplizieren (Drift-Gefahr), enthält `AGENTS.md` nur die ~10 nicht-verhandelbaren Hard-Rules + Verweis. Alternative „AGENTS.md kanonisch, CLAUDE.md = Symlink" verworfen: `CLAUDE.md` ist groß, etabliert und git-getrackt; ein Bruch wäre invasiv.
- **gofmt-Hook als separates Skript, nicht inline in JSON.** `scripts/claude-gofmt-hook.sh` (bereits vorhanden) wird referenziert; vermeidet brüchiges JSON-Escaping und ist einzeln testbar. Matcher `Edit|Write`, no-op außer bei `*.go`.
- **Permissions: breite, sichere Wildcards in committeter `settings.json`.** Read-only/idempotente Operationen (`rtk git/go/grep/find/read/ls/make/diff`, `gofmt`, `go test|build|vet|run`, `pnpm -C web build|test|lint`, `openspec list|show|validate|status`, read-only `sqlite3`). Keine destruktiven oder Deploy-Befehle. `settings.local.json` wird erst danach entrümpelt, damit nie eine Lücke entsteht (additiv-sicher).
- **`verify-change` als Slash-Command, nicht als Hook/Subagent.** On-demand vor „fertig"; ein blockierender Stop-Hook wäre für einen Solo-Workflow zu aufdringlich. Bündelt `make test`/`make lint`/`pnpm -C web build` + Projekt-Invarianten-Checkliste.
- **Doku-Fixes minimal-invasiv.** Nur die zwei faktisch falschen Stellen (Go 1.23→1.25; Test-Regel) plus ein neuer, kurzer Harness-Abschnitt in `CLAUDE.md`.

## Risks / Trade-offs

- **Permissions zu eng kuratiert → mehr Prompts statt weniger.** → Breite Wildcards wählen und gegen die bestehende `settings.local.json` abgleichen; im Zweifel großzügiger.
- **`settings.local.json` ist gitignored → Entrümpeln betrifft nur die lokale Maschine.** → Akzeptiert; die geteilte Baseline lebt in `settings.json`, der lokale Rest ist persönlich.
- **gofmt-Hook könnte bei fehlendem `python3`/`gofmt` still scheitern.** → Skript ist defensiv (`|| true`, Pfad-Fallback); ein Fehlschlag formatiert lediglich nicht und blockiert nichts. Der `pre-commit`-Hook fängt Formatierung ohnehin als Sicherheitsnetz.
- **`AGENTS.md` und `CLAUDE.md` können auseinanderdriften.** → `AGENTS.md` bewusst klein halten und ausdrücklich auf `CLAUDE.md` als Quelle verweisen.

## Migration Plan

1. `AGENTS.md`, `.claude/settings.json`, `.claude/commands/verify-change.md` neu anlegen.
2. Doku-Fixes in `CLAUDE.md` und `openspec/config.yaml`.
3. `settings.local.json` entrümpeln (zuletzt, nach Verifikation, dass `settings.json` die Routine-Fälle abdeckt).
4. Verifizieren: gofmt-Hook feuert bei `.go`-Edit; `/verify-change` läuft durch; Routine-Befehle prompten nicht mehr.

Rollback: alle Artefakte sind additiv bzw. reine Config/Doku — Entfernen der Dateien bzw. `git checkout` stellt den Zustand wieder her. Kein Produktionsrisiko.

## Open Questions

- Keine offen. Scope und Entscheidungen wurden mit dem Nutzer geklärt (Level 1+2, lokal, Permissions kuratieren).
