## Context

TeamWERK läuft auf einem IONOS VPS (1 GB RAM, Linux XS). Das Auth-System trägt bereits `ClubFunctions[]` im JWT und `HasFunction()` in der Middleware. Ordnerstruktur und Berechtigungen sollen vollständig durch Nutzer verwaltbar sein — ohne hartkodierten Folder-Typ. Berechtigungen vererben sich additiv vom Wurzel- zum Blattordner (kein DENY möglich).

## Goals / Non-Goals

**Goals:**
- Dynamische Ordnerhierarchie (beliebige Tiefe, user-defined)
- ACL pro Ordner: Vergabe an `everyone`, Rolle, Vereinsfunktion oder Einzeluser
- Additive Vererbung: Kind erbt alle Rechte der Elternkette (Union)
- Anti-Eskalation: Nutzer können nur Rechte vergeben ≤ ihrer eigenen
- Upload, Download, Löschen mit Berechtigungsprüfung
- Desktop: Zwei-Panel-Layout (Ordnerbaum + Dateiliste)
- Mobile: Card-Layout, Breadcrumb-Navigation, ⋮-Dropdown

**Non-Goals:**
- DENY-Overrides (Unterordner kann Eltern-Rechte nicht entziehen)
- Volltextsuche in Dokumenten
- Datei-Versionierung
- Quota-Enforcement pro Nutzer
- Video-Streaming (separater Change)

## Decisions

### Datenmodell: Drei Tabellen

```sql
file_folders (
  id, name, parent_id FK nullable,  -- null = Wurzelordner
  created_by FK, created_at
)

folder_permissions (
  id, folder_id FK,
  principal_type CHECK('everyone','role','club_function','user'),
  principal_ref  TEXT,  -- null für 'everyone', sonst Rollenname/Funktion/user_id
  can_read  INTEGER NOT NULL DEFAULT 0,
  can_write INTEGER NOT NULL DEFAULT 0
)

files (
  id, folder_id FK, original_name, disk_name, size, mime_type,
  uploaded_by FK, created_at
)
```

**Warum drei Tabellen:** Saubere Trennung zwischen Struktur (folders), Berechtigungen (permissions) und Inhalt (files). Berechtigungen sind unabhängig von Dateioperationen änderbar.

### Berechtigungs-Auflösung: Pfad-Union

```
canRead(userID, folderID):
  path = [folderID] + alle Vorfahren bis zur Wurzel
  permissions = SELECT * FROM folder_permissions WHERE folder_id IN (path)
  RETURN any permission in permissions that matches user AND can_read = 1

Match-Logik (in Reihenfolge Spezifität):
  1. principal_type = 'user'          AND principal_ref = userID       → höchste Priorität
  2. principal_type = 'club_function' AND principal_ref IN user.funcs
  3. principal_type = 'role'          AND principal_ref = user.role
  4. principal_type = 'everyone'                                        → niedrigste Priorität
```

Additiv: Ein `can_read = 1` aus einem Vorfahren reicht. Kein DENY.

**Warum kein rekursiver SQL-CTE:** SQLite unterstützt rekursive CTEs (>= 3.8.3), aber die Pfad-Auflösung in Go ist einfacher zu testen und zu debuggen. Anzahl Ebenen ist in der Praxis gering (< 10).

### Anti-Eskalation beim Vergeben

Beim `POST /api/folders/:id/permissions` prüft der Handler:
- Rufer hat `can_write` auf den Ordner (eigene Auflösung)
- Jede neue Permission darf nur Rechte gewähren ≤ den Rechten des Rufers auf diesen Ordner
- Admin (`role = 'admin'`) ist ausgenommen — darf immer alles vergeben

### Storage: Lokales Dateisystem, UUID-Dateiname

Dateien liegen unter `/var/lib/teamwerk/files/<uuid>.<ext>`. Der Original-Name lebt nur in der DB. Download setzt `Content-Disposition: attachment; filename="<original_name>"`.

**Warum flaches Verzeichnis statt Ordner-Hierarchie auf Disk:** Vereinfacht Umzüge von Ordnern (kein `mv`). Berechtigungen werden ausschließlich in der DB gehalten.

### Navigation: Modul „Mitglieder"

```ts
{ to: '/dokumente', label: 'Dokumente', roles: [] }
```

Eintrag direkt nach `/profil` im NavModule „Mitglieder".

### Frontend-Layout

**Desktop (≥ 640px):**
- Zwei-Panel: linke Spalte Ordnerbaum (aufklappbar), rechte Spalte Dateiliste als `<table>`
- Breadcrumb oben für aktuellen Pfad
- Primary-Buttons oben rechts: „↑ Hochladen", „+ Neuer Ordner" (nur bei can_write)
- Zeilen-Aktionen: Download-Icon, ⋮-Dropdown (Löschen, Berechtigungen)

**Mobile (< 640px):**
- Kein Ordnerbaum-Sidebar — Navigation durch Antippen von Ordner-Cards
- Breadcrumb sticky oben (`sticky top-0 z-10`)
- Buttons nebeneinander, `py-2.5` (min. 44px Touch-Target)
- Datei-Cards statt Tabelle: Name, Typ, Größe, Datum, Uploader, ⋮-Dropdown
- Berechtigungs-Modal als volles Modal (nicht Inline)

## Risks / Trade-offs

- **Disk-Vollläufer** → Mitigation: Max 50 MB pro Upload; `df`-Check im Deployment
- **Pfad-Auflösung bei tiefer Hierarchie** → Mitigation: Go-seitig, Tiefe in Praxis < 10; bei Performance-Problemen CTE nachrüsten
- **VPS-Ausfall = Datenverlust** → Mitigation: Backup `/var/lib/teamwerk/files/` empfohlen (außerhalb dieses Changes)
- **Zirkuläre Eltern-Kind-Referenz** → Mitigation: Backend prüft bei `parent_id`-Änderung auf Zyklus

## Migration Plan

1. Migration `005_dateiablage.up.sql`: Tabellen `file_folders`, `folder_permissions`, `files`
2. VPS: `mkdir -p /var/lib/teamwerk/files` (in `setup-vps.sh` ergänzen)
3. Env-Variable `FILES_DIR=/var/lib/teamwerk/files` in VPS-Env-Datei
4. Rollback: `005_dateiablage.down.sql` + Verzeichnis löschen

## Open Questions

- Maximale Upload-Größe: 50 MB pro Datei — ausreichend für PDF/DOCX/Bilder?
- Wurzelordner: werden initial einige Standard-Ordner angelegt (z.B. „Allgemein"), oder startet man mit leerem Root?
