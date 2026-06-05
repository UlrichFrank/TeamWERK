## Context

Der Nav-Link „Mitglieder" ist bereits für Trainer ausgeblendet (AppShell.tsx), aber der Route-Guard in App.tsx erlaubt Trainern weiterhin den Direktzugriff per URL. Zusätzlich sind `GET /api/members` und `GET /api/members/{id}` in der allgemeinen Authenticated-Gruppe registriert, d. h. jeder eingeloggte Nutzer kann die API direkt aufrufen.

Das vorhandene `members`-Spec-Szenario „Teamleiter sieht nur eigene Teammitglieder" impliziert noch einen eingeschränkten Lesezugriff für Trainer — dieser Lesepfad fällt mit diesem Change komplett weg.

## Goals / Non-Goals

**Goals:**
- Trainer werden auf der Frontend-Seite zuverlässig auf die Startseite weitergeleitet, wenn sie `/mitglieder` oder `/mitglieder/:id` direkt aufrufen
- Die Backend-API antwortet mit 403, wenn ein Nutzer ohne vorstand-/admin-Funktion `GET /api/members` oder `GET /api/members/{id}` aufruft

**Non-Goals:**
- Änderungen an Schreib-Endpunkten (`POST`, `PUT`, `DELETE /api/members/...`) — diese sind bereits korrekt eingeschränkt
- Änderungen an Kader-Endpunkten (`/api/admin/kader/...`) — Trainer-Zugriff dort bleibt unverändert

## Decisions

**Frontend: Route Guard**
`roles`-Array in `RoleRoute` für beide Mitglieder-Routen von `['admin','vorstand','trainer']` auf `['admin','vorstand']` reduzieren. `RoleRoute` redirectet bereits auf `/` wenn keine passende Rolle — kein zusätzlicher Mechanismus nötig.

**Backend: Gruppenverschiebung statt neuem Middleware**
`GET /api/members` und `GET /api/members/{id}` werden aus der allgemeinen `auth.Middleware`-Gruppe in die bestehende `RequireClubFunction("vorstand")`-Gruppe verschoben (dieselbe, die schon `POST /api/members`, `PUT /api/members/{id}` etc. enthält). Das ist der minimale Eingriff und nutzt die vorhandene Gruppenstruktur.

Alternative: Separaten Check im Handler — verworfen, weil die Chi-Middleware-Gruppen bereits sauber die Zugriffskontrolle kapseln und Handler-seitige Checks schwerer auditierbar sind.

## Risks / Trade-offs

[Bestehende Trainer-Nutzung von GET /api/members] → Kein Kader-Code ruft diese Endpunkte direkt auf. Der Kader-Handler hat eigene SQL-Queries. `AdminUsersPage` ruft `GET /api/members` zur Mitgliedersuche auf, ist aber bereits auf vorstand/admin eingeschränkt — kein Regressionsrisiko.

[Members-Spec-Szenario „Teamleiter sieht nur eigene Teammitglieder"] → Dieses Szenario wird im Members-Spec als veraltet markiert. Es wird durch ein neues Szenario ersetzt, das explizit 403 für Trainer dokumentiert.
