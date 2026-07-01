## Why

Der tus-basierte Video-Upload (`web/src/pages/VideoUploadPage.tsx`) bricht mit HTTP 401 ab, sobald der 15-minütige JWT-Access-Token während des Uploads abläuft. `tus-js-client` fährt eigene `fetch`/`XHR`-Requests und geht **nicht** durch die Axios-Instanz, die in `web/src/lib/api.ts` einen Auto-Refresh-Interceptor bei 401 hat. Aktuell steht `headers: { Authorization: Bearer ${getAccessToken() ?? ''} }` als **statisches** Objekt in der `tus.Upload`-Konfiguration — der Token wird einmal beim Konstruktor-Aufruf gelesen und danach mit jedem PATCH-Chunk unverändert gesendet, bis er expired und die Auth-Middleware jeden weiteren Chunk mit 401 abweist.

Konkret gemeldet am 2026-07-01 mit einem 768-MB-Upload nach ~3½ Minuten (Journal des neuen Servers zeigt ~12 erfolgreiche PATCH-Chunks mit HTTP 204, dann sofortiger 401 in 245 µs — klassisches Ablauf-Muster). Bei bislang typischen kleinen Testvideos war der Fehler unauffällig, wird aber mit produktivem Betrieb (mehrere GB pro Spiel) zum harten Blocker.

## What Changes

- **`web/src/lib/api.ts`**: `refreshAccessToken(): Promise<string>` als exportierte Funktion herauslösen — nutzt das bereits vorhandene `refreshPromise`-In-Flight-Guard (Single-Flight-Pattern), damit paralleles `refresh` von Axios-Interceptor und tus-Hook sich nicht ins Gehege kommen.
- **`web/src/pages/VideoUploadPage.tsx`**: `tus.Upload`-Konfiguration bekommt zwei zusätzliche Hooks:
  - `onBeforeRequest(req)`: setzt `Authorization: Bearer ${getAccessToken() ?? ''}` frisch aus dem Store vor jedem PATCH/POST/HEAD (statt statischem `headers`-Objekt beim Konstruktor).
  - `onShouldRetry(err, retryAttempt, options)`: prüft, ob `err.originalResponse?.getStatus() === 401`. Wenn ja, `await refreshAccessToken()` aufrufen und `true` zurückgeben (tus retryt automatisch, der Retry liest den neuen Token via `onBeforeRequest`). Wenn `refreshAccessToken` selbst wirft (Refresh-Token abgelaufen), `setAccessToken(null)` + Redirect auf `/login` (analog zu `api.ts`) und `false` zurückgeben.
- **`web/src/pages/__tests__/VideoUploadPage.test.tsx`**: Zwei zusätzliche Test-Fälle — Access-Token läuft während Upload ab (Chunk 1 = 401, Refresh wird aufgerufen, Chunk 1-Retry mit neuem Token = 204) und Refresh-Token abgelaufen (Refresh selbst gibt 401, Upload bricht mit klarer Fehlermeldung ab, User wird auf `/login` umgeleitet).

**Nicht Teil dieser Änderung:**
- Kein proaktiver Refresh vor Ablauf (z. B. Timer, der 60 s vor JWT-Expiry aktualisiert). Reaktives Refresh-on-401 reicht — der Trade-off ist ein einziger fehlgeschlagener Chunk pro Ablauf-Zyklus (~15 s Upload-Zeit verloren, tus retryt idempotent), gegen deutlich einfachere Implementierung.
- Keine Anpassung anderer tus-Konsumenten — aktuell ist `VideoUploadPage.tsx` die einzige Stelle, die tus nutzt.

## Capabilities

### New Capabilities
- `video-upload-token-refresh`: Beim tus-Video-Upload wird der JWT-Access-Token vor jedem Chunk-Request aus dem geteilten Store gelesen und bei Chunk-401 automatisch über denselben Refresh-Endpoint erneuert, den auch die Axios-Instanz nutzt.

### Modified Capabilities

(keine — es gibt aktuell keine spec für Video-Upload; die neue Capability beschreibt ausschließlich das Token-Refresh-Verhalten des Upload-Clients.)

## Impact

- **`web/src/lib/api.ts`**: neue Export `refreshAccessToken`. Der Body des Response-Interceptors nutzt danach dieselbe Funktion (statt die Promise inline zu bauen) → Verhalten identisch, nur Kapselung besser.
- **`web/src/pages/VideoUploadPage.tsx`**: `headers`-Feld wird gestrichen (`onBeforeRequest` ersetzt es). `onShouldRetry` neu hinzu.
- **`web/src/pages/__tests__/VideoUploadPage.test.tsx`**: zwei neue Testfälle. Bestehende Tests bleiben unverändert.
- **Kein Backend-Code angefasst**: `/api/auth/refresh` existiert bereits und wird unverändert benutzt. `/api/videos/upload/*` mit JWT-Auth-Middleware bleibt wie ist — der Fix ist rein Frontend-seitig.
- **PWA/Service-Worker**: keine Auswirkung — der Service-Worker cached `/api/*`-Requests network-first, tus-PATCHes gehen weiter direkt an den Server.
- **Race-Condition**: durch Wiederverwendung des existierenden `refreshPromise`-Guards in `api.ts` sind gleichzeitig laufende Refresh-Aufrufe (z. B. paralleler Axios-401 + tus-401) automatisch dedupliziert — beide warten auf dasselbe Promise.
- **Betriebs-Risiko**: minimal. Wenn der neue Code Bugs hat, bekommt der User im schlimmsten Fall dieselbe Fehlermeldung wie vorher („Upload fehlgeschlagen: unauthorized") — kein Datenverlust, tus-Fingerprint bleibt im localStorage, Upload lässt sich weiter fortsetzen.
