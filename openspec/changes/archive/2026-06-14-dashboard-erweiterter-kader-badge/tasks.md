## 1. Backend: GET /teams/my

- [x] 1.1 `Team`-Struct in `internal/teams/handler.go` um `IsExtended bool \`json:"isExtended"\`` erweitern
- [x] 1.2 `ListMyTeams`-Query auf UNION-Muster umbauen: Stammkader/Trainer/Eltern → `is_extended=0`, Nur-Erweiterter-Kader → `is_extended=1` (mit NOT EXISTS-Guard)
- [x] 1.3 Test: Spieler nur in `kader_extended_members` → `isExtended: true`; Stammkader-Spieler → `isExtended: false`

## 2. Backend: GET /dashboard → meineTermine

- [x] 2.1 `NextEvent`-Struct in `internal/dashboard/handler.go` um `IsExtended bool \`json:"isExtended"\`` erweitern
- [x] 2.2 `queryNextEvents` um CTE `extended_teams` ergänzen; CASE WHEN in SELECT für `is_extended`
- [x] 2.3 Scan in `queryNextEvents` das neue Feld einlesen

## 3. Frontend: Dashboard-Komponenten

- [x] 3.1 `NextEvent`-Interface in `DashboardPage.tsx` um `isExtended: boolean` erweitern
- [x] 3.2 `MeinTeamSection`: Teams-Fetch auf neues `isExtended`-Feld auswerten; Badge „Erw. Kader" rendern wenn `true`
- [x] 3.3 `MeineTermineSection`: `isExtended`-Feld auswerten; Zusatz „(Erw. Kader)" in Teamzeile rendern wenn `true`

## 4. Test-Anforderungen

- [x] 4.1 `TestListMyTeams_IsExtended`: User nur in `kader_extended_members` → `isExtended: true`
- [x] 4.2 `TestListMyTeams_IsNotExtended`: User in `kader_members` → `isExtended: false`
- [x] 4.3 `TestDashboard_MeineTermine_IsExtended`: Training-Event eines Extended-Teams hat `isExtended: true`
