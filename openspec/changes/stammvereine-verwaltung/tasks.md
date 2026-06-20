# Tasks: Stammvereine-Verwaltung

## 1. Datenbank-Migration (Schema, Schritt A — automatisch)

- [x] 1.1 `internal/db/migrations/047_stammvereine.up.sql` — **nur Schema, keine Daten-Mutation** (siehe design.md §2.1):
  - `CREATE TABLE stammvereine (id, name UNIQUE, aktiv DEFAULT 1, sort_order DEFAULT 0, created_at)` gemäß design.md §1.1.
  - Seed der 8 Vereine aus `Mitgliedsvereine[]` mit aufsteigender `sort_order`.
  - `ALTER TABLE members ADD COLUMN home_club_id INTEGER REFERENCES stammvereine(id);` (alle Bestands-Mitglieder erhalten `NULL`).
  - **Kein** `UPDATE members …` in dieser Migration (Backfill ist der reviewbare Schritt 2).
  - Commit: `chore(db): Migration 047 — stammvereine-Tabelle und members.home_club_id`
- [x] 1.2 `.down.sql`: `home_club_id` via `DROP COLUMN` entfernen (wie Migration 035), `DROP TABLE stammvereine`.
- [x] 1.3 Up-Migration auf Temp-DB verifiziert: 8 Seed-Vereine vorhanden; `home_club_id`-Spalte existiert und ist überall `NULL`.

## 2. Daten-Backfill home_club → home_club_id (reviewbar, Schritte B + C)

- [x] 2.1 Preview-SQL als Datei `deploy/stammverein-mapping-preview.sql` eingecheckt (read-only SELECT aus design.md §2.2). **Ausführung gegen Produktiv-DB durch Betreiber** (`sqlite3 /var/lib/teamwerk/teamwerk.db < deploy/stammverein-mapping-preview.sql`).
- [ ] 2.2 **Review durch Vorstand/Kassierer** (manuell, nach Deploy): jede `UNMATCHED`-Zeile entscheiden — später im Mitglied zuweisen oder bewusst `NULL` (= `aktiv_ohne`). Ergebnis dokumentieren.
- [x] 2.3 Apply-SQL als Datei `deploy/stammverein-mapping-apply.sql` eingecheckt (nur exakte Treffer, kein Fuzzy). **Manuelle Ausführung nach Freigabe**, nicht Teil der Auto-Migration.
- [ ] 2.4 Verifizieren (manuell, nach Apply): Anzahl gesetzter `home_club_id` entspricht den `exakt`-Zeilen der Preview; `UNMATCHED` bleibt `NULL` (Kontroll-SELECT ist Teil der Apply-Datei).

## 3. Backend — Stammverein-Package

- [x] 3.1 Neues Package `internal/stammvereine/` mit `Handler{db, hub}`. In `internal/arch/arch_test.go` als domain klassifiziert.
- [x] 3.2 `GET /api/stammvereine` — aktive Vereine; `?include_inactive=1` nur vorstand/admin.
- [x] 3.3 `POST /api/stammvereine` (vorstand) — anlegen, 409 bei doppeltem Namen, `Broadcast("stammvereine")`.
- [x] 3.4 `PUT /api/stammvereine/{id}` (vorstand) — umbenennen / `aktiv` togglen, `Broadcast`.
- [x] 3.5 `DELETE /api/stammvereine/{id}` (vorstand) — Soft-Delete `aktiv=0`, `Broadcast`.
- [x] 3.6 Routen in `internal/app/router.go` eingehängt (GET → Authenticated, Mutationen → Vorstand); `NewHandler(db, hub)` in `main.go` verdrahtet. Zusätzlich in `internal/permissions/matrix_test.go` (Permission-Matrix) gepflegt.
- [x] 3.7 Tests `internal/stammvereine/handler_test.go`: POST 201/409/403, GET 200/401, PUT 403, DELETE soft + 403, include_inactive nur Vorstand.

## 4. Backend — Member-Integration

- [x] 4.1 `Update`-Request um `home_club_id *int` erweitert; persistiert; FK-Existenz geprüft → 400 bei ungültiger ID. Honorar-Status setzt `home_club_id=NULL`.
- [x] 4.2 `GET /api/members/{id}` gibt `home_club_id` und aufgelösten `home_club_name` zurück (LEFT JOIN stammvereine).
- [x] 4.3 Tests `internal/members/stammverein_test.go`: Zuweisen, Entfernen (`null`), ungültige ID → 400.

## 5. Backend — Beitragslauf deterministisch

- [x] 5.1 `LoadMembersForLauf` (`query.go`): `home_club_id IS NOT NULL` als `HasHomeClub` selektiert.
- [x] 5.2 `computeItem` (`handler.go`): Kategorie aus `m.HasHomeClub` statt `MatchHomeClub`; `home_club_unklar`-Warnung entfernt.
- [x] 5.3 `MatchHomeClub`/`Mitgliedsvereine[]` in `compute.go` als `Deprecated` kommentiert (kein Aufruf mehr im Lauf; bleibt als Migrations-Hilfsmittel).
- [x] 5.4 Tests `internal/beitragslauf/`: aktiv mit `home_club_id` → `aktiv_mit` (96 €); aktiv ohne → `aktiv_ohne` (226 €); keine `home_club_unklar`-Warnung mehr.

## 6. Frontend — Settings-Tab

- [x] 6.1 `AdminSettingsPage.tsx`: Tab „Stammvereine" in `TABS`. Cap = `manage_seasons` (vorstand/admin) statt `manage_club` — deckt sich mit den vorstand-only-Mutationen im Backend; Kassierer sieht den Tab bewusst nicht.
- [x] 6.2 CRUD-Tabelle (Anlegen/Umbenennen/Deaktivieren/Aktivieren) mit brand-Klassen (INPUT/BTN_SM/BTN_DANGER_SM, Card/Tabelle aus CLAUDE.md).
- [x] 6.3 `useLiveUpdates(e => e === 'stammvereine' && load())`.

## 7. Frontend — Mitglied-Auswahl

- [x] 7.1 `MemberStammdatenTab.tsx`: Freitext-Input durch `<select>` ersetzt. Optionen: „Kein Stammverein" + aktive Vereine; aktuell zugeordneter inaktiver Verein zusätzlich „(deaktiviert)" angezeigt.
- [x] 7.2 Member-Form-State (`MemberDetailPage.tsx`) auf `home_club_id` umgestellt; im PUT-Payload enthalten.

## 8. Verifikation & Abschluss

- [x] 8.1 Verifikation: `go vet` sauber, `openspec validate --strict` ok (beide Changes), Frontend-Lint sauber. Hinweis: `golangci-lint` lokal nicht ausführbar (installiertes v2-Binary vs. `.golangci.yml` im v1-Format — Umgebungsproblem, nicht von dieser Änderung); läuft im pre-push-Hook mit gepinnter Version.
- [x] 8.2 `go test ./...` grün (inkl. `stammvereine`, `members`, `beitragslauf`, Architektur- und Permission-Matrix-Test).
- [x] 8.3 `pnpm -C web build` + `test` (341) + `lint` grün.
- [ ] 8.4 Proposal archivieren (nach Merge/Deploy).
