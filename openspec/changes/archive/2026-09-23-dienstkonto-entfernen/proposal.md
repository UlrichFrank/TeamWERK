# Proposal

## Why

Das Dienstkonto (`duty_accounts` mit `soll`/`ist`) ist eine Altlast. Der Stub
`dienstkonto-ist-buchung` hat den Befund festgehalten: `Fulfill` und `CashSubstitute`
buchen nie auf `ist`, nur das Löschen eines Termins rechnet für die betroffenen Nutzer neu.
`GET /api/duty-accounts` und der CSV-Export liefern damit eine Zahl, die vom
Löschverhalten des Vorstands abhängt, statt von den geleisteten Diensten.

Inzwischen liest die Oberfläche keine der beiden Routen mehr. Die Dienst-Bilanz auf dem
Dashboard und die Rangliste rechnen live über `internal/dutyfairness` aus
`duty_slots`/`duty_assignments`, und zwar bewusst ohne `duty_accounts`. Eine Reparatur der Buchung
würde also eine zweite, unbenutzte Zahl für dieselbe Größe am Leben halten. Deshalb wird entfernt statt repariert.

## What Changes

- **BREAKING (API):** `GET /api/duty-accounts` und `GET /api/duty-accounts/export` entfallen samt
  Handlern (`duties.Accounts`, `duties.ExportAccounts`). Kein Frontend-Aufrufer.
- `Claim` legt keine `duty_accounts`-Zeile mehr an.
- `DeleteGame` rechnet `duty_accounts.ist` nicht mehr nach; die Sammlung der betroffenen
  Nutzer (`fulfilledUIDs`) entfällt, soweit sie nur dafür existiert.
- `DeleteUser` löscht nicht mehr explizit aus `duty_accounts` (die Tabelle hängt per
  `ON DELETE CASCADE` an `users`).
- Tests, Permission-Matrix, Doku (`docs/berechtigungen.md`, Gotchas, Paketkommentar
  `dutyfairness`) werden nachgezogen.
- Der Stub `dienstkonto-ist-buchung` wird verworfen; sein Befund ist mit diesem Change erledigt.
- **Die Tabelle `duty_accounts` bleibt vorerst bestehen** (siehe design.md): ein `DROP TABLE`
  würde `make deploy-rollback` brechen, weil das Vorgänger-Binary sie beim Löschen von
  Terminen und Nutzern noch beschreibt. Der Drop folgt als eigene Migration, sobald dieser
  Stand deployt und stabil ist.

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `duties`: Stunden-Auswertungen lesen `duty_slots.hours_value`; kein `duty_accounts.ist` mehr.
- `game-deletion-cascade`: Requirement „Konto-Konsistenz bei Cascade-Delete" entfällt.
- `dienst-fuer-familienmitglied`: Stellvertreter-Claim zählt in der Dienst-Bilanz des Kindes statt im Konto.
- `test-auth`: Cascade beim Nutzer-Löschen ohne `duty_accounts`.
- `test-duties`: Claim ohne Konto-Seiteneffekt; Requirement „Dienstkonten-Sichtbarkeit" entfällt.
- `test-duties-gaps`: Fulfill-Invariante zu `duty_accounts.ist` entfällt.
- `permissions`: `GET /api/duty-accounts` und `/export` aus den Tier-Listen entfernt.
- `mobile-table-cards`: die nicht mehr existierende `DutyAccountsPage` wird gestrichen.

## Impact

- `internal/app/router.go`, `internal/duties/handler.go`, `internal/games/handler.go`,
  `internal/auth/handler.go`, `internal/dutyfairness/fairness.go` (Kommentar)
- Tests: `internal/duties/handler_test.go`, `internal/games/handler_test.go`,
  `internal/permissions/matrix_test.go`
- Doku: `docs/berechtigungen.md`, `docs/agent/06-gotchas.md`
- Keine Migration, kein Frontend.

## Test-Anforderungen

Keine neue Route. Entfernte Routen und Invarianten:

| Route | Test | Erwartet |
|---|---|---|
| `GET /api/duty-accounts` | Permission-Matrix (Eintrag entfernt; Drift-Schutz meldet jede verbliebene Route) | Route existiert nicht mehr |
| `GET /api/duty-accounts/export` | dito | Route existiert nicht mehr |
| `POST /api/duty-board/{slotId}/claim` | bestehender Claim-Test ohne `duty_accounts`-Assertion | 204, Zuweisung angelegt |
| `DELETE /api/games/{id}` | bestehende Cascade-Tests ohne `ist`-Assertion | 204, Slots und Zuweisungen weg |

**Invariante:** Kein Anwendungscode liest oder schreibt `duty_accounts` (prüfbar per `git grep duty_accounts -- internal web/src` → nur Migrationen).
