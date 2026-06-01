## Why

Überall wo Personen in der App erscheinen (Dienst-Assignees, Trainer, Spieler) sieht man bisher nur einen Namen — ohne Möglichkeit, schnell Kontakt aufzunehmen. Das zentrale Datenschutz-Modell (`user_visibility`) ist bereits vorhanden, wurde aber bisher nur für die Dienstbörse genutzt.

## What Changes

- Eine geteilte `PersonChip`-Komponente ersetzt alle bisherigen Personen-Anzeigen in der App
- Hover (Desktop) oder Tap (Mobile) öffnet einen Tooltip mit Kontaktdaten gemäß individuellen Freigaben
- Kontaktdaten werden **lazy** per neuem Endpoint `GET /api/users/:id/contact` abgerufen — kein Vorladen, kein Overhead im Board-Response
- Ein session-scoped React Context cached die Daten pro `user_id`, verhindert Doppel-Requests und wird bei Logout geleert
- Personen ohne Nutzer-Account (Member ohne Login) degradieren zu Plain-Text ohne Tooltip
- Gilt für: Duty-Board-Assignees, Kader-Trainer, Mitglieder-Liste

## Capabilities

### New Capabilities
- `person-contact`: Lazy-fetching Kontaktdaten einer Person per `user_id`, privacy-gefiltert, gecacht per Session

### Modified Capabilities
- `duty-assignee-display`: `AssigneeChip` wird durch `PersonChip` ersetzt; Board-Response liefert nur noch `user_id + name + photo_url` (keine Phones/Address mehr inline)
- `duties`: `boardSlot.assignees[]` vereinfacht zu `[{ user_id, name, photo_url? }]`
- `kader`: `trainerRow` um `user_id *int` erweitert (LEFT JOIN auf `users`)

## Impact

### Backend
- `cmd/teamwerk/main.go`: neue Route `GET /api/users/{id}/contact` registrieren (auth-protected)
- `internal/members/handler.go` oder neuer `internal/users/handler.go`: `GetContact`-Handler mit privacy-gefiltertem SQL
- `internal/duties/handler.go`: `publicAssignee`-Struct vereinfachen (`UserID` hinzufügen, `Phones`/`Address` entfernen)
- `internal/kader/handler.go`: `loadTrainers()` um LEFT JOIN auf `users` erweitern → `user_id` Optional

### Frontend
- `web/src/contexts/PersonContactContext.tsx`: neuer Context + `usePersonContact()`-Hook; Cache `Map<userId, PersonContact | 'loading' | 'error'>`; bei Logout leeren
- `web/src/components/PersonChip.tsx`: neue geteilte Komponente; Props `{ userId: number, name: string, photoUrl?: string }`
- `web/src/components/DutySlotList.tsx`: `PublicAssignee`-Interface um `user_id` erweitern, `Phones`/`Address` entfernen; `AssigneeChip` durch `PersonChip` ersetzen
- `web/src/pages/AdminKaderPage.tsx`: Trainer-Chips auf `PersonChip` umstellen
- `web/src/pages/MembersPage.tsx`: Mitglieder-Namen auf `PersonChip` umstellen (wo `user_id` vorhanden)
- Keine neuen DB-Tabellen, keine DB-Migrations
