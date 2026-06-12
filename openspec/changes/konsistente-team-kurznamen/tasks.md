## 1. Backend — neuer Endpoint

- [x] 1.1 Handler `ListTeamNames` in `internal/games/handler.go` implementieren: Query aller aktiven Teams mit `id, age_class, gender, team_number, group_count` via Kader-Join (analog zu `ListTeamsForUser`, ohne Rollenfilter)
- [x] 1.2 Route `GET /api/teams/names` in `cmd/teamwerk/main.go` im authenticated-Block registrieren

## 2. Frontend — teamName.ts aufräumen

- [x] 2.1 `buildTeamDisplayNames` aus `web/src/lib/teamName.ts` entfernen

## 3. Frontend — KalenderPage

- [x] 3.1 `/api/teams/names` parallel zu den anderen Initial-Loads in `loadInitialData` laden, Ergebnis in neuem State `allTeamNames`
- [x] 3.2 `shortNames` aus `allTeamNames` statt aus `teams` berechnen
- [x] 3.3 `displayNames`-Berechnung und alle Verwendungen entfernen, `buildTeamDisplayNames`-Import entfernen

## 4. Frontend — übrige Komponenten

- [x] 4.1 `GameEditModal.tsx`: `buildTeamDisplayNames` → `buildTeamShortNames`, Import anpassen
- [x] 4.2 `AdminTrainingsPage.tsx`: `buildTeamDisplayNames` → `buildTeamShortNames`, Import anpassen
- [x] 4.3 `TerminePage.tsx`: `buildTeamDisplayNames` → `buildTeamShortNames`, Import anpassen
- [x] 4.4 `ChatPage.tsx`: `buildTeamDisplayNames` → `buildTeamShortNames`, Import anpassen
