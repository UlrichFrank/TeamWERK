## 1. Provider-agnostischer Einstieg

- [x] 1.1 `AGENTS.md` im Repo-Root anlegen: destillierte Hard-Rules (pnpm statt npm; Tests via `/usr/local/go/bin/go`; brand-* Tokens; `hub.Broadcast` bei jeder Mutation + Frontend `useLiveUpdates`; neue Route → Happy-Path + Fehlerfall-Test; Rollen/Vereinsfunktionen-Modell; keine Unicode-Icons; nächste freie Migrationsnummer) + Verweis auf `CLAUDE.md` als kanonische Quelle.
- [x] 1.2 Konsistenz prüfen: jede Hard-Rule in `AGENTS.md` widerspricht keiner Aussage in `CLAUDE.md`.

## 2. Selbstkorrektur- und Permissions-Layer

- [x] 2.1 `.claude/settings.json` (committet) anlegen mit `hooks.PostToolUse`: Matcher `Edit|Write` → `command` ruft `scripts/claude-gofmt-hook.sh` auf.
- [x] 2.2 In derselben `settings.json` kuratierte `permissions.allow`-Liste: breite, sichere Wildcards (`rtk git/go/grep/find/read/ls/make/diff *`, `git add/commit/switch/checkout *`, `/usr/local/go/bin/go test|build|vet|run *`, `gofmt *`, `pnpm -C web build|test|lint`, `openspec list|show|validate|status *`, read-only `sqlite3 *`). Keine destruktiven/Deploy-Befehle.
- [x] 2.3 Prüfen, dass die geteilte Liste die häufigen Routine-Fälle aus `settings.local.json` abdeckt; dann `.claude/settings.local.json` entrümpeln (redundante Einzelfall-Einträge entfernen, nur maschinen-/personenspezifische Reste behalten).

## 3. Pre-Completion-Checkliste

- [x] 3.1 `.claude/commands/verify-change.md` anlegen: Ablauf `make test` (inkl. Architektur-Test) → `make lint` → `pnpm -C web build` → Invarianten-Checkliste (Route→Tests, Mutation→`hub.Broadcast`/`useLiveUpdates`, keine Raw-Tailwind-Farben, keine Unicode-Icons, nächste freie Migrationsnummer, `openspec validate`).

## 4. Doku-Drift-Fixes

- [x] 4.1 `CLAUDE.md`: Go-Version 1.23 → 1.25 (Abgleich mit `go.mod`).
- [x] 4.2 `CLAUDE.md`: neuer Abschnitt „## Harness / Verifikation" — dokumentiert `make hooks` (Git-Hooks), Architektur-Test (`internal/arch/`), gofmt-PostToolUse-Hook, `/verify-change`-Checkliste, `AGENTS.md`.
- [x] 4.3 `openspec/config.yaml`: Regel „Go-Handler-Tests optional (kein Test-Framework eingerichtet)" durch Verweis auf den verbindlichen Test-Standard ersetzen.

## 5. Verifikation

- [x] 5.1 gofmt-Hook: eine `*.go`-Datei via Edit unformatiert schreiben → ist danach gofmt-konform; eine `*.md`/`*.tsx`-Datei → Hook ohne Wirkung/Fehler.
- [x] 5.2 Permissions: einen Routine-Befehl (`go test`, `pnpm -C web build`) ausführen → kein Prompt.
- [x] 5.3 `/verify-change` aufrufen → läuft durch alle Gates.
- [x] 5.4 `openspec validate complete-agent-harness --strict` ist grün.

## Test-Anforderungen

- **Keine neuen HTTP-Routen und keine geänderte Geschäftslogik** — dieser Change betrifft ausschließlich Entwickler-/Agenten-Infrastruktur (Config, Doku, Slash-Command). Daher keine Go-Handler-Tests erforderlich.
- **Invariante (gofmt-Hook)**: Jede via Edit/Write geänderte `*.go`-Datei ist anschließend gofmt-konform; Nicht-Go-Dateien bleiben unberührt und der Hook blockiert nie den Toolaufruf. → verifiziert in 5.1.
- **Invariante (Permissions additiv-sicher)**: Routine-Operationen prompten nach Kuratierung nicht häufiger als zuvor. → verifiziert in 5.2.
- **Invariante (Doku-Konsistenz)**: Go-Version in `CLAUDE.md` == `go.mod`; `openspec/config.yaml`-Test-Regel bezeichnet Tests nicht als optional. → verifiziert beim Lesen nach 4.1/4.3.
