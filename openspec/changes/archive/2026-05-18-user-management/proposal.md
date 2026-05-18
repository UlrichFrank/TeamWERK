## Why

Die Admin-Oberfläche bietet derzeit nur einen Einladungsworkflow für neue Nutzer, aber keine Übersicht über bestehende Konten. Admins können weder sehen, welche Nutzer aktiv sind, noch einzelne Konten löschen — z. B. bei Vereinsaustritt oder fehlerhafter Registrierung.

## What Changes

- Neue Nutzerverwaltungs-Tabelle im Admin-Bereich zeigt alle registrierten Nutzer (E-Mail, Name, Rolle, zugehöriges Team)
- Admins können einzelne Nutzer löschen (inkl. Cleanup der zugehörigen Refresh-Tokens und Family-Links)
- Die bestehende Einladungsfunktion bleibt unverändert und wird in die neue Seite integriert

## Capabilities

### New Capabilities

- `user-list`: Tabellarische Übersicht aller Nutzer mit Filterung nach Rolle und Team, inkl. Einladungs-Button
- `user-delete`: Löschen eines Nutzerkontos durch Admins mit kaskadierten Datenbereinigungen

### Modified Capabilities

<!-- Keine bestehenden Specs vorhanden -->

## Impact

- **Backend:** Neuer `GET /api/admin/users` Endpunkt existiert bereits (laut CLAUDE.md) — muss ggf. erweitert werden. Neuer `DELETE /api/admin/users/{id}` Endpunkt erforderlich.
- **Frontend:** Neue Seite `web/src/pages/AdminUsers.tsx` ersetzt oder ergänzt den bestehenden Nutzer-Tab. Route und Nav-Eintrag in `App.tsx` / `AppShell.tsx` anpassen.
- **Datenbank:** Kein Schema-Change — kaskadierende Deletes via vorhandene Foreign-Key-Constraints (refresh_tokens, family_links, duty_assignments etc.).
- **Kein Risiko für RAM/externe Dienste.**
