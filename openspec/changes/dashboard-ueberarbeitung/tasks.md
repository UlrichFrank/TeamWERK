## 1. Backend: Meine Termine

- [ ] 1.1 `queryNextEvents` in `internal/dashboard/handler.go` implementieren: UNION-Query auf `training_sessions` und `games` für User-Teams, frühestes Datum ermitteln, alle Events dieses Datums zurückgeben
- [ ] 1.2 Response-Struct `NextEvent` mit Feldern `id`, `type`, `date`, `time`, `title`, `team_name`, `detail_url` definieren
- [ ] 1.3 `Response`-Struct um `MeineTermine []NextEvent` erweitern, altes `NextGames`-Feld entfernen

## 2. Backend: Meine Dienste

- [ ] 2.1 `queryMeineDienste` implementieren: nächstes `game` mit `duty_slots` für aktive Saison und User-Teams ermitteln
- [ ] 2.2 Logik: eigene Slots des Users abfragen (`status IN ('assigned','fulfilled','cash_substitute')`), sonst `openSlotsCount` berechnen
- [ ] 2.3 Response-Struct `MeineDienste` mit `nextGame`, `mySlots`, `openSlotsCount`, `dutyAccount` definieren
- [ ] 2.4 `Response`-Struct um `MeineDienste` erweitern, alte `Actions`- und separate `DutyAccount`-Felder entfernen

## 3. Backend: Mein Team Roster

- [ ] 3.1 Neuen Handler-Endpoint `GET /api/teams/:id/roster` in passendem Package anlegen (z.B. `internal/config` oder neues `internal/teams`)
- [ ] 3.2 Berechtigungsprüfung via `user_accessible_teams` implementieren (HTTP 403 bei fehlendem Zugriff)
- [ ] 3.3 Query für Trainer (via `kader_trainers` + `members` + `users`) implementieren
- [ ] 3.4 Query für Spieler (via `kader_members` + `members` + verlinkter `users`) implementieren
- [ ] 3.5 Query für Eltern (via `family_links`, nur Eltern deren Kinder im Kader sind) implementieren
- [ ] 3.6 Route in `cmd/teamwerk/main.go` registrieren

## 4. Backend: Fahrgemeinschaften

- [ ] 4.1 `queryCarpoolingHint` umbauen: nur `confirmed`-Paarungen des Users, über nächste max. 3 Auswärtsspiele
- [ ] 4.2 Response-Typ von single object auf Array (`[]CarpoolingConfirmed`) ändern
- [ ] 4.3 Felder `bieteCount`, `sucheCount`, `myEntry`, `openEntries` aus Response entfernen

## 5. Frontend: Dashboard Kacheln

- [ ] 5.1 `DashboardPage.tsx`: Kachel "Diese Woche" entfernen, neue Kachel "Meine Termine" mit `meineTermine`-Daten und Links zu `detail_url` rendern
- [ ] 5.2 `DashboardPage.tsx`: Kachel "Dienstkonto" entfernen, neue Kachel "Meine Dienste" implementieren (eigene Slots oder offene Anzahl + Saldo)
- [ ] 5.3 `DashboardPage.tsx`: Kachel "Dein Team" umbenennen in "Mein Team", einen Link pro Team des Users anzeigen (statt generischem Link)
- [ ] 5.4 `DashboardPage.tsx`: Kachel "Fahrgemeinschaften" auf neues Array-Format umstellen, nur bestätigte Paare darstellen
- [ ] 5.5 TypeScript-Interfaces für `NextEvent`, `MeineDienste`, `CarpoolingConfirmed` aktualisieren

## 6. Frontend: Mein Team Seite

- [ ] 6.1 Neue Seite `web/src/pages/MeinTeamPage.tsx` erstellen: API-Call zu `/api/teams/:id/roster`, gestapelte Tabellen (Trainer / Spieler / Eltern) je Team
- [ ] 6.2 Route `/mein-team` in `web/src/App.tsx` registrieren
- [ ] 6.3 Nav-Eintrag "Mein Team" in `AppShell.tsx` für Trainer, Spieler und Eltern sichtbar schalten
