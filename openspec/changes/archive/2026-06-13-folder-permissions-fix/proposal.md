## Why

Die Dokumentenverwaltung (`/dokumente`) hat einen Sicherheitsfehler in der Berechtigungsauflösung: Berechtigungen aller Vorfahren-Ordner werden additiv vereint (OR-Logik), sodass eine Einschränkung in einem Unterordner durch eine weiträumigere Freigabe im Elternordner umgangen wird. Ein Standard-Nutzer, der einen Ordner mit `everyone: read` im Elternordner hat, kann dadurch auch geschützte Unterordner lesen.

## What Changes

- **Nearest-Ancestor-Wins:** `resolveAccess()` iteriert statt eines einzigen Multi-Ordner-Querys von Zielordner Richtung Root und stoppt beim ersten Ordner, der eigene Berechtigungen definiert. Dessen Regeln gelten exklusiv; Vorfahren-Regeln werden ignoriert.
- **Family-Context-Erweiterung:** Nutzer mit einer `family_links`-Verknüpfung erhalten bei der Berechtigungsprüfung auch die Vereinsfunktionen und User-IDs ihrer verknüpften Kinder berücksichtigt. Gilt für `principal_type=club_function` und `principal_type=user`.
- **Benutzerfreundliche User-Anzeige:** `GET /api/folders/{id}/permissions` liefert für `principal_type=user`-Einträge zusätzlich den Anzeigenamen des Nutzers. Im Frontend wird statt der rohen User-ID der Name angezeigt; die Eingabe erfolgt über ein Dropdown mit allen Nutzern statt einem ID-Textfeld.

## Capabilities

### New Capabilities

- `folder-permission-resolution`: Korrektes Berechtigungsmodell für Ordnerhierarchien — Nearest-Ancestor-Wins statt additiver Vererbung, mit Family-Context-Unterstützung für Elternteil-Nutzer.
- `folder-permission-ux`: Benutzerfreundliche Darstellung von User-Berechtigungen — Anzeigename statt ID, Nutzer-Dropdown statt Freitextfeld.

### Modified Capabilities

## Impact

- `internal/files/handler.go`: `resolveAccess()`, `folderPath()`, `permResponse`, `ListPermissions()`
- `web/src/pages/DocumentsPage.tsx`: `PermissionsModal` — Anzeige und Eingabe für `user`-Typ
- Keine Datenbankmigrationen
- Alle bestehenden Routen unter `/api/folders/*` und `/api/files/*` betroffen (nutzen `resolveAccess`)
- Tests für `resolveAccess` müssen neu geschrieben werden
