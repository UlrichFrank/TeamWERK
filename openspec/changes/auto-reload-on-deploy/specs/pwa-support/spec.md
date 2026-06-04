## ADDED Requirements

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
