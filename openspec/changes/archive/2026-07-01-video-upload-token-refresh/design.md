## Context

`web/src/lib/api.ts` implementiert das TeamWERK-Auth-Modell:

- Modul-lokale Variable `accessToken: string | null` speichert den kurzlebigen JWT (15 min).
- `getAccessToken()`/`setAccessToken()` als Zugriff (kein React-State, damit auch außerhalb von Komponenten lesbar).
- Axios-Instanz hat einen Request-Interceptor, der den Token als Bearer-Header setzt, und einen Response-Interceptor, der bei 401 `POST /api/auth/refresh` aufruft, den neuen Token speichert und den Original-Request wiederholt.
- `refreshPromise: Promise<string> | null` als Single-Flight-Guard verhindert Duplicate-Refresh-Calls bei parallelen 401-Responses.

Der Video-Upload nutzt `tus-js-client@4.3.x` (siehe `web/package.json`). tus fährt eigene HTTP-Requests über `fetch`/`XHR`, nicht über Axios — der Auto-Refresh greift dort nicht. Aktueller Code (`VideoUploadPage.tsx` Zeile 176):

```ts
headers: { Authorization: `Bearer ${getAccessToken() ?? ''}` }
```

Das Objekt wird **einmal** beim `new tus.Upload(...)`-Aufruf ausgewertet. Alle PATCH-Chunks tragen denselben Token, bis der Upload endet oder — häufiger — der Token nach 15 min abläuft und der nächste Chunk 401 bekommt.

Beispiel aus dem Journal des neuen Servers am 2026-07-01 (Bootstrap-Tag):
```
17:34:58 → PATCH 204  (Offset 0 MB)      Chunk 1
17:35:15 → PATCH 204  (Offset 64 MB)     Chunk 2
…
17:38:20 → PATCH 204  (Offset 768 MB)    Chunk 12
17:38:34 → PATCH 401  in 245 µs          Auth-Middleware kickt Chunk 13 ab
```

## Goals / Non-Goals

**Goals:**

- Video-Uploads jeglicher Dauer laufen zuverlässig durch, solange der Refresh-Token (7 Tage) gültig ist.
- Wiederverwendung des existierenden Single-Flight-Refresh-Guards, damit paralleles Axios-401 und tus-401 sich nicht mit doppelten `/api/auth/refresh`-Calls beharken (Refresh-Tokens rotieren serverseitig — der zweite Call würde einen 401 kriegen).
- Klarer Fehler-Pfad, wenn der Refresh-Token selbst abgelaufen ist: Redirect auf `/login`, kein stiller Upload-Fehler.
- Keine neuen Abhängigkeiten.

**Non-Goals:**

- **Kein proaktiver Refresh vor JWT-Ablauf** (z. B. `setInterval`-Timer, der 60 s vor Expiry refresht). Kostenersparnis: eine deutlich einfachere Implementierung, Trade-off ist ein einzelner fehlgeschlagener Chunk pro Ablauf-Zyklus (~15 s Upload-Zeit, tus retryt idempotent).
- **Keine Refactoring anderer tus-Konsumenten**: `VideoUploadPage.tsx` ist aktuell die einzige Stelle im Frontend, die `tus-js-client` benutzt (`rg tus-js-client web/src`).
- **Keine Änderung des Backend-Auth-Verhaltens**: `/api/videos/upload/*` bleibt mit JWT-Bearer-Auth geschützt; `/api/auth/refresh` behält seine bisherige Semantik.
- **Kein neues Store-Mechanismus**: die vorhandene Modul-lokale `accessToken`-Variable in `api.ts` bleibt Single Source of Truth.

## Decisions

### D1 — `onBeforeRequest` statt dynamischer `headers`-Callback

`tus-js-client@4` unterstützt `headers` nur als statisches Objekt oder als Funktion, die pro *Upload-Instanz* aufgerufen wird — beides ist zu grobkörnig für unseren Fall (der Token soll pro *Request* frisch sein). Die dokumentierte, saubere Alternative ist `onBeforeRequest(req)`, ein Hook, der pro HTTP-Request aufgerufen wird und über `req.setHeader(name, value)` Header setzen kann.

```ts
onBeforeRequest: (req) => {
  const t = getAccessToken()
  if (t) req.setHeader('Authorization', `Bearer ${t}`)
}
```

**Alternative verworfen:** `upload.options.headers` nach jedem Refresh mutieren. Funktioniert, ist aber implizit und race-anfällig, weil `tus.Upload` intern Optionen kopieren kann.

### D2 — `onShouldRetry` liest Status via `err.originalResponse.getStatus()`

tus-js-client übergibt bei Fehlern ein `Error`-Objekt mit `originalRequest` und `originalResponse` (`HttpRequest`/`HttpResponse`-Interfaces). `originalResponse?.getStatus() === 401` ist der zuverlässige Weg, den HTTP-Status zu prüfen — die Message ist nicht standardisiert und nicht garantiert.

```ts
onShouldRetry: async (err, retryAttempt, options) => {
  const status = err.originalResponse?.getStatus?.()
  if (status !== 401) return retryAttempt < options.retryDelays.length
  try {
    await refreshAccessToken()   // aktualisiert Store, onBeforeRequest liest neuen Wert
    return true
  } catch {
    setAccessToken(null)
    window.location.href = '/login'
    return false
  }
}
```

Wichtig: Bei Non-401-Fehlern das **Default-Verhalten** von tus beibehalten (Retry mit `retryDelays`), sonst brechen transiente Netzwerkfehler den Upload ab. Der Vergleich `retryAttempt < options.retryDelays.length` reproduziert die Default-Logik.

**Alternative verworfen:** Alle 401 einfach `return true` ohne Refresh. Führt zu einem Endlos-Retry mit dem gleichen abgelaufenen Token — kein Fortschritt, nur mehr Log-Spam.

### D3 — `refreshAccessToken()` als eigene Export-Funktion in `api.ts`

Bisher steht die Refresh-Logik inline im Response-Interceptor. Zur Wiederverwendung im tus-Hook wird sie herausgezogen:

```ts
export function refreshAccessToken(): Promise<string> {
  if (!refreshPromise) {
    refreshPromise = axios
      .post('/api/auth/refresh', {}, { withCredentials: true })
      .then(res => {
        const t = res.data.access_token as string
        setAccessToken(t)
        return t
      })
      .finally(() => { refreshPromise = null })
  }
  return refreshPromise
}
```

Der Response-Interceptor nutzt danach dieselbe Funktion. Semantik unverändert; die Extraktion ist rein struktureller Refactor.

**Alternative verworfen:** Zweiter unabhängiger Refresh-Guard im tus-Hook. Bricht die Single-Flight-Garantie — bei parallelen 401 (Axios + tus) würden zwei `refresh`-Calls parallel raus, der zweite bekäme 401 (Refresh-Token wurde vom ersten rotiert und ist ungültig).

### D4 — Bei Refresh-Failure: Redirect auf `/login`, `return false`

Wenn `refreshAccessToken()` selbst wirft (Refresh-Token abgelaufen, Netzwerkfehler beim Refresh-Endpoint), wird:
1. `setAccessToken(null)` — verhindert weitere autorisierte Requests mit ungültigem Token.
2. `window.location.href = '/login'` — konsistent zum Axios-Interceptor-Verhalten in derselben Situation.
3. `return false` aus `onShouldRetry` — tus stoppt den Upload, ruft `onError` mit dem 401, User sieht den bekannten „Upload fehlgeschlagen"-Alert. Der Fingerprint bleibt im localStorage — nach neuem Login klickt der User „Upload fortsetzen?" und die tus-Session springt beim letzten erfolgreichen Chunk weiter.

**Alternative verworfen:** Nur `return false` ohne Redirect. Dann sitzt der User weiter auf der Upload-Seite mit einem Alert, ohne zu wissen, dass er ausgeloggt ist. Erst der nächste Klick würde ihn auf `/login` schicken (via Axios-Interceptor).

### D5 — Test-Setup: `msw` für `/api/auth/refresh`, tus-Mock für PATCH

Bestehende Tests in `VideoUploadPage.test.tsx` mocken tus vermutlich bereits (Konvention prüfen — falls nicht, sind alle Tests „nur oberflächlich"). Für die zwei neuen Fälle:

- **Test A (Token-Refresh mid-upload)**: msw-Handler antwortet auf ersten PATCH mit 401, auf zweiten mit 204. `/api/auth/refresh`-Handler liefert neuen Token. Assertion: Upload läuft durch, `refreshAccessToken` wurde einmal aufgerufen, zweiter PATCH trägt neuen Bearer.
- **Test B (Refresh selbst 401)**: `/api/auth/refresh`-Handler antwortet mit 401. Assertion: `onError` wurde mit 401 aufgerufen, `window.location.href` wurde auf `/login` gesetzt (via `vi.spyOn(window.location, 'href')` oder analog).

Falls tus in den bestehenden Tests real läuft (was mit einem File-Upload im jsdom schwierig ist), werden wir stattdessen die tus-Konfiguration extrahieren und die Hooks isoliert testen. Diese Entscheidung fällt bei der Umsetzung.

## Risks / Trade-offs

- **Trade-off:** ein Chunk (~15 s Upload-Zeit / ~64 MB) geht bei jedem Ablauf-Zyklus (alle 15 min) verloren. Bei 4h-Uploads sind das ~16 verlorene Chunks / ~1024 MB re-uploaded von insgesamt vielleicht 60 GB — vernachlässigbar. **Mitigation:** proaktives Refresh vor Ablauf wäre eine mögliche Folge-Change, aktuell nicht nötig.
- **Risiko:** wenn Backend `/api/auth/refresh` unter Last einen kurzzeitigen 5xx zurückgibt, würde `refreshAccessToken` werfen und der Upload bräche ab, obwohl der Refresh-Token an sich noch gültig ist. **Mitigation:** In `onShouldRetry` unterscheiden zwischen „Refresh warf mit 401" (Redirect) und „Refresh warf mit anderem Fehler" (kein Redirect, `return retryAttempt < options.retryDelays.length` — tus retryt normal, refresh wird nochmal versucht). Umsetzungs-Detail für die Implementierungs-Task.
- **Risiko:** die extrahierte `refreshAccessToken()`-Funktion ändert das Timing des Response-Interceptors minimal (Promise-Kette geht durch eine zusätzliche Funktion). **Mitigation:** Semantik ist identisch, bestehende `api.ts`-Tests laufen unverändert grün — falls sie brechen, hat der Refactor einen echten Bug entdeckt.
- **Trade-off:** die Test-Fälle setzen voraus, dass wir tus's `onBeforeRequest`/`onShouldRetry` verlässlich mocken oder das Hook-Verhalten isoliert testen können. Wenn die bestehende Test-Suite tus vollständig mockt (also der Hook nie live läuft), muss der Test das Hook-Verhalten direkt aufrufen — potenziell weniger integrativ, aber ausreichend als Regressions-Schutz.

## Migration Plan

Rein Frontend-Deploy: `make build && make deploy` schiebt den neuen Bundle raus, User sehen die Änderung nach dem nächsten Page-Load. Kein DB-Migrations-Bedarf, kein Backend-Restart-Bedarf (Handler unverändert).

**Rollback:** git-Revert des Change-Commits, `make deploy`. Fingerprints im localStorage der User werden nicht ungültig — tus setzt einfach mit dem alten (statischen-Header-)Verhalten fort.

## Open Questions

- **Bestehende Test-Suite:** wie tief mockt `VideoUploadPage.test.tsx` bereits `tus-js-client`? Antwort steuert D5 — bei vollem Mock testen wir die Hook-Funktionen isoliert, bei live-tus mit msw-Interception müssen wir File-Blobs in jsdom simulieren.
- **`retryDelays.length` vs. anderes Default-Kriterium:** tus-js-client v4 könnte die Retry-Anzahl nicht ausschließlich über `retryDelays.length` steuern — vor Umsetzung Doku prüfen und ggf. `retryAttempt < 3` oder `undefined` (= default) zurückgeben.
- **Zeitraum bis Follow-up:** wenn der proaktive Refresh doch noch eingebaut werden soll (weil der eine verlorene Chunk pro Ablauf-Zyklus stört), lohnt ein separater Change später — nicht Teil dieses Fixes.
