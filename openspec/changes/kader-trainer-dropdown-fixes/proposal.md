## Why

Die Kader-Seite hat drei Probleme: Trainer können einem Kader nicht zugewiesen werden. Dazu kommen zwei UI-Bugs beim Jahrgangs-Dropdown — Clipping durch `overflow-hidden` und ein Fokus-Problem beim erneuten Öffnen. Zusätzlich fehlt im Mitgliederprofil die Möglichkeit, eine Vereinsfunktion (Trainer, Vorstand, Vorstands-Beisitzer) zu hinterlegen. Diese Funktion ist bewusst von der System-Benutzerrolle (`users.role`) getrennt, da Berechtigungen und Vereinsfunktionen unabhängige Konzepte sind.

## What Changes

- **Neu**: Feld `club_function` auf `members`-Tabelle — nullable, Werte: `trainer`, `vorstand`, `vorstand_beisitzer`
- **Neu**: Vereinsfunktion in Mitgliederdetail und Mitgliederprofil anzeigbar und bearbeitbar
- **Neu**: Pro Kader-Karte können mehrere Trainer zugewiesen werden; Auswahl aus Mitgliedern mit `club_function = 'trainer'`
- **Neu**: DB-Migration 015 — Spalte `club_function` auf `members`
- **Neu**: DB-Migration 016 — Junction-Tabelle `kader_trainers (kader_id, member_id)`
- **Neu**: `GET /api/admin/kader` liefert `trainers: [{id, name}]`; `PUT /api/admin/kader/{id}` akzeptiert `trainers_add` / `trainers_remove` (member_id)
- **Fix**: `overflow-hidden` vom Kader-Karten-Container entfernt → Dropdown-Clipping behoben
- **Fix**: Fokus-Problem beim Jahrgangs-Dropdown behoben

## Capabilities

### New Capabilities

- `member-club-function`: Mitglieder können eine Vereinsfunktion (Trainer / Vorstand / Vorstands-Beisitzer) tragen, unabhängig von ihrer System-Benutzerrolle; die Funktion ist im Mitgliederprofil editierbar
- `kader-trainer-assignment`: Admin kann pro Kader-Eintrag beliebig viele Trainer (aus Mitgliedern mit `club_function = 'trainer'`) zuweisen und entfernen; zugewiesene Trainer werden als Chips angezeigt

### Modified Capabilities

*(keine bestehenden Specs betroffen)*

## Impact

- `internal/db/migrations/015_member_club_function.up/down.sql` (neu)
- `internal/db/migrations/016_kader_trainers.up/down.sql` (neu)
- `internal/members/handler.go`: `club_function` in Scan, Insert, Update und Response ergänzen
- `web/src/pages/MemberDetailPage.tsx`: Vereinsfunktion-Select anzeigen/bearbeiten
- `internal/kader/handler.go`: `kaderDetail` um `Trainers []trainerRow` erweitern; `UpdateKader` mit trainers_add/remove
- `web/src/pages/AdminKaderPage.tsx`: Trainer-Chips + Add-Select; `overflow-hidden` entfernen; Dropdown-Fix
- Keine API-Breaking-Changes (neue optionale Felder)
