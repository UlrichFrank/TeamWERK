## Why

TeamWERK hat keine Möglichkeit, Vereinsdokumente zentral abzulegen und gezielt freizugeben. Mitglieder, Trainer und Vorstand tauschen Dokumente aktuell über externe Kanäle aus — ohne Zugangskontrolle und ohne Übersicht. Eine flexible, selbst verwaltbare Ordnerstruktur mit feingranularen Berechtigungen schließt diese Lücke.

## What Changes

- Neuer Nav-Eintrag „Dokumente" im Modul „Mitglieder" (alle eingeloggten Nutzer)
- Frontend: Datei-Browser mit dynamischer Ordnernavigation, Upload und Berechtigungsverwaltung
- Backend: Hierarchische Ordnerstruktur (beliebige Tiefe, Vererbung additiv)
- Berechtigungssystem per Ordner: Vergabe an Gruppe „alle", Rolle, Vereinsfunktion oder Einzelperson
- Anti-Eskalation: Nutzer können nur Rechte vergeben, die sie selbst besitzen
- Storage auf dem VPS-Dateisystem; Metadaten und ACLs in SQLite

## Capabilities

### New Capabilities

- `file-storage`: Dateien hochladen, herunterladen und verwalten; Storage auf VPS-Dateisystem mit Metadaten in SQLite
- `file-folders`: Dynamische Ordnerstruktur mit beliebiger Tiefe und additiver Berechtigungsvererbung
- `file-permissions`: ACL-basierte Zugangskontrolle pro Ordner — Vergabe an everyone, Rolle, Vereinsfunktion oder Einzeluser; Anti-Eskalation

### Modified Capabilities

*(keine)*

## Impact

- Neues Package `internal/files/` (Handler, ACL-Auflösung)
- Neue DB-Migration: Tabellen `file_folders`, `folder_permissions`, `files`
- Neue API-Routen unter `/api/files/` und `/api/folders/`
- Neue Frontend-Seite `web/src/pages/DocumentsPage.tsx`
- Nav-Eintrag in `AppShell.tsx` unter Modul „Mitglieder", `roles: []`
- Kein neuer externer Dienst; Storage lokal auf VPS (`/var/lib/teamwerk/files/`)
