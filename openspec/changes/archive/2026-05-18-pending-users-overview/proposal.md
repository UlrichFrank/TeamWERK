## Why

Die Nutzerverwaltung zeigt aktuell nur registrierte Konten. Offene Einladungen (versendete Token) und Beitrittsanfragen sind nur auf separaten Seiten sichtbar und können nicht aus dem Admin-Bereich entfernt werden. Admins wollen auf einen Blick sehen, wer eingeladen wurde, wer einen Antrag gestellt hat — und beides bereinigen können.

## What Changes

- Die bestehende Nutzertabelle wird um zwei weitere Eintragstypen erweitert: aktive Einladungen und offene Beitrittsanfragen — erkennbar am Status-Badge
- Einladungen können gelöscht werden (invalidiert den Link, keine Benachrichtigung nötig)
- Beitrittsanfragen können gelöscht werden (zusätzlich zur bestehenden Genehmigen/Ablehnen-Funktion)
- Zwei neue Backend-Endpunkte: `GET /api/admin/invitations` und `DELETE /api/admin/invitations/{id}`
- Neuer Endpunkt: `DELETE /api/admin/membership-requests/{id}`

## Capabilities

### New Capabilities

- `invitation-list`: Liste aktiver (ungenutzter, nicht abgelaufener) Einladungstoken für Admins
- `invitation-delete`: Admin kann eine versendete Einladung widerrufen (Token aus DB löschen)
- `membership-request-delete`: Admin kann eine Beitrittsanfrage löschen (nicht nur ablehnen)

### Modified Capabilities

## Impact

- **Backend:** `internal/auth/handler.go` — drei neue Handler, Routen in `main.go`
- **Frontend:** `AdminUsersPage.tsx` — Nutzertabelle um Einladungs- und Anfragezeilen erweitern
- **Datenbank:** Kein Schema-Change — `invitation_tokens` und `membership_requests` existieren bereits
