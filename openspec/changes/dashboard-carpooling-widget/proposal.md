## Why

Das Dashboard-Widget „Fahrtgemeinschaften" zeigt einen unklaren Mix aus Event-Verlauf, Statusinformationen und reinen Zählern — mit einem „Invalid Date"-Bug und redundanten Nennungen derselben Person. Nutzer können nicht erkennen, wer konkret Mitfahrt anbietet oder sucht, ohne die vollständige Mitfahrseite zu öffnen.

## What Changes

- `recentEvents` wird aus dem `CarpoolingHint`-Modell entfernt
- Neues Feld `openEntries` liefert Namen + Typ der offenen Einträge anderer Nutzer (max. 5, Variante B: alle ohne `confirmed`-Status)
- `bieteCount`/`sucheCount` bleiben als Gesamtzähler für „+ X weitere"-Anzeige
- Das Dashboard-Widget wird neu gestaltet: Spielzeile kompakt, bestätigte Paarungen prominent, offene Einträge als Namensliste, `myEntry` als kleine Statuszeile

## Capabilities

### New Capabilities

_(keine neuen Capabilities — nur Verbesserung eines bestehenden Widgets)_

### Modified Capabilities

- `dashboard-carpooling-hint`: Das `CarpoolingHint`-Modell ändert sich (recentEvents → openEntries); das Widget-Rendering ändert sich grundlegend

## Impact

- **Backend:** `internal/dashboard/handler.go` — Struct `CarpoolingHint`, Event-Query entfernen, neue openEntries-Query
- **Frontend:** `web/src/pages/DashboardPage.tsx` — Interface, Hilfskomponenten, Render-Logik in `CarpoolingHintCard`
- **API-Kontrakt:** Breaking change für den `/api/dashboard`-Endpunkt (recentEvents entfällt, openEntries kommt dazu) — da Frontend und Backend im selben Repo im Gleichschritt deployt werden, kein externer Koordinierungsaufwand
