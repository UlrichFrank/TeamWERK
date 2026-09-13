## REMOVED Requirements

### Requirement: Access-Token wird pro Chunk-Request frisch aus dem Store gelesen
**Reason**: Diese Anforderung beschrieb ausschließlich das Verhalten von `web/src/pages/VideoUploadPage.tsx` und des geteilten Browser-Auth-Stores (`web/src/lib/api.ts`). Die Seite entfällt mit diesem Change — Video-Uploads laufen ausschließlich über das Offline-Encoding-Tool.
**Migration**: Äquivalentes Verhalten (frischer Token pro Request) ist jetzt Teil von `video-offline-encoding-tool`, Requirement „Token-Refresh während eines langen Uploads", dort aber über einen `http.Client`-Cookie-Jar statt eines Browser-Auth-Stores realisiert.

### Requirement: 401 auf einem Chunk löst Token-Refresh und Retry aus
**Reason**: Browser-spezifisches tus-Hook-Verhalten von `VideoUploadPage.tsx`, die mit diesem Change entfällt.
**Migration**: Siehe `video-offline-encoding-tool`, Requirement „Token-Refresh während eines langen Uploads" — dieselbe Grundidee (401 löst Refresh + Retry desselben Chunks aus), jetzt im Tool statt im Browser implementiert.

### Requirement: Refresh-Token abgelaufen bricht Upload sauber ab und leitet auf Login um
**Reason**: Der Redirect auf `/login` ist ein Browser-Konzept und setzt `VideoUploadPage.tsx` voraus, die entfällt.
**Migration**: Siehe `video-offline-encoding-tool`, Requirement „Token-Refresh während eines langen Uploads" — das Tool bricht bei abgelaufenem Refresh-Token den Upload ab und fordert zur erneuten Anmeldung im Tool selbst auf (kein Browser-Redirect).

### Requirement: Refresh-Requests werden über einen In-Flight-Guard dedupliziert
**Reason**: Der beschriebene Single-Flight-Guard (`refreshPromise` in `web/src/lib/api.ts`) ist spezifisch für das gemeinsame Nutzen von Axios-Interceptor und Browser-tus-Client, die mit `VideoUploadPage.tsx` entfällt.
**Migration**: Das Tool hat keinen parallelen Axios-Interceptor, gegen den dedupliziert werden müsste — es besteht nur ein einzelner Upload-Prozess pro Tool-Instanz, ein Single-Flight-Guard ist dort nicht erforderlich.

### Requirement: `refreshAccessToken` ist eine exportierte Funktion in `api.ts`
**Reason**: Reine Implementierungsanforderung an das Browser-Frontend-Modul `web/src/lib/api.ts`, die mit dem Wegfall des Browser-Video-Uploads für diesen Anwendungsfall gegenstandslos wird (die Funktion bleibt für andere Axios-Aufrufe im Frontend bestehen, ist aber nicht mehr Teil dieser Capability).
**Migration**: Keine — das Tool implementiert seine eigene Refresh-Logik in Go, siehe `video-offline-encoding-tool`.

### Requirement: Regressions-Tests für Token-Refresh-Mid-Upload und Refresh-Failure
**Reason**: Die referenzierte Testdatei `web/src/pages/__tests__/VideoUploadPage.test.tsx` entfällt mit der Seite selbst.
**Migration**: Äquivalente Regressionstests entstehen im Tool-Repository (Go-Tests für den Refresh-Retry-Pfad), siehe Tasks von `video-offline-encoding-tool`.
