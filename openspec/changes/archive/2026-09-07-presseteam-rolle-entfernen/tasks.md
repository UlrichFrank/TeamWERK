## 1. Migration

- [x] 1.1 `internal/db/migrations/056_presseteam_rolle_entfernen.up.sql`: `UPDATE users/invitation_tokens SET role='standard' WHERE role='presseteam'`, danach Tabellen-Rebuild mit `CHECK (role IN ('admin','standard'))` (12-Schritt-Recipe wie `019`)
- [x] 1.2 `.down.sql`: CHECK wieder auf drei Werte erweitern (kein Daten-Restore möglich, im Kopf dokumentieren)
- [x] 1.3 Go-Test: Bestandszeile wird `standard`, `INSERT … 'presseteam'` scheitert am CHECK

## 2. Backend

- [x] 2.1 `internal/auth/roles.go`: `RolePressTeam` entfernen, Hierarchie-Kommentar korrigieren
- [x] 2.2 `internal/auth/handler.go`: Rollen-Validierung (Einladung + `PUT /users/{id}/role`) auf `admin|standard`
- [x] 2.3 `internal/app/router.go`: Autor-Gruppe der Match-Reports verliert `RequireRole(...)` → Tier Authenticated
- [x] 2.4 `internal/matchreports`: `isPressTeamOrAdmin` entfernen (Aufrufer in `Create`, `MyList`)
- [x] 2.5 `internal/duties/match_report_guard.go` + Aufruf in `ClaimSlot` entfernen (`role_required` entfällt)
- [x] 2.6 `internal/policy/rules.go`: Nav „Spielberichte" für alle Eingeloggten
- [x] 2.7 `internal/testutil/fixtures.go`: `CreatePressTeamUser` entfernen; Tests auf `standard` umstellen
- [x] 2.8 `internal/arch/authz_test.go` + `internal/permissions/matrix_test.go`: Rollen-Konstante, Allowlist-Einträge und Autor-Tier-Erwartung nachziehen

## 3. Frontend

- [x] 3.1 `web/src/App.tsx`: `SYSTEM_ROLES` ohne `presseteam`; `/spielberichte` und `/spielberichte/:id` ohne Rollen-Gate
- [x] 3.2 `web/src/pages/AdminUsersPage.tsx`: Rolle aus Labels, `ALL_ROLES` und Auswahlfeld entfernen

## 4. Abschluss

- [x] 4.1 `make test`, `golangci-lint`, `pnpm -C web build/test/lint` grün
- [x] 4.2 `openspec validate presseteam-rolle-entfernen --strict`
