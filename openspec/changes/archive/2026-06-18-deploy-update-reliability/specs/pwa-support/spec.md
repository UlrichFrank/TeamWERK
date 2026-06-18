## MODIFIED Requirements

### Requirement: Service Worker mit Cache-First-Strategie für statische Assets

Statische Assets (JS-Bundles, CSS, Schriften, Bilder unter `assets/` und `icons/`) SHALL der Service Worker via Workbox-Precache mit Cache-First-Strategie bedienen: zuerst Cache, bei Cache-Miss Netzwerk. HTML-Dateien — insbesondere `index.html` — SHALL **nicht** im Workbox-Precache enthalten sein; Navigationen werden über die separate `NetworkFirst`-Route (Cache `app-shell`) abgewickelt (siehe Capability `app-update-reliability`).

#### Scenario: Statisches Asset aus Cache
- **WHEN** der Nutzer die App lädt
- **WHEN** das Asset bereits im Workbox-Precache ist
- **THEN** wird es sofort aus dem Cache geladen ohne Netzwerkanfrage

#### Scenario: index.html ist nicht im Precache
- **WHEN** der Service Worker installiert
- **THEN** enthält das Precache-Manifest keine `.html`-Dateien
- **THEN** wird `index.html` ausschließlich über die `NetworkFirst`-Navigationsroute geliefert

### Requirement: Service Worker meldet erkannte Updates an die App

Der Service Worker SHALL erkannte Updates (neues Bundle nach `make deploy`) via `onNeedRefresh`-Callback an die App melden. Die App SHALL bei Erhalt dieses Callbacks denselben Update-Banner anzeigen wie bei SSE-basierter Versionserkennung. Der Reload-Flow SHALL beim Klick auf „Jetzt laden" sicherstellen, dass die App den neuen Stand sieht — auch wenn der neue Service Worker zum Klick-Zeitpunkt noch nicht im `waiting`-State ist:

1. Wenn `registration.waiting` direkt verfügbar ist: `SKIP_WAITING` senden, auf `controllerchange` warten, dann `location.reload()`.
2. Sonst: `registration.update()` aufrufen, bis ~5 s auf `registration.waiting` pollen.
3. Wenn nach dem Poll ein `waiting`-SW da ist: weiter wie in (1).
4. Wenn weiterhin keiner: alle Caches mit Namen-Präfix `workbox-precache` löschen, `app-shell`-Cache löschen, `api-cache`-Cache löschen, dann `location.reload()`. Andere Caches (Google Fonts) bleiben unangetastet.

#### Scenario: SW erkennt neues Bundle und zeigt Banner

- **WHEN** nach einem Deployment ein neuer Service Worker heruntergeladen wurde
- **WHEN** der SW in den `waiting`-State übergeht
- **THEN** ruft Workbox `onNeedRefresh` auf
- **THEN** zeigt die App den Update-Banner „Neue Version verfügbar"

#### Scenario: Reload mit bereits wartendem SW

- **WHEN** der Nutzer im Banner auf „Jetzt laden" klickt
- **WHEN** `registration.waiting` zu diesem Zeitpunkt einen wartenden SW enthält
- **THEN** wird `SKIP_WAITING` an den wartenden SW gesendet
- **THEN** lädt die Seite nach `controllerchange` neu

#### Scenario: Reload, wenn SW noch kein Update gefunden hat

- **WHEN** der Nutzer „Jetzt laden" klickt
- **WHEN** `registration.waiting` zu diesem Zeitpunkt `null` ist (SSE hat den Update früher gemeldet als der SW)
- **THEN** ruft die App `registration.update()` auf
- **THEN** pollt die App bis zu 5 s auf `registration.waiting`
- **WHEN** innerhalb des Timeouts ein `waiting`-SW erscheint
- **THEN** läuft der `SKIP_WAITING`+Reload-Pfad

#### Scenario: Reload-Fallback, wenn kein neuer SW kommt

- **WHEN** der Nutzer „Jetzt laden" klickt
- **WHEN** auch nach 5 s `registration.waiting` weiterhin `null` ist
- **THEN** löscht die App alle Caches mit Namen-Präfix `workbox-precache`
- **THEN** löscht die App den `app-shell`-Cache
- **THEN** löscht die App den `api-cache`-Cache
- **THEN** ruft die App `location.reload()` auf
- **THEN** sieht der Nutzer weder einen veralteten Precache-Shell noch veraltete API-Daten

#### Scenario: Fallback lässt Google-Fonts-Caches unberührt

- **WHEN** der Fallback-Pfad ausgeführt wird
- **THEN** wird der Cache `google-fonts-cache` NICHT gelöscht
- **THEN** wird der Cache `google-fonts-static-cache` NICHT gelöscht

#### Scenario: PWA Standalone erkennt Update beim App-Start

- **WHEN** die installierte PWA nach einem Deployment neu gestartet wird
- **WHEN** Netzwerkzugang besteht
- **THEN** liefert der Service Worker die neue `index.html` direkt aus der NetworkFirst-Route, ohne dass der Banner geklickt werden muss
