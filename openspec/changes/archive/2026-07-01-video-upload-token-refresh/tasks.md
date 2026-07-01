## 1. `api.ts` — Refresh-Funktion extrahieren

- [x] 1.1 In `web/src/lib/api.ts` eine neue exportierte Funktion `refreshAccessToken(): Promise<string>` einfügen. Body: wenn `refreshPromise` null ist, `axios.post('/api/auth/refresh', {}, { withCredentials: true })`, im `.then` den `access_token` extrahieren, `setAccessToken` aufrufen, den Token zurückgeben; in `.finally` `refreshPromise = null` setzen. Immer die aktuelle `refreshPromise` zurückgeben. Semantik unverändert.
- [x] 1.2 Response-Interceptor umbauen: den Inline-Refresh-Block durch `const newToken = await refreshAccessToken()` ersetzen. `original.headers.Authorization = 'Bearer ' + newToken; return api(original)` bleibt. Verhalten identisch, keine Test-Änderungen erwartet.
- [x] 1.3 `make -C web test -- api` (oder wie die Test-Suite es startet) — bestehende `api.ts`-Tests müssen grün bleiben. Falls sie brechen, prüfen ob die Extraction ein bislang verstecktes Verhalten geändert hat.

## 2. `VideoUploadPage.tsx` — Hooks eintragen

- [x] 2.1 In `VideoUploadPage.tsx` das statische `headers`-Feld aus dem `new tus.Upload(...)`-Call entfernen. Kommentar drüber, warum es weg ist (Verweis auf Requirement „Access-Token wird pro Chunk-Request frisch aus dem Store gelesen"). Import `getAccessToken`, `refreshAccessToken`, `setAccessToken` aus `../lib/api`.
- [x] 2.2 `onBeforeRequest`-Hook ergänzen: `onBeforeRequest: (req) => { const t = getAccessToken(); if (t) req.setHeader('Authorization', 'Bearer ' + t) }`. Docstring 1-Zeile: „setzt Bearer-Token pro Request frisch aus dem Store (kein statisches headers-Objekt, weil Access-Token während langer Uploads rotiert)".
- [x] 2.3 `onShouldRetry`-Hook ergänzen: `async (err, retryAttempt, options) => { … }`. Logik:
    1. `const status = err.originalResponse?.getStatus?.()`
    2. Wenn `status !== 401`: `return retryAttempt < options.retryDelays.length` (Default-Retry-Verhalten für transiente Fehler).
    3. Wenn `status === 401`: `try { await refreshAccessToken(); return true } catch (refreshErr) { … }`.
    4. Im catch: `const refreshStatus = refreshErr?.response?.status`. Wenn `refreshStatus === 401` → `setAccessToken(null); window.location.href = '/login'; return false`. Wenn nicht 401 (Netzwerk/5xx) → `return retryAttempt < options.retryDelays.length` (kein Redirect, tus retryt normal, nächster Chunk-Refresh-Versuch nach `retryDelay`).
    - **Umsetzungs-Abweichung (tus 4.3.1):** `onShouldRetry` wurde **synchron** implementiert (nicht `async`). tus 4.x wertet den Rückgabewert direkt in `if (shouldRetry(...))` aus (`upload.js:456`); eine `async`-Funktion lieferte ein *immer truthy* Promise und hebelte `return false` (Abbruch) aus. Deshalb: Hook synchron → Refresh via `refreshAccessToken().catch(...)` fire-and-forget anstoßen; der Retry wartet den Single-Flight-Refresh in `onBeforeRequest` ab (tus **awaited** `onBeforeRequest`, `upload.js:982`) und trägt so den neuen Token. Refresh-401 setzt einen `sessionExpired`-Merker (Redirect + `setAccessToken(null)`), sodass ein Folge-401 sofort `false` liefert. Dafür `getPendingRefresh()` neu aus `lib/api.ts` exportiert. Alle Requirement-Garantien bleiben erfüllt.
- [x] 2.4 Sicherstellen, dass an ALLEN Aufruf-Stellen von `new tus.Upload(...)` in `VideoUploadPage.tsx` beide Hooks konsistent gesetzt sind (Datei enthält aktuell zwei: Upload und Probe-Instanz). Für die Probe-Instanz (findPreviousUploads) reicht ggf. nur `onBeforeRequest` — prüfen und entscheiden.
- [ ] 2.5 Manuell im Browser testen: mit devtools-Netzwerk-Filter auf `/api/videos/upload/` einen echten Upload starten, in devtools-Application → LocalStorage tus-Fingerprint sehen, während Upload läuft `setAccessToken('kaputt')` per devtools-Konsole aufrufen (`import(...).then(m => m.setAccessToken('kaputt'))`) und beobachten: erster PATCH danach = 401 → refresh-Request → Retry mit neuem Token = 204. Alternativ (weniger Aufwand): Access-Token-Ablauf serverseitig auf 30 s reduzieren, komplettes Video hochladen, prüfen dass es durchläuft.

## 3. Tests

- [x] 3.1 In `web/src/pages/__tests__/VideoUploadPage.test.tsx` erst prüfen, wie tus aktuell gemockt ist: `rg "tus" web/src/pages/__tests__/`. Wenn tus vollständig gemockt ist (jest/vitest mock module), müssen wir den Hook-Test isolieren: die Optionen-Objekte, die an `new tus.Upload(...)` übergeben werden, capturen und die Hooks direkt aufrufen. Wenn tus mit msw live läuft: msw-Handler für PATCH-Reihenfolge nutzen.
- [x] 3.2 Test „Token-Refresh mid-upload" schreiben (siehe Requirement-Szenario). Erster PATCH liefert 401, `/api/auth/refresh` liefert `{ access_token: 'neu' }`, zweiter PATCH (Retry) liefert 204. Assertions: Upload endet mit `onSuccess`, der Retry-PATCH trug `Authorization: Bearer neu`, `POST /api/auth/refresh` wurde genau EINmal aufgerufen.
- [x] 3.3 Test „Refresh-Token abgelaufen" schreiben. PATCH liefert 401, `/api/auth/refresh` liefert 401. Assertions: `onError` wurde mit einem Fehler aufgerufen, `setAccessToken(null)` wurde aufgerufen (Spy), `window.location.href` wurde auf `/login` gesetzt (via `Object.defineProperty(window, 'location', ...)` oder `vi.spyOn`).
- [x] 3.4 Beide Tests einzeln laufen lassen und beobachten, dass sie NUR mit dem neuen Code grün werden. Kontrollprobe: temporär den Fix in `VideoUploadPage.tsx` auskommentieren, Tests müssen dann rot werden — sonst testen sie nicht das richtige.
- [x] 3.5 `pnpm -C web test` gesamt grün.

## 4. Verifikation gegen produktive Größenordnung

- [ ] 4.1 Auf dem neuen Server (Testphase, `teamwerk.team-stuttgart.org`): Upload eines echten Videos > 1 GB durchführen. Log auf dem Server prüfen: PATCHes müssen 204 liefern; wenn ein 401 zwischendurch auftaucht, dann muss sofort danach ein 204 folgen (Retry mit neuem Token). Kein Upload-Abbruch.
- [ ] 4.2 Falls möglich: Access-Token-TTL im Backend temporär auf 60 s reduzieren (nur auf Test-Server), damit man während eines mittelgroßen Uploads mehrere Refresh-Zyklen sieht. TTL nach dem Test wieder auf 15 min setzen. **Nicht auf Prod.**

## 5. OpenSpec/Doku

- [x] 5.1 `openspec validate video-upload-token-refresh --strict` läuft grün.
- [x] 5.2 Nach Merge: `openspec archive video-upload-token-refresh` (bzw. `/opsx:archive`).
- [x] 5.3 Kurzen Nachtrag in `docs/agent/06-gotchas.md` unter neuem Abschnitt „Video-Upload Auth-Refresh" (2–3 Sätze: tus geht nicht durch Axios, deshalb eigene onBeforeRequest+onShouldRetry-Hooks in `VideoUploadPage.tsx` für Token-Refresh).
