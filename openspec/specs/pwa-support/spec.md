# pwa-support Specification

## Purpose

Diese Spezifikation beschreibt die Capability `pwa-support`. (Automatisch normalisiert; Purpose bei Bedarf verfeinern.)

## Requirements

### Requirement: Installierbare Progressive Web App
TeamWERK SHALL als Progressive Web App installierbar sein. Browser MÜSSEN auf kompatiblen Geräten die „Zum Homescreen hinzufügen"-Option anzeigen.

#### Scenario: Installationsaufforderung im Browser
- **WHEN** ein Nutzer TeamWERK im Browser öffnet
- **WHEN** der Browser PWA-Installation unterstützt
- **THEN** zeigt der Browser die Option, die App zum Homescreen hinzuzufügen

#### Scenario: App startet im Standalone-Modus
- **WHEN** die App über den Homescreen-Icon gestartet wird
- **THEN** startet die App ohne Browser-Chrome (kein URL-Balken, kein Tab-Bar)
- **THEN** zeigt die App die Markenfarben in der Statusleiste (Theme-Color `#000000`)

### Requirement: Web App Manifest
Die App SHALL ein Web App Manifest (`/manifest.json`) bereitstellen.

#### Scenario: Manifest enthält alle Pflichtfelder
- **WHEN** das Manifest geladen wird
- **THEN** enthält es: `name: "TeamWERK"`, `short_name: "TeamWERK"`, `theme_color: "#000000"`, `background_color: "#FFFFFF"`, `display: "standalone"`, `start_url: "/"`

#### Scenario: App-Icons verfügbar
- **WHEN** das Manifest geladen wird
- **THEN** sind Icons in den Größen 192×192 und 512×512 als PNG verlinkt
- **THEN** ist das 512×512-Icon als `maskable` markiert (für Android-Adaptive Icons)

### Requirement: Service Worker mit Network-First-Strategie für API
Die App SHALL einen Service Worker registrieren. Reguläre API-Aufrufe (`/api/*`) MÜSSEN mit Network-First-Strategie gecacht werden: zuerst Netzwerk, bei Fehler Cache-Fallback. **SSE-Endpoints SHALL davon ausgenommen sein:** `/api/events` und `/api/chat/events` SHALL mit `NetworkOnly` ausgeliefert werden, damit long-lived `text/event-stream`-Verbindungen nicht von Workbox' Response-Klon-, Timeout- oder Cache-Logik beeinträchtigt werden. Die `NetworkOnly`-Routes für die SSE-Endpoints MÜSSEN in `sw.ts` **vor** der allgemeinen `/api/*`-`NetworkFirst`-Route registriert werden, damit das Routing-Match sie zuerst trifft.

#### Scenario: SSE-Endpoints laufen ungefiltert
- **WHEN** der Browser eine Verbindung zu `/api/events` oder `/api/chat/events` öffnet
- **THEN** wird die Verbindung vom Service Worker mit `NetworkOnly` direkt durchgereicht
- **THEN** versucht Workbox NICHT, den `text/event-stream`-Response zu cachen oder bei Timeout aus dem Cache zu beantworten

#### Scenario: API-Aufruf mit Netzwerk
- **WHEN** der Nutzer online ist und Daten abruft (nicht SSE)
- **THEN** liefert der Service Worker frische Daten vom Server
- **THEN** speichert der Service Worker die Antwort im Cache

#### Scenario: API-Aufruf ohne Netzwerk
- **WHEN** der Nutzer offline ist und eine Seite aufruft (nicht SSE)
- **THEN** liefert der Service Worker die zuletzt gecachte Antwort (sofern vorhanden)

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

### Requirement: Offline-Fallback-Seite
Die App SHALL bei fehlendem Netzwerk und fehlendem Cache eine Offline-Fallback-Seite (`/offline.html`) anzeigen.

#### Scenario: Keine Verbindung, kein Cache
- **WHEN** der Nutzer offline ist
- **WHEN** die angefragte Seite nicht gecacht ist
- **THEN** zeigt der Browser die Offline-Fallback-Seite mit dem Hinweis „Sie sind offline"
- **THEN** enthält die Seite das TeamWERK-Logo und Markenfarben

### Requirement: App-Icons aus Logo-SVG generiert
Die PWA-Icons SHALL aus dem vorhandenen TeamWERK-Logo-SVG generiert werden. Das Logo MUSS auf schwarzem (`#000000`) Hintergrund zentriert dargestellt werden mit 10% Safe-Zone-Padding für Maskable Icons.

#### Scenario: Icons sehen im App-Launcher korrekt aus
- **WHEN** die App auf dem Homescreen installiert ist
- **THEN** ist das Icon auf Android-Geräten rund zugeschnitten (Maskable)
- **THEN** ist das Logo auf dem Icon erkennbar und nicht abgeschnitten
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

