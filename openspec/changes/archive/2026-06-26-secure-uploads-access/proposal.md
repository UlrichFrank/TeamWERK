## Why

`ServeUpload` (`internal/upload/handler.go:462-470`) ist im Public-Tier registriert (`internal/app/router.go:93`, vor jeder Auth-Middleware), obwohl der Doc-Kommentar `// GET /api/uploads/* — Auth required` lautet — ein Intent/Implementierungs-Widerspruch. Der Handler prüft nur auf `..`-Traversal (nicht ausnutzbar dank `CleanPath` + `filepath.Join`) und streamt danach **jede** Datei unter `uploadDir` ohne JWT, Token oder Ownership-Check. SEPA-PDFs sind dank ZK-Ciphertext unkritisch, aber **Mitglieder-/Nutzerfotos** (PII, ggf. Minderjährige) sind betroffen: Die Foto-URL erscheint in vielen API-Antworten und `<img>`-Tags und leakt über Referrer, Browser-History, Proxy-/CDN-Logs und PWA-Cache — danach ohne Login dauerhaft abrufbar (Sicherheitsaudit 2026-06-26, **B-5**). Die UUIDv4-Dateinamen sind nur „Security by unguessable URL".

## What Changes

- **`/api/uploads/*` aus dem Public-Tier entfernen** und in die **Cookie-Auth-Group** (`auth.CookieMiddleware`) verschieben — dasselbe Muster, mit dem die SSE-Routen das „kein Bearer im `<img>`/`EventSource`"-Problem lösen. Same-origin `<img src>` sendet das HttpOnly-Refresh-Cookie automatisch → **kein Frontend-Umbau** nötig. (Die zunächst erwogene Download-Token-Variante wurde als unverhältnismäßig für einen Low-Befund verworfen, siehe `design.md` D1.)
- **Doc-Kommentar korrigieren** und mit der tatsächlichen, jetzt geschützten Mountierung in Einklang bringen.
- **Härtung der Foto-Antworten:** `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store`. UUID-Dateinamen bleiben als Defense-in-Depth.
- **Bewusste Grenze:** kein Pro-Foto-Sichtbarkeitscheck (nur „authentifiziert"); der behobene Befund war der unauthentifizierte Zugriff.

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `permissions`: Die Anforderung „Public Endpoints sind ohne Auth zugänglich" wird angepasst — `GET /api/uploads/*` ist **nicht mehr** ohne Autorisierung erreichbar. Neue Anforderung: Upload-Auslieferung erfordert ein gültiges, kurzlebiges Download-Token mit vorab geprüfter Berechtigung.

## Impact

- **Code:** `internal/app/router.go` (`/api/uploads/*` in die Cookie-Auth-Group verschieben), `internal/upload/handler.go` (`ServeUpload`: Härtungs-Header; Kommentar korrigieren).
- **Frontend:** keine Änderung (same-origin `<img>` sendet das Cookie automatisch).
- **API-Verhalten:** **BREAKING** für unauthentifizierte `/api/uploads/...`-Zugriffe (bisher offen) — beabsichtigt; eingeloggte Nutzer unverändert.
- **Tests:** unauthentifiziert → 401; mit Cookie → 200 + Härtungs-Header.
- **Daten/Migration:** keine.
