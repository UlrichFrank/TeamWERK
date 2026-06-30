# Tasks — Halber Beitrag bei Ein-/Austritt und im ersten Abrechnungsjahr

## 1. Schema & Migration

- [x] 1.1 Migration `014_half_fee.up.sql`: `ALTER TABLE members ADD COLUMN exit_date DATE;`
- [x] 1.2 Migration: `ALTER TABLE seasons ADD COLUMN is_inaugural INTEGER NOT NULL DEFAULT 0;`
- [x] 1.3 Migration: Backfill `members.join_date` für NULL-Zeilen auf einen Tag **vor** dem
  frühesten `seasons.start_date` (Fallback fixes frühes Datum, falls keine Saison existiert).
- [x] 1.4 `014_half_fee.down.sql`: Spalten via Tabellen-Rebuild entfernen (SQLite) bzw.
  dokumentierter No-op, konsistent mit den übrigen down-Migrationen des Projekts.

## 2. Backend — Beitragslauf-Kern

- [x] 2.1 `query.go`: `MemberRow` um `JoinDate`, `ExitDate` erweitern; `LoadMembersForLauf`
  lädt `join_date`/`exit_date` und ändert WHERE auf `status NOT IN ('honorar','anwaerter')`
  (ausgetretene bleiben drin).
- [x] 2.2 `handler.go` `loadSeason`: zusätzlich `end_date` und `is_inaugural` liefern; Signatur
  auf eine `seasonInfo`-Struktur umstellen (start, end, inaugural, label).
- [x] 2.3 `compute.go`: Funktion `halfFee(m, season) (bool, reason)` (Priorität erstjahr →
  eintritt → austritt) + `inWindow(date, start, end)`-Helfer; exakte Halbierung `betrag/2`.
- [x] 2.4 `compute.go`/`computeItem`: Inklusions-Sonderfall für unterjährige Austritte
  (`ausgetreten` + exit_date im Fenster → Kategorie aus `home_club_id`, einbeziehen);
  früher/ohne exit_date → `status_inaktiv`.
- [x] 2.5 `handler.go` `PreviewItem`: Felder `Half bool json:"half"`,
  `HalfReason string json:"half_reason,omitempty"` ergänzen; in `computeItem` setzen.
- [x] 2.6 `compute.go`: veralteten `MatchHomeClub`/`Mitgliedsvereine`/`levenshtein`-Block
  unangetastet lassen (nicht im Pfad); Paket-Doku-Kommentar (kein Pro-rata) aktualisieren.

## 3. Backend — Members (exit_date, join_date-Pflicht)

- [x] 3.1 `Member`-Struct: `ExitDate *string json:"exit_date,omitempty"`.
- [x] 3.2 GetMember-SELECT + Scan um `exit_date` erweitern.
- [x] 3.3 Update-Handler: `ExitDate`/`JoinDate` aus Body lesen; UPDATE-SQL um `exit_date`,
  `join_date` ergänzen.
- [x] 3.4 Validierung Update **und** Create: fehlt `join_date` → HTTP 400; Status
  `ausgetreten` ohne `exit_date` → HTTP 400.
- [x] 3.5 Create-Handler-Struct + INSERT um `join_date` (Pflicht) erweitern.

## 4. Backend — Seasons (is_inaugural)

- [x] 4.1 `config`: Season-Struct + ListSeasons-SELECT um `is_inaugural`.
- [x] 4.2 CreateSeason/UpdateSeason: `IsInaugural` lesen + in INSERT/UPDATE schreiben.

## 5. Frontend

- [x] 5.1 `MemberDetailPage.tsx`: `exit_date` in `Member`-Interface, Form-State,
  applyMemberToForm, handleSave-Body.
- [x] 5.2 `MemberStammdatenTab.tsx`: Austrittsdatum-Date-Input, sichtbar/Pflicht bei
  `status === 'ausgetreten'`; Eintrittsdatum als Pflichtfeld kennzeichnen.
- [x] 5.3 `AdminSettingsPage.tsx`: „Erstes Abrechnungsjahr"-Checkbox in Saison-Create/-Edit;
  `is_inaugural` in Interface, State, post/put-Payload.
- [x] 5.4 `BeitragslaufPage.tsx`: `PreviewItem` um `half`/`half_reason`; Hinweis-Badge
  „halber Beitrag (Eintritt/Austritt/erstes Jahr)" in Desktop-Tabelle + Mobile-Card.

## 6. Docs

- [x] 6.1 `docs/agent/06-gotchas.md`: „kein Pro-rata" → Halbierungsregel beschreiben.
- [x] 6.2 `docs/agent/04-api-db.md`: Schema-Notizen (`exit_date`, `is_inaugural`,
  join_date-Pflicht).
- [x] 6.3 `CHANGELOG`/Release-Note falls vorhanden.

## 7. Verifikation

- [x] 7.1 `make test` grün (inkl. neuer Tests, Architektur-Test).
- [x] 7.2 `make lint` + `pnpm -C web build/test/lint` grün.
- [x] 7.3 `openspec validate pro-rata-half-fee --strict`.

---

## Test-Anforderungen

Garantierte Invariante je Test (fachlich, keine Coverage-Dummies):

### Compute (reine Funktion, `internal/beitragslauf/compute_test.go`)

| Test | Invariante |
|---|---|
| `TestHalfFee_Eintritt` | join_date im Fenster → `half=true, reason="eintritt"`, Betrag = voll/2 |
| `TestHalfFee_Austritt` | ausgetreten + exit_date im Fenster → `half=true, reason="austritt"` |
| `TestHalfFee_Erstjahr` | is_inaugural → `half=true, reason="erstjahr"` unabhängig von Daten |
| `TestHalfFee_Ganzjaehrig` | join vor Start, kein Austritt, nicht inaugural → `half=false`, voller Betrag |
| `TestHalfFee_KeinStacking` | join **und** exit im Fenster → genau einmal halbiert (= voll/2, nicht /4) |
| `TestHalfFee_ExakteHalbierung` | ungerader Cent-Betrag → ganzzahlige Halbierung (Abschnitt) |
| `TestInWindow_Grenzen` | start_date und end_date inklusive |

### Handler (`internal/beitragslauf/handler_test.go`, via testutil)

| Route → Test | Erwartung |
|---|---|
| `GET /preview` → `TestPreview_NeumitgliedZahltHalbenBeitrag` (umgekehrter Alttest) | 200, betrag = halb, `half=true` |
| `GET /preview` → `TestPreview_UnterjaehrigerAustrittEinbezogen` | 200, ausgetreten-im-Fenster `included=true`, halb |
| `GET /preview` → `TestPreview_FruehererAustrittAusgeschlossen` | 200, exit_date vor Saison → `included=false` (`status_inaktiv`) |
| `GET /preview` → `TestPreview_ErstjahrAlleHalb` | 200, is_inaugural → alle Eingeschlossenen halb |
| `GET /preview` → `TestPreview_GanzjaehrigVoll` | 200, Bestandsmitglied voll |

### Members (`internal/members/handler_test.go`)

| Route → Test | Erwartung |
|---|---|
| `POST /api/members` → `TestCreateMember_OhneEintrittsdatum400` | 400 |
| `PUT /api/members/{id}` → `TestUpdateMember_AustrittOhneAustrittsdatum400` | 400 |
| `PUT /api/members/{id}` → `TestUpdateMember_AustrittMitDatumOK` | 200, exit_date persistiert |
| `GET /api/members/{id}` → liefert `exit_date` im Body | 200 |

### Seasons (`internal/config/*_test.go`)

| Route → Test | Erwartung |
|---|---|
| `POST /api/seasons` (is_inaugural=true) → `TestCreateSeason_Inaugural` | 200/201, Flag persistiert |
| `PUT /api/seasons/{id}` → `TestUpdateSeason_ToggleInaugural` | 200, Flag aktualisiert |
