## Voraussetzung

`mannschaft-aufgaben-strafen` muss gemergt und archiviert sein (Baseline-Spec `openspec/specs/mannschaftsstrafen/` existiert), sonst schlägt `openspec validate` für die MODIFIED-Requirements dieses Changes fehl.

## 1. Datenmodell & Migration

- [x] 1.1 Migration `0NN_mannschaftskasse_und_strafen_einheiten.up.sql` + `.down.sql` (nächste freie Nummer nach `031`)
- [x] 1.2 `penalty_settings(kader_id PK REFERENCES kader ON DELETE CASCADE, unit CHECK IN ('euro','striche') DEFAULT 'euro')`
- [x] 1.3 Backfill in `up.sql`: `INSERT INTO penalty_settings(kader_id, unit) SELECT id, 'euro' FROM kader` (idempotent via `INSERT OR IGNORE`)
- [x] 1.4 `team_cashbook_entries(id PK, kader_id FK, amount_cent INTEGER NOT NULL [signed], note TEXT NOT NULL, entered_by_member_id FK, entered_at DATETIME DEFAULT CURRENT_TIMESTAMP)`
- [x] 1.5 `kader_kassenwarte(kader_id, member_id) PRIMARY KEY (kader_id, member_id)` mit CASCADE-FKs
- [x] 1.6 `.down.sql` FK-sicher (Drop in umgekehrter Reihenfolge)
- [x] 1.7 Migration up/down verifiziert (Teil von `go test ./internal/db/...`)

## 2. Backend — Gates (`internal/teams/access.go`)

- [x] 2.1 `isKassenwartOfTeam(ctx, memberID, teamID) (bool, error)` (kader_kassenwarte-Lookup, admin-Bypass)
- [x] 2.2 `canManageCashbook = isTrainerOfTeam OR isKassenwartOfTeam`
- [x] 2.3 `canReadCashbook` = benannter Alias / Wiederverwendung von `canReadPenalties` (Spieler ∨ Trainer ∨ Erw. Kader; Eltern/Extern → false)
- [x] 2.4 Gates durch die Handler-Tests abgedeckt (403-Fälle je Route)

## 3. Backend — Strafen-Einheiten (`internal/teams/penalty_settings.go`)

- [x] 3.1 `GET /api/teams/{id}/penalty-settings` — Read für Team-Interne, liefert `{ unit: "euro" | "striche" }`
- [x] 3.2 `GET /api/teams/{id}/penalty-settings/preview?to=<unit>` — Trainer-only, liefert `{ from, to, affected: n, roundedUp: n, entries: [{ id, oldAmount, newAmount, unitLabel }], catalog: [...] }` ohne DB-Mutation
- [x] 3.3 `PUT /api/teams/{id}/penalty-settings` — Trainer-only, TX: Katalog + alle `team_penalties` umrechnen (`ceil(cent/100)` bei →striche, `n*100` bei →euro), `penalty_settings` setzen, Broadcast `penalty-settings` + `penalties`
- [x] 3.4 Betrag-Validierung in `POST /api/teams/{id}/penalties` und `POST /api/teams/{id}/penalty-types`: bei aktueller `unit='striche'` Ganzzahl-Check (Cent-Feld muss durch 100 teilbar sein → also Ganzzahl-Striche), sonst HTTP 400
- [x] 3.5 Routen in `internal/app/router.go` (`BuildRouter`) eintragen
- [x] 3.6 Tests: Read 200, Trainer 200, NonTrainer 403, EurToStriche_RoundsUpAndConverts (multi-row, TX), StricheToEur_ExactConversion, Preview_NoMutation, Penalty_NonInteger_400

## 4. Backend — Mannschaftskasse (`internal/teams/cashbook.go`)

- [x] 4.1 `GET /api/teams/{id}/cashbook` mit `canReadCashbook`; Response `{ entries: [...], balanceCent }`
- [x] 4.2 `POST /api/teams/{id}/cashbook` (Trainer ∨ Kassenwart): signed `amountCent`, `note`, Broadcast `cashbook`
- [x] 4.3 `DELETE /api/teams/{id}/cashbook/{eid}` (Trainer ∨ Kassenwart): hard delete, Broadcast `cashbook`
- [x] 4.4 Kassenwart-Ernennung: `GET`/`POST`/`DELETE /api/teams/{id}/treasurers[/{mid}]`, Trainer-only für POST/DELETE, Broadcast `treasurers`
- [x] 4.5 Routen in `internal/app/router.go` eintragen
- [x] 4.6 Read-Gate-Tests: Player/Trainer/Extended 200, Parent/Outsider 403, Roster-Exclusion-Invariante
- [x] 4.7 Write-Gate-Tests: Trainer 200, Kassenwart 200, Spieler 403, ForeignTeamKassenwart 403
- [x] 4.8 Ernennungs-Tests: Trainer 200, NonTrainer 403, `TestClubFunctions_NoKassenwartValue`

## 5. Test-Fixtures (`internal/testutil/`)

- [x] 5.1 `AppointKassenwart(t, db, kaderID, memberID)`
- [x] 5.2 `CreateCashbookEntry(t, db, kaderID, memberID, amountCent, note)`
- [x] 5.3 `SetPenaltyUnit(t, db, kaderID, unit)`

## 6. Frontend — MeinTeamPage (`web/src/pages/MeinTeamPage.tsx`)

- [x] 6.1 „Mannschaftskasse"-Kopfzeile aus Strafen-Tab entfernen (Zeilen 484–487)
- [x] 6.2 Neuer Tab „Kasse" mit `canReadCashbook`-Gate (403 versteckt Sektion); Ledger + Saldo, Buchung anlegen (Trainer/Kassenwart), Eintrag löschen
- [x] 6.3 Einheiten-Umschaltung in Verwalten-Tab: Radio/Toggle `Euro | Striche`, Preview-Modal mit Delta-Liste, PUT nach Bestätigung
- [x] 6.4 Strafen-Anzeige: Betrag je nach `unit` als „X,XX €" oder „N Striche" (Utility `fmtPenaltyAmount(cent, unit)`)
- [x] 6.5 Levy-Form: `<input>` bei `unit='striche'` erhält `type="number" step="1" min="1"`, Placeholder „Anzahl Striche"; bei `unit='euro'` bleibt Decimal wie heute
- [x] 6.6 Verwalten-Tab: Kassenwart-Sektion direkt unter Strafenwart-Sektion (Kopie des Bausteins, andere Endpoints/Events)
- [x] 6.7 Bold-Me: eigene Zeile in Roster-Tabs (Team/Trainer/Eltern), Strafen-Übersicht, Kassenbuch als `font-semibold` (Utility `isMe(memberId)`)
- [x] 6.8 `useLiveUpdates` um `cashbook`, `treasurers`, `penalty-settings` erweitern (jeweils passenden Reload triggern)

## 7. Verifikation

- [x] 7.1 `go test ./...` (inkl. Architektur-, Broadcast- + Permission-Gate) grün
- [x] 7.2 `golangci-lint` grün
- [x] 7.3 `pnpm -C web build` + `pnpm -C web lint` grün (0 Errors, keine Raw-Farben/Emojis)
- [x] 7.4 `openspec validate mannschaftskasse-und-strafen-einheiten --strict` grün
- [ ] 7.5 Merge nach `main` + Deploy
