## Why

Im Code, in der DB und in der UI wird das Feature inkonsistent als „Mitfahrgelegenheiten", „Mitfahrten" und „Mitfahrt-Paarungen" bezeichnet. Im Navigations-Label heißt es bereits „Mitfahrten" (sowohl in `internal/policy/rules.go` als auch in `web/src/components/AppShell.tsx`), während Routen, Tabelle, SSE-Event und Page-Komponente noch das längere „Mitfahrgelegenheiten" tragen. Ein einheitlicher Name reduziert kognitive Last, vereinfacht Suche und Refactoring und macht die Specs konsistent.

## What Changes

- **BREAKING** API-Routen umbenannt: `/api/mitfahrgelegenheiten*` → `/api/mitfahrten*` (3 Routen: `GET`/`POST`/`DELETE`, registriert in `internal/app/router.go`).
- **BREAKING** Frontend-Route umbenannt: `/mitfahrgelegenheiten` → `/mitfahrten`.
- **BREAKING** DB-Tabelle umbenannt: `mitfahrgelegenheiten` → `mitfahrten` (neue Migration **002**, per `ALTER TABLE … RENAME TO`).
- SSE-Event-Name umbenannt: `mitfahrgelegenheiten` → `mitfahrten` (Backend-`Broadcast` + Frontend-`useLiveUpdates`).
- Page-Komponente und Datei umbenannt: `MitfahrgelegenheitenPage.tsx` → `MitfahrtenPage.tsx`.
- Backend-Navigation `internal/policy/rules.go`: `NavItem{"Mitfahrten", "/mitfahrgelegenheiten"}` → Pfad `/mitfahrten`.
- Push-Deep-Links und Notification-Titel angepasst: Deep-Link `/mitfahrgelegenheiten` → `/mitfahrten`, Titel „Mitfahrgelegenheit" → „Mitfahrt", Body „sucht eine Mitfahrgelegenheit" → „sucht eine Mitfahrt".
- Doku (`CLAUDE.md`, `docs/anleitung-*.md`, `docs/berechtigungen.md`, `web/public/CHANGELOG.md`) nachgezogen.
- Bestehende Specs: nur die **Routen-/Pfad-/Event-/Tabellen-/Komponenten-Referenzen** in den betroffenen Live-Specs werden per MODIFIED-Delta auf die neuen Bezeichner gebracht.

### Bewusst **nicht** im Scope (Begründung in design.md)

- **Go-Package-Rename** `internal/carpooling/` — englischer Domain-Begriff, kollidiert nicht mit deutschen UI-Texten, hoher Diff ohne Nutzergewinn.
- **Tabelle `mitfahrt_paarungen`, `carpooling_events`** — bereits konsistent bzw. internes Eventlog.
- **OpenSpec-Capability-/Spec-Folder-Rename** — die Folder behalten ihre Namen (`mitfahrgelegenheiten-board`, `-nav`, `-team-filter`, `-meine-filter`), symmetrisch zur Go-Package-Entscheidung. Nur der **Inhalt** (Routen/Pfade/Begriffe) wird angepasst, nicht der Folder-Name. Vermeidet driftanfällige REMOVED+ADDED-Volltextkopien.
- **Frontend-Redirect** `/mitfahrgelegenheiten` → `/mitfahrten` — interne App, keine öffentlichen Bookmarks; bei Bedarf nachreichbar.
- Funktionale Änderungen am Feature.

## Capabilities

### New Capabilities

(keine)

### Modified Capabilities

Nur Bezeichner-Referenzen (Routen, Frontend-Pfad, SSE-Event-String, DB-Tabelle, Komponentenname, Push-Titel) werden in den folgenden Live-Specs aktualisiert — Folder-Namen und fachliche Logik bleiben:

- `mitfahrgelegenheiten-board` — Routen/Pfad/Begriff in den vorhandenen UI-Requirements.
- `mitfahrgelegenheiten-nav` — Pfad `/mitfahrten`, Komponente `MitfahrtenPage`, Nav-Label.
- `mitfahrgelegenheiten-team-filter` — `GET /api/mitfahrten`.
- `carpooling-team-filter` — `GET /api/mitfahrten` (Rollenfilter + `?team_id`).
- `carpooling-elternzugang` — `POST/GET/DELETE /api/mitfahrten`.
- `carpooling-notifications` — `POST/DELETE /api/mitfahrten`, Push-Titel „Mitfahrt", Deep-Link `/mitfahrten`.
- `dashboard-migration` — Link/Pfad `/mitfahrten`.
- `dashboard-carpooling-hint` — Tabellenref `mitfahrten.treffpunkt`.
- `dashboard-offene-gesuche` — Tabellenref `mitfahrten.typ='suche'`.
- `sse-live-updates` — Event-String `mitfahrten`, Route, Komponente `MitfahrtenPage`.
- `number-spinner` — Datei `MitfahrtenPage.tsx`.
- `permissions` — Routenliste + Frontend-Pfad-Zeile.

> `mitfahrgelegenheiten-meine-filter` enthält keine umbenannten Bezeichner → kein Delta nötig. `mitfahrt-paarungen` nutzt nur `/api/mitfahrt-paarungen` (bleibt) → kein Delta nötig, sofern keine Tabellen-Referenz vorliegt.

## Impact

- **Backend (Code):** `internal/app/router.go` (Routen-Registrierung), `internal/carpooling/handler.go`, `internal/carpooling/paarungen_handler.go` (SQL, SSE-`Broadcast`, Push-Pfade/-Titel, Kommentare), `internal/dashboard/handler.go` (SQL-JOINs), `internal/scheduler/scheduler.go` (SQL + Push-Deep-Link), `internal/policy/rules.go` (Nav-Pfad).
- **Backend (Tests):** `internal/carpooling/handler_test.go`, `internal/carpooling/team_push_test.go`, `internal/dashboard/handler_test.go`, `internal/permissions/matrix_test.go` (Routen-Pfade + Insert-SQL auf `mitfahrten`).
- **DB:** Neue Migration `002_rename_mitfahrgelegenheiten.up.sql` + `.down.sql`. `ALTER TABLE mitfahrgelegenheiten RENAME TO mitfahrten`. Die FK-Referenzen aus `mitfahrt_paarungen.biete_id`/`suche_id` auf `mitfahrgelegenheiten(id)` werden von SQLite bei `PRAGMA foreign_keys=ON` automatisch auf `mitfahrten(id)` umgeschrieben (modernc.org/sqlite ≥ 3.40). Die partiellen Unique-Indizes (`idx_mitfahr_biete_unique`, `idx_mitfahr_suche_unique`) heißen schon `mitfahr*` und ziehen die Tabellenreferenz automatisch mit — kein Index-Rename nötig.
- **Frontend:** `web/src/App.tsx` (Route + Import), `web/src/components/AppShell.tsx` (Nav-`to`), `web/src/pages/MitfahrgelegenheitenPage.tsx` (Datei-Rename + Komponentenname + API-Pfade + Event-Filter), `web/src/pages/DashboardPage.tsx` (Links + `useLiveUpdates`-Event), `web/src/test/renderAsPersona.tsx` (Routenliste).
- **Specs:** MODIFIED-Deltas (Folder behalten ihre Namen) — siehe „Modified Capabilities".
- **Externe Konsumenten:** keine — interne App, kein öffentliches API.
- **Deploy-Atomarität:** Single-Binary mit `embed.FS`; Frontend-Bundle und Backend werden gemeinsam deployt → kein Übergangszustand altes-Frontend/neues-Backend, kein Doppel-Broadcast nötig.
