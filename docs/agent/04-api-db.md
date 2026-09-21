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
| Authenticated | alle Eingeloggten (Profil, Dienstbörse, Spiele, Chat, BWHV-Staffeln: Tabelle/Spielplan/Ranglisten/Spielbericht/Saisonstatistik, …) |
| Trainer + sportliche_leitung | Slots, Anfragen, Training |
| Vorstand (+ Trainer/sL) | Spiele, Kader, Duty-Slots, Saisons (lesen), Venues (CRUD), `/api/practice-groups` (Übungsgruppen) |
| Vorstand | Mitglieder-CRUD, Verein, Teams, Nutzer, Einladungen, Duty-Types/-Templates, H4A-Spielimport (`POST /api/games/import/h4a/preview` + `/apply`), Massen-Dienstregeneration (`POST /api/duty-slots/bulk-regen/preview` + `/apply`), BWHV-Staffel-Katalog (`GET /api/bwhv/staffel-katalog`) und manueller Abruf (`POST /api/staffeln/{id}/poll`) |
| Vorstand + Kassierer | Mitglieder lesen, `PUT /members/{id}/bank-details` (Feld-Whitelist), Fee-Run |
| Admin only | Impersonate |

## Schema-Konventionen (nicht-ableitbar)

- **Geldbeträge in Cent** (z.B. `beitrags_saetze.betrag_eur`).
- **`player_memberships` ist eine View** über `kader_members` — kein direktes INSERT; stattdessen `INSERT INTO kader_members (kader_id, member_id) …`.
- **Beitragslauf-Protokoll ist keine Tabelle**, sondern append-only Textdatei pro Saison unter `BEITRAGSLAUF_DIR` (`./storage/beitragslauf-protokolle`) — ins Backup aufnehmen.
- **Status-Felder** sind CHECK-Constraints (z.B. `members.status`: `aktiv|verletzt|pausiert|ausgetreten`) — gültige Werte in der jeweiligen Migration nachsehen.
- **`venues.hall_number`** (BWHV-Hallennummer) und **`games.external_id`** (BWHV-Spielnummer) sind die Fremdschlüssel des H4A-Imports (Migration `042`). Beide nullable: `hall_number` mit **Partial-Unique-Index** (`WHERE hall_number IS NOT NULL` — nicht zuordenbare Nicht-BWHV-Orte bleiben NULL), `external_id` **ohne** UNIQUE (manuell angelegte Spiele koexistieren; die Eindeutigkeit prüft der Import fachlich).
- **`members.join_date`/`exit_date`** steuern die Beitrags-Halbierung (Migration `014`): `join_date` ist App-Pflichtfeld (DB nullbar), `exit_date` Pflicht bei `status='ausgetreten'`. **`seasons.is_inaugural`** (INTEGER 0/1) markiert das erste Abrechnungsjahr (alle zahlen halb). Details siehe Gotcha „SEPA-Beitragslauf".
- **`bwhv_games` ist kein `games`.** Der Staffel-Spielplan des Verbands (~810 Begegnungen je Saison, überwiegend fremde Vereine) lebt ausschließlich in `bwhv_games`; der Abruf legt **nie** eine `games`-Zeile an. Verknüpft wird nur über `bwhv_games.game_id`, wo die BWHV-Spielnummer einer vorhandenen `games.external_id` entspricht. `games` trägt deshalb auch weiterhin **keine Ergebnisspalten** — der Spielstand steht in `bwhv_games`, für eigene und fremde Begegnungen gleichermaßen.
- **`kader.staffel`** trägt den Staffelcode (`gClassSname`, z.B. `mB-RL-BW`), nicht die Handball4All-Klassen-ID: die wechselt mit der Saison, der Code nicht. Nur an Kadern mit `kind='team'` (Übungsgruppen → HTTP 409).
- **`user_events`** (Migration `050`, Event-Log) trägt **keinen** Fremdschlüssel auf ein Domänen-Objekt — kein `ref_type`/`ref_id`, nur ein Sprungziel `url`, das ins Leere zeigen darf. Die Zeile ist zum Sendezeitpunkt eingefroren, weil das referenzierte Objekt (gelöschter Termin, entfernter Dienst-Slot) danach oft nicht mehr existiert. `category` hat einen `CHECK` über acht Werte **ohne** `chat` — Chat läuft über einen eigenen Kanal (`push.SendToUserWithBadge`) und schreibt bewusst nicht hierher; eine neunte Kategorie braucht eine Migration, keinen Code-Pfad. Details siehe Gotcha „Event-Log".

## Paginierung

`GET /api/members` und `GET /api/users`: `?search=&limit=50&offset=0` → `{ items: [...], total: N }`. Frontend: serverseitige Suche (auf Mobile `sticky top-0 z-10`) + „Mehr laden"-Button, kein clientseitiges `filter()`.
