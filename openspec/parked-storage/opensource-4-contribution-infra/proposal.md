## Why

Ziel ist ein **echtes, gepflegtes Produkt** (nicht nur ein Code-Showcase): externe Vereine sollen es selbst hosten und Verbesserungen beitragen können. Dafür braucht das öffentliche Repo die übliche Contribution-Infrastruktur — Beitrags-Leitfaden, Verhaltenskodex, Sicherheits-Meldeweg, Issue-/PR-Vorlagen — und eine **öffentliche CI**, die dasselbe Gate erzwingt wie der heutige `pre-push`-Hook (`go vet`, `go test -race`, `golangci-lint`, `pnpm build/test/lint`, `openspec validate`). Ergänzend ein **Self-Hosting-Guide**, damit ein fremder Verein die Instanz aufsetzen kann.

Außerdem verlangt AGPL-3.0 §13: Wer den Dienst über ein Netzwerk anbietet, muss Nutzern den **Quellcode-Zugang** ermöglichen — das erfordert einen Source-Link in der laufenden App.

## What Changes

- **`CONTRIBUTING.md`** — Workflow (OpenSpec, Conventional Commits, Hard Rules), lokales Setup, Test-/Lint-Gate
- **`CODE_OF_CONDUCT.md`** — Contributor Covenant
- **`SECURITY.md`** — privater Meldeweg für Schwachstellen (kein öffentliches Issue)
- **Issue-/PR-Templates** unter `.github/`
- **Öffentliche CI** (`.github/workflows/ci.yml`) — spiegelt das `pre-push`-Gate
- **`docs/SELF_HOSTING.md`** — VPS-Setup, ENV-Referenz, Migrationen, Backup (inkl. Beitragslauf-Protokoll-Dir), Web-Push/VAPID
- **AGPL §13 Source-Link** im App-Footer (Verweis auf das öffentliche Repo)

## Capabilities

### New Capabilities

- `contribution-infrastructure`: Prozesse und Automatisierung, die externe Beiträge ermöglichen und Qualität mechanisch sichern (CI-Gate, Templates, Leitfäden), plus AGPL-§13-Quellzugang.

### Modified Capabilities

*(keine Anwendungs-Capability — Prozess/Infra; der Source-Link berührt das Frontend minimal)*

## Impact

- Neue Dateien: `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `.github/ISSUE_TEMPLATE/*`, `.github/pull_request_template.md`, `.github/workflows/ci.yml`, `docs/SELF_HOSTING.md`
- Minimale Frontend-Ergänzung: Source-Link (Footer/Über-Seite) — verweist auf konfigurierte Repo-URL (Überschneidung mit ② Config)
- CI nutzt Go 1.25 + pnpm; spiegelt `make`-Targets; keine Secrets nötig für Build/Test
- Voraussetzung: ③ (Doku verlinkt aus CONTRIBUTING) und ② (Repo-URL/Branding aus Config)

## Test-Anforderungen

| Einheit | Testname | Erwartete Invariante |
|---|---|---|
| CI-Workflow | (CI selbst) | `ci.yml` läuft grün auf einem sauberen Checkout und bricht bei Test-/Lint-Fehler ab |
| Frontend Footer | `TestFooter_RendersSourceLink` (oder Vitest-Äquivalent) | Source-Link auf das öffentliche Repo ist vorhanden (AGPL §13) |

**Garantierte Invariante:** Die öffentliche CI erzwingt dieselben Gates wie der lokale `pre-push`-Hook; ein PR, der Tests/Lint/Build/`openspec validate` bricht, kann nicht grün werden.
