## Why

Als Admin ist es aufwändig, das UI-Verhalten für verschiedene Rollen und Vereinsfunktionen zu testen, weil dafür bisher separate Logins nötig sind. Eine Impersonation-Funktion ermöglicht schnelles Umschalten in die Sicht eines beliebigen Users — ohne Passwort, ohne Tab-Wechsel.

## What Changes

- Neuer Backend-Endpoint `POST /api/admin/impersonate/{userId}` gibt ein kurzlebiges JWT mit den Claims des Ziel-Users zurück
- Der Admin-Refresh-Cookie bleibt unverändert — "Beenden" ist ein normaler `/auth/refresh`-Call
- AuthContext bekommt `impersonating`-State sowie `startImpersonation` / `stopImpersonation`
- AppShell zeigt einen gelben Banner während aktiver Impersonation mit "Beenden"-Button
- AdminUsersPage erhält pro User-Zeile einen "Testen als"-Button (nur für Admins, nicht für sich selbst)

## Capabilities

### New Capabilities
- `admin-impersonation`: Admin kann in der Nutzerverwaltung einen User auswählen und dessen JWT-Claims (role, club_functions, is_parent) temporär übernehmen; alle API-Calls laufen dann authentisch mit diesem Token; Rückkehr zur Admin-Session via Refresh-Cookie

### Modified Capabilities

## Impact

- `internal/auth/handler.go`: neuer Handler `Impersonate`
- `cmd/teamwerk/main.go`: neue Route `POST /api/admin/impersonate/{userId}`
- `web/src/contexts/AuthContext.tsx`: `impersonating` State + zwei neue Funktionen
- `web/src/components/AppShell.tsx`: ImpersonationBanner-Komponente
- `web/src/pages/AdminUsersPage.tsx`: "Testen als"-Button in Desktop-Tabelle und Mobile-Cards
- Keine neuen DB-Tabellen, keine neuen Abhängigkeiten
