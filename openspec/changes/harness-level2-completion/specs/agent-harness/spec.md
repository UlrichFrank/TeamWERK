## ADDED Requirements

### Requirement: Deterministische Doku-Drift-Validierung (make audit)

Der Harness SHALL ein `make audit`-Target bereitstellen (Logik in `scripts/audit.sh`), das Drift zwischen Projekt-Dokumentation und tatsächlichem Code deterministisch erkennt. Die folgenden Checks SHALL **harte** Fehler sein (Exit ≠ 0): Go-Version in `CLAUDE.md` stimmt mit `go.mod` überein; keine Raw-Tailwind-Farben (`bg-gray-*`, `text-gray-*`, `text-red-*`, `bg-red-*`, …) in `web/src`; keine Unicode-Icons/Emojis in JSX; der Architektur-Test (`go test ./internal/arch/`) ist grün. Der Check „in `CLAUDE.md` dokumentierter `/api/`-Pfad existiert nicht mehr in `internal/app/router.go`" SHALL nur eine Warnung sein und den Exit-Code nicht beeinflussen. Jeder Treffer SHALL mit Datei und Zeile ausgegeben werden.

#### Scenario: Drift führt zu hartem Fehler
- **WHEN** die in `CLAUDE.md` genannte Go-Version von `go.mod` abweicht (oder eine Raw-Tailwind-Farbe / ein Unicode-Icon im Frontend auftaucht)
- **THEN** beendet sich `make audit` mit Exit ≠ 0 und nennt die konkrete Fundstelle

#### Scenario: Sauberer Stand
- **WHEN** keine harten Drift-Verletzungen vorliegen
- **THEN** beendet sich `make audit` mit Exit 0 (etwaige Route-Warnungen ausgenommen)

#### Scenario: Veraltete dokumentierte Route ist nur Warnung
- **WHEN** ein in `CLAUDE.md` dokumentierter `/api/`-Pfad nicht mehr im Router existiert
- **THEN** gibt `make audit` eine Warnung aus, beendet sich aber dennoch mit Exit 0 (sofern kein harter Check verletzt ist)

### Requirement: Projektspezifische Review-Instanz

Der Harness SHALL einen Review-Subagenten unter `.claude/agents/teamwerk-reviewer.md` bereitstellen, der einen Diff unabhängig gegen die TeamWERK-Invarianten prüft. Er SHALL mindestens abdecken: brand-* Tokens statt Raw-Tailwind; `lucide-react`-Icons statt Unicode; `hub.Broadcast` bei Mutationen plus `useLiveUpdates` im Frontend; korrektes Rollen-/Vereinsfunktionen-Modell (`RequireRole`/`RequireClubFunction`/`IsParent`); Test-Standard (neue Route → Happy-Path + Fehlerfall); Migrationsregeln; Architektur-Layering. Er SHALL `CLAUDE.md`/`AGENTS.md` als kanonische Quelle referenzieren statt Regelinhalte zu duplizieren.

#### Scenario: Diff verletzt eine Invariante
- **WHEN** der Reviewer einen Diff prüft, der eine Mutations-Route ohne `hub.Broadcast` einführt
- **THEN** meldet er dies als Finding mit Bezug auf die betroffene Stelle

#### Scenario: Reviewer dupliziert keine Regeln
- **WHEN** der Reviewer auf eine Konventionsfrage stößt
- **THEN** verweist er auf `CLAUDE.md`/`AGENTS.md` als Quelle statt die Regel inline neu zu definieren
