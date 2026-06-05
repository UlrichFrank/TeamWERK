## ADDED Requirements

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
Die App SHALL einen Service Worker registrieren. API-Aufrufe (`/api/*`) MÜSSEN mit Network-First-Strategie gecacht werden: zuerst Netzwerk, bei Fehler Cache-Fallback.

#### Scenario: API-Aufruf mit Netzwerk
- **WHEN** der Nutzer online ist und Daten abruft
- **THEN** liefert der Service Worker frische Daten vom Server
- **THEN** speichert der Service Worker die Antwort im Cache

#### Scenario: API-Aufruf ohne Netzwerk
- **WHEN** der Nutzer offline ist und eine Seite aufruft
- **THEN** liefert der Service Worker die zuletzt gecachte Antwort (sofern vorhanden)

### Requirement: Service Worker mit Cache-First-Strategie für statische Assets
Statische Assets (JS-Bundles, CSS, Schriften, Bilder) SHALL der Service Worker mit Cache-First-Strategie bedienen: zuerst Cache, bei Cache-Miss Netzwerk.

#### Scenario: Statisches Asset aus Cache
- **WHEN** der Nutzer die App lädt
- **WHEN** das Asset bereits gecacht ist
- **THEN** wird es sofort aus dem Cache geladen ohne Netzwerkanfrage

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
## Requirements
### Requirement: Service Worker meldet erkannte Updates an die App

Der Service Worker SHALL erkannte Updates (neues Bundle nach `make deploy`) via `onNeedRefresh`-Callback an die App melden. Die App SHALL bei Erhalt dieses Callbacks denselben Update-Banner anzeigen wie bei SSE-basierter Versionserkennung.

#### Scenario: SW erkennt neues Bundle und zeigt Banner

- **WHEN** nach einem Deployment ein neuer Service Worker heruntergeladen wurde
- **WHEN** der SW in den `waiting`-State übergeht
- **THEN** ruft Workbox `onNeedRefresh` auf
- **THEN** zeigt die App den Update-Banner „Neue Version verfügbar"

#### Scenario: Klick auf Reload aktiviert neuen Service Worker

- **WHEN** der Nutzer im Banner auf „Jetzt neu laden" klickt
- **THEN** wird `updateServiceWorker(true)` aufgerufen
- **THEN** aktiviert der neue SW sofort (`skipWaiting`) und lädt die Seite neu

#### Scenario: PWA Standalone erkennt Update beim App-Start

- **WHEN** die installierte PWA nach einem Deployment neu gestartet wird
- **WHEN** Netzwerkzugang besteht
- **THEN** prüft der SW beim Start auf Updates
- **THEN** erscheint bei verfügbarem Update der Banner beim nächsten Seitenbesuch

