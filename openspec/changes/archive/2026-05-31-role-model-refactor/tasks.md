## 1. Datenbank-Migration

- [x] 1.1 Neue Migration-Datei anlegen (`00N_role_model_refactor.up.sql` / `.down.sql`)
- [x] 1.2 Junction-Tabelle `member_club_functions` erstellen (member_id FK, function CHECK, PRIMARY KEY)
- [x] 1.3 Bestehende `members.club_function`-Werte in `member_club_functions` migrieren
- [x] 1.4 `users`-Tabelle rekreieren: CHECK-Constraint auf `('admin','standard')`, alle Nicht-Admin-Rollen auf `'standard'` setzen
- [x] 1.5 `invitation_tokens`-Tabelle rekreieren: `target_role` CHECK auf `('admin','standard')`
- [x] 1.6 `members.club_function`-Spalte entfernen (Tabelle rekreieren)
- [x] 1.7 Down-Migration vollständig implementieren (alle Schritte umkehren)
- [x] 1.8 Migration lokal testen: `make migrate-up` und `make migrate-down`

## 2. Backend — Auth (JWT & Middleware)

- [x] 2.1 `Claims`-Struct in `internal/auth/tokens.go` erweitern: `ClubFunctions []string`, `IsParent bool`
- [x] 2.2 `IssueAccessToken` anpassen: Login-Query befüllt `ClubFunctions` aus `member_club_functions` und `IsParent` aus `family_links`
- [x] 2.3 Hilfsmethode `(c *Claims) HasFunction(f string) bool` implementieren
- [x] 2.4 `RequireRole`-Middleware bleibt für `admin`-Guards; neue Middleware `RequireClubFunction(f ...string)` hinzufügen
- [x] 2.5 `roleRank`-Map in `auth/handler.go` entfernen; Admin-Invite-Guard ersetzen durch: nur `admin` darf `admin` einladen
- [x] 2.6 E-Mail-Benachrichtigung bei Beitrittsantrag: Query `WHERE role IN ('trainer','admin')` → `JOIN member_club_functions` für Trainer-Lookup

## 3. Backend — Handler-Anpassungen

- [x] 3.1 `dashboard/handler.go`: `teamQueryForUser(role string)` → `teamQueryForUser(clubFunctions []string, isParent bool)`; `switch role` durch additive Checks ersetzen
- [x] 3.2 `dashboard/handler.go`: `effectivePersona(clubFunctions []string, isParent bool) string` implementieren (Priorität: trainer > spieler > elternteil)
- [x] 3.3 `dashboard/handler.go`: alle `role == "elternteil"` → `claims.IsParent`; alle `role == "trainer"` → `claims.HasFunction("trainer")`
- [x] 3.4 `members/handler.go`: `claims.Role == "elternteil"` → `claims.IsParent`; `claims.Role == "trainer"/"vorstand"` → `claims.HasFunction(...)`
- [x] 3.5 `members/handler.go`: Query-Parameter `?club_function=` → JOIN auf `member_club_functions` statt WHERE-Klausel auf Spalte
- [x] 3.6 `members/handler.go`: Member-Create/Update-API akzeptiert `club_functions []string`; schreibt in Junction-Tabelle (DELETE alle + INSERT neu)
- [x] 3.7 `members/handler.go`: Member-Read-Query lädt `club_functions` per `GROUP_CONCAT` oder separater Query; gibt `[]string` zurück
- [x] 3.8 `games/handler.go`: alle `claims.Role == "trainer"` → `claims.HasFunction("trainer")`
- [x] 3.9 `main.go`: `RequireRole("trainer", "admin")` → `RequireClubFunction("trainer")` kombiniert mit `auth.Middleware`; alle Route-Gruppen prüfen

## 4. Frontend — AuthContext & API-Types

- [x] 4.1 `AuthContext.tsx`: `user`-Typ erweitern um `clubFunctions: string[]` und `isParent: boolean`
- [x] 4.2 JWT-Parsing im AuthContext: neue Felder aus Token-Payload lesen
- [x] 4.3 Hilfsfunktion `hasFunction(user, f: string): boolean` im AuthContext oder als Utility exportieren

## 5. Frontend — Routing & Navigation

- [x] 5.1 `App.tsx`: `RoleRoute` auf neues Modell umstellen — `roles=['admin','vorstand','trainer']` → admin-Check + `hasFunction`-Check
- [x] 5.2 `AppShell.tsx`: Nav-Sichtbarkeit für Kader-Eintrag auf `hasFunction('trainer')` oder `admin` umstellen
- [x] 5.3 `App.tsx` / `AppShell.tsx`: alle verbleibenden Referenzen auf `'elternteil'`, `'spieler'`, `'vorstand'`, `'trainer'` als Rolle bereinigen

## 6. Frontend — Seiten & Komponenten

- [x] 6.1 `AdminUsersPage.tsx`: `ALL_ROLES`, `ROLE_LABELS`, `roleRank` auf `['admin','standard']` reduzieren; Invite-Formular vereinfachen
- [x] 6.2 `MemberStammdatenTab.tsx`: `<select club_function>` → Checkbox-Gruppe für Mehrfachauswahl der Vereinsfunktionen
- [x] 6.3 `MemberStammdatenTab.tsx`: `isSpieler`-Flag → `hasSpieler = clubFunctions.includes('spieler')`
- [x] 6.4 `MembersPage.tsx`: `clubFunctionFilter` bleibt, aber sendet Array-fähigen Filter-Parameter
- [x] 6.5 `DashboardPage.tsx`: alle `role === 'elternteil'` → `user.isParent`; alle `role === 'trainer'` → `hasFunction(user,'trainer')`
- [x] 6.6 `AdminDutyTemplatesPage.tsx`: `target_role`-Dropdown auf verbleibende Werte prüfen (elternteil/spieler/trainer bleiben als Dienstpflicht-Kategorien)

## 7. Abschluss

- [ ] 7.1 Lokaler End-to-End-Test: Login als Admin, Standard-Nutzer, Trainer, Elternteil
- [ ] 7.2 Sicherstellen dass `make build` fehlerfrei durchläuft
- [ ] 7.3 `make deploy` auf VPS ausführen (migrate up läuft automatisch)
