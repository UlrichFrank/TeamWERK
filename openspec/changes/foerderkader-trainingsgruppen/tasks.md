# Tasks

Reihenfolge: Backend (API/DB) vor Frontend. Ein Commit pro Task. Jede neue Route
bekommt Happy-Path- + Fehlerfall-Test (Fixtures in `internal/testutil`).

## 1. Datenbank-Migration (033)

- [x] 1.1 Migration `034_foerderkader_trainingsgruppen.up.sql` + `.down.sql`
  anlegen (034, da 033 bereits `member_series_unavailabilities` belegt).
  `PRAGMA legacy_alter_table=ON` nötig: vier Views (team_/player_/trainer_memberships,
  user_accessible_teams) referenzieren `members`, sonst scheitert DROP/RENAME.
  - `members`-Rebuild: `members_new` mit CHECK
    `status IN ('aktiv','verletzt','pausiert','ausgetreten','passiv','honorar','anwaerter','foerderkind')`,
    vollständige Spaltenliste per `INSERT … SELECT`, `DROP`/`RENAME`, alle Indizes
    neu (`idx_members_member_number` u. a.). Vorbild: Rebuild in Migration 018.
  - `training_group_categories (name PK, sort_order, created_at)` + Seed in
    Wunsch-Ordnung: `('Perspektivkader', 1), ('Förderkader', 2)`.
  - `.down.sql` spiegelbildlich (CHECK ohne `foerderkind`; Tabelle droppen). Hinweis:
    down bei bereits vorhandenen `foerderkind`-Rows scheitert am CHECK — im
    down-Skript dokumentieren/abfangen.
- [x] 1.2 Up/Down-Round-Trip auf Temp-DB verifiziert: Up → v34, `foerderkind`
  erlaubt, Kategorien in Ordnung (Perspektivkader=1, Förderkader=2), Views intakt,
  Bestandsdaten erhalten; Down → CHECK zurückgesetzt, Kategorientabelle entfernt.

## 2. Backend — Trainingsgruppen-Kategorien (Config-Package)

- [ ] 2.1 `GET /api/training-group-categories` (authentifiziert) in
  `internal/config` — liest `training_group_categories ORDER BY sort_order`.
  Read-Caching wie andere Referenzdaten.
- [ ] 2.2 `POST /api/training-group-categories` + `DELETE /.../{name}` unter dem
  Vorstand-Tier in `router.go`; beide rufen
  `h.hub.Broadcast("training-group-categories-changed")` (Broadcast-Gate). Löschen
  einer verwendeten Kategorie ist zulässig (kein FK-Check).
- [ ] 2.3 Tests: Read (Happy-Path, Seed sichtbar); POST/DELETE Happy-Path;
  403 für Nicht-Vorstand; Löschen einer verwendeten Kategorie lässt Kader intakt.

## 3. Backend — Member-Status `foerderkind`

- [ ] 3.1 Mitglieder-Anlage/-Bearbeitung (`internal/members`): `foerderkind`
  akzeptieren; Pflichtfeld-Validierung wie bei `anwaerter` lockern (nur Vorname,
  Nachname, Geburtsdatum Pflicht — kein `join_date`-Zwang). Ungültiger Status → 400.
- [ ] 3.2 `internal/testutil/fixtures.go`: `CreateMember` um Status-Option erweitern
  (falls nötig), damit Tests Förderkinder seeden können.
- [ ] 3.3 Tests: Förderkind anlegen ohne `join_date` (Erfolg); ungültiger Status
  (400); Zuordnung eines Förderkinds zu Kader + erweitertem Kader (Erfolg,
  erscheint in Roster-Ableitung); Mehrfach-Kader-Zuordnung.

## 4. Backend — Beitragslauf-Ausschluss

- [ ] 4.1 `internal/beitragslauf/query.go`: `status NOT IN ('honorar','anwaerter')`
  → `… ,'foerderkind')`. Kommentar (`query.go:73-76`) mit anpassen.
- [ ] 4.2 Test: Mitglied `status='foerderkind'` mit sonst vollständigen Daten wird
  aus der Vorschau ausgeschlossen (`included=false`, nicht in Summen).

## 4b. Kanonische Sortierreihenfolge (Backend + Frontend)

- [ ] 4b.1 `internal/db.AgeClassSortKey(col)` (Sibling von `TeamDisplayShort`) —
  SQL-Ausdruck: Nicht-Trainingsgruppen (Block `0`, alphabetisch) vor
  Trainingsgruppen (Block `1`, nach `training_group_categories.sort_order`).
- [ ] 4b.2 Alle Kader-/Team-Ordnungen auf den Sortkey umstellen (Primärkriterium),
  Sekundär `gender, team_number` beibehalten: `internal/kader/handler.go:236`;
  `internal/games/handler.go` (≈5× `ORDER BY t.age_class …`); `internal/teams/handler.go`
  (wo Kader-Ordnung gemeint, statt `ORDER BY t.name`). Vollständiges Audit der
  Team-/Kader-`ORDER BY`-Stellen.
- [ ] 4b.3 `web/src/lib/teamName.ts`: `compareAgeClass(a, b, categories)` mit
  gleicher Logik; `AdminKaderPage:117,318` (`.sort()`) darauf umstellen.
- [ ] 4b.4 Tests: A–D unverändert alphabetisch; Perspektivkader vor Förderkader
  (nicht alphabetisch); Sekundärsortierung nach `team_number` (2016 vor 2017).

## 5. Frontend — Kader-Anlage

- [ ] 5.1 `AdminKaderPage`: zusätzlich `GET /api/training-group-categories` laden;
  Altersklasse-`<select>` unioniert Spiel-Altersklassen + Trainingsgruppen
  (visuell gruppiert, z. B. optgroup „Wettkampf" / „Trainingsgruppen").
- [ ] 5.2 Jahrgang-`<select>`: bei gewählter Trainingsgruppen-Kategorie freie
  Jahresliste (Saison-Startjahr − 4 … − 14) statt Bracket-Jahre; bei
  Spiel-Altersklasse unverändert `bracketYears`.
- [ ] 5.3 `useLiveUpdates` auf `training-group-categories-changed` → Liste neu laden.
- [ ] 5.4 `team_number` bei Trainingsgruppen-Kategorien nach aufsteigendem
  `dedicated_birth_year` vergeben (statt reiner Anlage-Reihenfolge), damit
  `gF1`/`gF2` deterministisch dem Jahrgang folgen. Kurzname-Formel
  (`teamName.ts` / `team_display_short.go`) **nicht** anfassen — A–D unberührt.
- [ ] 5.5 Verifizieren (kein Code): `display_short` liefert `gP`/`gF1`/`gF2`, der
  Jahrgang erscheint als Badge; A–D-Kurznamen unverändert. Bestehende
  `team-name-display`-Tests müssen grün bleiben.

## 6. Frontend — Mitglieder-Status Förderkind

- [ ] 6.1 Mitglieder-Anlage/-Bearbeitung: `foerderkind` als Status-Option
  (analog `anwaerter`); UI erzwingt kein `join_date` für diesen Status.
- [ ] 6.2 Mitgliederliste: „Förderkind"-Badge + Status-Filter (bestehende
  `member-list-filters`-Muster wiederverwenden).

## 7. (Optional) Frontend — Kategorien-Verwaltung

- [ ] 7.1 Kleine Vorstand-Verwaltung der Trainingsgruppen-Kategorien
  (Anlegen/Löschen) unter Einstellungen. Kann entfallen, wenn der Seed genügt und
  Änderungen selten per Migration erfolgen — dann diesen Task streichen.

## 8. Verifikation

- [ ] 8.1 `/verify-change` bzw. volles Gate: `go vet`, `go test ./...` (inkl.
  Architektur- + Broadcast-Gate), `golangci-lint`, `pnpm -C web build/test/lint`,
  `openspec validate foerderkader-trainingsgruppen --strict`.
- [ ] 8.2 Manueller Rauchtest: Förderkader 2016 anlegen → Gastkind
  (`foerderkind`) anlegen → dem Kader zuordnen → Trainings-Serie/-Session +
  RSVP → Beitragslauf-Vorschau zeigt das Kind als ausgeschlossen.
