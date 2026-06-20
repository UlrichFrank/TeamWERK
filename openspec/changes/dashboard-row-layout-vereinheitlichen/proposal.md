## Why

Das Dashboard hat heute vier Sektionen (Meine Termine, Meine Dienste, Mein Team, Fahrgemeinschaften) mit drei unterschiedlichen Zeilen-Layouts:

- **Meine Termine**: vier-spaltige Zeile `Datum | Icon | Titel+Subtitel | →`
- **Meine Dienste**: Gruppen-Header „Datum · Gegner", darunter Liste `Check + dutyTypeName + Zeit`
- **Fahrgemeinschaften**: Gruppen-Header pro Spiel, getrennte Sub-Header „Bestätigt" / „Offene Gesuche"
- **Mein Team**: zweispaltig `Name | →`

Das wirkt visuell inkonsistent. Da Termine, Dienste und Fahrgemeinschaften alle „terminbezogene Aktionen am Tag X" sind, sollen sie identisch wie eine Tabelle aussehen — gleiches Spaltenraster über die Sektionen hinweg. Zusätzlich ist die heutige Reihenfolge `Mein Team` vor `Fahrgemeinschaften` ungünstig: die navigationslastige Team-Sektion gehört ans Ende, terminbezogene Sektionen nach oben.

Für die Fahrt-Zusagen fehlt im API-Payload der Treffpunkt des Partners — die Subtitel-Spalte hätte sonst nur den Gegnernamen als einzigen Token und würde aus dem Spaltenraster fallen.

## What Changes

**Frontend (`web/src/pages/DashboardPage.tsx`):**
- Section-Reihenfolge: `Termine → Dienste → Fahrgemeinschaften → Mein Team`
- Neue gemeinsame Row-Komponente `DashboardRow` mit Spaltenraster `w-10 | w-4 Icon | flex-1 min-w-0 | w-4 →`
- Spalte 3 ist immer zweizeilig: `text-sm font-medium text-brand-text truncate` + `text-xs text-brand-text-muted`
- `MeineTermineSection` nutzt `DashboardRow` (Umstrukturierung, kein Inhaltswechsel)
- `MeineDiensteSection`: kein Gruppen-Header mehr; jeder Slot ist eine `DashboardRow` mit Spiel-Datum links, ✓-Icon, `dutyTypeName` als Titel, `opponent · eventTime` als Subtitel. Fallback-Zeile „N offene Dienste verfügbar" mit Info-Icon. Dienstkonto-Toggle bleibt unverändert unten.
- `FahrgemeinschaftenSection`: keine Sub-Header mehr; flache Liste aus Zusagen (✓-Icon) und offenen Gesuchen (🔍-Icon), chronologisch sortiert.

**Backend (`internal/dashboard/handler.go`):**
- `carpoolingConfirmed[].paarungen[]` wird um `partnerTreffpunkt` (string, leer wenn nicht gesetzt) erweitert.
- Der Wert ist der Treffpunkt der **Partner-Seite** der Paarung: bin ich Bieter, ist es der Treffpunkt des Sucher-Eintrags; bin ich Sucher, ist es der Treffpunkt des Bieter-Eintrags. Dieselbe Logik gilt für Kinder-Paarungen.

**Keine Änderungen** an: API-Routen, DB-Schema, Auth, anderen Frontend-Seiten, Tests von nicht betroffenen Domänen.

## Capabilities

### New Capabilities

_(keine neuen Capabilities)_

### Modified Capabilities

- `dashboard-carpooling-hint`: Payload `carpoolingConfirmed[].paarungen[]` erhält Feld `partnerTreffpunkt`.

## Test-Anforderungen

| Route | Test | Erwartung |
|---|---|---|
| `GET /api/dashboard` | `TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_AsBieter` | `200`; eigene Bieter-Paarung liefert `partnerTreffpunkt = <Treffpunkt des Sucher-Eintrags>` |
| `GET /api/dashboard` | `TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_AsSucher` | `200`; eigene Sucher-Paarung liefert `partnerTreffpunkt = <Treffpunkt des Bieter-Eintrags>` |
| `GET /api/dashboard` | `TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_Empty` | `200`; wenn Partner keinen Treffpunkt gesetzt hat, ist `partnerTreffpunkt = ""` |
| `GET /api/dashboard` | `TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_KindAsBieter` | `200`; Eltern-User sieht für Kind-Bieter-Paarung den Sucher-Treffpunkt |

**Invariante:** `partnerTreffpunkt` ist immer der Treffpunkt der **Gegenseite** der Paarung — nie der eigene und nie das Spiel-Venue.

## Impact

- **Backend-Dateien:** `internal/dashboard/handler.go`, `internal/dashboard/handler_test.go`
- **Frontend-Dateien:** `web/src/pages/DashboardPage.tsx`
- **DB-Migration:** keine
- **API-Schema:** additives Feld `partnerTreffpunkt` in `carpoolingConfirmed[].paarungen[]` (bestehende Clients ignorieren es)
- **Risiken:** UI-Änderung — Nutzer müssen sich an neue Reihenfolge gewöhnen. Kein Daten- oder Berechtigungsrisiko.
- **Rollback:** rein Frontend + additives Feld — durch Revert eines Commits rückgängig machbar.
