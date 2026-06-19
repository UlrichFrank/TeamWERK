# CLAUDE.md

Guidance for Claude Code working in this repository.
Provider-agnostische Kurzfassung der Hard-Rules: [`AGENTS.md`](./AGENTS.md).

## Hard Rules

- **`pnpm`, nie `npm`** für alle Frontend-/npm-Operationen.
- **Go-Befehle mit `/usr/local/go/bin/go`** (1.25) — nicht dem Homebrew-`go` (1.26+, inkompatibel mit go.mod).
- **Nur `brand-*`-Tokens**, keine Raw-Tailwind-Farben (`bg-gray-50`, `text-red-600`, …).
- **Keine Unicode-Icons/Emojis in JSX** — `lucide-react`.
- **Jede Mutations-Route ruft `h.hub.Broadcast(...)`**, das Frontend abonniert mit `useLiveUpdates` (siehe Gotcha SSE).
- **Jede neue Route bekommt Tests** (Happy-Path + Fehlerfall).
- **Kein ORM** — direktes `database/sql`.

---

## Überblick

TeamWERK — interne Verwaltungsplattform für Team Stuttgart (Handball), läuft unter
`https://internal.team-stuttgart.org` auf einem IONOS VPS (Linux XS, 1 GB RAM).

**Stack:** Go 1.25 + Chi v5 · SQLite (WAL, `modernc.org/sqlite`, kein CGo) · React 18 + Tailwind v3 · Vite · JWT-Auth.

**Struktur:** Entrypoint `cmd/teamwerk/main.go` (`embed.FS`, Subcommands wie `migrate`/`scheduler:run`/`create-admin`/`gen-vapid`, baut `app.Handlers` und mountet `app.BuildRouter`). Der **Routenbaum mit allen Auth-Tiers liegt in `internal/app/router.go` (`BuildRouter`)**. Je ein Package pro Domäne unter `internal/` (`auth`, `members`, `duties`, `games`, …). Migrations in `internal/db/migrations/`. Frontend in `web/` (Vite + React; `App.tsx` = Routen-Baum, `lib/api.ts` = Axios mit Auto-Refresh, eine Datei pro Route in `pages/`). Deploy-Skripte in `deploy/`. Bei Bedarf per `ls`/Glob erkunden statt aus dem Gedächtnis.

---

## Entwicklungsworkflow

```bash
# Lokaler Start (zwei Prozesse)
go run ./cmd/teamwerk        # Backend :8080  (braucht web/dist/ wegen //go:embed — sonst web/dist/.gitkeep anlegen)
cd web && pnpm dev           # Vite :5173, proxyt /api → :8080

make build                   # pnpm build + go build → bin/teamwerk
make deploy                  # build + rsync auf VPS + systemctl restart (führt automatisch migrate up aus)
make migrate-up / migrate-down
make test / lint / coverage
```

**Neue Migration:** `internal/db/migrations/00N_beschreibung.up.sql` + `.down.sql` mit der **nächsten freien Nummer**. Nie eine Nummer ≤ aktueller DB-Version — golang-migrate überspringt sie lautlos.

---

## Go-Konventionen

**Handler-Pattern** — ein Package pro Domäne unter `internal/`:

```go
type Handler struct{ db *sql.DB; hub *hub.EventHub }
func NewHandler(db *sql.DB, hub *hub.EventHub) *Handler { return &Handler{db, hub} }
func (h *Handler) MethodName(w http.ResponseWriter, r *http.Request) { … }
```

**Router (Chi v5):** `r.Get("/api/members/{id}", membH.Get)`, auslesen mit `r.PathValue("id")`.

**Auth/JWT:** `claims := auth.ClaimsFromCtx(r.Context())`. Access-Token 15 min (HS256, im Frontend im Memory), Refresh-Token 7 Tage (opaque, SHA256-Hash in DB, HttpOnly-Cookie). Gating via `auth.RequireRole(...)` und `auth.RequireClubFunction(...)`.

**DB-Zugriff:** Kein ORM. Bei DB-Open automatisch `PRAGMA journal_mode=WAL; foreign_keys=ON;`. Nullable Felder über `sql.NullInt64` & Co. scannen, dann `.Valid` prüfen.

**E-Mail:** `h.mailer.Send(to, subject, body)` (net/smtp, Config aus `.env`).

### Rollen und Vereinsfunktionen (zwei orthogonale Dimensionen)

**System-Rolle** (`users.role`, JWT-Claim `role`, via `auth.RequireRole(...)`):

| Rolle | Bedeutung |
|---|---|
| `admin` | Vollzugriff; umgeht alle `RequireClubFunction`-Checks |
| `standard` | Default; Zugriff aus Vereinsfunktionen + Ownership |

**Vereinsfunktion** (`member_club_functions.function`, 0–n pro Member, JWT-Claim `club_functions: string[]`, via `auth.RequireClubFunction(...)` / `claims.HasFunction(...)`): `spieler`, `trainer`, `sportliche_leitung`, `vorstand`, `vorstand_beisitzer`, `kassierer`.

- **Eltern** sind keine Vereinsfunktion → `claims.IsParent` (aus `family_links`), nie `HasFunction("elternteil")`.
- `sportvorstand` existiert **nicht** (= `vorstand` + `sportliche_leitung`).

---

## API & Datenbank — Quelle der Wahrheit ist der Code

**Routen:** Die maßgebliche Liste steht in `internal/app/router.go` (`BuildRouter`, nach Auth-Tier gruppiert). Dort nachschlagen statt aus dem Gedächtnis — eine Doku-Kopie würde driften.

**Schema:** Maßgeblich sind die Migrations in `internal/db/migrations/` (`*.up.sql`). Dort die Tabellen/Spalten/CHECK-Constraints lesen.

### Namens- & Sprachkonvention

- **Backend-API-Routen: englisch**, lowercase/kebab-case, generische REST-Struktur `/api/{resource}/{id}/{action}` (z.B. `/api/members/{id}/bank-details`). Bestehende deutsche Ausnahmen (`/api/mitfahrgelegenheiten`) nicht als Vorbild nehmen.
- **Frontend-Routen (`App.tsx`, sichtbare Pfade): deutsch** (z.B. `/admin/saisons`, `/admin/beitragslauf`).
- Alle Frontend-API-Calls relativ zu `/api/` (Prefix in `lib/api.ts`: `baseURL: '/api'`).

### Auth-Tiers (wo gehört eine neue Route hin?)

| Tier | Zugriff |
|---|---|
| Public | Login, Register, Passwort-Reset, Beitrittsantrag, Downloads |
| Authenticated | alle Eingeloggten (Profil, Dienstbörse, Spiele, Chat, …) |
| Trainer + sportliche_leitung | Slots, Anfragen, Training, Venues |
| Vorstand (+ Trainer/sL) | Spiele, Kader, Duty-Slots, Saisons (lesen) |
| Vorstand | Mitglieder-CRUD, Verein, Teams, Nutzer, Einladungen, Duty-Types/-Templates |
| Vorstand + Kassierer | Mitglieder lesen, `PUT /members/{id}/bank-details` (Feld-Whitelist), Fee-Run |
| Admin only | Impersonate |

### Schema-Konventionen (nicht-ableitbar)

- **Geldbeträge in Cent** (z.B. `beitrags_saetze.betrag_eur`).
- **`player_memberships` ist eine View** über `kader_members` — kein direktes INSERT; stattdessen `INSERT INTO kader_members (kader_id, member_id) …`.
- **Beitragslauf-Protokoll ist keine Tabelle**, sondern append-only Textdatei pro Saison unter `BEITRAGSLAUF_DIR` (`./storage/beitragslauf-protokolle`) — ins Backup aufnehmen.
- **Status-Felder** sind CHECK-Constraints (z.B. `members.status`: `aktiv|verletzt|pausiert|ausgetreten`) — gültige Werte in der jeweiligen Migration nachsehen.

### Paginierung

`GET /api/members` und `GET /api/users`: `?search=&limit=50&offset=0` → `{ items: [...], total: N }`. Frontend: serverseitige Suche (auf Mobile `sticky top-0 z-10`) + „Mehr laden"-Button, kein clientseitiges `filter()`.

---

## Frontend-Konventionen

**Auth:** `const { user, login, logout, loading } = useAuth()` — `user` hat `email`, `role` (aus JWT).
**API:** `import { api } from '../lib/api'` → `api.get('/members')` (Bearer + Auto-Refresh bei 401).
**Neue Seite:** Datei in `web/src/pages/`, Route in `App.tsx` unter dem `AppShell`-Outlet, ggf. Nav-Eintrag in `AppShell.tsx` (`roles`-Array).

### Styling (Tailwind v3)

Keine eigene CSS-Datei außer `index.css` (nur `@tailwind`). Schrift: Hanken Grotesk.
Marke: Schwarz `#181310`, Gelb `#FDE400`, Weiß `#FFFFFF`; sekundär Blau `#3E4A98`, Grün `#6EB42E`.
**Keine raw Tailwind-Farben** — immer `brand-*`-Tokens (`tailwind.config.js`):

| Token | Wert | Ersetzt |
|---|---|---|
| `brand-surface-card` | `#F9FAFB` | `bg-gray-50` |
| `brand-text` | `#111827` | `text-gray-900`, `text-black` |
| `brand-text-muted` | `#6B7280` | `text-gray-500` |
| `brand-text-subtle` | `#9CA3AF` | `text-gray-400`, Placeholder |
| `brand-border` | `#D1D5DB` | `border-gray-300` |
| `brand-border-subtle` | `#E5E7EB` | `border-gray-200`, Divider |
| `brand-danger` | `#C0253A` | `text-red-600`, destruktiv |
| `brand-danger-light` | `#FCEEF1` | `bg-red-50/100` in Alerts |
| `brand-info` | `#3B82F6` | Info-Alert |
| `brand-table-select` | `#E5E7EB` | Row-Hover |

**Verbindliche Klassen-Strings:**

- **Button Primary:** `bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed`
- **Button Small (Tabellen):** `bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed`
- **Button Danger:** `bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed`
- **Input:** `w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow`
- **Card:** `bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6` (Tabellen-Container: `… overflow-hidden`)
- **Modal:** `bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6`
- **Alert Info:** `p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text`
- **Alert Fehler:** `p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger`
- **Tabellen-Header (th):** `bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left`
- **Tabellen-Row:** `hover:bg-brand-table-select transition-colors` / Zelle `px-4 py-3 text-sm text-brand-text`

**Icons (lucide-react):** Keine Unicode/Emojis in JSX. `☰`→`<Menu>`, `✕`→`<X>`, `⋮`→`<MoreVertical>`, `▸/▾`→`<ChevronRight>/<ChevronDown>`, `✓`→`<Check>`, `⚠`→`<AlertTriangle>`, `🗑`→`<Trash2>`, `«/»`→`<ChevronsLeft>/<ChevronsRight>`, Heim→`<Home>`, Auswärts→`<MapPin>`. Größen `w-4 h-4` (inline) · `w-5 h-5` (Buttons/Nav) · `w-6 h-6` (standalone). Icon-only-Buttons brauchen `aria-label`.

**Button-Position:** Listen → Primär oben rechts neben `<h1>`; Formulare → unten; Inline-Form in Karte → unten in der Karte.

### Mobile & PWA

- **Breakpoint:** `sm:` (640px) ist die einzige Mobile/Desktop-Grenze. Keine `md:`-Logik für Mobile.
- **Navigation:** Hamburger (`<Menu>`) öffnet die Sidebar als Fixed-Overlay (`z-50`) mit Backdrop. Desktop-Sidebar immer sichtbar. Main-Padding Mobile `px-4 py-4` statt `p-8`; Deko-Klassen (`rounded-tl-3xl …`) nur `sm:`.
- **Tabellen auf Mobile:** Card-Layout statt `<table>`; Actions hinter `<MoreVertical>`-Dropdown; Multi-Feld-Inline-Edit als Modal. Shared: `MobileCard`, `ActionMenu`, `EditModal` in `web/src/components/`.
- **Touch-Targets:** min. 44px Höhe → `py-2.5` auf Mobile (`sm:py-1.5`).
- **PWA** (`vite-plugin-pwa`): Service Worker network-first für `/api/*`, cache-first für Assets. Manifest `web/public/manifest.json`, Icons `web/public/icons/`. Offline-Shell mit Hinweis.

---

## Bekannte Gotchas

**SQLite DATE-Felder:** API gibt Datumsfelder als ISO-Timestamp zurück (`"2026-05-30T00:00:00Z"`). Im Frontend immer `.slice(0, 10)` für Vergleiche und `date + 'T12:00:00'`-Konstruktionen.

**Aktive Saison:** Spielplan, Dienst-Erstellung und Dienst-Konten setzen eine aktive Saison voraus (Verwaltung `/admin/saisons`). Ohne aktive Saison schlagen game- und slot-Inserts mit FK-Fehler fehl.

**SSE Live-Updates:** Jede Mutations-Route (`POST`/`PUT`/`DELETE`) muss `h.hub.Broadcast("domain-event")` aufrufen; das Frontend abonniert mit `useLiveUpdates((event) => { if (event === 'domain-event') reload() })`. `Handler`-Structs mit Mutationen brauchen ein `hub *hub.EventHub`-Feld (in `main.go` via `NewHandler(db, hub)` übergeben). Fehlt `Broadcast` (Backend) **oder** `useLiveUpdates` (Frontend), bleibt die Seite nach fremden Änderungen stumm.

**Push Notifications:** Infrastruktur in `internal/notifications/`. VAPID-Keys via `go run ./cmd/teamwerk gen-vapid` in `.env` (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_EMAIL`). Senden immer als Goroutine (darf den HTTP-Response nicht blockieren):

```go
go notifications.SendToUsers(h.db, h.cfg, []int{userID1, userID2}, "Titel", "Text", "/ziel-url")
```

Frontend-Hook `usePushSubscription` (in `AppShell.tsx`) registriert automatisch beim App-Start. iOS nur als Homescreen-PWA (`display-mode: standalone`). Subscriptions in `push_subscriptions`; ungültige Endpoints (HTTP 410) werden bereinigt. Scheduled Notifications → Job im `internal/scheduler/`, idempotent via `notification_log`.

**Auto-Duty-Regen:** Jede Spieländerung (`POST/PUT/DELETE /api/games/{id}`) triggert Regeneration der Dienst-Slots für Event-Datum ± 1 Tag (beachtet `same_day_behavior`/`adjacent_day_behavior` der Duty-Types). Slots mit `is_custom=1` (manuell angelegt/editiert) werden geschont. Response enthält `regen_summary`. Vor Deploy manuell-editierte Bestandsslots mit `UPDATE duty_slots SET is_custom=1 WHERE id IN (...)` schützen.

**SEPA-Beitragslauf:** `/admin/beitragslauf` (`vorstand`, `kassierer`, `admin`). Bewusst einfach: **kein Pro-rata** (voller Jahresbeitrag), Fälligkeit **immer 01.07.** der Saison, alle Lastschriften **RCUR** (keine FRST), Spieler gelten als Kinder. Vor dem ersten Lauf müssen die SEPA-Stammdaten (`glaeubiger_id`, `iban`, `bic`, `kontoinhaber`) unter Einstellungen → Verein gepflegt sein, sonst liefert `POST /api/fee-run/export` HTTP 400. Beitragsmatrix unter Einstellungen → Beiträge (3 Kategorien, Cent, Historie via `valid_from`). „Lauf bestätigen" schreibt das append-only Saison-Protokoll. Kassierer darf Mitglieder lesen + Bankdaten via `PUT /api/members/{id}/bank-details` korrigieren.

---

## Test-Standard

Jede neue HTTP-Route **muss** mindestens **Happy-Path** (Erfolg) und **Fehlerfall** (401/403/400/404/409) abdecken. Tests prüfen fachliche Invarianten — keine Dummy-Assertions zur Coverage-Erhöhung.

Jeder OpenSpec-Proposal mit neuen Routen / geänderter Geschäftslogik braucht einen Abschnitt `## Test-Anforderungen` (Route → Testname + erwarteter Status, plus die garantierte Invariante).

**Fixtures** in `internal/testutil/`: `NewDB`, `NewServer`, `CreateUser`, `CreateMember`, `CreateSeason`, `CreateTeam`, `CreateGame`, `CreateKader`, `CreateDutyType`, `CreateDutySlot`, `CreateInvitationToken`, `CreatePasswordResetToken`, `CreateRefreshToken`.

`make coverage` → stdout + HTML nach `/tmp/teamwerk-coverage.html`. Coverage ist Indikator, kein Gate.

---

## Harness / Verifikation

Konventionen werden mechanisch durchgesetzt, nicht nur dokumentiert.

- **Git-Hooks** (`make hooks`, in `make init`): `pre-commit` = gofmt auf gestagete Go-Dateien; `pre-push` = volles Gate (`go vet`, `go test -race ./...` inkl. Architektur-Test, `golangci-lint`, `pnpm -C web build/test/lint`, `openspec validate`). Notausgang: `git push --no-verify`.
- **Architektur-Test** `internal/arch/arch_test.go` (stdlib, Teil von `make test`): Domain-Packages importieren sich nicht gegenseitig; Foundation importiert keine Domain/Composition; jedes neue `internal/`-Package muss klassifiziert werden.
- **gofmt-Selbstkorrektur:** `PostToolUse`-Hook (`scripts/claude-gofmt-hook.sh`) formatiert via Edit/Write geänderte `*.go`-Dateien.
- **Pre-Completion:** Slash-Command **`/verify-change`** prüft Build/Test/Lint + Projekt-Invarianten (Route→Tests, Mutation→`Broadcast`/`useLiveUpdates`, brand-Tokens, lucide-Icons, Migrationsnummer, `openspec validate`).
- **Permissions:** geteilte Routine-Befehle in `.claude/settings.json`; `.claude/settings.local.json` (gitignored) nur maschinenspezifisch.

---

## OpenSpec-Workflow

Spezifikationsgetrieben über `openspec/` (Proposal → Design → Specs → Tasks → Apply → Archive).

**Conventional Commits** verpflichtend — Format `<type>(<scope>): <beschreibung>`.
Typen: `feat`, `fix`, `refactor`, `chore`, `docs`, `style`, `test`. Scope = Domänen-Package (`duties`, `members`, `auth`, `db`, `pwa`, …).
Beispiel: `feat(duties): Dienstbörse zeigt offene Slots nach Datum sortiert`.

**Ein Commit pro OpenSpec-Task** (nicht alle Tasks zusammenfassen); abschließender Commit archiviert ggf. das Proposal.

---

## Deployment & VPS

IONOS VPS Linux XS · Binary `/usr/local/bin/teamwerk` · systemd-Service `teamwerk` · Nginx Reverse Proxy 443→8080 (Certbot). Config `/etc/teamwerk/env` (PORT, DB_PATH, JWT_SECRET, SMTP_*). DB `/var/lib/teamwerk/teamwerk.db`. Scheduler-Cronjob `* * * * * /usr/local/bin/teamwerk scheduler:run`. Erstaufbau: `bash deploy/setup-vps.sh` (root).

SSH-Alias `vServer` (in `.env`), direkt `https://217.160.118.39`. Domain + Certbot-Zertifikat noch ausstehend.

```bash
make migrate-remote-up                               # Migrationen auf VPS
make create-admin-remote EMAIL=… PASSWORD=… NAME=…   # Admin anlegen
```
