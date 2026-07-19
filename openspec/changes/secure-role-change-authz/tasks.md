## 1. Handler-Guard

- [x] 1.1 In `UpdateUserRole` (`internal/auth/handler.go`) vor dem Setzen prüfen: Ziel-Account aktuelle Rolle laden; wenn `caller.Role != "admin"` und (Ziel-Rolle == `admin` ODER `caller.UserID == targetID`) → 403
- [x] 1.2 Bestehende „nur admin darf admin vergeben"-Prüfung beibehalten

## 2. Permissions-Matrix

- [x] 2.1 `internal/permissions/matrix_test.go`: Erwartung für `PUT /api/users/{id}/role` an das neue Verhalten anpassen (ggf. eigenes Expected-Set)

## 3. Tests

- [x] 3.1 Nicht-Admin-`vorstand` degradiert Admin → 403, Ziel bleibt `admin`
- [x] 3.2 Nicht-Admin ändert eigene Rolle → 403
- [x] 3.3 `admin` setzt Rolle → erfolgreich (kein Regress)
- [x] 3.4 Nicht-Admin vergibt `admin` → 403 (Bestandsverhalten)

## 4. Verifikation

- [x] 4.1 `/verify-change` + `openspec validate secure-role-change-authz --strict`
