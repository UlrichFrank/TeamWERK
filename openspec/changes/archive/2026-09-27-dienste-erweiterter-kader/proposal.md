## Why

Der erweiterte Kader ist bei Terminen, Chat, Anwesenheit, Kalender, Terminmeldungen und
Videos inzwischen gleichgestellt. **Bei den Diensten nicht:** die Dienstbörse
(`duties.Board`) löst ihre Team-Quelle ausschließlich über `player_memberships` (=
Stammkader) auf. Ein Spieler, der regelmäßig in der höheren Mannschaft aushilft, und
seine Eltern sehen deren Dienste nicht und können sie deshalb nicht belegen — obwohl sie
bei genau diesen Spielen ohnehin in der Halle sind und Hilfe anbieten wollen.

Die Gleichstellung darf aber nicht in eine **Pflicht** kippen: der erweiterte Kader schuldet
dem fremden Team keine Dienststunden (so begründet in `audienceAllowlist` für
`duties.eligibleDutyRecipients`). Aushilfe muss deshalb überall als solche erkennbar sein
und in der Dienst-Bilanz getrennt stehen.

Nebenbei deckt der Change einen stillen Fehler auf: belegt heute jemand einen Dienst eines
Teams, in dem er nicht im Stammkader steht (möglich, weil `Claim` kein Team prüft), rechnet
`dutyfairness` die Zuweisung über Stufe 3 („eigene Mitglieder unabhängig vom Team“) seinem
**Stammteam** gut. Mit sichtbaren Aushilfe-Diensten würde aus dem Randfall der Normalfall.

## What Changes

- **Dienstbörse:** Team-Quelle und Eltern-Zielgruppe (`'eltern'`) von `GET /api/duty-board`
  umfassen den **erweiterten Kader** der aktiven Saison und dessen **Eltern** (via
  `family_links`). Statusfilter `members.status <> 'ausgetreten'` (Förderkinder).
- **Kennzeichnung am Slot:** jede Board-Gruppe trägt `aushilfe: true`, wenn der Betrachter
  nur über den erweiterten Kader (selbst oder Kind) mit ihren Teams verbunden ist. Das
  Frontend zeigt einen Chip „Aushilfe“.
- **Kennzeichnung an der Belegung:** jeder Eingetragene trägt `aushilfe: true`, wenn seine
  Zuweisung nach der Zurechnungsregel eine Aushilfe ist — für alle Betrachter sichtbar.
- **Dienst-Bilanz (`dutyfairness`):** zwei neue Zurechnungsstufen vor der bisherigen
  Stufe 3. Aushilfe-Zuweisungen zählen **je Mitglied × Aushilfe-Team** getrennt, gehen
  **nicht** ins Geleistet/Vorhersage des Mitglieds (also weder ins Stammteam noch in eine
  Rangliste) und haben **kein Soll**. Das Soll des fremden Teams ändert sich nicht.
- **Dashboard „Meine Dienste“:** der bestehende Block bleibt beim Stammteam
  (unverändert). Darunter ein **getrennter Block „Aushilfe“**: eigene kommende
  Aushilfe-Zusagen und das nächste Spiel eines erweiterten Teams mit offenen, passenden
  Slots. Die Dienst-Bilanz-Kachel bekommt einen eigenen Abschnitt „Aushilfe“ (nur
  Geleistet/Geplant).
- **Rangliste:** ein Team, mit dem der Nutzer über den erweiterten Kader verbunden ist,
  wird im Team-Filter angeboten. Der Block des Teams bekommt einen **separaten,
  nicht gerankten Abschnitt „Aushilfen“** — gleiche Anonymisierung wie die Rangliste.
- **Keine Push:** `eligibleDutyRecipients` bleibt beim Stammkader. Bewusste Abweichung von
  „Empfänger = Sichtbarkeit“ in `push-duties`: Empfänger ⊆ Sichtbare.

## Capabilities

### New Capabilities

- `dienste-aushilfe-erweiterter-kader`: Aushilfe-Kennzeichen (Board-Gruppe, Eingetragene),
  Aushilfe-Block im Dashboard „Meine Dienste“, Aushilfe-Abschnitt in Bilanz-Kachel und
  Rangliste.

### Modified Capabilities

- `duties`: Sichtbarkeit und Eltern-Zielgruppe der Dienstbörse umfassen den erweiterten
  Kader und dessen Eltern.
- `dienste-familien-rangliste`: Zurechnungsstufen um Aushilfe erweitert; Rangliste bietet
  Aushilfe-Teams an und weist Aushilfen getrennt aus.
- `push-duties`: Empfängermenge ausdrücklich ohne erweiterten Kader (Teilmenge der
  Sichtbarkeit statt Gleichheit).
- `api-routes`: `GET /api/teams?scope=duties` (Team-Filter der Dienstbörse) umfasst den
  erweiterten Kader.
- `terminmeldung-empfaenger`: Präzisierung — die Absage eines Dienstes erreicht alle
  Eingetragenen, auch eine Aushilfe aus dem erweiterten Kader.

## Impact

- `internal/duties/handler.go` — `Board` (Team-Quelle, `'eltern'`-Audience,
  `aushilfe`-Flags), Kommentar an `eligibleDutyRecipients`
- `internal/dutyfairness/fairness.go` — erweiterte Kader-Mitgliedschaften laden,
  Zurechnungsstufen, Aushilfe-Zähler je (Mitglied, Team); `handler.go` — Rangliste
- `internal/dashboard/handler.go` — `audienceMatchClauseSQL`, neuer Aushilfe-Block,
  Bilanz-Aushilfe
- Frontend: `DutySlotList.tsx`, `DashboardPage.tsx`, `DienstRanglistePage.tsx`
- Keine Migration, keine neue Route, kein neues Hub-Event (Broadcasts bestehen schon).
- Benutzerhandbuch (`web/public/benutzerhandbuch.html`) — Abschnitt Dienste.

## Test-Anforderungen

Keine neue Route; geänderte Geschäftslogik an bestehenden Routen.

| Route / Einheit | Test | Erwartet | Invariante |
|---|---|---|---|
| `GET /api/duty-board` | `TestBoard_ErweiterterKaderSiehtTeamDienste` | 200, Gruppen von Team B enthalten, `aushilfe: true` | Sichtbarkeit umfasst den erweiterten Kader |
| `GET /api/duty-board` | `TestBoard_ElternErweiterterKaderSehenElternSlot` | 200, `audiences=["eltern"]`-Slot enthalten | Eltern-Zielgruppe umfasst Kinder im erweiterten Kader |
| `GET /api/duty-board` | `TestBoard_AusgetretenImErweitertenKaderSiehtNichts` | 200, keine Gruppe von Team B | Statusfilter `<> 'ausgetreten'` |
| `GET /api/duty-board` | `TestBoard_FoerderkindImErweitertenKader` | 200, Gruppen von Team B | Förderkinder nicht per `= 'aktiv'` ausgeschlossen |
| `GET /api/duty-board` | `TestBoard_StammSchlaegtErweitert` | Gruppe eines A+B-Spiels `aushilfe: false` | Stamm schlägt erweitert |
| `GET /api/duty-board` | `TestBoard_AssigneeAushilfeKennzeichen` | Eintrag `aushilfe: true` für Aushilfe, `false` für Stammkader-Elternteil | Kennzeichen für alle Betrachter gleich |
| `GET /api/duty-board` | `TestBoard_FremdesTeamWeiterhinUnsichtbar` | keine Gruppe eines unverbundenen Teams C | Fehlerfall: keine Ausweitung über den Kader hinaus |
| `POST /api/duty-slots` | `TestCreateSlot_ErweiterterKaderBekommtKeinePush` | 201, erweiterter Kader + Eltern nicht in Empfängern | Empfänger ⊆ Sichtbarkeit |
| `dutyfairness.Compute` | `TestCompute_AushilfeZaehltNichtAufsStammteam` | Stamm-`geleistet` unverändert, Aushilfe für B = 1 | keine stille Gutschrift ans Stammteam |
| `dutyfairness.Compute` | `TestCompute_AushilfeAendertSollNicht` | Soll/Total/PlayerCount von B unverändert | fremdes Soll unberührt |
| `dutyfairness.Compute` | `TestCompute_StammkindSchlaegtAushilfekind` | Zuweisung voll beim Stammkader-Kind | Stufenreihenfolge |
| Board ↔ Bilanz | `TestAushilfePraedikat_BoardUndBilanzDeckungsgleich` | gleiche Aushilfe-Aussage für dieselbe Konstellation | ein Prädikat, zwei Formen |
| `GET /api/dashboard` | `TestDashboard_MeineDienste_ErweiterterKaderZaehltNicht` (bestehend) | unverändert grün | Stamm-Block unverändert |
| `GET /api/dashboard` | `TestDashboard_MeineDienste_AushilfeBlock` | `aushilfe.nextGame` = Spiel von B, `openSlotsCount` korrekt; Zusage nur in `aushilfe.mySlots` | Trennung der Blöcke |
| `GET /api/dashboard` | `TestDashboard_MeineDienste_OhneErweitertenKaderKeinBlock` | `aushilfe: null` | kein leerer Block |
| `GET /api/dashboard` | `TestDashboard_DutyAccountAushilfe` | `dutyAccountAushilfe` ohne `soll` | Bilanz getrennt |
| `GET /api/duty-fairness/rangliste` | `TestRangliste_AushilfeTeamImFilter` | 200, Team B angeboten | Filter umfasst erweiterten Kader |
| `GET /api/duty-fairness/rangliste` | `TestRangliste_AushilfenAnonymisiert` | fremde Aushilfe ohne `name`/`memberId` | Anonymisierung gilt auch für Aushilfen |
| `GET /api/duty-fairness/rangliste` | `TestRangliste_FremdesTeam_403` (bestehend) | 403 für unverbundenes Team | Fehlerfall bleibt |
| Frontend | `DutySlotList.aushilfe.test.tsx`, `DashboardPage.aushilfe.test.tsx`, `DienstRanglistePage.aushilfe.test.tsx` | Chip/Badge/Abschnitt erscheinen nur bei `aushilfe` | Kennzeichnung sichtbar |
