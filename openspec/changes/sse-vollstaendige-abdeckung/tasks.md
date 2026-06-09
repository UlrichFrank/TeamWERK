## 1. Backend: kader/handler.go – Hub-Integration

- [x] 1.1 `kader/handler.go`: `hub *hub.EventHub`-Feld zur `Handler`-Struct hinzufügen und `NewHandler(db *sql.DB, h *hub.EventHub)` Signatur anpassen
- [x] 1.2 `cmd/teamwerk/main.go`: `kader.NewHandler(database)` → `kader.NewHandler(database, hub)` aktualisieren
- [x] 1.3 `kader/handler.go` `UpdateKader`: `h.hub.Broadcast("kader")` nach erfolgreicher DB-Operation hinzufügen
- [x] 1.4 `kader/handler.go` `InitializeKader` / `createSingleKader`: `h.hub.Broadcast("kader")` hinzufügen
- [x] 1.5 `kader/handler.go` `DeleteKader`: `h.hub.Broadcast("kader")` hinzufügen
- [x] 1.6 `kader/handler.go` `CopyFromSeason`: genau ein `h.hub.Broadcast("kader")` am Ende der gesamten Operation hinzufügen
- [x] 1.7 `kader/handler.go` `AutoAssign`: genau ein `h.hub.Broadcast("kader")` am Ende hinzufügen
- [x] 1.8 `kader/handler.go` `PatchGamesPerSeason`: `h.hub.Broadcast("kader")` hinzufügen

## 2. Backend: members/handler.go – fehlende Broadcasts

- [x] 2.1 `LinkUser`: `h.hub.Broadcast("members")` nach erfolgreicher DB-Operation hinzufügen
- [x] 2.2 `CreateFamilyLink`: `h.hub.Broadcast("members")` hinzufügen
- [x] 2.3 `DeleteFamilyLink`: `h.hub.Broadcast("members")` hinzufügen
- [x] 2.4 `UpdateProfile`: `h.hub.Broadcast("members")` hinzufügen
- [x] 2.5 `AddPhone`: `h.hub.Broadcast("members")` hinzufügen
- [x] 2.6 `UpdatePhone`: `h.hub.Broadcast("members")` hinzufügen
- [x] 2.7 `DeletePhone`: `h.hub.Broadcast("members")` hinzufügen
- [x] 2.8 `UpdateVehicle`: `h.hub.Broadcast("members")` hinzufügen
- [x] 2.9 `UpdateChildAccount`: `h.hub.Broadcast("members")` hinzufügen (bereits vorhanden in UpdateChildMember – prüfen ob doppelt)
- [x] 2.10 `UpdateChildBank`: `h.hub.Broadcast("members")` hinzufügen

## 3. Backend: games/handler.go – Template-Broadcasts

- [x] 3.1 `CreateTemplate`: `h.hub.Broadcast("games")` nach erfolgreicher DB-Operation hinzufügen
- [x] 3.2 `UpdateTemplate`: `h.hub.Broadcast("games")` hinzufügen
- [x] 3.3 `DeleteTemplate`: `h.hub.Broadcast("games")` hinzufügen

## 4. Frontend: Dashboard

- [x] 4.1 `DashboardPage.tsx`: `useLiveUpdates`-Callback um `"games"`, `"trainings"`, `"duties"`, `"absences"` erweitern (silent reload mit `load(true)`)

## 5. Frontend: Admin-Trainings

- [x] 5.1 `AdminTrainingsPage.tsx`: `useLiveUpdates` importieren und bei `"trainings"`-Event `loadSeries()` und `loadStandalone()` aufrufen

## 6. Frontend: Admin-Einstellungen

- [x] 6.1 `AdminSettingsPage.tsx`: `useLiveUpdates` importieren und bei `"settings"`-Event die Club-Daten (`/club`), Saisons (`load()`) und Altersklassen-Regeln (`/age-class-rules`) still neu laden

## 7. Frontend: Mitglieds-Detail

- [x] 7.1 `MemberDetailPage.tsx`: `useLiveUpdates` importieren und bei `"members"`-Event die Mitgliedsdaten des aktuell angezeigten Mitglieds still neu laden (vorhandene load-Logik im `useEffect` referenzieren)

## 8. Frontend: Admin-Kader

- [x] 8.1 `AdminKaderPage.tsx`: `useLiveUpdates` importieren und bei `"kader"`-Event `loadKader(selectedSeason.id)` aufrufen (Guard: nur wenn `selectedSeason` vorhanden)

## 9. Frontend: Mein-Team

- [x] 9.1 `MeinTeamPage.tsx`: `useLiveUpdates` importieren und bei `"members"` oder `"kader"`-Event die Team-Daten still neu laden

## 10. Frontend: Admin-Benutzer

- [x] 10.1 `AdminUsersPage.tsx`: `useLiveUpdates` importieren und bei `"members"`-Event die Seite still neu laden (Einladungen, Anfragen, User-Liste)

## 11. Frontend: Mitgliedschaftsanfragen

- [x] 11.1 `MembershipRequestsPage.tsx`: `useLiveUpdates` importieren und bei `"members"`-Event `load()` aufrufen

## 12. Frontend: Admin-Dienst-Typen

- [x] 12.1 `AdminDutyTypesPage.tsx`: `useLiveUpdates` importieren und bei `"duties"`-Event `load()` aufrufen

## 13. Frontend: Admin-Duty-Templates (Liste)

- [x] 13.1 `AdminDutyTemplatesPage.tsx`: `useLiveUpdates` importieren und bei `"games"`-Event die Template-Liste still neu laden

## 14. Frontend: Admin-Duty-Template-Detail

- [x] 14.1 `AdminDutyTemplateDetailPage.tsx`: `useLiveUpdates` importieren und bei `"games"`-Event das aktuelle Template still neu laden

## 15. Verifikation

- [x] 15.1 `go build ./...` – keine Kompilierfehler
- [ ] 15.2 Manuelle Prüfung: Kader-Mutation auf AdminKaderPage in Tab A → AdminKaderPage in Tab B aktualisiert sich
- [ ] 15.3 Manuelle Prüfung: Dashboard zeigt aktualisierte Daten nach Game-/Training-Änderung ohne Reload
- [ ] 15.4 Manuelle Prüfung: Familienlink anlegen → MembersPage in Tab B aktualisiert sich
