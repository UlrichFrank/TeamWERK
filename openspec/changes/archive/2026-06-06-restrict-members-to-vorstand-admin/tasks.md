## 1. Backend

- [x] 1.1 In `cmd/teamwerk/main.go` die Zeilen `r.Get("/api/members", membH.List)` und `r.Get("/api/members/{id}", membH.Get)` aus der allgemeinen Authenticated-Gruppe entfernen und in die `RequireClubFunction("vorstand")`-Gruppe verschieben

## 2. Frontend Route Guard

- [x] 2.1 In `web/src/App.tsx` `'trainer'` aus dem `roles`-Array der Route `path="mitglieder"` entfernen (→ `['admin','vorstand']`)
- [x] 2.2 In `web/src/App.tsx` `'trainer'` aus dem `roles`-Array der Route `path="mitglieder/:id"` entfernen (→ `['admin','vorstand']`)

## 3. Verifikation

- [x] 3.1 Mit einem Trainer-Account prüfen, dass `/mitglieder` auf `/` weiterleitet
- [x] 3.2 Mit einem Trainer-Account prüfen, dass `GET /api/members` 403 zurückgibt
- [x] 3.3 Mit einem Vorstand-Account prüfen, dass `/mitglieder` und `GET /api/members` weiterhin funktionieren
