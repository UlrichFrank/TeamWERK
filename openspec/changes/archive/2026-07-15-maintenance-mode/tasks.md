## 1. Datenbank — `system_settings`-Tabelle

- [x] 1.1 Nächste freie Migrationsnummer prüfen (aktuell letzte: `022_press_photo_consent`). Neue Dateien anlegen: `internal/db/migrations/023_system_settings.up.sql` und `.down.sql`.
- [x] 1.2 `023_system_settings.up.sql` schreiben: Tabelle `system_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_by INTEGER REFERENCES users(id))` und `INSERT OR IGNORE INTO system_settings (key, value) VALUES ('maintenance_mode', 'off')`.
- [x] 1.3 `023_system_settings.down.sql` schreiben: `DROP TABLE system_settings`.
- [x] 1.4 Migrations-Test in `internal/db/migrations_test.go` (falls existent) oder neu: doppelter `Migrate up`-Lauf gegen frische SQLite-DB → keine Fehler, genau eine Row `maintenance_mode`. Testname: `TestMigration_SystemSettings_Idempotent`. — als `TestMigration023_SystemSettings_Idempotent` ergänzt (folgt Namenskonvention der bestehenden 011/016/018-Tests).
- [ ] 1.5 `make migrate-up` gegen lokale Dev-DB ausführen; `sqlite3` prüfen, dass Schema wie erwartet. — durch Migration-Test in Phase 1.4 bereits gegen frische DB verifiziert; separater `make migrate-up`-Lauf beim finalen Verify (Phase 10).

## 2. Backend — Settings-Package und Store

- [x] 2.1 Neues Package `internal/settings/` anlegen.
- [x] 2.2 `internal/settings/store.go`: Typ `Store` mit `db *sql.DB`, `maintenanceOn atomic.Bool`, `pollInterval time.Duration`. Konstruktor `NewStore(ctx, db)` lädt initialen Wert per SELECT und startet Goroutine, die alle 10 s neu lädt. `NewStoreForTest` ohne Poll für Tests. Methoden `MaintenanceMode()`, `SetMaintenanceMode(ctx, enabled, updatedBy)`, `Snapshot(ctx)`, `Reload(ctx)`.
- [x] 2.3 `internal/settings/store_test.go`: Tests für Initialwert, `SetMaintenanceMode` (DB-Row aktualisiert, `updated_by` gesetzt, Cache sofort synchron), `Reload` (nach externem UPDATE greift Cache-Refresh), `Snapshot` (Metadaten inkl. E-Mail-Anzeige).
- [x] 2.4 `internal/arch/arch_test.go` erweitern: `settings` in `foundation`-Map ergänzt. `TestArchitecture_AllPackagesClassified` grün.

## 3. Backend — Maintenance-Middleware

- [x] 3.1 `internal/settings/middleware.go`: `MaintenanceMiddleware(store *Store, jwtSecret string)`. Signatur nutzt `string` (nicht `[]byte`) für Konsistenz mit bestehendem `auth.ParseAccessToken`.
- [x] 3.2 `internal/settings/middleware_test.go`: 7 Test-Funktionen decken alle Szenarien aus dem spec ab (Modus off, blockierte Non-Admin-Mutation, Admin-Bypass, Auth-Route-Whitelist, GET/HEAD/OPTIONS-Durchlass, unauth-Mutation-Block, kaputter Token). Alle grün.

## 4. Backend — HTTP-Handler

- [x] 4.1 `internal/settings/handler.go`: `Handler` mit `store`, `hub`. Routen: `GetPublicStatus` (public), `GetAdminStatus` (admin, mit Metadaten), `SetMaintenanceMode` (admin, POST) inkl. `hub.Broadcast("settings-changed")`.
- [x] 4.2 `internal/settings/handler_test.go`: 6 Testfälle (public-Status ohne Auth, public-Status reflektiert Toggle, Toggle als Admin + Broadcast, Toggle als Non-Admin → 403, Toggle unauth → 401, Admin-Status mit Metadaten). Nutzt eigenen `testHTTPServer`-Helper, weil `testutil.NewServer` auth-Middleware forciert.
- [x] 4.3 `internal/app/router.go`: `Settings` + `SettingsStore` in `Handlers`; `MaintenanceMiddleware` nach CORS registriert (nil-Store-tolerant); Public-Route `GET /api/maintenance-status`; Admin-Routen `GET`/`POST /api/admin/maintenance-mode`. `cmd/teamwerk/main.go` verdrahtet `settings.NewStore(ctx, database)` (Poll-Loop endet mit SIGTERM).
- [x] 4.4 `internal/app/maintenance_router_test.go`: E2E-Test — bei aktivem Modus liefert `POST /api/games` als Non-Admin 503 mit `X-Maintenance-Mode: 1` und maintenance_mode-Body; `GET /api/maintenance-status` bleibt 200; Admin-POST wird von der Maintenance-Middleware nicht abgelehnt.

## 5. Backend — CLI-Subcommand

- [x] 5.1 `cmd/teamwerk/main.go`: Subcommand `maintenance on|off [--db <pfad>]`. Wrapper `runMaintenance()` + testbare Logik `maintenanceToggle(args)`. Öffnet DB direkt, setzt `system_settings.value`, `updated_by=NULL` (CLI hat keinen User-Kontext).
- [x] 5.2 `cmd/teamwerk/maintenance_test.go`: `TestCLI_MaintenanceOn`, `TestCLI_MaintenanceOff`, `TestCLI_MaintenanceInvalidArg` gegen frische Temp-DB mit vollen Migrationen.
- [x] 5.3 Runbook-Ergänzung — in Phase 9.1 erledigt.

## 6. Frontend — Status-Hook und Banner

- [x] 6.1 `web/src/hooks/useMaintenanceStatus.ts`: Hook fetcht `GET /maintenance-status` bei Mount, abonniert `settings-changed`-SSE, refetcht dann. Fail-open bei Fehler.
- [x] 6.2 `web/src/hooks/useMaintenanceStatus.test.tsx`: 4 Tests (initial fetch, fail-open, event triggert refetch, andere Events ignoriert).
- [x] 6.3 `web/src/components/MaintenanceBanner.tsx`: persistent, `role=status`, `bg-brand-danger-light` + `<AlertTriangle>`, nicht dismissable.
- [x] 6.4 `web/src/components/MaintenanceBanner.test.tsx`: 3 Tests (sichtbar bei enabled, null bei disabled, null während loading).
- [x] 6.5 `web/src/components/AppShell.tsx`: `<MaintenanceBanner />` oberhalb `<TransitionalHostnameBanner />` gemountet.

## 7. Frontend — Axios-Interceptor

- [x] 7.1 `web/src/lib/api.ts`: `setMaintenanceHandler(fn|null)`-Registrierung, Interceptor prüft `status===503 && headers['x-maintenance-mode']==='1'`. Kein DOM-Import.
- [x] 7.2 `web/src/lib/api.test.ts`: 4 Tests via axios-custom-adapter (Header-503 triggert Handler, generischer 503 nicht, kein Handler unproblematisch, 200 lässt Handler in Ruhe).
- [x] 7.3 `AppShell.tsx`: `setMaintenanceHandler` in useEffect registriert, zeigt 5-Sekunden-Toast unten mit Wartungshinweis.

## 8. Frontend — Admin-UI-Seite

- [x] 8.1 `web/src/pages/admin/WartungsmodusPage.tsx`: lädt Zustand, zeigt Metadaten, Toggle-Button (Primary bei on → aus, Danger bei off → ein), inline Fehleranzeige, plus CLI-Fallback-Hinweisbox.
- [x] 8.2 `WartungsmodusPage.test.tsx`: 3 Tests (Zustand-Anzeige, Toggle-Klick sendet POST + reload, Fehler beim Laden).
- [x] 8.3 `web/src/App.tsx`: Route `/wartung` unter `RoleRoute roles={['admin']}`.
- [x] 8.4 Nav: sowohl `navModules` in `AppShell.tsx` (Client-Layout) als auch `policy.rules.NavItems` (Server-driven Filter) um „Wartungsmodus" → `/wartung` (nur Rolle `admin`) ergänzt.

## 9. Runbook-Ergänzung

- [x] 9.1 `docs/agent/10-deployment.md`: Abschnitt „Wartungsmodus" mit UI-Weg (`/wartung`), CLI-Fallback + Sync-Verhalten-Hinweis ergänzt.
- [ ] 9.2 (Optional, außerhalb dieser Change): Follow-up-Commit an `deploy/internal-alias-cutover-runbook.md`, der „vor Phase B: Wartungsmodus aktivieren, nach Phase E deaktivieren" ergänzt. — bewusst nicht in dieser Change, um Scope sauber zu halten.

## 10. Verifikation

- [x] 10.1 `openspec validate maintenance-mode --strict` → „is valid".
- [x] 10.2 Alle Gates grün: `go test ./...` 1221 passed / 44 pkg, `go vet` clean, `golangci-lint run ./...` No issues, `pnpm test` 507 passed / 52 files, `pnpm lint` 0 errors (6 pre-existing set-state-in-effect Warnings, konsistent mit Codebase-Stil), `pnpm build` + `go build` grün.
- [ ] 10.3 Manueller Smoke-Test lokal: Admin schaltet Modus ein → zweite Browsersession (Nicht-Admin) sieht Banner, Klick auf Schreiben löst freundlichen Dialog aus. Admin schaltet aus → Banner verschwindet. — vom Betreuer zu machen, Automat kann nicht in zwei parallelen Browsersessions klicken.
- [ ] 10.4 `/verify-change` durchlaufen lassen. — separater manueller Command am Ende.
