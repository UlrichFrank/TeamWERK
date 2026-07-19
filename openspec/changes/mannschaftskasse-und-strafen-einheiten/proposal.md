## Why

Der Basis-Change `mannschaft-aufgaben-strafen` hat Strafen sichtbar in der App verankert, aber zwei Dinge verwechselt bzw. offengelassen:

1. **Einheit**: Handball-Mannschaften rechnen Strafen in der Praxis entweder in **Euro** oder in **Strichen** (Notches). Der aktuelle Stand kennt nur Euro (`amount_cent`), Striche-Teams müssten „umrechnen im Kopf".
2. **Mannschaftskasse ≠ Strafen**: Der Strafen-Tab zeigt heute an oberster Stelle die Zeile „Mannschaftskasse: 42,50 €" — reine Summe aller offenen Strafen. Das suggeriert eine Kasse, die es gar nicht gibt. Was fehlt, ist ein echtes kleines **Kassenbuch** (Einzahlungen, Ausgaben, Saldo), unabhängig von den Strafen — realistisch, weil Teams Geld auch für Trainergeschenke, Kabinen-Bier oder Startgelder ausgeben, nicht nur für Strafen.

Zusätzlich eine kleine UX-Verbesserung: In den Listen (Strafen, Kasse, Rosters) soll der eingeloggte Nutzer **fett** dargestellt werden, damit er seine eigene Zeile sofort findet.

## What Changes

- **Strafen bekommen eine wählbare Einheit** (`euro` oder `striche`) pro Kader/Saison. Default `euro`. Bei `striche` sind nur ganze Zahlen zulässig; Katalog-Default und Vergabe-Betrag werden im Backend gegen `n % 1 == 0` validiert.
- **Wechsel der Einheit rechnet automatisch um**, mit Bestätigungs-Preview: `1 € = 1 Strich` (Identitäts-Rate). Euro → Striche rundet Kommazahlen auf (`ceil(cent / 100)`), Striche → Euro multipliziert exakt (`n * 100`). Katalog-Defaults und alle bestehenden Strafen dieses Kaders werden in einer Transaktion umgerechnet.
- **Neue Capability `mannschaftskasse`**: eigenes kader-scoped Kassenbuch (`team_cashbook_entries`) mit signierten Beträgen (Einzahlung positiv, Ausgabe negativ), Notiz, Datum, Buchender. Saldo = SQL-Summe.
- **Neue per-Kader-Appointment `Kassenwart`** (Table `kader_kassenwarte`, Sibling zu `kader_strafenwarte`), vom Trainer ernannt. **Trainer ODER Kassenwart** dürfen buchen; **Trainer** ernennt. Kein neuer globaler `member_club_functions`-Wert (analog zum Strafenwart-Modell aus dem Basis-Change).
- **Neuer Tab „Kasse"** unter „Mein Team", sichtbar für Spieler + Trainer + Erweiterten Kader (nicht für Eltern, nicht für Außenstehende — dasselbe Read-Gate wie Strafen).
- **Strafen-Tab bereinigt**: die irreführende „Mannschaftskasse"-Kopfzeile (Zeilen 484–487 in `MeinTeamPage.tsx`) wird entfernt. Der Strafen-Tab zeigt fortan nur noch offene Strafen pro Spieler.
- **Verwalten-Tab bekommt Kassenwart-Sektion** direkt unter der Strafenwart-Sektion (gleicher UI-Baustein).
- **Eigene Zeile fett**: In den Roster-Tabs (Team/Trainer/Eltern), der Strafen-Übersicht und im Kassenbuch wird die Zeile des eingeloggten Nutzers `font-semibold`. Kriterium: `roster.mymember.id === row.memberId` (oder analog).
- **SSE**: neue Broadcast-Events `cashbook`, `kassenwarte`, `penalty-settings`; jede Mutations-Route ruft `h.hub.Broadcast(...)`, Frontend abonniert via `useLiveUpdates`.

## Capabilities

### New Capabilities
- `mannschaftskasse`: kader-scoped Kassenbuch (Einzahlungen/Ausgaben mit Notiz, Saldo), `Kassenwart` als per-Kader-Appointment, Read-Gate wie Strafen (keine Eltern), Trainer und Kassenwart dürfen buchen, Trainer ernennt.

### Modified Capabilities
- `mannschaftsstrafen`: Strafen bekommen die Einheit **Euro | Striche** (per Kader). Einheit-Wechsel rechnet vorhandene Beträge um. „Mannschaftskasse"-Header verschwindet aus dem Strafen-Tab. Read-/Write-Gates und Snapshot-Semantik bleiben unverändert.

## Impact

- **Neue Migration** (`internal/db/migrations/`, nächste freie Nummer, `.up.sql` + `.down.sql`): drei Tables — `penalty_settings(kader_id PK, unit CHECK IN ('euro','striche'))`, `team_cashbook_entries(id, kader_id, amount_cent [signed], note, entered_by_member_id, entered_at)`, `kader_kassenwarte(kader_id, member_id) PK`. Alle FK auf `kader(id) ON DELETE CASCADE`.
- **Backend:** `internal/teams/access.go` — neuer Helper `isKassenwartOfTeam`, ergänzt bestehendes `canReadPenalties` als `canReadCashbook` (identisches Prädikat, benannt für Klarheit). Neue Handler-Dateien `cashbook.go`, `penalty_settings.go` in `internal/teams`. Router `internal/app/router.go` — neue Routen unter Authenticated- (Read) und Trainer-Tier (Write). Strafen-Handler (`penalties.go`) erhält Umrechnungs-Transaktion und Preview-Route.
- **SSE (Hard Rule):** jede Mutations-Route ruft `h.hub.Broadcast(...)` (Events `cashbook`, `kassenwarte`, `penalty-settings`, plus `penalties` beim Einheiten-Wechsel wegen Massen-Update); vom `internal/arch/broadcast_test.go` automatisch erfasst.
- **Frontend:** `web/src/pages/MeinTeamPage.tsx` — neuer Tab „Kasse", Strafen-Tab-Header raus, Einheiten-Umschaltung (Trainer-Verwaltung mit Preview-Modal), Verwalten-Tab-Sektion für Kassenwart, `useLiveUpdates` erweitert um neue Events. Bold-Me-Logik als kleiner Utility-Vergleich in den relevanten Listen. `brand-*`-Tokens, `lucide-react`.
- **Tests (Hard Rule):** Happy-Path + Fehlerfall je neuer Route; Fixtures in `internal/testutil/` um `AppointKassenwart`, `CreateCashbookEntry`, `SetPenaltyUnit` erweitert.
- **Abhängigkeit:** setzt voraus, dass `mannschaft-aufgaben-strafen` in `openspec/specs/mannschaftsstrafen/` archiviert ist (die MODIFIED-Requirements referenzieren den dortigen Baseline-Text). Reihenfolge: Basis-Change mergen + archivieren → dieser Change.

## Test-Anforderungen

Jede neue Route bekommt Happy-Path + Fehlerfall. Route → Testname → erwarteter Status → garantierte Invariante:

**Einheiten (`mannschaftsstrafen` modified)**

- `GET /api/teams/{id}/penalty-settings` → `TestPenaltySettings_Read_200` → 200 → jeder Team-Leser sieht die aktive Einheit.
- `PUT /api/teams/{id}/penalty-settings` → `TestPenaltySettings_Trainer_200` → 200 → Trainer darf Einheit setzen.
- `PUT /api/teams/{id}/penalty-settings` → `TestPenaltySettings_NonTrainer_403` → 403 → kein Nicht-Trainer.
- `PUT /api/teams/{id}/penalty-settings` → `TestPenaltySettings_EurToStriche_RoundsUpAndConverts` → 200 → `5,50 € → 6 Striche`, atomar in einer TX; alle betroffenen Rows aktualisiert.
- `PUT /api/teams/{id}/penalty-settings` → `TestPenaltySettings_StricheToEur_ExactConversion` → 200 → `6 Striche → 6,00 €`, keine Rundung.
- `POST /api/teams/{id}/penalties` → `TestPenaltyCreate_StricheUnit_NonInteger_400` → 400 → Betrag mit Kommastellen bei Einheit `striche` wird rejected.
- `GET /api/teams/{id}/penalty-settings/preview?to=striche` → `TestPenaltySettings_Preview_NoMutation` → 200 → Preview enthält Delta-Liste + Anzahl aufgerundet, aber DB bleibt unverändert.

**Kassenbuch (`mannschaftskasse` neu)**

- `GET /api/teams/{id}/cashbook` → `TestCashbookRead_Player_200` / `_Trainer_200` / `_Extended_200` → 200 → Ledger + Saldo für Team-Interne.
- `GET /api/teams/{id}/cashbook` → `TestCashbookRead_Parent_403` → 403 → **Eltern sehen die Kasse nicht** (Kern-Invariante analog Strafen).
- `GET /api/teams/{id}/cashbook` → `TestCashbookRead_Outsider_403` → 403 → Außenstehende gesperrt.
- `POST /api/teams/{id}/cashbook` → `TestCashbookCreate_Trainer_200` / `_Kassenwart_200` → 200/201 → Trainer und Kassenwart dürfen buchen.
- `POST /api/teams/{id}/cashbook` → `TestCashbookCreate_Spieler_403` → 403 → normaler Spieler ohne Rolle darf nicht buchen.
- `POST /api/teams/{id}/cashbook` → `TestCashbookCreate_ForeignTeamKassenwart_403` → 403 → **Kassenwart Team A kann Team B nicht bebuchen** (keine Row angelegt).
- `DELETE /api/teams/{id}/cashbook/{eid}` → `TestCashbookDelete_Kassenwart_200` → 200 → einzelne Buchung hart gelöscht.
- `POST /api/teams/{id}/kassenwarte` → `TestKassenwartAppoint_Trainer_200` / `_NonTrainer_403` → 200/403 → nur Trainer ernennt.
- Invariante `TestCashbook_ExcludedFromRoster` → 200 → Roster-Response enthält keinerlei Kassendaten (analog zur Strafen-Ausschluss-Invariante).
- Invariante `TestClubFunctions_NoKassenwartValue` → CHECK-Constraint von `member_club_functions` enthält keinen `kassenwart`-Wert.
- Broadcast-Gate (`internal/arch/broadcast_test.go`) erfasst alle Mutations-Routen automatisch (`cashbook`/`kassenwarte`/`penalty-settings`).

## Non-Goals

- **Keine Buchungs-Historie mit Audit-Trail über Änderungen**: Kassenbuch-Einträge werden hart gelöscht wie Strafen — Historie „wer hat wann was editiert" ist ein separater Follow-up-Change, falls sich das je als nötig erweist.
- **Keine Kopplung Kasse↔Strafen**: Eine Buchung markiert **keine** Strafe als „bezahlt". Das würde die bewusst entkoppelte Architektur unterlaufen. Ausgleich bleibt manuell (Kassenwart bucht + Strafenwart resettet).
- **Kein konfigurierbarer Wechselkurs**: `1 € = 1 Strich` ist fest verdrahtet. Andere Raten sind nicht vorgesehen.
- **Kein automatischer Saldo-Warnstand / keine Push-Benachrichtigungen**: die Kasse ist eine reine Anzeige.
- **Keine Änderung am orthogonalen zweidimensionalen Berechtigungsmodell**: `Kassenwart` ist per-Kader-Appointment, kein globaler Vereinsfunktions-Wert und kein JWT-Claim.
