## Why

`ServeUpload` (`internal/upload/handler.go:462-470`) ist im Public-Tier registriert (`internal/app/router.go:93`, vor jeder Auth-Middleware), obwohl der Doc-Kommentar `// GET /api/uploads/* — Auth required` lautet — ein Intent/Implementierungs-Widerspruch. Der Handler prüft nur auf `..`-Traversal (nicht ausnutzbar dank `CleanPath` + `filepath.Join`) und streamt danach **jede** Datei unter `uploadDir` ohne JWT, Token oder Ownership-Check. SEPA-PDFs sind dank ZK-Ciphertext unkritisch, aber **Mitglieder-/Nutzerfotos** (PII, ggf. Minderjährige) sind betroffen: Die Foto-URL erscheint in vielen API-Antworten und `<img>`-Tags und leakt über Referrer, Browser-History, Proxy-/CDN-Logs und PWA-Cache — danach ohne Login dauerhaft abrufbar (Sicherheitsaudit 2026-06-26, **B-5**). Die UUIDv4-Dateinamen sind nur „Security by unguessable URL".

## What Changes

- **`/api/uploads/*` aus dem Public-Tier entfernen** und Zugriff auf authentifizierte, berechtigte Aufrufer beschränken. Da `<img src>` keinen Bearer-Header sendet, erfolgt der Zugriff über dasselbe **kurzlebige, signierte Download-Token-Muster** wie `SepaDownloadToken`/`SepaDownload` (Capability `file-download-token`) statt über einen Bearer-geschützten XHR.
- **Ownership-/Sichtbarkeits-Check** beim Ausstellen des Tokens (analog `policy.MemberCan`): wer ein Foto sehen darf, erhält ein Token; das Streamen validiert das Token.
- **Doc-Kommentar korrigieren** und mit der tatsächlichen Mountierung in Einklang bringen.
- **Härtung der Foto-Antworten:** `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store`. UUID-Dateinamen bleiben als Defense-in-Depth erhalten.

## Capabilities

### New Capabilities
<!-- keine -->

### Modified Capabilities
- `permissions`: Die Anforderung „Public Endpoints sind ohne Auth zugänglich" wird angepasst — `GET /api/uploads/*` ist **nicht mehr** ohne Autorisierung erreichbar. Neue Anforderung: Upload-Auslieferung erfordert ein gültiges, kurzlebiges Download-Token mit vorab geprüfter Berechtigung.

## Impact

- **Code:** `internal/app/router.go` (Mountierung von `/api/uploads/*` verschieben), `internal/upload/handler.go` (`ServeUpload`: Token-Validierung statt offener Auslieferung; Token-Ausgabe-Endpoint analog `SepaDownloadToken`; Härtungs-Header; Kommentar korrigieren).
- **Frontend:** Foto-URLs (`photoURL`) müssen das Download-Token mitführen (Token-Endpoint aufrufen, dann `<img src=...token>`); betroffene Stellen in `web/src/` anpassen.
- **API-Verhalten:** **BREAKING** für direkte, tokenlose `/api/uploads/...`-Zugriffe (bisher offen) — beabsichtigt.
- **Tests:** tokenloser Zugriff → 401/403; gültiges Token → 200; Token für ein nicht sichtbares Foto wird nicht ausgestellt (403).
- **Daten/Migration:** keine.
