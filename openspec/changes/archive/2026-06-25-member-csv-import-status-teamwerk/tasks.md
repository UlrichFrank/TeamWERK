# Tasks — member-csv-import-status-teamwerk

> Ein Commit pro Task. Scope abhängig vom Task (`db`, `members`, `web`).

## 1. Migration

- [x] 1.1 `internal/db/migrations/007_beitragsfrei_grund.up.sql`: `ALTER TABLE members ADD COLUMN beitragsfrei_grund TEXT;`
- [x] 1.2 `internal/db/migrations/007_beitragsfrei_grund.down.sql`: SQLite-konformer Drop (Spalten-Drop oder Tabellen-Rebuild gemäß Projektkonvention).
- [x] 1.3 `make migrate-up` lokal grün; `make migrate-down` und erneut up testen.

  _Commit:_ `feat(db): Spalte members.beitragsfrei_grund (Migration 007)`

## 2. Backend: GET/PUT-Mapping für neues Feld

- [x] 2.1 `internal/members/handler.go`: `GetMember`-Select um `COALESCE(beitragsfrei_grund,'')` ergänzen; Response-Typ `Member` um `BeitragsfreiGrund string \`json:"beitragsfrei_grund"\``.
- [x] 2.2 `Update` (PUT `/api/members/{id}`): Feld lesen und schreiben; **wenn `beitragsfrei=false`, `beitragsfrei_grund := NULL` erzwingen** (siehe `design.md` D4).
- [x] 2.3 Unit-Test `TestGetMember_BeitragsfreiGrundField` + `TestUpdateMember_BeitragsfreiFalseClearsGrund`.

  _Commit:_ `feat(members): beitragsfrei_grund in GET/PUT mit Clear-Invariante`

## 3. Backend: Bankdaten-Whitelist erweitern

- [x] 3.1 `UpdateBankdaten` (`handler.go:794`): Request-Struct um `Beitragsfrei bool` + `BeitragsfreiGrund string` ergänzen; UPDATE-Statement um beide Spalten erweitern; Clear-Invariante D4 anwenden.
- [x] 3.2 Tests `TestBankdaten_KassiererPflegtBeitragsfreiGrund`, `TestBankdaten_BeitragsfreiFalseClearsGrund`, `TestBankdaten_SpielerForbidden` (403-Pfad weiter grün).

  _Commit:_ `feat(members): Kassierer pflegt Beitragsfrei + Grund via bank-details`

## 4. Backend: CSV-Import-Mapping umstellen

- [x] 4.1 `Import` (`handler.go:1512`): Spalten-Mapping umbauen:
      - „Status" nicht mehr lesen (Insert- und Update-Pfad).
      - „Status TeamWERK" → `normalizeStatus` → `members.status`.
      - „beitragsfrei" → direkter Bool (`"ja"` → 1, sonst 0); im Enrich-Modus nur `0 → 1` zulassen (D6).
      - „Grund für Beitragsfreiheit" → `members.beitragsfrei_grund` (Enrich überschreibt nicht).
      - Block „beitragsfrei aus Status ableiten" (`handler.go:1953–1960`) entfernen.
- [x] 4.2 `IMPORT_FIELDS`-äquivalente Backend-Validation: in `fieldAllowed`-Pfaden alle drei Spalten unabhängig prüfen (`status`, `beitragsfrei`, `beitragsfrei_grund`).
- [x] 4.3 Tests:
      - `TestImport_StatusTeamWERK_AppendNew`
      - `TestImport_BeitragsfreiSpalte_DirectMap`
      - `TestImport_BeitragsfreiGrund_Append`
      - `TestImport_BeitragsfreiGrund_EnrichLeaves`
      - `TestImport_AlteStatusSpalteWirdIgnoriert`
      - `TestImport_GekuendigtBleibtAlias`

  _Commit:_ `feat(members): CSV-Import nutzt Status TeamWERK + Beitragsfrei-Spalten`

## 5. Frontend: Bankdaten-Tab editierbar

- [x] 5.1 `web/src/pages/MemberDetailPage.tsx`: Typ `Member` + `MemberForm` um `beitragsfrei_grund?: string`. Initialwert aus GET-Response übernehmen.
- [x] 5.2 `web/src/components/admin/MemberKontaktTab.tsx`: unter Checkbox „Beitragsfrei" konditionales Textinput „Grund". Sichtbarkeit `form.beitragsfrei === true`. Toggle aus → `onFormChange({ beitragsfrei: false, beitragsfrei_grund: '' })`. `brand-*`-Tokens, `lucide-react` nicht nötig.
- [x] 5.3 Submit-Logik: bei `PUT /members/{id}` und `PUT /members/{id}/bank-details` Feld immer mitsenden (leerer String → server clear).
- [x] 5.4 `pnpm -C web test` für `MemberKontaktTab.permissions.test.tsx` grün halten; ggf. Erwartungen anpassen.

  _Commit:_ `feat(web): Grund für Beitragsfreiheit im Bankdaten-Tab editierbar`

## 6. Frontend: Import-Dialog

- [x] 6.1 `web/src/pages/MembersPage.tsx`: `IMPORT_FIELDS` aufspalten:
      - `{ col: 'status', label: 'Status' }`
      - `{ col: 'beitragsfrei', label: 'Beitragsfrei' }`
      - `{ col: 'beitragsfrei_grund', label: 'Grund für Beitragsfreiheit' }`
- [x] 6.2 Default-Auswahl: alle drei vorausgewählt (Konsistenz mit bisherigem Verhalten).
- [x] 6.3 `pnpm -C web build` + `lint` grün.

  _Commit:_ `feat(web): Import-Dialog mit getrennten Whitelist-Checkboxen für Beitragsfrei`

## 7. Abschluss

- [x] 7.1 `openspec validate member-csv-import-status-teamwerk --strict` grün.
- [x] 7.2 `/verify-change` ausführen (Build/Test/Lint + Invarianten).
- [x] 7.3 `CHANGELOG.md` ergänzen (`feat(members): …`, `feat(web): …`).
- [ ] 7.4 Change archivieren: `openspec archive member-csv-import-status-teamwerk`.

  _Commit:_ `chore(openspec): Change member-csv-import-status-teamwerk archivieren`
