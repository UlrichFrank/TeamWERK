## 1. Handler-Guard

- [ ] 1.1 In `UpdateUserRole` (`internal/auth/handler.go`) vor dem Setzen prüfen: Ziel-Account aktuelle Rolle laden; wenn `caller.Role != "admin"` und (Ziel-Rolle == `admin` ODER `caller.UserID == targetID`) → 403
- [ ] 1.2 Bestehende „nur admin darf admin vergeben"-Prüfung beibehalten

## 2. Permissions-Matrix

- [ ] 2.1 `internal/permissions/matrix_test.go`: Erwartung für `PUT /api/users/{id}/role` an das neue Verhalten anpassen (ggf. eigenes Expected-Set)

## 3. Tests

- [ ] 3.1 Nicht-Admin-`vorstand` degradiert Admin → 403, Ziel bleibt `admin`
- [ ] 3.2 Nicht-Admin ändert eigene Rolle → 403
- [ ] 3.3 `admin` setzt Rolle → erfolgreich (kein Regress)
- [ ] 3.4 Nicht-Admin vergibt `admin` → 403 (Bestandsverhalten)

## 4. Verifikation

- [ ] 4.1 `/verify-change` + `openspec validate secure-role-change-authz --strict`
