## ADDED Requirements

### Requirement: HTTP-Cache-Header für statische Assets

Der Backend-Handler, der `web/dist/`-Dateien ausliefert (`spaHandler` in `cmd/teamwerk/main.go`), SHALL pfadabhängige `Cache-Control`-Header setzen. Dateien unter `assets/` mit einem Content-Hash im Dateinamen (Muster `assets/<name>-<hash>.<ext>`, mindestens 8 Zeichen Hash aus `[A-Za-z0-9_-]`) SHALL den Header `Cache-Control: public, max-age=31536000, immutable` erhalten. Alle anderen Dateien — insbesondere `index.html`, `sw.js`, `manifest.webmanifest`, `manifest.json`, `registerSW.js` und alle Dateien unter `icons/` — SHALL den Header `Cache-Control: no-cache, must-revalidate` erhalten.

#### Scenario: Hashed JS-Bundle wird langfristig gecacht
- **WHEN** der Client `GET /assets/index-AbCd1234.js` aufruft
- **THEN** liefert der Server die Datei mit Header `Cache-Control: public, max-age=31536000, immutable`

#### Scenario: index.html wird immer revalidiert
- **WHEN** der Client `GET /` oder `GET /index.html` aufruft
- **THEN** liefert der Server die Datei mit Header `Cache-Control: no-cache, must-revalidate`

#### Scenario: Service Worker wird immer revalidiert
- **WHEN** der Client `GET /sw.js` aufruft
- **THEN** liefert der Server die Datei mit Header `Cache-Control: no-cache, must-revalidate`

#### Scenario: Manifest wird immer revalidiert
- **WHEN** der Client `GET /manifest.webmanifest` aufruft
- **THEN** liefert der Server die Datei mit Header `Cache-Control: no-cache, must-revalidate`

### Requirement: ETag-basierte Revalidierung für nicht-hashed Dateien

Für alle Dateien mit `Cache-Control: no-cache, must-revalidate` SHALL der Handler einen ETag-Header setzen, der sich beim nächsten Deployment ändert. Der ETag-Wert SHALL aus dem aktuellen `buildHash` (Git-Commit-SHA, der ohnehin via `-ldflags` injiziert wird) und einer kurzen, deterministischen Ableitung aus dem angefragten Pfad gebildet werden. Bei `If-None-Match`-Header im Request mit identischem ETag-Wert SHALL der Handler mit Status `304 Not Modified` und ohne Response-Body antworten.

#### Scenario: Erster Request liefert ETag
- **WHEN** der Client `GET /index.html` ohne `If-None-Match` aufruft
- **THEN** liefert der Server Status 200 mit Datei-Body und Header `ETag: "<buildHash>-<pathHash>"`

#### Scenario: Conditional Request mit gleichem ETag liefert 304
- **WHEN** der Client `GET /index.html` mit `If-None-Match: "<aktueller-ETag>"` aufruft
- **THEN** liefert der Server Status 304 ohne Response-Body
- **THEN** ist der `Content-Length` 0 oder fehlt

#### Scenario: Nach Deploy unterscheidet sich der ETag
- **WHEN** der `buildHash` sich nach `make deploy` ändert
- **WHEN** der Client den alten ETag via `If-None-Match` mitschickt
- **THEN** liefert der Server Status 200 mit neuem Body und neuem ETag

### Requirement: Navigationen laufen via NetworkFirst

Der Service Worker SHALL Navigationsrequests (`request.mode === 'navigate'`) über die Workbox-Strategie `NetworkFirst` mit Cache-Name `app-shell` und Network-Timeout von 3 Sekunden ausliefern. Diese Route MUSS vor allen anderen Routen registriert werden, damit das Routing-Match sie zuerst trifft. Der zugehörige Workbox-Cache SHALL maximal einen Eintrag halten (die jeweils letzte erfolgreich geladene Shell).

#### Scenario: Online-Cold-Start lädt aktuelle Shell vom Netz
- **WHEN** die installierte PWA mit Netzverbindung gestartet wird
- **WHEN** der Server innerhalb von 3 Sekunden antwortet
- **THEN** liefert der Service Worker die frische `index.html` aus der Netz-Response
- **THEN** aktualisiert der Service Worker den `app-shell`-Cache mit dieser Response

#### Scenario: Offline-Cold-Start fällt auf gecachte Shell zurück
- **WHEN** die installierte PWA ohne Netzverbindung gestartet wird
- **WHEN** der `app-shell`-Cache eine zuvor gespeicherte Shell enthält
- **THEN** liefert der Service Worker die gecachte Shell aus

#### Scenario: Langsames Netz nach 3 s Timeout fällt auf Cache zurück
- **WHEN** die installierte PWA gestartet wird
- **WHEN** der Server nicht innerhalb von 3 Sekunden antwortet
- **WHEN** der `app-shell`-Cache eine zuvor gespeicherte Shell enthält
- **THEN** liefert der Service Worker die gecachte Shell aus

### Requirement: index.html ist nicht im Workbox-Precache

Das Workbox-Precache-Manifest (`__WB_MANIFEST`, gebaut von `vite-plugin-pwa` über `injectManifest.globPatterns`) SHALL keine HTML-Dateien enthalten. Hashed Asset-Dateien (`*.js`, `*.css`, `*.woff2`, Icons, statische Bilder) bleiben im Precache.

#### Scenario: Precache-Manifest enthält kein .html
- **WHEN** der Build (`pnpm build`) das Service-Worker-Bundle erzeugt
- **THEN** taucht im resultierenden `__WB_MANIFEST`-Array keine `.html`-Datei auf

#### Scenario: Hashed Assets bleiben im Precache
- **WHEN** der Build das Service-Worker-Bundle erzeugt
- **THEN** sind alle Dateien aus `assets/` (JS, CSS) im `__WB_MANIFEST`

### Requirement: Reload-Fallback löscht Precache und App-Shell-Cache

Wenn der Klick auf "Jetzt laden" weder einen `registration.waiting` findet noch innerhalb von 5 Sekunden via `registration.update()` einen produziert, SHALL die App im Fallback-Pfad zusätzlich zum bestehenden `api-cache` auch alle Caches mit Namen-Präfix `workbox-precache` sowie den `app-shell`-Cache löschen, bevor sie `location.reload()` aufruft. Andere Caches (insbesondere Google-Fonts-Caches) SHALL unangetastet bleiben.

#### Scenario: Fallback löscht workbox-precache und app-shell
- **WHEN** der Nutzer "Jetzt laden" klickt
- **WHEN** weder waiting SW noch `registration.update()` einen neuen SW liefern
- **THEN** ruft die App `caches.delete()` für alle Cache-Namen mit Präfix `workbox-precache` auf
- **THEN** ruft die App `caches.delete('app-shell')` auf
- **THEN** ruft die App `caches.delete('api-cache')` auf
- **THEN** ruft die App `location.reload()` auf

#### Scenario: Fallback lässt Google-Fonts-Caches unberührt
- **WHEN** der Fallback-Pfad ausgeführt wird
- **THEN** wird der Cache `google-fonts-cache` NICHT gelöscht
- **THEN** wird der Cache `google-fonts-static-cache` NICHT gelöscht

### Requirement: Update nach Deploy ohne Logout-Workaround

Nach `make deploy` SHALL jeder eingeloggte Nutzer mit Netzverbindung innerhalb derselben Session die neue Version erhalten, ohne sich ausloggen oder die PWA komplett beenden zu müssen. Mindestens einer der folgenden Pfade MUSS dafür greifen: (a) Klick auf den "Jetzt laden"-Banner, (b) Cold-Start der PWA nach Hintergrundphase.

#### Scenario: "Jetzt laden" liefert neue Version
- **WHEN** nach einem Deploy der SSE-`__version:`-Hash oder workbox' `onNeedRefresh` den Banner triggert
- **WHEN** der Nutzer "Jetzt laden" klickt
- **THEN** lädt die App die neue Version (neuer `buildHash` im Footer sichtbar)

#### Scenario: PWA-Cold-Start lädt neue Version vom Netz
- **WHEN** nach einem Deploy der Nutzer die PWA aus dem Hintergrund holt oder vom Homescreen neu öffnet
- **WHEN** Netzverbindung besteht
- **THEN** liefert der Service Worker die neue `index.html` aus dem Netz (NetworkFirst)
- **THEN** sieht der Nutzer die neue Version, ohne den Banner klicken zu müssen
