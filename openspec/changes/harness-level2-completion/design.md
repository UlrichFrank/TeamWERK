## Context

Der Harness hat bereits: `CLAUDE.md`/`AGENTS.md` (Kontext), `internal/arch/arch_test.go` + Git-Hooks (Constraints, lokal erzwungen), `.claude/settings.json` gofmt-Hook (Middleware), `/verify-change` (Pre-Completion). Es fehlen aus dem Level-2-Katalog noch die Documentation-as-Code-Validierung und die agentenbasierte Review-Checkliste. Constraint: Solo/kleines Team, lokale Durchsetzung, kein Rauschen-Toleranz (ein verrauschtes Tool wird ignoriert).

## Goals / Non-Goals

**Goals:**
- Doku-gegen-Code-Drift deterministisch und reproduzierbar sichtbar machen.
- Eine unabhängige, projektkundige Review-Perspektive bereitstellen, die ein Diff selbstständig gegen die Invarianten prüft.
- Beide Bausteine rauschfrei halten (sonst werden sie ignoriert).

**Non-Goals:**
- Kein autonomer/geplanter Agent, keine Observability, kein A/B/Dashboard (Level 3).
- Kein Cloud-CI (lokale Durchsetzung bleibt die bewusste Wahl).
- Keine Aktivierung der Hooks als Change-Arbeit (`make hooks` ist Operations, 1 Befehl).

## Decisions

- **`make audit` ist hart (Exit ≠ 0) für deterministische Checks.** Ein weicher Audit, der nur Text ausgibt, wird ignoriert. Hart macht ihn als Gate verwendbar (später optional in `pre-push`). Alternative „nur informativ" verworfen.
- **Route-Drift bleibt Warnung, nicht Fehler.** `CLAUDE.md` dokumentiert bewusst eine *kuratierte Teilmenge* der 221 Routen — ein harter Vollständigkeits-Check würde dauerhaft falsch-positiv sein. Geprüft wird nur die nützliche Richtung: dokumentierter `/api/`-Pfad, der nicht mehr im Router existiert (veraltete Doku). Als Warnung, weil die Pfad-Extraktion aus Markdown heuristisch ist.
- **Audit-Logik in `scripts/audit.sh`, nicht inline im Makefile.** Testbar, lesbar, vom Makefile **und** später ggf. vom `pre-push`-Hook aufrufbar (DRY).
- **Review als Subagent (`.claude/agents/`), nicht als Slash-Command.** Der Artikel verlangt eine „agent-specific" Review-Instanz. Ein Subagent hat eigenen Kontext und liest den Diff autonom (komplementär zu `/verify-change`, das der arbeitende Agent an sich selbst ausführt). Eine zweite Slash-Command-Checkliste wäre nur Selbstprüfung in grün — kein unabhängiges Urteil. Der generische `code-review`-Skill kennt die TeamWERK-Invarianten nicht; deshalb ein projektspezifischer Reviewer.
- **Erweiterung der bestehenden `agent-harness`-Capability** (ADDED Requirements), keine neue Capability — es ist dieselbe Harness-Domäne.

## Risks / Trade-offs

- **[Raw-Tailwind/Icon-Grep produziert False-Positives]** (z. B. `text-red-` in einem Kommentar/String) → Grep eng fassen (auf `className`/JSX-Kontext zielen), und Treffer mit Datei:Zeile ausgeben, damit man sie schnell prüfen kann.
- **[Route-Extraktion aus Markdown ist heuristisch]** → bewusst nur Warnung, blockiert nie.
- **[Reviewer-Subagent driftet gegenüber CLAUDE.md]** → Reviewer verweist auf `CLAUDE.md`/`AGENTS.md` als Quelle statt Regeln zu duplizieren; hält nur die Review-Linse fest, nicht die Regelinhalte.
- **[`make audit` als hartes Gate nervt im Alltag]** → vorerst NICHT automatisch in `pre-push`; bleibt manuell, bis es sich bewährt. Einhängen ist eine separate, spätere Entscheidung.

## Migration Plan

1. `scripts/audit.sh` + `make audit`-Target anlegen.
2. `.claude/agents/teamwerk-reviewer.md` anlegen.
3. `CLAUDE.md`-Harness-Abschnitt ergänzen.
4. Voraussetzung umsetzen/dokumentieren: `make hooks` aktiviert die bestehenden Git-Hooks.

Rollback: reine Config/Skripte/Doku — Dateien entfernen bzw. `git checkout`. Kein Produktionsrisiko.

## Open Questions

- Keine offen. `make audit` wird vorerst NICHT in `pre-push` eingehängt (separate spätere Entscheidung, sobald rauschfrei bestätigt).
