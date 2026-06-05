## 1. Datenbank

- [x] 1.1 Migration `005_dateiablage.up.sql` anlegen: Tabellen `file_folders` (id, name, parent_id FK nullable, created_by FK, created_at), `folder_permissions` (id, folder_id FK, principal_type CHECK('everyone','role','club_function','user'), principal_ref TEXT, can_read INTEGER, can_write INTEGER), `files` (id, folder_id FK, original_name, disk_name, size, mime_type, uploaded_by FK, created_at)
- [x] 1.2 Migration `005_dateiablage.down.sql` anlegen
- [x] 1.3 `make migrate-up` lokal ausführen und Schema prüfen

## 2. Backend — Berechtigungs-Engine

- [x] 2.1 Package `internal/files/` anlegen mit `Handler struct{ db *sql.DB; storageDir string }`
- [x] 2.2 Funktion `folderPath(db, folderID) []int` — liefert alle Vorfahren-IDs bis zur Wurzel
- [x] 2.3 Funktion `resolveAccess(db, claims, folderID) (canRead, canWrite bool)` — Union aller Permissions im Pfad
- [x] 2.4 Funktion `checkAntiEscalation(db, claims, folderID, newPerm)` — prüft ob neue Berechtigung ≤ eigene Rechte; Admin ausgenommen

## 3. Backend — Ordner-API

- [x] 3.1 `GET /api/folders` — Wurzelordner auflisten (nur sichtbare, d.h. can_read)
- [x] 3.2 `POST /api/folders` — Ordner anlegen; prüft can_write auf parent_id
- [x] 3.3 `GET /api/folders/:id/contents` — Unterordner + Dateien auflisten (can_read)
- [x] 3.4 `DELETE /api/folders/:id` — Ordner löschen (nur wenn leer, can_write)
- [x] 3.5 `GET /api/folders/:id/permissions` — ACL-Einträge des Ordners (can_write)
- [x] 3.6 `POST /api/folders/:id/permissions` — ACL-Eintrag anlegen mit Anti-Eskalations-Check
- [x] 3.7 `DELETE /api/folders/:id/permissions/:permId` — ACL-Eintrag entfernen (can_write)

## 4. Backend — Datei-API

- [x] 4.1 `POST /api/folders/:folderId/files` — Upload via multipart/form-data, max 50 MB, UUID-Dateiname auf Disk
- [x] 4.2 `GET /api/files/:id/download` — Datei streamen mit Content-Disposition-Header (can_read)
- [x] 4.3 `DELETE /api/files/:id` — DB-Eintrag und Datei auf Disk löschen (can_write)
- [x] 4.4 Alle Routen in `cmd/teamwerk/main.go` in der authenticated-Gruppe registrieren

## 5. Deployment-Vorbereitung

- [x] 5.1 `deploy/setup-vps.sh` um `mkdir -p /var/lib/teamwerk/files` ergänzen
- [x] 5.2 Env-Variable `FILES_DIR=/var/lib/teamwerk/files` in VPS-Env-Datei dokumentieren
- [x] 5.3 `storageDir` aus Env in Config-Struct laden

## 6. Frontend — Navigation

- [x] 6.1 Nav-Eintrag `{ to: '/dokumente', label: 'Dokumente', roles: [] }` in `AppShell.tsx` nach „Mein Profil" eintragen
- [x] 6.2 Route `/dokumente` in `App.tsx` registrieren

## 7. Frontend — Desktop-Layout

- [x] 7.1 `web/src/pages/DocumentsPage.tsx` anlegen: Zwei-Panel-Layout (Ordnerbaum links, Inhalt rechts)
- [x] 7.2 Ordnerbaum: rekursiv aufklappbar, aktiver Ordner hervorgehoben
- [x] 7.3 Breadcrumb-Pfad oberhalb der Dateiliste
- [x] 7.4 Buttons „↑ Hochladen" und „+ Neuer Ordner" (nur bei can_write) oben rechts
- [x] 7.5 Dateiliste als `<table>` mit Spalten: Name, Typ, Größe, Datum, Uploader, Aktionen
- [x] 7.6 Zeilen-Aktionen: Download-Icon; ⋮-Dropdown mit „Löschen" und „Berechtigungen" (nur bei can_write)

## 8. Frontend — Mobile-Layout

- [x] 8.1 Sticky Breadcrumb `sticky top-0 z-10` oben
- [x] 8.2 Ordner als Cards (kein Sidebar-Baum); Antippen navigiert in den Ordner
- [x] 8.3 Dateien als Cards: Name, Typ, Größe, Datum, Uploader, ⋮-Dropdown
- [x] 8.4 Buttons `py-2.5` (min. 44px Touch-Target), nebeneinander unter dem Breadcrumb

## 9. Frontend — Modals

- [x] 9.1 Upload-Modal: Datei-Picker, Fortschrittsbalken, Erfolgs-/Fehlermeldung
- [x] 9.2 Neuer-Ordner-Modal: Eingabefeld für Name, Speichern-Button
- [x] 9.3 Berechtigungs-Modal: Liste bestehender ACL-Einträge, Formular zum Hinzufügen (Principal-Typ Dropdown, Referenz-Feld, Lesen/Schreiben Checkboxen), Entfernen-Button pro Eintrag
