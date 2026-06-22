## ADDED Requirements


### Requirement: Kanonische Agenten-Konventionsquelle

Das Repository SHALL die verbindlichen Konventionen für Coding-Agenten zentral in `CLAUDE.md` bereitstellen. `CLAUDE.md` SHALL die nicht-verhandelbaren Hard-Rules im Kopf führen und die thematischen Detail-Kapitel aus `docs/agent/*.md` per `@`-Import vollständig in den Kontext laden. Es SHALL keine konkurrierende Konventionsdatei (z. B. eine separate `AGENTS.md`) geben, die `CLAUDE.md` widersprechen könnte.

#### Scenario: Agent findet die Hard-Rules
- **WHEN** ein Coding-Agent das Projekt öffnet und `CLAUDE.md` liest
- **THEN** findet er die Kern-Regeln (pnpm statt npm, Go 1.26 via `/usr/local/go/bin/go`, brand-* Tokens statt Raw-Tailwind, `hub.Broadcast` bei jeder Mutation, neue Route braucht Happy-Path- + Fehlerfall-Test, Rollen/Vereinsfunktionen-Modell, nächste freie Migrationsnummer)
- **AND** die thematischen Detail-Kapitel sind über `@docs/agent/*`-Importe verfügbar

#### Scenario: Keine konkurrierende Quelle
- **WHEN** im Repo-Root nach Konventionsdateien gesucht wird
- **THEN** ist `CLAUDE.md` die einzige maßgebliche Quelle (keine separate `AGENTS.md`, die driften könnte)

### Requirement: Automatische Go-Formatierung editierter Dateien

Der Harness SHALL sicherstellen, dass jede von einem Agenten via Edit/Write geänderte Go-Datei automatisch mit gofmt formatiert wird, bevor sie in den `pre-commit`-Hook gerät. Die Verdrahtung SHALL in der committeten `.claude/settings.json` als `PostToolUse`-Hook (Matcher `Edit|Write`) erfolgen und das vorhandene `scripts/claude-gofmt-hook.sh` aufrufen. Der Hook DARF für Nicht-Go-Dateien keine Wirkung haben und DARF bei fehlenden Tools nicht den Toolaufruf blockieren.

#### Scenario: Go-Datei wird nach Edit formatiert
- **WHEN** ein Agent eine `*.go`-Datei mit unformatiertem Code schreibt
- **THEN** ist die Datei unmittelbar danach gofmt-konform

#### Scenario: Nicht-Go-Datei bleibt unberührt
- **WHEN** ein Agent eine `*.tsx`- oder `*.md`-Datei schreibt
- **THEN** läuft der gofmt-Hook ohne Wirkung und ohne Fehler durch

### Requirement: Vorab freigegebene Routine-Befehle

Die committete `.claude/settings.json` SHALL eine geteilte `permissions.allow`-Liste enthalten, die langlebige, sichere Routine-Operationen ohne Prompt erlaubt. Sie DARF KEINE destruktiven oder Deploy-Befehle vorab freigeben. Die persönliche `.claude/settings.local.json` (gitignored) SHALL anschließend so entrümpelt werden, dass nur das verbleibt, was die geteilte Liste nicht abdeckt.

#### Scenario: Routine-Befehl prompted nicht
- **WHEN** ein Agent eine durch die geteilte Liste abgedeckte Routine-Operation ausführt (z. B. `go test`, `gofmt`, `pnpm -C web build`, `openspec validate`)
- **THEN** erscheint kein Permission-Prompt

#### Scenario: Destruktiver Befehl bleibt geschützt
- **WHEN** ein Agent einen nicht freigegebenen oder destruktiven Befehl ausführen will (z. B. Deploy, `rm -rf`)
- **THEN** ist er nicht durch die geteilte `settings.json` vorab erlaubt

### Requirement: Pre-Completion-Verifikationsroutine

Der Harness SHALL einen Slash-Command `.claude/commands/verify-change.md` bereitstellen, der vor Abschluss einer Änderung die Qualitäts-Gates und die Projekt-Invarianten prüft. Er SHALL mindestens umfassen: Backend-Tests (`make test`, inkl. Architektur-Test), Lint (`make lint`), Frontend-Build (`pnpm -C web build`) sowie die Invarianten-Checkliste (neue Route → Happy-Path + Fehlerfall-Test; Mutation → `hub.Broadcast` + Frontend `useLiveUpdates`; keine Raw-Tailwind-Farben; keine Unicode-Icons in JSX; neue Migration = nächste freie Nummer; `openspec validate` für offene Changes).

#### Scenario: Checkliste deckt alle Gates ab
- **WHEN** `/verify-change` aufgerufen wird
- **THEN** führt er durch Backend-Tests, Lint, Frontend-Build und die Projekt-Invarianten

### Requirement: Driftfreie Projekt-Dokumentation

Die Projekt-Dokumentation SHALL mit dem tatsächlichen Stand übereinstimmen. Die in `CLAUDE.md` genannte Go-Version SHALL der in `go.mod` gepinnten Version entsprechen. Die Test-Regel in `openspec/config.yaml` SHALL mit dem verbindlichen Test-Standard (`CLAUDE.md`: jede neue Route braucht Happy-Path- + Fehlerfall-Test) übereinstimmen und DARF Tests nicht als optional bezeichnen.

#### Scenario: Go-Version stimmt überein
- **WHEN** die Go-Version in `CLAUDE.md` mit `go.mod` verglichen wird
- **THEN** sind beide identisch (1.25)

#### Scenario: Test-Regel ist konsistent
- **WHEN** die `tasks`-Regeln in `openspec/config.yaml` gelesen werden
- **THEN** verweisen sie auf den verbindlichen Test-Standard und bezeichnen Go-Handler-Tests nicht als optional
