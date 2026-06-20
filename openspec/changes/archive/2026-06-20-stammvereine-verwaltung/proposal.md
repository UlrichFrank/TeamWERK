## Why

Der Stammverein eines Mitglieds entscheidet über den Aktiv-Beitrag (96 € `aktiv_mit` statt 226 € `aktiv_ohne`). Heute ist das ein **Freitextfeld** `members.home_club` (Migration 016), das der Beitragslauf per **Fuzzy-Matching** (`MatchHomeClub`, Levenshtein ≤ 3) gegen eine **hardcodierte** Liste von 8 Vereinen (`internal/beitragslauf/compute.go:34`) abgleicht.

Das ist die Ursache des Problems „aktive Spieler mit Stammverein zahlen mal 96 €, mal 226 €": Tippfehler, Abkürzungen oder Zusätze über der Fuzzy-Toleranz werden nicht erkannt → das Mitglied fällt auf `aktiv_ohne` (226 €). Die Zuordnung ist nicht-deterministisch und für den Kassierer nicht nachvollziehbar.

Außerdem lässt sich die Vereinsliste nur per Code-Deploy ändern.

Ziel:
1. Stammvereine in einem **eigenen Settings-Tab** verwalten (CRUD).
2. Stammverein auf der Mitgliederseite aus einer **festen Liste auswählen** (Dropdown statt Freitext).
3. Beitragsberechnung wird **deterministisch**: `aktiv_mit` ⇔ ein Stammverein ist zugeordnet, sonst `aktiv_ohne`. Kein Raten mehr.

„Ohne Stammverein" bleibt ein **gültiger Zustand** (→ `aktiv_ohne`, 226 €); das Dropdown enthält dafür eine explizite Option „Kein Stammverein".

## What Changes

**DB-Migration (neu, 047):**
- Neue Tabelle `stammvereine (id, name UNIQUE, aktiv INTEGER DEFAULT 1, sort_order, created_at)`.
- Seed der 8 bestehenden Vereine aus `Mitgliedsvereine[]`.
- `members` erweitern: `home_club_id INTEGER REFERENCES stammvereine(id)` (nullable).
- **Daten-Migration**: bestehende `members.home_club`-Freitexte einmalig per kanonischem Abgleich (Logik aus `MatchHomeClub`) auf `home_club_id` mappen. Nicht zuordenbare Werte bleiben `NULL` und werden protokolliert.
- `members.home_club` (Freitext) bleibt als **Audit-Spur** erhalten (kein Drop in dieser Migration).

**Backend (neu) — Stammverein-Verwaltung:**
- `GET /api/stammvereine` (authenticated) — Liste aktiver Vereine für das Dropdown; `?include_inactive=1` (vorstand) für die Verwaltung.
- `POST /api/stammvereine` (vorstand) — neuen Verein anlegen (`name`).
- `PUT /api/stammvereine/{id}` (vorstand) — umbenennen / `aktiv` umschalten.
- `DELETE /api/stammvereine/{id}` (vorstand) — **Soft-Delete** (`aktiv=0`), niemals Hard-Delete, solange Mitglieder referenzieren; FK bleibt intakt.
- Alle Mutationen rufen `h.hub.Broadcast("stammvereine")` auf.

**Backend (geändert):**
- `UpdateMemberRequest` akzeptiert `home_club_id *int` (nullable) zusätzlich/statt `home_club`-Freitext; Whitelist in `internal/members/handler.go` entsprechend ergänzen.
- `internal/beitragslauf`: Kategorisierung nutzt `home_club_id IS NOT NULL` statt `MatchHomeClub`. `LoadMembersForLauf` lädt `home_club_id`. `MatchHomeClub`/`Mitgliedsvereine[]` werden nur noch vom einmaligen Migrations-Schritt benötigt und danach als deprecated markiert (kein Aufruf mehr im Lauf).
- `home_club_unklar`-Warnung entfällt (es gibt keine unsichere Zuordnung mehr).

**Frontend (neu):**
- Settings-Tab „Stammvereine" in `AdminSettingsPage.tsx` (Capability `manage_club`), CRUD-Tabelle mit Anlegen/Umbenennen/Deaktivieren, abonniert `useLiveUpdates('stammvereine')`.

**Frontend (geändert):**
- `MemberStammdatenTab.tsx`: Freitext-Input durch `<select>` ersetzen (Optionen: „Kein Stammverein" + aktive Vereine + ggf. der aktuell zugeordnete inaktive Verein).

## Impact

- Betroffene Specs: neue Capability `stammverein-verwaltung`; geänderte Capability `sepa-beitragslauf` (deterministische Kategorisierung), `members` (home_club_id).
- Betroffener Code: `internal/db/migrations/047_*`, `internal/stammvereine/` (neu), `internal/members/handler.go`, `internal/beitragslauf/{query,handler,compute}.go`, `internal/app/router.go`, `web/src/pages/AdminSettingsPage.tsx`, `web/src/components/admin/MemberStammdatenTab.tsx`.
- Reihenfolge: setzt **`fix-passiv-beitragssatz`** nicht zwingend voraus, sollte aber danach deployt werden.

## Offene Entscheidungen

- **Capability**: wiederverwendet `manage_club` (wie der „Verein"-Tab) statt einer neuen `manage_stammvereine` — feiner granular nur bei Bedarf.
- **`home_club`-Freitext**: bleibt vorerst (Audit). Ein späterer Cleanup-Change kann ihn droppen, sobald die Migration verifiziert ist.
