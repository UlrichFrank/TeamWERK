## Why

Trainers können `/mitglieder` und `/mitglieder/:id` durch direkte URL-Eingabe aufrufen, obwohl der Nav-Link bereits ausgeblendet ist. Der Frontend-Route-Guard hat `trainer` in der Rollenliste, und die Backend-API `GET /api/members` ist für alle authentifizierten Nutzer offen — beide Lücken müssen geschlossen werden.

## What Changes

- **Frontend**: `trainer` aus den `RoleRoute`-Rollen für `/mitglieder` und `/mitglieder/:id` in `App.tsx` entfernen
- **Backend**: `GET /api/members` und `GET /api/members/{id}` aus der allgemeinen Authenticated-Gruppe in die `RequireClubFunction("vorstand")`-Gruppe verschieben

## Capabilities

### New Capabilities

_Keine neuen Capabilities._

### Modified Capabilities

- `members`: Das Szenario „Teamleiter sieht nur eigene Teammitglieder" wird entfernt — Trainer haben keinen Lesezugriff mehr auf die Mitgliederliste und -details. Stattdessen: explizites Szenario, dass `GET /api/members` und `GET /api/members/{id}` ohne vorstand/admin-Funktion mit 403 antworten.

## Impact

- `web/src/App.tsx`: Zeilen 81–82, `roles`-Array der beiden Mitglieder-Routen
- `cmd/teamwerk/main.go`: Zeilen 131–132, Verschiebung von `GET /api/members` und `GET /api/members/{id}` in die vorstand-Gruppe
- Kader-Feature bleibt unberührt — Trainer greifen auf Mitgliedersuche über `/api/admin/kader/{id}/member-suggestions` zu, das bereits in der `RequireClubFunction("vorstand","trainer")` Gruppe liegt
- `AdminUsersPage` ruft ebenfalls `GET /api/members` auf — diese Seite ist bereits auf vorstand/admin eingeschränkt, kein Regressionspotenzial
