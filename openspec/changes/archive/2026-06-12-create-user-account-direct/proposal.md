## Why

Bislang können Vorstand und Admin neue Nutzer nur über den Einladungs-Flow anlegen, der eine funktionierende E-Mail-Zustellung voraussetzt. Für Fälle, in denen Zugangsdaten direkt übergeben werden sollen (z.B. Ersteinrichtung, Vor-Ort-Onboarding), fehlt eine direkte Account-Anlage.

## What Changes

- Neuer Backend-Endpunkt `POST /api/users` zum direkten Anlegen eines login-fähigen Accounts (email, first_name, last_name, password als bcrypt-Hash)
- Neuer Dropdown-Eintrag „Account anlegen" auf `/admin/nutzer` neben „CSV importieren"
- Neues Modal mit E-Mail, Vorname, Nachname und automatisch generiertem Passwort (Frontend-seitig, ~16 Zeichen) mit Copy-Button

## Capabilities

### New Capabilities

- `direct-user-creation`: Vorstand und Admin können einen vollständigen, sofort login-fähigen Nutzeraccount direkt anlegen — ohne Einladungs-E-Mail. Das generierte Passwort wird im Modal angezeigt und kann per Button in die Zwischenablage kopiert werden.

### Modified Capabilities

<!-- keine bestehenden Specs betroffen -->

## Impact

- **Backend:** `internal/auth/handler.go` — neuer Handler `CreateUser`; Route in `cmd/teamwerk/main.go` (Vorstand-Gruppe)
- **Frontend:** `web/src/pages/AdminUsersPage.tsx` — erweitertes Dropdown, neues Modal
- **Keine neuen Dependencies**, keine Datenbankmigrationen (nutzt vorhandene `users`-Tabelle)
- **Zugriff:** Vorstand-Routegroup (schließt admin implizit ein)
