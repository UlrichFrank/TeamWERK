## Why

Das Dashboard zeigt aktuell fünf Kacheln, von denen zwei ("Diese Woche" und "Dienstkonto") inhaltlich überlappen und deren Trennung Nutzer verwirrt. Gleichzeitig fehlen Trainings in der Terminübersicht, und die Fahrgemeinschafts-Kachel zeigt zu viel unstrukturierten Inhalt. Die Überarbeitung reduziert auf vier fokussierte Kacheln mit klarer Aufgabentrennung.

## What Changes

- **Entfernt**: Kachel "Diese Woche" (Dienst-Aktionen der laufenden Woche)
- **Entfernt**: Kachel "Dienstkonto" (separater Saison-Saldo-Block)
- **Neu**: Kachel "Meine Termine" — tagesbasiert, alle Event-Typen (Trainings + Spiele), Navigation zu Termindetails
- **Neu**: Kachel "Meine Dienste" — Dienst-Slots des nächsten Spiels mit Slots + Saison-Saldo
- **Umbenannt + erweitert**: "Dein Team" → "Mein Team" — Dashboard zeigt Links je Team, neue Detailseite mit Trainer/Spieler/Eltern-Tabellen
- **Gefiltert**: "Fahrgemeinschaften" — nur beidseitig bestätigte Paare, über nächste max. 3 Auswärtsspiele

## Capabilities

### New Capabilities

- `dashboard-meine-termine`: Tagesbasierte Terminübersicht auf dem Dashboard — nächster Tag mit Terminen, alle Events dieses Tages (training_sessions + games), Navigation zu Detailseiten
- `dashboard-meine-dienste`: Zusammengeführte Dienst-Kachel — Slots des nächsten Spiels mit Diensten (eigene Zusagen oder offene Anzahl) + Saison-Saldo
- `mein-team-seite`: Neue Seite `/mein-team` mit gestapelten Tabellen je Team (Trainer, Spieler, Eltern mit Kontaktdaten), neuer Backend-Endpoint `GET /api/teams/:id/roster`

### Modified Capabilities

- `dashboard-carpooling-hint`: Kachel zeigt nur noch beidseitig bestätigte Paare (statt alle offenen Einträge), Zeitfenster auf nächste 3 Auswärtsspiele erweitert

## Impact

**Backend:**
- `internal/dashboard/handler.go`: `queryNextGames` → `queryNextEvents` (UNION training_sessions + games), neue `queryMeineDienste`-Funktion, angepasste `queryCarpoolingHint`
- Neues Package oder Erweiterung in `internal/members` oder `internal/config`: Endpoint `GET /api/teams/:id/roster`

**Frontend:**
- `web/src/pages/DashboardPage.tsx`: Alle vier Kacheln überarbeitet
- Neue Seite `web/src/pages/MeinTeamPage.tsx`
- `web/src/App.tsx`: Route `/mein-team` hinzufügen
- `web/src/components/AppShell.tsx`: Nav-Eintrag für Mein Team prüfen
