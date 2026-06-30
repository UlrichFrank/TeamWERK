# Überblick

TeamWERK — interne Verwaltungsplattform für Team Stuttgart (Handball), läuft unter
`https://internal.team-stuttgart.org` auf einem IONOS VPS (Linux XS, 1 GB RAM).

**Stack:** Go 1.26 + Chi v5 · SQLite (WAL, `modernc.org/sqlite`, kein CGo) · React 19 + Tailwind v3 · Vite · JWT-Auth.

**Struktur:** Entrypoint `cmd/teamwerk/main.go` (`embed.FS`, Subcommands wie `migrate`/`scheduler:run`/`create-admin`/`gen-vapid`, baut `app.Handlers` und mountet `app.BuildRouter`). Der **Routenbaum mit allen Auth-Tiers liegt in `internal/app/router.go` (`BuildRouter`)**. Je ein Package pro Domäne unter `internal/` (`auth`, `members`, `duties`, `games`, …). Migrations in `internal/db/migrations/`. Frontend in `web/` (Vite + React; `App.tsx` = Routen-Baum, `lib/api.ts` = Axios mit Auto-Refresh, eine Datei pro Route in `pages/`). Deploy-Skripte in `deploy/`. Bei Bedarf per `ls`/Glob erkunden statt aus dem Gedächtnis.
