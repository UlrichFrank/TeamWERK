## 1. DB-Migration

- [x] 1.1 Migration `022_erweiterter_kader_access.up.sql` erstellen: `user_accessible_teams` View neu definieren mit zusätzlichem UNION-Arm für `kader_extended_members`
- [x] 1.2 Migration `022_erweiterter_kader_access.down.sql` erstellen: View auf Stand von Migration 018 zurücksetzen
- [x] 1.3 `make migrate-up` lokal ausführen und verifizieren dass abgesetzte Spieler nun in `user_accessible_teams` erscheinen

## 2. Backend — Roster-API

- [x] 2.1 `internal/teams/handler.go`: `RosterResponse` um Feld `ExtendedPlayers []PlayerEntry` ergänzen
- [x] 2.2 `internal/teams/handler.go` `GetRoster`: Query für extended players hinzufügen (JOIN auf `kader_extended_members` + `kader` + `members`)
- [x] 2.3 Test `TestGetRoster_ExtendedPlayers`: Team mit regulären + abgesetzten Spielern → `extended_players` korrekt befüllt, Spieler nicht doppelt in `players`

## 3. Backend — ListMyGames

- [x] 3.1 `internal/games/handler.go` `ListMyGames`: Team-Filter-Bedingung für `spieler`-Rolle um UNION-Arm für `kader_extended_members` erweitern
- [x] 3.2 `internal/games/handler.go` `ListMyGames`: SELECT um Subquery `in_regular_kader` (EXISTS über `kader_members`) ergänzen; `args`-Slice entsprechend erweitern
- [x] 3.3 `internal/games/handler.go` `ListMyGames`: Scan um `inRegularKader`-Variable erweitern; Auto-Confirm nur setzen wenn `inRegularKader == true`
- [x] 3.4 Test `TestListMyGames_ExtendedKaderSiehtSpiel`: abgesetzter Spieler sieht Spiel seines erweiterten Teams
- [x] 3.5 Test `TestListMyGames_ExtendedKaderKeinAutoConfirm`: opt-out-Spiel → abgesetzter Spieler bekommt `my_rsvp: null`
- [x] 3.6 Test `TestListMyGames_RegularKaderAutoConfirmBleibt`: opt-out-Spiel → reguläres Mitglied bekommt weiterhin `my_rsvp: confirmed`

## 4. Frontend — MeinTeamPage

- [x] 4.1 `web/src/pages/MeinTeamPage.tsx`: Interface `TeamRoster` um `extendedPlayers: PlayerEntry[]` ergänzen
- [x] 4.2 `web/src/pages/MeinTeamPage.tsx` `RosterSection`: im Team-Tab nach der Spielertabelle, wenn `roster.extendedPlayers.length > 0`, Abschnitt „Erweiterter Kader" mit gleichem Tabellenlayout (# + Name) rendern

## 5. Frontend — TermineDetailPage

- [x] 5.1 `web/src/pages/TermineDetailPage.tsx` `ResponseTable`: `rows` in zwei Arrays aufteilen — `regularRows` (`!is_extended`) und `extendedRows` (`is_extended`)
- [x] 5.2 `ResponseTable`: reguläre Zeilen wie bisher rendern; wenn `extendedRows.length > 0` einen Trenner mit Heading „Erweiterter Kader" + eigene `<tbody>`-Gruppe darunter rendern; „Erw."-Badge entfernen (visuelle Trennung ersetzt ihn)
