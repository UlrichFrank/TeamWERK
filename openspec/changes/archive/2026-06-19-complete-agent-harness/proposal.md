## Why

Der Agent-Harness (nach „Harness-Engineering": Agentenverhalten durch Infrastruktur lenken statt nur durch Prompting) ist erst zur Hälfte gebaut. Architektur-Test und Git-Hooks (Säule 2) sind bereits gemergt; es fehlen die provider-agnostische Kontext-Schicht, die lokale Selbstkorrektur-/Verifikations-Schleife und die Bereinigung von Doku-Drift. Ohne diese Teile bleiben Konventionen für Agenten teils unauffindbar (nur in `CLAUDE.md`, nicht in `AGENTS.md`), routinemäßige Befehle erzeugen unnötige Permission-Prompts, und zwei dokumentierte Fakten widersprechen dem echten Stand.

## What Changes

- **Provider-agnostischer Einstieg**: Neue `AGENTS.md` im Repo-Root mit den destillierten Hard-Rules und Verweis auf `CLAUDE.md` als kanonische Quelle (lesbar für Codex/Cursor/Gemini, nicht nur Claude).
- **Selbstkorrektur-Hook**: Committete `.claude/settings.json` mit `PostToolUse`-Hook, der `scripts/claude-gofmt-hook.sh` auf editierten Go-Dateien ausführt (gofmt), damit der bereits gemergte `pre-commit`-Hook nicht an Formatierung scheitert.
- **Kuratierte geteilte Permissions**: In derselben `settings.json` eine schlanke `permissions.allow`-Liste langlebiger, sicherer Wildcards für Routine-Operationen; danach Entrümpeln der persönlichen `.claude/settings.local.json` (gitignored) von ~130 Einzelfall-Einträgen.
- **Pre-Completion-Checkliste**: Neuer Slash-Command `.claude/commands/verify-change.md`, der vor „fertig" Build/Test/Lint und die Projekt-Invarianten (neue Route → Tests, Mutation → `hub.Broadcast`, keine Raw-Tailwind-Farben, keine Unicode-Icons, nächste freie Migrationsnummer) durchgeht.
- **Doku-Drift-Fixes**: `CLAUDE.md` Go-Version 1.23 → 1.25 (Abgleich mit `go.mod`) plus neuer Abschnitt „Harness / Verifikation"; `openspec/config.yaml`-Regel „Go-Handler-Tests optional" an den verbindlichen Test-Standard angleichen.

## Capabilities

### New Capabilities
- `agent-harness`: Garantien für die Agenten-Infrastruktur — provider-agnostischer Kontext-Einstieg, automatische Go-Formatierung editierter Dateien, vorab freigegebene Routine-Befehle, eine Pre-Completion-Verifikationsroutine und driftfreie Projekt-Dokumentation.

### Modified Capabilities
<!-- Keine spec-level Verhaltensänderung an bestehenden Capabilities. Die Anpassungen an CLAUDE.md und openspec/config.yaml sind Doku-Drift-Korrekturen, keine Requirement-Änderungen. -->

## Impact

- **Neue Dateien**: `AGENTS.md`, `.claude/settings.json`, `.claude/commands/verify-change.md`.
- **Geänderte Dateien**: `.claude/settings.local.json` (entrümpeln, gitignored), `CLAUDE.md`, `openspec/config.yaml`.
- **Bereits gemergt (Kontext, nicht Teil dieses Changes)**: `internal/arch/arch_test.go`, `.githooks/pre-commit`, `.githooks/pre-push`, `scripts/claude-gofmt-hook.sh`, `Makefile` (`hooks`-Target).
- **Kein** Produktionscode (Go-Backend / React-Frontend) betroffen; keine DB-Migration; keine API-Routen-Änderung. Reine Entwickler-/Agenten-Infrastruktur.
- **Risiko**: Permissions-Kuratierung muss additiv-sicher sein (breite Wildcards), sonst entstehen mehr Prompts statt weniger.
