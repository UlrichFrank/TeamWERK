## Why

Die Dashboard-Sektion „Fahrgemeinschaften" zeigt heute ausschließlich **bestätigte** Paarungen (`carpoolingConfirmed`, gefüllt von `queryCarpoolingConfirmed` in `internal/dashboard/handler.go`). **Offene Gesuche** — `mitfahrgelegenheiten`-Einträge mit `typ='suche'`, die noch keine `confirmed`-Paarung haben — werden fürs Dashboard nirgends abgefragt.

Folge: Ein Mitglied mit freien Plätzen sieht auf dem Dashboard nicht, dass jemand im eigenen Team zu einem kommenden Spiel eine Mitfahrt sucht. Die Information existiert (`/mitfahrgelegenheiten`), liegt aber eine Navigation entfernt. Die nächsten Termine stehen bereits unter „Meine Termine" — daneben gehören die offenen Gesuche dazu.

Diese Änderung bringt nur die **eigenen Teams** ins Dashboard. Die teamübergreifende Variante (gleicher Tag/Ort) ist bewusst als Folge-Proposal `dashboard-offene-gesuche-cross-team` ausgegliedert.

## What Changes

- Neue Backend-Funktion `queryCarpoolingOpenRequests(userID, seasonID)` in `internal/dashboard/handler.go`.
- Neues Feld `carpoolingOpenGroups` in der `GET /api/dashboard`-Antwort. Pro kommendem Spiel der eigenen Teams (nächste max. 3 Spiele, **alle** `event_type`) eine Gruppe mit den offenen Suche-Einträgen.
- „Offen" = `typ='suche'` **ohne** `mitfahrt_paarungen`-Eintrag mit `status='confirmed'`. Eine nur `pending`-Paarung zählt weiter als offen.
- Frontend: `FahrgemeinschaftenSection` (`web/src/pages/DashboardPage.tsx`) bekommt unter den bestätigten Paarungen einen Block „Offene Gesuche". Kein zusätzliches Live-Update-Wiring nötig — `useLiveUpdates('mitfahrgelegenheiten')` reloadet das Dashboard bereits.
- `carpoolingConfirmed` und `queryCarpoolingConfirmed` bleiben **unverändert**.
- Keine Migration, keine neue Route.

## Capabilities

### New Capabilities

- `dashboard-offene-gesuche`: Das Dashboard zeigt zu den kommenden Spielen der eigenen Teams die offenen Mitfahr-Gesuche.

### Modified Capabilities

_(keine — `dashboard-carpooling-hint` bleibt unberührt)_

## Test-Anforderungen

| Route | Testname (Vorschlag) | Erwartung / Invariante |
|---|---|---|
| `GET /api/dashboard` | `TestDashboard_OffeneGesuche_OwnTeam` | 200; offenes `suche` an kommendem Spiel eines eigenen Teams erscheint in `carpoolingOpenGroups`. |
| `GET /api/dashboard` | `TestDashboard_OffeneGesuche_ConfirmedExcluded` | `suche` mit `confirmed`-Paarung erscheint **nicht** in `carpoolingOpenGroups` (aber weiter in `carpoolingConfirmed`). |
| `GET /api/dashboard` | `TestDashboard_OffeneGesuche_PendingStillOpen` | `suche` mit nur `pending`-Paarung erscheint **weiter** als offen. |
| `GET /api/dashboard` | `TestDashboard_OffeneGesuche_OtherTeamExcluded` | `suche` an einem Spiel, das keinem eigenen Team gehört, erscheint **nicht** (Cross-Team ist Folge-Proposal). |

**Garantierte Invariante:** In `carpoolingOpenGroups` erscheint ein `suche` genau dann, wenn (a) das zugehörige Spiel in der aktiven Saison liegt, künftig ist und einem eigenen Team gehört, und (b) keine `confirmed`-Paarung darauf existiert.

## Impact

- **Datei:** `internal/dashboard/handler.go` — neue Methode + neues Response-Feld.
- **Datei:** `internal/dashboard/handler_test.go` — neue Tests.
- **Datei:** `web/src/pages/DashboardPage.tsx` — neuer Block in `FahrgemeinschaftenSection`, Typ-Erweiterung der Dashboard-Response.
- **Kein** Schema-/Migrations-Change, **keine** neue Route.
