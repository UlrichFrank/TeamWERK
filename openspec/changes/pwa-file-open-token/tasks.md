## 1. Backend: Token-Generierung

- [x] 1.1 `generateDownloadToken(fileID, userID int, secret string) string` in `internal/files/` implementieren — HMAC-SHA256 über JSON-Payload `{fid, uid, exp}`, Base64URL-kodiert
- [x] 1.2 `validateDownloadToken(token string, fileID int, secret string) (userID int, err error)` implementieren — prüft Signatur, TTL und File-ID-Match
- [x] 1.3 Handler `HandleDownloadToken` anlegen: authentifiziert via Middleware, prüft Lesezugriff auf Datei-Ordner, gibt `{ "token": "..." }` zurück
- [x] 1.4 Route `GET /api/files/{id}/download-token` in `main.go` unter dem `auth.Middleware`-Block registrieren

## 2. Backend: Download per Token

- [x] 2.1 Bestehenden `HandleDownload`-Handler erweitern: wenn `?token=` vorhanden und kein `Authorization`-Header, Token validieren statt JWT aus Header lesen
- [x] 2.2 Nach Token-Validierung: Lesezugriff auf Ordner der Datei erneut prüfen (gleiche ACL-Logik wie authenticated path)
- [x] 2.3 Fehlerbehandlung: abgelaufenes Token → 401, manipuliertes Token → 401, File-ID-Mismatch → 401, kein Zugriff → 403

## 3. Frontend: openFile() umbauen

- [x] 3.1 In `DocumentsPage.tsx` die Funktion `openFile()` umschreiben: erst `GET /api/files/{id}/download-token` aufrufen, dann `window.open('/api/files/${file.id}/download?token=${token}', '_blank')` öffnen
- [x] 3.2 Blob-Download, `createObjectURL`, `revokeObjectURL` und den `isViewable`-Branch entfernen
- [x] 3.3 Fehlerfall abfangen: wenn Token-Anfrage fehlschlägt, kurze Fehlermeldung anzeigen (kein `alert()`, stattdessen inline error state)
