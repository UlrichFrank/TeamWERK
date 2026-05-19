# CLAUDE.md

Guidance for Claude Code working in this repository.

## Überblick

TeamWERK (TeamWERK — Where Engagement Really Klicks) ist die interne Verwaltungsplattform für Team Stuttgart (Handball). Sie läuft unter `https://intern.team-stuttgart.org` auf einem IONOS VPS (Linux XS, 1 GB RAM).

**Stack:** Go 1.23 + Chi v5 · SQLite (WAL) · React 18 + Tailwind v3 · Vite · JWT-Auth

Die öffentliche TYPO3-Homepage (`team-stuttgart.org`) ist ein separates Repo und hat keinen Code-Overlap — lediglich ein Link im TYPO3-Header verweist hierher.

---

## Verzeichnisstruktur

```
teamwerk/
├── cmd/teamwerk/main.go   ← Einstiegspunkt: Router, embed.FS, Subcommands
├── internal/
│   ├── auth/                 ← JWT-Tokens, Middleware, Auth-Handler
│   ├── config/               ← Config-Struct (.env-Laden), Club/Season/Team-Handler
│   ├── db/                   ← SQLite öffnen + WAL setzen, Migrations
│   ├── duties/               ← Diensttypen, Slots, Assignments, Accounts
│   ├── mailer/               ← SMTP-Versand (net/smtp)
│   ├── members/              ← Mitglieder, Familien-Links, Fahrzeuginfo
│   └── scheduler/            ← Expired-Token-Bereinigung
├── internal/db/migrations/   ← golang-migrate .up/.down SQL-Dateien (embedded im Binary)
├── web/                      ← Vite + React-Projekt
│   ├── src/
│   │   ├── App.tsx           ← Routen-Baum (BrowserRouter)
│   │   ├── components/       ← AppShell (Sidebar-Nav)
│   │   ├── contexts/         ← AuthContext (User + JWT in Memory)
│   │   ├── lib/api.ts        ← Axios-Instanz mit Auto-Refresh-Interceptor
│   │   └── pages/            ← Eine Datei pro Route
│   └── vite.config.ts        ← Proxy /api → :8080
├── deploy/                   ← setup-vps.sh, nginx-intern.conf, teamwerk.service
├── Makefile
└── .env.example
```

---

## Entwicklungsworkflow

### Lokaler Start (zwei Prozesse)

```bash
# Terminal 1 — Go-Backend auf :8080
go run ./cmd/teamwerk

# Terminal 2 — Vite Dev-Server auf :5173 (proxyt /api → :8080)
cd web && npm run dev
```

`make dev` startet beide, aber das Go-Backend läuft dann im Hintergrund ohne sauberes Beenden.

> **Wichtig:** `go run ./cmd/teamwerk` erfordert `web/dist/` (wegen `//go:embed all:web/dist`).  
> Im reinen Backend-Dev einfach `web/dist/.gitkeep` anlegen oder `make build` einmal laufen lassen.

### Build & Deploy

```bash
make build    # npm run build + go build → bin/teamwerk
make deploy   # build + rsync auf VPS + systemctl restart
```

### Datenbank-Migrations lokal

```bash
make migrate-up    # go run ./cmd/teamwerk migrate up --db ./teamwerk.db
make migrate-down
```

Neue Migration anlegen: `internal/db/migrations/00N_beschreibung.up.sql` + `.down.sql`.

> **Warnung:** Nie eine Nummer einfügen, die ≤ der aktuellen DB-Version ist — golang-migrate überspringt sie lautlos. Immer die nächste freie Nummer verwenden.

---

## Go-Konventionen

### Package-Struktur

Jede Domäne (`auth`, `members`, `duties`, …) ist ein Package unter `internal/`. Pattern:

```go
type Handler struct{ db *sql.DB }
func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }
func (h *Handler) MethodName(w http.ResponseWriter, r *http.Request) { … }
```

### Router-Patterns (Chi v5)

```go
r.Get("/api/members/{id}", membH.Get)     // Pfadparameter
id := r.PathValue("id")                  // Auslesen (Go 1.22+ std, Chi wraps it)
```

Route-Gruppen:
- **Public**: Login, Register, Passwort-Reset, Beitrittsantrag
- **Authenticated** (`auth.Middleware`): alle eingeloggten Nutzer
- **Admin + Trainer** (`auth.RequireRole("admin","trainer")`): Slot-Verwaltung, Anfragen
- **Vorstand** (`auth.RequireRole("vorstand")`): Vereinskonfig, Nutzer
- **Admin only** (`auth.RequireRole("admin")`): Export

### Auth / JWT

```go
// Claims-Felder im JWT
type Claims struct {
    UserID int    `json:"uid"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

// Im Handler die Claims aus dem Context holen
claims := auth.ClaimsFromCtx(r.Context())
```

Access Token: 15 min, HS256, im Frontend im Memory.  
Refresh Token: 7 Tage, opaque (SHA256-Hash in DB), als HttpOnly-Cookie.

### Rollen

| Rolle | Bedeutung |
|-------|-----------|
| `admin` | Vollzugriff |
| `vorstand` | Vereinskonfig |
| `trainer` | Eigenes Team: Mitglieder sehen, Slots verwalten, Anfragen bearbeiten |
| `elternteil` | Nur eigene Kinder (via `family_links`), Dienstbörse, Konto |
| `spieler` | Eigenes Profil, Dienstbörse, Konto |

### Datenbankzugriff

Kein ORM, direktes `database/sql`. SQLite via `modernc.org/sqlite` (pure Go, kein CGo).  
Bei DB-Open wird automatisch gesetzt: `PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`

Nullable Felder:

```go
var jerseyNum sql.NullInt64
rows.Scan(…, &jerseyNum, …)
if jerseyNum.Valid { n := int(jerseyNum.Int64); m.JerseyNumber = &n }
```

### E-Mail-Versand

```go
h.mailer.Send(to, subject, body)  // net/smtp, SMTP-Config aus .env
```

---

## Datenbankschema

### 001 – Auth

| Tabelle | Schlüsselfelder |
|---------|----------------|
| `users` | `id`, `email` UNIQUE, `name`, `password` (bcrypt), `role` CHECK('admin','trainer','elternteil','spieler'), `team_id` FK |
| `refresh_tokens` | `user_id` FK, `token_hash` UNIQUE, `expires_at` |
| `invitation_tokens` | `email`, `team_id`, `role`, `token` UNIQUE, `expires_at`, `used_at` |
| `password_reset_tokens` | `user_id` FK, `token` UNIQUE, `expires_at`, `used_at` |
| `membership_requests` | `name`, `email`, `team_id` FK, `status` CHECK('pending','approved','rejected'), `handled_by`, `handled_at` |

### 002 – Core

| Tabelle | Schlüsselfelder |
|---------|----------------|
| `clubs` | `id`, `name`, `address`, `founded_year` |
| `seasons` | `id`, `name`, `start_date`, `end_date`, `is_active` (max. 1 aktiv) |
| `teams` | `id`, `name`, `age_class`, `gender` CHECK('m','f','mixed'), `is_active` |
| `team_trainers` | `team_id` FK, `user_id` FK (UNIQUE-Paar) |

### 003 – Members

| Tabelle | Schlüsselfelder |
|---------|----------------|
| `members` | `id`, `first_name`, `last_name`, `date_of_birth`, `pass_number` UNIQUE, `jersey_number`, `position`, `status` CHECK('aktiv','verletzt','pausiert','ausgetreten'), `user_id` FK |
| `team_memberships` | `member_id` FK, `team_id` FK, `season_id` FK, `is_primary`, UNIQUE(member,team,season) |
| `family_links` | `parent_user_id` FK, `member_id` FK (PK zusammen) |
| `vehicle_info` | `user_id` PK FK, `seats`, `notes` |

### 004 – Duties

| Tabelle | Schlüsselfelder |
|---------|----------------|
| `duty_types` | `id`, `name`, `hours_value`, `cash_substitute` |
| `duty_slots` | `id`, `duty_type_id` FK, `event_name`, `event_date`, `slots_total`, `slots_filled`, `role_description`, `season_id` FK |
| `duty_assignments` | `id`, `duty_slot_id` FK, `user_id` FK, `status` CHECK('pending','fulfilled','cash_substitute'), `cash_amount`, `fulfilled_at`, UNIQUE(slot,user) |
| `duty_accounts` | `user_id` PK FK, `season_id` PK FK, `soll`, `ist` |
| `duty_season_targets` | `season_id` FK, `duty_type_id` FK, `soll_hours` |

---

## API-Routen (Übersicht)

### Public
```
POST /api/auth/login
POST /api/auth/refresh
POST /api/auth/logout
POST /api/auth/request-membership
POST /api/auth/register
POST /api/auth/forgot-password
POST /api/auth/reset-password
```

### Authenticated (alle eingeloggt)
```
GET  /api/members
POST /api/members
GET  /api/members/export
GET  /api/members/{id}
PUT  /api/members/{id}
PUT  /api/members/{id}/status
POST /api/members/{id}/team-assignment
GET  /api/profile/me
GET  /api/profile/vehicle
PUT  /api/profile/vehicle
GET  /api/duty-board
POST /api/duty-board/{slotId}/claim
GET  /api/duty-accounts
GET  /api/duty-slots
GET  /api/duty-slots/{id}/assignments
```

### Admin + Trainer
```
POST /api/duty-slots
PUT  /api/duty-slots/{id}
POST /api/duty-assignments/{id}/fulfill
POST /api/duty-assignments/{id}/cash-substitute
GET  /api/admin/membership-requests
POST /api/admin/membership-requests/{id}/approve
POST /api/admin/membership-requests/{id}/reject
POST /api/auth/invite
```

### Admin only
```
GET/PUT  /api/admin/club
GET/POST /api/admin/seasons
PUT      /api/admin/seasons/{id}/activate
PUT      /api/admin/seasons/{id}/duty-targets
GET/POST /api/admin/teams
PUT      /api/admin/teams/{id}
POST     /api/admin/teams/{id}/assign-trainer
GET      /api/admin/users
POST     /api/admin/family-links
GET/POST /api/admin/duty-types
PUT      /api/admin/duty-types/{id}
GET      /api/admin/duty-accounts/export
```

---

## Frontend-Konventionen

### Auth-Context

```tsx
const { user, login, logout, loading } = useAuth()
// user hat: email, role (aus JWT-Payload dekodiert)
```

### API-Calls

```tsx
import { api } from '../lib/api'

api.get('/members')           // automatisch Bearer-Token + Auto-Refresh bei 401
api.post('/members', { … })
```

Alle Pfade relativ zu `/api/` (der Prefix wird in api.ts gesetzt: `baseURL: '/api'`).

### Neue Seite anlegen

1. Datei `web/src/pages/MeineSeite.tsx` erstellen
2. In `App.tsx` importieren und Route unter dem `AppShell`-Outlet anlegen
3. In `AppShell.tsx` ggf. Nav-Eintrag mit `roles`-Array eintragen

### Styling

Tailwind v3 via CDN-Klassen. Keine eigene CSS-Datei außer `index.css` (nur `@tailwind`-Direktiven).  
Marken-Primärfarben: Schwarz `#000000`, Gelb `#FAE806`, Weiß `#FFFFFF`.  
Marken-Sekundärfarben: Blau `#3E4A98`, Grün `#6EB42E`.  
Sidebar-Hintergrund: `bg-[#3E4A98]`. Primär-Buttons: `bg-[#FAE806] hover:bg-[#000000]`.  
Logo-SVG: `../team-stuttgart-org/team-stuttgart-site/Resources/Public/Images/logo.svg`  
Schrift: Hanken Grotesk (Google Fonts, entspricht TYPO3-Site)

---

## Deployment

**Ziel:** IONOS VPS Linux XS · `/usr/local/bin/teamwerk` · systemd-Service `teamwerk`  
**Nginx:** Reverse Proxy Port 443 → 8080, Zertifikat via Certbot  
**Konfiguration:** `/etc/teamwerk/env` (enthält PORT, DB_PATH, JWT_SECRET, SMTP_*)  
**DB:** `/var/lib/teamwerk/teamwerk.db`  
**Scheduler:** Cronjob `* * * * * /usr/local/bin/teamwerk scheduler:run`

Für einen Erstaufbau: `bash deploy/setup-vps.sh` auf dem VPS ausführen (root).

---

## Bekannte Gotchas

**SQLite DATE-Felder:** API gibt Datumsfelder als ISO-Timestamp zurück (`"2026-05-30T00:00:00Z"`). Im Frontend immer `.slice(0, 10)` verwenden für Vergleiche und `date + 'T12:00:00'`-Konstruktionen.

**Aktive Saison:** Spielplan, Dienst-Erstellung und Dienst-Konten setzen eine aktive Saison voraus. Verwalten unter `/admin/saisons`. Ohne aktive Saison schlagen game- und slot-Inserts mit FK-Fehler fehl.

**`make deploy`** führt automatisch `migrate up` aus — der neue Binary hat die Migrations eingebettet.

---

## Offene VPS-Aufgaben (manuell)

Diese Tasks aus Phase 1 erfordern SSH-Zugang zum IONOS-VPS und wurden noch nicht ausgeführt:

- SSH-Zugang prüfen, Ubuntu-Version notieren
- `deploy/setup-vps.sh` ausführen (Nginx, Certbot, systemd, Cron)
- DNS: Subdomain `intern.team-stuttgart.org` → VPS-IP setzen
- `/etc/teamwerk/env` befüllen (JWT_SECRET, SMTP_PASS)
- `make deploy` ausführen (erster Deployment)
