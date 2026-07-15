## Why

Bei Server-Umzügen, DB-Migrationen oder Compliance-Incidents muss die App vorübergehend gegen Schreibzugriffe abgesichert werden, ohne den Dienst komplett offline zu nehmen. Konkret steht der VPS-Wechsel `internal.*` → `teamwerk.*` an, dessen finaler `server-sync-data`-Lauf und die DNS-Propagation (aktuell TTL 10800 s = 3 h) ein Fenster ergeben, in dem Alt- und Neu-DB divergieren würden, sobald Nutzer weiter auf der Alt-Instanz schreiben. Ein Nginx-Level-Block hätte funktioniert, verlagert die Regel aber weg vom Anwendungs-Kern und kennt die UI nicht — Nutzer sähen kryptische 503er. Ein Dauer-Wartungsmodus im Backend + Frontend löst dieselbe Migration sauber **und** steht für alle künftigen Wartungsfenster bereit (weitere Umzüge, Notfall-Freezes, größere Deployment-Slots).

## What Changes

- **Neue Middleware `MaintenanceMiddleware`** im Request-Pfad, blockiert `POST/PUT/PATCH/DELETE` mit HTTP 503, wenn der Modus aktiv ist. Ausnahmen: `/api/auth/*` (verhindert Selbst-Aussperr-Falle), Requester mit System-Rolle `admin` (Bypass, damit Admin den Modus wieder ausschalten kann).
- **Neue Tabelle `system_settings`** (`key TEXT PK, value TEXT, updated_at, updated_by INTEGER REFERENCES users(id)`) mit initialem Row `key='maintenance_mode', value='off'`. Generisches Key-Value-Schema, damit spätere Toggles (z. B. `read_only_kader`, `disable_signups`) ohne Schema-Änderung möglich sind — aber im Rahmen dieser Change wird ausschließlich der eine Key genutzt.
- **Migration** `0NN_system_settings.up.sql`/`.down.sql` mit dem oben beschriebenen Schema und idempotentem `INSERT OR IGNORE` für die Default-Row.
- **Neuer Toggle-Endpoint** `POST /api/admin/maintenance-mode` (`auth.RequireRole("admin")`), Body `{"enabled": true|false}`, schreibt `system_settings`-Row und ruft `h.hub.Broadcast("settings-changed")`.
- **Neuer Public-Read-Endpoint** `GET /api/maintenance-status`, unauthenticated (weil der Banner auch auf der Login-Seite sichtbar sein muss), liefert `{"enabled": bool}`.
- **503-Response-Kontrakt** der Middleware: JSON-Body `{"error":"maintenance_mode","message":"…"}` **und** Response-Header `X-Maintenance-Mode: 1`, damit der Frontend-Interceptor den Fall eindeutig erkennt (getrennt von Load-Balancer-503s).
- **Frontend-Banner-Komponente** `MaintenanceBanner` in `web/src/components/`, gemountet in `AppShell.tsx` **oberhalb** des `TransitionalHostnameBanner`. Sichtbar wenn Status `enabled=true`; nutzt `useMaintenanceStatus`-Hook, der via `useLiveUpdates('settings-changed')` auf SSE reagiert.
- **Frontend Axios-Interceptor** in `web/src/lib/api.ts`: fängt Responses mit `status === 503 && headers['x-maintenance-mode'] === '1'` ab, zeigt einen freundlichen Modal/Toast („Wartungsmodus aktiv — bitte gleich noch mal versuchen.") und unterdrückt den generischen Fehler.
- **Admin-UI-Seite** `/admin/wartung` (oder Integration in bestehende Admin-Nav) mit Ein/Aus-Button, aktueller Zustand, letzter Umschaltender + Zeitpunkt.
- **CLI-Fallback** `teamwerk maintenance on|off` als Subcommand in `cmd/teamwerk/main.go` — defensiver Ausschalt-Weg, falls das Admin-UI mal nicht erreichbar ist (Login kaputt, JS-Fehler, etc.). SSE-Broadcast entfällt hier, weil der Betrieb-Prozess keinen `hub` in der Hand hat; das Frontend synchronisiert sich beim nächsten Poll/Reload.
- **Frontend-UX-Tiefe bewusst begrenzt** auf zwei Layer: (1) persistenter Banner, (2) 503-Interceptor-Dialog. **Kein** per-Button-Disable-Layer — das wäre über alle Formulare hinweg viel Oberfläche für ein Wartungs-Feature und würde die Wartbarkeit belasten.

## Capabilities

### New Capabilities
- `maintenance-mode`: Admin-schaltbarer Zustand, der auf dem Server alle Mutationen (POST/PUT/PATCH/DELETE) mit HTTP 503 abweist — außer Auth-Routen und Requests von System-Rolle `admin` — und im Frontend über einen persistenten Banner plus 503-Dialog kommuniziert. Toggle via Admin-UI oder CLI-Subcommand, Zustand in `system_settings`, Live-Verteilung via SSE-Event `settings-changed`.

### Modified Capabilities
(keine — der Wartungsmodus ist eigenständig, greift aber quer in Request-Handling und Frontend-Shell ein. Bestehende Capabilities wie `auth` oder `api-routes` werden nicht in ihren Requirements geändert.)

## Impact

- **`internal/db/migrations/`**: neue Migration mit nächster freier Nummer (aktuell zu prüfen), `system_settings`-Tabelle + Default-Row.
- **`internal/app/router.go`** (`BuildRouter`): neue `MaintenanceMiddleware` zwischen CORS/Recover und Auth-Middleware; neue Routen `POST /api/admin/maintenance-mode` (admin-Tier) und `GET /api/maintenance-status` (public).
- **Neues Package `internal/settings/`** (oder Ergänzung eines bestehenden Config-Packages — `internal/config/` ist env-basiert und passt nicht): Handler + Store für `system_settings`. Store cached den Wert in-memory (atomic bool) und invalidiert bei jedem `POST`, damit die Middleware auf dem Hot-Path keinen DB-Roundtrip macht.
- **`internal/hub/`**: SSE-Kanal `settings-changed` (falls nicht schon vorhanden — ist ein neues Event-Label).
- **`cmd/teamwerk/main.go`**: neuer Subcommand `maintenance on|off`, ruft direkt in den Settings-Store, ohne HTTP.
- **Frontend**:
  - `web/src/components/MaintenanceBanner.tsx` (neu) + Unit-Test.
  - `web/src/hooks/useMaintenanceStatus.ts` (neu) + Unit-Test.
  - `web/src/components/AppShell.tsx`: `<MaintenanceBanner />` oberhalb `<TransitionalHostnameBanner />` mounten.
  - `web/src/lib/api.ts`: Response-Interceptor um Maintenance-503-Erkennung erweitern; Dialog-/Toast-Trigger.
  - `web/src/pages/AdminMaintenancePage.tsx` (neu) + Route in `App.tsx` + Nav-Eintrag in `AppShell.tsx` (nur `admin`).
- **Tests**: siehe Test-Anforderungen unten. Zusätzlich Architektur-Test (`internal/arch/arch_test.go`) muss das neue Package klassifizieren.
- **Sicherheit**: keine Auswirkung auf Auth-, Krypto-, Rate-Limit-Modell. Die Middleware sitzt **vor** den Auth-Routen, dokumentiert per Kommentar, warum Auth trotzdem durchgelassen wird.
- **Performance**: Middleware kostet einen atomic-Load pro Nicht-GET-Request. Kein DB-Zugriff auf dem Hot-Path.
- **Migration-Interaktion mit `server-sync-data`**: Bewusstes Feature — der `maintenance_mode=on`-Zustand wandert bei einem DB-Snapshot mit auf den Neu-VPS. Nach dem DNS-Switch sehen Neu-VPS-Nutzer weiter den Banner, bis der Admin dort explizit deaktiviert. Wird im Runbook dokumentiert.

## Test-Anforderungen

| Route / Ort | Testname | Erwarteter Status / Invariante |
|---|---|---|
| `POST /api/admin/maintenance-mode` | `TestMaintenanceMode_Toggle_AsAdmin_Returns200` | Admin toggelt Flag auf `on`; Response 200; `system_settings.value = 'on'`; `updated_by` = Admin-User-ID. |
| `POST /api/admin/maintenance-mode` | `TestMaintenanceMode_Toggle_AsNonAdmin_Returns403` | Vorstand/Kassierer/Trainer erhalten 403; Flag unverändert. |
| `POST /api/admin/maintenance-mode` | `TestMaintenanceMode_Toggle_Unauthenticated_Returns401` | Ohne Auth-Header 401. |
| `GET /api/maintenance-status` | `TestMaintenanceStatus_Public_Returns200` | Antwort 200 auch ohne Auth-Header; Body `{"enabled": bool}`. |
| Middleware | `TestMaintenanceMiddleware_BlocksMutation_When_Active` | `POST /api/games` bei aktivem Modus → 503, JSON `{"error":"maintenance_mode"}`, Header `X-Maintenance-Mode: 1`. |
| Middleware | `TestMaintenanceMiddleware_AllowsGet_When_Active` | `GET /api/games` bei aktivem Modus → 200 (unverändert). |
| Middleware | `TestMaintenanceMiddleware_AllowsAuthRoutes_When_Active` | `POST /api/auth/login`, `POST /api/auth/refresh`, `POST /api/auth/logout` liefern reguläre Antworten (nicht 503). |
| Middleware | `TestMaintenanceMiddleware_AllowsAdmin_When_Active` | Admin-JWT: `POST /api/games` → normal (200/201/400 je nach Payload, aber **nicht** 503). |
| Middleware | `TestMaintenanceMiddleware_NoOverhead_When_Inactive` | Bei Flag=off passiert die Middleware jeden Request unverändert durch (kein 503, kein Header). |
| Frontend `MaintenanceBanner` | Component-Test | Rendert bei `status.enabled=true`; rendert `null` bei `false`. |
| Frontend Axios-Interceptor | Unit-Test | Response `status:503, headers:{'x-maintenance-mode':'1'}` triggert einen erkennbaren Nutzer-Hinweis (spy/mock auf Dialog-Fn); reguläre 503 (ohne Header) fällt durch. |
| Frontend `useMaintenanceStatus` | Hook-Test | Initialer Fetch von `/api/maintenance-status`; auf SSE-Event `settings-changed` erneuter Fetch. |
| CLI | `TestCLI_MaintenanceOn` / `TestCLI_MaintenanceOff` | Subcommand setzt `system_settings.value` korrekt; Exit 0. |
| Migration | `TestMigration_SystemSettings_Idempotent` | Zweimaliges Migrate-up ohne Fehler; genau eine Row `maintenance_mode`. |

**Garantierte fachliche Invariante**: Bei aktivem Wartungsmodus akzeptiert das Backend **keine** Mutations-Requests von Nicht-Admin-Nutzern (außer Auth-Routen), **und** jede App-Session — auch die noch nicht authentifizierte Login-Seite — zeigt einen sichtbaren Wartungshinweis; der Admin bleibt jederzeit voll handlungsfähig, kann den Modus über UI **oder** CLI ein- und ausschalten.
