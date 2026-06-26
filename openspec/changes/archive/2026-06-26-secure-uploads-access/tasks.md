## 1. Backend: Cookie-authentifizierte Auslieferung

- [x] 1.1 `/api/uploads/*` in `internal/app/router.go` aus dem Public-Mount in die Cookie-Auth-Group (`auth.CookieMiddleware`, wie SSE) verschieben
- [x] 1.2 `ServeUpload` (`internal/upload/handler.go`): `Referrer-Policy: no-referrer` und `Cache-Control: private, no-store` setzen; UUID-/`..`-Abwehr beibehalten
- [x] 1.3 Irreführenden Doc-Kommentar korrigieren (jetzt tatsächlich geschützt)

## 2. Tests & Verifikation

- [x] 2.1 Unauthentifizierter `GET /api/uploads/<datei>` → 401
- [x] 2.2 Mit gültigem Refresh-Cookie → 200 + `Referrer-Policy: no-referrer` + `Cache-Control: private, no-store`
- [x] 2.3 `/verify-change` + `openspec validate secure-uploads-access --strict`

## 3. Hinweis (kein Frontend-Change nötig)

- [x] 3.1 Bestätigt: same-origin `<img src="/api/uploads/...">` sendet das HttpOnly-Refresh-Cookie automatisch (SameSite=Strict, Path=/) — keine `photoURL`-Anpassungen im Frontend erforderlich
