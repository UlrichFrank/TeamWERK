# video-upload-token-refresh Specification

## Purpose
TBD - created by archiving change video-upload-token-refresh. Update Purpose after archive.
## Requirements
### Requirement: Access-Token wird pro Chunk-Request frisch aus dem Store gelesen
Der tus-Video-Upload-Client MUST/MUSS vor jedem HTTP-Request (POST, HEAD, PATCH) den aktuellen Access-Token aus dem geteilten Auth-Store (`getAccessToken()` in `web/src/lib/api.ts`) lesen und als `Authorization: Bearer <token>`-Header setzen. Ein statisches `headers`-Objekt beim Konstruktor-Aufruf ist NICHT zulässig, weil es den Wert beim Upload-Start einfriert und Token-Rotationen während des Uploads ignoriert.

#### Scenario: Frischer Token bei jedem Chunk
- **WHEN** ein Upload mehrere PATCH-Chunks nacheinander sendet und zwischen zwei Chunks der Store per `setAccessToken(neu)` aktualisiert wurde (z. B. durch parallelen Axios-Refresh)
- **THEN** trägt der nächste PATCH-Chunk den neuen Token, ohne dass tus neu initialisiert werden musste

#### Scenario: Kein Token im Store
- **WHEN** `getAccessToken()` zum Zeitpunkt eines Requests `null` liefert
- **THEN** wird der `Authorization`-Header nicht gesetzt (statt Bearer-Leer-String), damit der Server eine klare Auth-Fehlermeldung liefert und nicht mit einem ungültigen Bearer verwirrt wird

---

### Requirement: 401 auf einem Chunk löst Token-Refresh und Retry aus
Wenn ein Chunk-Request mit HTTP 401 abgelehnt wird, MUST/MUSS der tus-Client `refreshAccessToken()` (aus `web/src/lib/api.ts`) aufrufen und den fehlgeschlagenen Chunk anschließend automatisch retryen. Der Retry MUST/MUSS den neuen Token verwenden (über den Per-Request-Header-Hook aus dem vorherigen Requirement).

#### Scenario: Access-Token läuft mitten im Upload ab
- **WHEN** ein PATCH-Chunk 401 zurückgibt und `refreshAccessToken()` gibt einen neuen Token zurück
- **THEN** wird der Chunk mit dem neuen Bearer-Token wiederholt und liefert 204, der Upload läuft ohne User-Sichtbarkeit weiter

#### Scenario: Refresh-Endpoint transient nicht erreichbar
- **WHEN** `refreshAccessToken()` mit einem Nicht-401-Fehler wirft (Netzwerkfehler, 5xx)
- **THEN** wird der Upload NICHT abgebrochen — tus retryt nach der normalen `retryDelays`-Strategie (so als wäre es ein transienter Fehler ohne 401), damit der nächste Refresh-Versuch nach ein paar Sekunden erfolgen kann

---

### Requirement: Refresh-Token abgelaufen bricht Upload sauber ab und leitet auf Login um
Wenn `refreshAccessToken()` selbst mit HTTP 401 wirft (also der Refresh-Token abgelaufen ist), MUST/MUSS der tus-Client:
1. `setAccessToken(null)` aufrufen, damit keine weiteren Requests einen ungültigen Token tragen,
2. `window.location.href = '/login'` setzen (analog zum Axios-Response-Interceptor in `api.ts`),
3. `false` aus dem Retry-Hook zurückgeben, damit tus den Upload beendet und `onError` mit dem 401-Fehler auslöst.

#### Scenario: Refresh-Token 7 Tage abgelaufen
- **WHEN** `refreshAccessToken()` mit einem 401 wirft (weil der Refresh-Cookie abgelaufen ist)
- **THEN** wird der Access-Token gelöscht, der Browser wird auf `/login` navigiert, und der Upload beendet sich mit einer nutzerfreundlichen Fehlermeldung

#### Scenario: tus-Fingerprint überlebt für Wiederaufnahme
- **WHEN** der Upload wegen Refresh-401 abgebrochen wurde und der User sich neu einloggt
- **THEN** bietet die Upload-Seite dem User „Upload fortsetzen?" an (`storeFingerprintForResuming: true` bleibt aktiv), und tus fährt beim letzten erfolgreichen Chunk weiter

---

### Requirement: Refresh-Requests werden über einen In-Flight-Guard dedupliziert
`refreshAccessToken()` MUST/MUSS das bereits vorhandene Single-Flight-Muster (`refreshPromise`-Variable in `web/src/lib/api.ts`) wiederverwenden, damit parallele Refresh-Aufrufe aus dem Axios-Interceptor und dem tus-Hook auf dieselbe Promise warten. Es DARF (MUST NOT) NICHT zwei parallele `POST /api/auth/refresh`-Requests entstehen, weil der Server den Refresh-Token nach dem ersten Call rotiert und der zweite dann fälschlich 401 bekäme.

#### Scenario: Gleichzeitiger Refresh von Axios und tus
- **WHEN** ein Axios-Request und ein tus-Chunk zur exakt gleichen Zeit 401 zurückbekommen und beide den Refresh anstoßen
- **THEN** wird `POST /api/auth/refresh` genau EINmal an den Server geschickt, beide Callers warten auf dieselbe Promise, beide bekommen denselben neuen Token

#### Scenario: Refresh-Promise wird nach Erfolg zurückgesetzt
- **WHEN** ein Refresh erfolgreich zurückgekommen ist und ein späterer 401 einen neuen Refresh anstößt
- **THEN** wird ein neuer `POST /api/auth/refresh` geschickt (nicht der alte Promise wiederverwendet — sonst würde ein einmal erfolgreicher Refresh die nächsten 15 min „gecached" bleiben)

---

### Requirement: `refreshAccessToken` ist eine exportierte Funktion in `api.ts`
`web/src/lib/api.ts` MUST/MUSS eine `refreshAccessToken(): Promise<string>` als Modul-Export bereitstellen, die dieselbe Logik wie der bisherige Response-Interceptor-Body ausführt (Single-Flight, `setAccessToken` mit dem Ergebnis). Der Response-Interceptor MUST/MUSS diese Funktion wiederverwenden, damit es nur EINE Quelle der Wahrheit für das Refresh-Verhalten gibt.

#### Scenario: Interceptor nutzt dieselbe Funktion
- **WHEN** ein Axios-Request 401 bekommt und der Interceptor refreshen will
- **THEN** ruft der Interceptor `refreshAccessToken()` auf (nicht mehr eine inline aufgebaute Promise) und retryt mit dem zurückgegebenen Token

#### Scenario: Test kann Refresh isoliert triggern
- **WHEN** ein Unit-Test die Funktion direkt importiert und aufruft
- **THEN** wird `POST /api/auth/refresh` gesendet und `accessToken` im Modul aktualisiert (beobachtbar via `getAccessToken()`)

---

### Requirement: Regressions-Tests für Token-Refresh-Mid-Upload und Refresh-Failure
`web/src/pages/__tests__/VideoUploadPage.test.tsx` MUST/MUSS zwei zusätzliche Testfälle enthalten, die die neuen Hooks abdecken:

1. **„Access-Token läuft mitten im Upload ab"**: Mock antwortet auf den ersten PATCH mit 401, auf `/api/auth/refresh` mit einem neuen Token, auf den PATCH-Retry mit 204. Assertion: Upload wird als erfolgreich gemeldet (`onSuccess` gerufen), `refreshAccessToken` wurde genau EINmal aufgerufen, der Retry-PATCH trägt den neuen Token.
2. **„Refresh-Token abgelaufen"**: Mock antwortet auf `/api/auth/refresh` mit 401. Assertion: `onError` wurde mit einem 401 aufgerufen, `setAccessToken(null)` wurde aufgerufen, Redirect auf `/login` wurde ausgelöst.

#### Scenario: Mid-Upload-Refresh-Test läuft grün
- **WHEN** die Testsuite den Mid-Upload-Refresh-Fall ausführt
- **THEN** bestätigt sie, dass der Upload trotz zwischenzeitlichem 401 als erfolgreich abschließt und der zweite Chunk den neuen Token trägt

#### Scenario: Refresh-Failure-Test läuft grün
- **WHEN** die Testsuite den Refresh-Failure-Fall ausführt
- **THEN** bestätigt sie, dass der Upload sauber abbricht und der Browser auf `/login` navigiert

