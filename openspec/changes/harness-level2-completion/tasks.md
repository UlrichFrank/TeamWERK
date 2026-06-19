## 0. Voraussetzung (Operations, kein Change-Code)

- [ ] 0.1 Git-Hooks aktivieren: `make hooks` ausführen (setzt `core.hooksPath` auf `.githooks`).

## 1. make audit (Documentation-as-Code)

- [ ] 1.1 `scripts/audit.sh` anlegen mit harten Checks (Exit ≠ 0): Go-Version `CLAUDE.md` vs `go.mod`; Raw-Tailwind-Farben in `web/src`; Unicode-Icons/Emojis in JSX; `go test ./internal/arch/`. Jeder Treffer mit Datei:Zeile. Grep eng auf JSX/className-Kontext fassen, um False-Positives zu vermeiden.
- [ ] 1.2 Warn-Check ergänzen (beeinflusst Exit-Code nicht): in `CLAUDE.md` dokumentierte `/api/`-Pfade, die nicht in `internal/app/router.go` vorkommen.
- [ ] 1.3 `make audit`-Target im Makefile (ruft `scripts/audit.sh`), Hilfetext + `.PHONY` ergänzen.

## 2. Review-Subagent

- [ ] 2.1 `.claude/agents/teamwerk-reviewer.md` anlegen: Review-Linse über brand-Tokens, SSE `hub.Broadcast`/`useLiveUpdates`, Rollen/Vereinsfunktionen-Modell, Test-Standard, Migrationsregeln, Architektur-Layering; verweist auf `CLAUDE.md`/`AGENTS.md` als Quelle.

## 3. Doku

- [ ] 3.1 `CLAUDE.md`-Abschnitt „Harness / Verifikation" um `make audit` und den `teamwerk-reviewer`-Subagent ergänzen; `make hooks` als Voraussetzung kenntlich machen.

## 4. Verifikation

- [ ] 4.1 `make audit` auf sauberem Stand → Exit 0; testweise einen Drift einbauen (z. B. falsche Go-Version in CLAUDE.md) → Exit ≠ 0 mit Fundstelle; danach zurücknehmen.
- [ ] 4.2 Reviewer-Subagent gegen einen kleinen Test-Diff laufen lassen → meldet die erwartete Invarianten-Verletzung.
- [ ] 4.3 `core.hooksPath` zeigt nach 0.1 auf `.githooks` (pre-commit/pre-push aktiv).
- [ ] 4.4 `openspec validate harness-level2-completion --strict` ist grün.

## Test-Anforderungen

- **Keine neuen HTTP-Routen und keine geänderte Geschäftslogik** — reine Entwickler-/Agenten-Infrastruktur (Skript, Subagent, Doku). Daher keine Go-Handler-Tests erforderlich.
- **Invariante (audit deterministisch & hart)**: Bei vorhandenem hartem Drift endet `make audit` mit Exit ≠ 0; auf sauberem Stand mit Exit 0. → verifiziert in 4.1.
- **Invariante (audit rauschfrei)**: Der Route-Check eskaliert nie zu Exit ≠ 0 (nur Warnung). → verifiziert in 4.1.
- **Invariante (Reviewer trifft)**: Eine bewusst eingebaute Invarianten-Verletzung wird als Finding gemeldet. → verifiziert in 4.2.
