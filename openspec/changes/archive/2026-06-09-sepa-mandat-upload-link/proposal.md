## Why

Der Upload-Button für SEPA-Mandat-Dokumente ist ein Stub — der API-Aufruf wurde nie implementiert. Dazu kommt ein Sicherheitsproblem: SEPA-Dokumente werden über `/api/uploads/*` ausgeliefert, einer **öffentlichen Route ohne Auth**. Jeder, der die UUID-URL kennt, kann das Dokument abrufen. Die URL wird ausserdem nur Admins zurückgegeben — Mitglieder, Eltern und Vorstand können ihr eigenes Mandat nicht sehen. Zusätzlich fehlt die Möglichkeit für Mitglieder, ein Mandat zurückzuziehen.

## What Changes

**Backend (neu):**
- `GET /api/members/{id}/sepa-mandat/download-token` — kurzlebiger HMAC-Token für Mitglied, Eltern, Vorstand, Admin
- `GET /api/members/{id}/sepa-mandat/download?token=...` — auth via Token, serviert die Datei (nicht über public `/api/uploads/`)
- `DELETE /api/members/{id}/sepa-mandat` — Mitglied (eigenes), Vorstand, Admin können Dokument löschen

**Backend (geändert):**
- `GET /api/members/{id}`: `sepa_mandat_url` auch für Mitglied (isOwn), Elternteil (family_link) und Vorstand zurückgeben — nicht nur Admin
- `sepa_mandat_url` zeigt künftig auf `/api/members/{id}/sepa-mandat/download` (token-geschützt), nicht auf `/api/uploads/...`

**Frontend:**
- `MemberDatenschutzTab` erhält `memberId: number` als Prop
- Upload-Logik fertigstellen (API-Aufruf war Stub)
- Öffnen-Link via Token-Flow (wie DocumentsPage), funktioniert auf iOS
- Löschen-Button für Mitglied (isOwn), Eltern, Vorstand, Admin
- `DocumentsPage`: iOS-Bug in `openFile` behoben (Fenster synchron vor async öffnen)

## Capabilities

### New Capabilities

- `sepa-mandat-upload`: Tatsächlicher Datei-Upload via API, auth-geschützter Download via Token, Löschen-Funktion, korrekte Berechtigungen

### Modified Capabilities

*(keine Spec-Level-Änderungen an bestehenden Capabilities — API-Verhalten von `GET /api/members/{id}` ändert sich, aber kein eigener Spec-Eintrag)*

## Impact

- `internal/upload/handler.go` — 3 neue Endpoints + Token-Logik
- `internal/members/handler.go` — `sepa_mandat_url` für mehr Rollen sichtbar, URL auf neuen Endpoint
- `web/src/components/admin/MemberDatenschutzTab.tsx` — Upload, Token-Download, Löschen
- `web/src/pages/MemberDetailPage.tsx` — `memberId`-Prop übergeben
- `web/src/pages/DocumentsPage.tsx` — iOS-Bug-Fix in `openFile`
- Keine neuen Dependencies, keine Migrations
