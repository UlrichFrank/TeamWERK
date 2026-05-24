## Why

Die Navigationssichtbarkeit in AppShell ist für die Trainer-Rolle falsch konfiguriert: „Mein Profil" fehlt, „Mitglieder" ist sichtbar (obwohl Trainer den Kader nutzen sollen), und die Kader-Verwaltung ist für Trainer gesperrt. Das führt dazu, dass Trainer keinen sinnvollen Einstieg in ihre Kernaufgaben haben.

## What Changes

- **„Mein Profil"** wird für alle Rollen außer `admin` sichtbar (neu: `excludeRoles`-Mechanismus in AppShell)
- **„Mitglieder"** wird auf `admin` und `vorstand` eingeschränkt — Trainer nicht mehr berechtigt
- **„Kader"** wird für `trainer` freigeschaltet (Navigation + Backend-Middleware)
- Backend: Kader-API-Routen erhalten `trainer` in der `RequireRole`-Liste

## Capabilities

### New Capabilities
- `nav-exclude-roles`: Neue `excludeRoles`-Eigenschaft für Nav-Items, die bestimmte Rollen ausschließt statt einer Whitelist

### Modified Capabilities
- `nav-visibility`: Sichtbarkeitsregeln der Navigation für Profil, Mitglieder und Kader ändern sich

## Impact

- `web/src/components/AppShell.tsx`: NavItem-Typ, Filter-Logik, Nav-Konfiguration
- `cmd/teamwerk/main.go`: `RequireRole` bei Kader-Routen
