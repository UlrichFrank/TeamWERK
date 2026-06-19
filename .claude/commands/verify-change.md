---
description: Pre-Completion-Checkliste — Build/Test/Lint + TeamWERK-Projekt-Invarianten prüfen, bevor eine Änderung als fertig gilt.
---

Du bist im Begriff, eine Änderung abzuschließen. Arbeite diese Checkliste **vollständig** ab,
bevor du dem Nutzer „fertig" meldest. Führe die Gates tatsächlich aus (nicht nur behaupten)
und prüfe die Invarianten gegen das aktuelle Diff (`git diff` / `git status`). Melde am Ende
eine kompakte Tabelle: jedes Item mit ✓ / ✗ / n/a und bei ✗ die konkrete Fundstelle.

## Gate 1 — Backend

- `make test` ist grün (Go race-Tests **inkl.** Architektur-Test `internal/arch/`, plus vitest).
- `make lint` ist grün (`golangci-lint`). Falls nicht installiert: Hinweis ausgeben, nicht als Fehler werten.

## Gate 2 — Frontend

- `pnpm -C web build` ist grün (= `tsc -b` + vite, also Typecheck inklusive).

## Gate 3 — Projekt-Invarianten (gegen das Diff prüfen)

- **Neue HTTP-Route?** → mindestens ein Happy-Path-Test (Erfolg) **und** ein Fehlerfall-Test (401/403/400/404/409). Keine Dummy-Assertions.
- **Mutations-Route (`POST`/`PUT`/`DELETE`) hinzugefügt/geändert?** → ruft `h.hub.Broadcast("<domain-event>")` auf, **und** die betroffene Frontend-Seite abonniert via `useLiveUpdates`.
- **Frontend-Änderung?** → keine Raw-Tailwind-Farben (`bg-gray-*`, `text-gray-*`, `text-red-*`, `bg-red-*`, …) — nur `brand-*`-Tokens. Keine Unicode-Icons/Emojis in JSX — `lucide-react` verwenden.
- **Neue Migration?** → Datei `internal/db/migrations/00N_*.up.sql` + `.down.sql` mit der **nächsten freien Nummer** (nie ≤ aktueller DB-Version).
- **Rollen/Funktionen korrekt?** → System-Rolle via `auth.RequireRole`, Vereinsfunktion via `auth.RequireClubFunction`/`claims.HasFunction`, Eltern via `claims.IsParent` (nie `HasFunction("elternteil")`).

## Gate 4 — OpenSpec

- Für jeden offenen Change: `openspec validate <change> --strict` ist grün.
- Tasks im aktiven Change, die diese Arbeit abdecken, sind als `- [x]` markiert.

## Gate 5 — Commit-Hygiene

- Conventional-Commit-Format bereit: `feat|fix|refactor|chore|docs|style|test(scope): …`.
- `git status` enthält keine versehentlich erfassten fremden Änderungen.

Wenn ein Gate ✗ ist: **nicht** „fertig" melden — die Ursache beheben oder dem Nutzer die Blockade klar benennen.
