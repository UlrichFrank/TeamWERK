# AGENTS.md

Provider-agnostischer Einstieg für Coding-Agenten (Claude, Codex, Cursor, Gemini, …).

> **Kanonische Quelle: [`CLAUDE.md`](./CLAUDE.md).** Diese Datei destilliert nur die
> nicht-verhandelbaren Regeln. Bei Konflikten oder Detailfragen gilt `CLAUDE.md`.

## Projekt in einem Satz

TeamWERK — interne Verwaltungsplattform für Team Stuttgart (Handball).
**Stack:** Go 1.26 + Chi v5 · SQLite (WAL, `modernc.org/sqlite`, kein CGo) · React 18 + Tailwind v3 · Vite · JWT-Auth.

## Hard Rules (nicht verhandelbar)

1. **`pnpm`, niemals `npm`** — für alle Frontend-/npm-Operationen.
2. **Go 1.26+** (go.mod: `go 1.26.0`). `/usr/local/go/bin/go` als Bootstrap zieht via `GOTOOLCHAIN` automatisch die 1.26-Toolchain; ein exportiertes `GOROOT=/usr/local/go` muss dafür ungesetzt sein.
3. **Nur `brand-*`-Tokens**, keine Raw-Tailwind-Farben (`bg-gray-50`, `text-red-600`, …). Tokens siehe `tailwind.config.js` / `CLAUDE.md`.
4. **Keine Unicode-Icons/Emojis in JSX** — `lucide-react` verwenden (`<Menu>`, `<X>`, `<Check>`, …).
5. **SSE-Pflicht:** Jede Mutations-Route (`POST`/`PUT`/`DELETE`) ruft `h.hub.Broadcast("domain-event")`; das Frontend abonniert mit `useLiveUpdates`. Fehlt eine Seite, bleibt sie nach fremden Änderungen stumm.
6. **Tests pro neuer Route (Pflicht):** mindestens Happy-Path (Erfolg) **und** Fehlerfall (401/403/400/404/409). Keine Dummy-Assertions.
7. **Rollen vs. Vereinsfunktionen** sind orthogonal:
   - System-Rolle (`users.role`): `admin` | `standard` — via `auth.RequireRole(...)`.
   - Vereinsfunktion (`member_club_functions.function`): `spieler`, `trainer`, `sportliche_leitung`, `vorstand`, `vorstand_beisitzer`, `kassierer` — via `auth.RequireClubFunction(...)` / `claims.HasFunction(...)`.
   - **Eltern** sind keine Vereinsfunktion → `claims.IsParent` (aus `family_links`), nie `HasFunction("elternteil")`.
   - `sportvorstand` existiert nicht (= `vorstand` + `sportliche_leitung`).
8. **Migrations:** neue Datei `internal/db/migrations/00N_*.up.sql` + `.down.sql` mit der **nächsten freien Nummer**; nie eine Nummer ≤ aktueller DB-Version (golang-migrate überspringt sie lautlos).
9. **Kein ORM** — direktes `database/sql`. Ein Package pro Domäne unter `internal/`, Handler-Struct-Pattern (`NewHandler(db, hub)`).
10. **Architektur-Layering** (per `internal/arch/arch_test.go` erzwungen): Domain-Packages importieren sich nicht gegenseitig; die Foundation-Schicht importiert keine Domain/Composition.

## Verifikation vor „fertig"

- `make hooks` einmalig (aktiviert Git-Hooks: pre-commit gofmt, pre-push Gate).
- `make test` (Backend race + vitest, inkl. Architektur-Test) · `make lint` · `pnpm -C web build`.
- Slash-Command **`/verify-change`** führt durch alle Gates + Projekt-Invarianten.

## Workflow

- Spezifikationsgetrieben über **OpenSpec** (`openspec/`): Proposal → Design → Specs → Tasks → Apply → Archive.
- **Conventional Commits** verpflichtend: `feat|fix|refactor|chore|docs|style|test(scope): …`.
