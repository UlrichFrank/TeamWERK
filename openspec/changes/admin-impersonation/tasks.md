## 1. Backend: Impersonation-Endpoint

- [x] 1.1 `Impersonate`-Handler in `internal/auth/handler.go` implementieren: User-Claims aus DB laden, Admin-und Selbst-Impersonation ablehnen (400), JWT via `IssueAccessToken` ausstellen, `{ access_token, user: { id, name } }` zurückgeben
- [x] 1.2 Route `POST /api/admin/impersonate/{userId}` in `cmd/teamwerk/main.go` unter `RequireRole("admin")` registrieren

## 2. Frontend: AuthContext erweitern

- [x] 2.1 `impersonating: { userId: number; name: string } | null` State zum AuthContext hinzufügen
- [x] 2.2 `startImpersonation(userId: number, name: string)` implementieren: `POST /api/admin/impersonate/{userId}` aufrufen, `setAccessToken` mit neuem Token, `user` State aus JWT-Payload aktualisieren, `impersonating` setzen
- [x] 2.3 `stopImpersonation()` implementieren: `POST /api/auth/refresh` aufrufen, `setAccessToken` mit Admin-Token, `user` State aktualisieren, `impersonating = null`
- [x] 2.4 `startImpersonation` und `stopImpersonation` im Context-Value und Interface exportieren

## 3. Frontend: ImpersonationBanner in AppShell

- [x] 3.1 `ImpersonationBanner`-Komponente in `AppShell.tsx` erstellen: gelber Streifen mit Name des impersonierten Users und "Beenden"-Button (`stopImpersonation` aufrufen), nur sichtbar wenn `impersonating != null`
- [x] 3.2 Banner im Layout oberhalb des `<main>`-Bereichs einbinden (innerhalb des `flex-1`-Containers, nach dem Mobile-Header)

## 4. Frontend: "Testen als"-Button in AdminUsersPage

- [x] 4.1 `startImpersonation` aus AuthContext in `AdminUsersPage.tsx` einbinden
- [x] 4.2 "Testen als"-Button in der Desktop-Tabelle pro User-Zeile hinzufügen (nur wenn `self?.role === 'admin' && u.id !== self?.id && u.role !== 'admin'`)
- [x] 4.3 "Testen als"-Eintrag im `actions`-Array der Mobile-Cards hinzufügen (gleiche Bedingung)
