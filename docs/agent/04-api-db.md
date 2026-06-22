# API & Datenbank — Quelle der Wahrheit ist der Code

**Routen:** Die maßgebliche Liste steht in `internal/app/router.go` (`BuildRouter`, nach Auth-Tier gruppiert). Dort nachschlagen statt aus dem Gedächtnis — eine Doku-Kopie würde driften.

**Schema:** Maßgeblich sind die Migrations in `internal/db/migrations/` (`*.up.sql`). Dort die Tabellen/Spalten/CHECK-Constraints lesen.

## Namens- & Sprachkonvention

- **Backend-API-Routen: englisch**, lowercase/kebab-case, generische REST-Struktur `/api/{resource}/{id}/{action}` (z.B. `/api/members/{id}/bank-details`). Bestehende deutsche Ausnahmen (`/api/mitfahrgelegenheiten`) nicht als Vorbild nehmen.
- **Frontend-Routen (`App.tsx`, sichtbare Pfade): deutsch** (z.B. `/admin/saisons`, `/admin/beitragslauf`).
- Alle Frontend-API-Calls relativ zu `/api/` (Prefix in `lib/api.ts`: `baseURL: '/api'`).

## Auth-Tiers (wo gehört eine neue Route hin?)

| Tier | Zugriff |
|---|---|
| Public | Login, Register, Passwort-Reset, Beitrittsantrag, Downloads |
| Authenticated | alle Eingeloggten (Profil, Dienstbörse, Spiele, Chat, …) |
| Trainer + sportliche_leitung | Slots, Anfragen, Training |
| Vorstand (+ Trainer/sL) | Spiele, Kader, Duty-Slots, Saisons (lesen), Venues (CRUD) |
| Vorstand | Mitglieder-CRUD, Verein, Teams, Nutzer, Einladungen, Duty-Types/-Templates |
| Vorstand + Kassierer | Mitglieder lesen, `PUT /members/{id}/bank-details` (Feld-Whitelist), Fee-Run |
| Admin only | Impersonate |

## Schema-Konventionen (nicht-ableitbar)

- **Geldbeträge in Cent** (z.B. `beitrags_saetze.betrag_eur`).
- **`player_memberships` ist eine View** über `kader_members` — kein direktes INSERT; stattdessen `INSERT INTO kader_members (kader_id, member_id) …`.
- **Beitragslauf-Protokoll ist keine Tabelle**, sondern append-only Textdatei pro Saison unter `BEITRAGSLAUF_DIR` (`./storage/beitragslauf-protokolle`) — ins Backup aufnehmen.
- **Status-Felder** sind CHECK-Constraints (z.B. `members.status`: `aktiv|verletzt|pausiert|ausgetreten`) — gültige Werte in der jeweiligen Migration nachsehen.

## Paginierung

`GET /api/members` und `GET /api/users`: `?search=&limit=50&offset=0` → `{ items: [...], total: N }`. Frontend: serverseitige Suche (auf Mobile `sticky top-0 z-10`) + „Mehr laden"-Button, kein clientseitiges `filter()`.
