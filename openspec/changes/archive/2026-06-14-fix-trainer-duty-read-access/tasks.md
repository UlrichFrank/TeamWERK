## 1. Backend: Router-Umstrukturierung

- [x] 1.1 In `cmd/teamwerk/main.go` die vier GET-Routen (`/api/duty-types`, `/api/duty-templates`, `/api/duty-templates/{id}`, `/api/duty-templates/{id}/preview`) aus der vorstand-only Gruppe (aktuell ~Zeile 358–368) herauslösen und in die vorstand+trainer+sportliche_leitung Gruppe (aktuell ~Zeile 375–390) verschieben
- [x] 1.2 Sicherstellen, dass POST/PUT/DELETE für duty-types und duty-templates weiterhin in der vorstand-only Gruppe registriert bleiben

## 2. Frontend: SpieltagDetailPage canEdit-Fix

- [x] 2.1 In `web/src/pages/SpieltagDetailPage.tsx` den Import um `hasFunction` aus `../contexts/AuthContext` ergänzen
- [x] 2.2 `canEdit`-Konstante (Zeile ~47) von `user?.role === 'trainer'`-Vergleich auf `hasFunction(user, 'trainer') || hasFunction(user, 'sportliche_leitung') || hasFunction(user, 'vorstand')` umstellen (analog zu `KalenderPage.tsx:664`)

## 3. Tests

- [x] 3.1 Go-Test für `GET /api/duty-types` mit Trainer-Nutzer: erwartet HTTP 200
- [x] 3.2 Go-Test für `GET /api/duty-types` mit Spieler ohne Vereinsfunktion: erwartet HTTP 403
- [x] 3.3 Go-Test für `GET /api/duty-templates` mit Trainer-Nutzer: erwartet HTTP 200
- [x] 3.4 Go-Test für `POST /api/duty-templates` mit Trainer-Nutzer: erwartet HTTP 403
