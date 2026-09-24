# Tasks

## 1. Vorbereitung

- [x] 1.1 Prod-Zählquery vorbereiten (nicht ausführen, dem Nutzer übergeben): Anzahl `duty_assignments` der aktiven Saison, deren Account weder selbst noch über ein Kind im Stammkader/als Trainer eines Slot-Teams steht, wohl aber im erweiterten Kader — Größenordnung der Bilanz-Verschiebung; Query als Kommentar in `design.md` unter „Risks“ ablegen
- [x] 1.2 SQL-Baustein `appdb.UserTeamsSQL(kind)` (stamm inkl. Trainer / extended, selbst + Kinder, aktive Saison, extended mit `status <> 'ausgetreten'`) in `internal/db` anlegen; verifiziert durch Unit-Test in `internal/db` mit Stamm-, Erweitert-, Förderkind- und Ausgetreten-Fall

## 2. Dienstbörse (Backend)

- [x] 2.1 `duties.Board`: Team-Quelle um `extended` erweitern und `'eltern'`-Audience-Match um Kinder im erweiterten Kader ergänzen; verifiziert durch `TestBoard_ErweiterterKaderSiehtTeamDienste`, `TestBoard_ElternErweiterterKaderSehenElternSlot`, `TestBoard_AusgetretenImErweitertenKaderSiehtNichts`, `TestBoard_FoerderkindImErweitertenKader`, `TestBoard_FremdesTeamWeiterhinUnsichtbar`
- [x] 2.2 `aushilfe`-Flag je Gruppe (Stamm schlägt erweitert) in die Board-Response; verifiziert durch `TestBoard_StammSchlaegtErweitert` und Assertion `aushilfe: true` in 2.1-Tests
- [x] 2.3 `aushilfe`-Flag je Eingetragenem in der bestehenden Assignee-Nachlade-Query; verifiziert durch `TestBoard_AssigneeAushilfeKennzeichen`
- [x] 2.4 Doc-Kommentar an `eligibleDutyRecipients` auf „Empfänger ⊆ Sichtbarkeit, erweiterter Kader bewusst ohne Push“ anpassen, Begründung in `audienceAllowlist` (`internal/arch/audience_test.go`) nachziehen; verifiziert durch `TestCreateSlot_ErweiterterKaderBekommtKeinePush` und grünen `go test ./internal/arch/...`

## 3. Dienst-Bilanz (`dutyfairness`)

- [x] 3.1 Snapshot lädt `kader_extended_members` der aktiven Saison als `extTeams`; Nur-erweitert-Mitglieder in `ownByUser`/`childrenByUser`, aber nicht in `Team.Members`/`PlayerCount`; verifiziert durch `TestCompute_AushilfeAendertSollNicht`
- [x] 3.2 Zurechnungsstufen 3/4 (Aushilfe) vor der bisherigen Stufe 3, Aushilfe-Zähler je (Mitglied, Team); verifiziert durch `TestCompute_AushilfeZaehltNichtAufsStammteam`, `TestCompute_StammkindSchlaegtAushilfekind`
- [x] 3.3 `Snapshot.AushilfeFor(userID)` und `Team.Aushilfen()` bereitstellen; verifiziert durch Unit-Tests in `fairness_test.go`
- [x] 3.4 Paritäts-Test Board ↔ Bilanz; verifiziert durch `TestAushilfePraedikat_BoardUndBilanzDeckungsgleich` (in `internal/duties/board_aushilfe_test.go`)

## 4. Dashboard (Backend)

- [x] 4.1 `audienceMatchClauseSQL` und `dutyTeamQuery` auf `appdb.UserTeamsSQL` umstellen, Stamm-Block unverändert; Stamm-`mySlots` schließt Aushilfe-Zusagen aus; verifiziert durch bestehenden `TestDashboard_MeineDienste_ErweiterterKaderZaehltNicht` (grün)
- [x] 4.2 `queryMeineDiensteAushilfe` → `meineDienste.aushilfe` (`mySlots` ≤ 5, `nextGame`, `openSlotsCount`, sonst `null`); verifiziert durch `TestDashboard_MeineDienste_AushilfeBlock`, `TestDashboard_MeineDienste_OhneErweitertenKaderKeinBlock`
- [x] 4.3 `meineDienste.dutyAccountAushilfe` aus `Snapshot.AushilfeFor`; verifiziert durch `TestDashboard_DutyAccountAushilfe`

## 5. Rangliste (Backend)

- [x] 5.1 `TeamsFor` um `extTeams` erweitern; verifiziert durch `TestRangliste_AushilfeTeamImFilter` und bestehenden `TestRangliste_FremdesTeam_403`
- [x] 5.2 Block-Feld `aushilfen` (ungerankt, gleiche Anonymisierung); verifiziert durch `TestRangliste_AushilfenAnonymisiert`

## 6. Frontend

- [ ] 6.1 `ExtendedBadge` nach `web/src/components/AushilfeBadge.tsx` verallgemeinern (Text-Prop, `brand-blue`-Tokens), Dashboard-Termine weiter „Erw. Kader“; verifiziert durch bestehende Dashboard-Tests grün
- [ ] 6.2 `DutySlotList`: Chip „Aushilfe“ an Gruppen mit `aushilfe`, Badge neben Eingetragenen mit `aushilfe`; verifiziert durch `DutySlotList.aushilfe.test.tsx`
- [ ] 6.3 `DashboardPage`: Abschnitt „Aushilfe (erw. Kader)“ unter „Meine Dienste“ (Icon `Handshake`, Link `/dienste`) und Abschnitt „Aushilfe“ unter der Bilanz ohne Soll-Balken; verifiziert durch `DashboardPage.aushilfe.test.tsx`
- [ ] 6.4 `DienstRanglistePage`: Abschnitt „Aushilfen“ je Block (Desktop-Tabelle + Mobile-Cards); verifiziert durch `DienstRanglistePage.aushilfe.test.tsx`
- [ ] 6.5 Live-Updates prüfen: bestehende `useLiveUpdates`-Abos auf Dienst-Events decken Board, Dashboard und Rangliste ab; verifiziert durch manuellen Zwei-Session-Test (Aushilfe belegt → Trainer-Session zeigt Badge ohne Reload)

## 7. Doku & Abschluss

- [ ] 7.1 Benutzerhandbuch (`web/public/benutzerhandbuch.html`), Abschnitt Dienste: Aushilfe im erweiterten Kader; verifiziert durch Sichtprüfung im Browser
- [ ] 7.2 Gotcha-Absatz „Dienst-Bilanz je Kind“ in `docs/agent/06-gotchas.md` um Stufen 3/4 und den Aushilfe-Zähler je (Mitglied, Team) ergänzen; verifiziert durch Review
- [ ] 7.3 `/verify-change` ausführen (Build, `make test`, Lint, `pnpm -C web test`, `openspec validate dienste-erweiterter-kader --strict`); verifiziert durch grünes Gate
