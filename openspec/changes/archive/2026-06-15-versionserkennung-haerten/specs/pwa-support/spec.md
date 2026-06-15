## MODIFIED Requirements

### Requirement: Service Worker mit Network-First-Strategie für API

Die App SHALL einen Service Worker registrieren. Reguläre API-Aufrufe (`/api/*`) SHALL mit Network-First-Strategie gecacht werden: zuerst Netzwerk, bei Fehler Cache-Fallback. **SSE-Endpoints SHALL davon ausgenommen sein:** `/api/events` und `/api/chat/events` SHALL mit `NetworkOnly` ausgeliefert werden, damit long-lived `text/event-stream`-Verbindungen nicht von Workbox' Response-Klon-, Timeout- oder Cache-Logik beeinträchtigt werden. Die `NetworkOnly`-Routes für die SSE-Endpoints MÜSSEN in `sw.ts` **vor** der allgemeinen `/api/*`-`NetworkFirst`-Route registriert werden, damit das Routing-Match sie zuerst trifft.

#### Scenario: SSE-Endpoints laufen ungefiltert

- **WHEN** der Browser eine Verbindung zu `/api/events` oder `/api/chat/events` öffnet
- **THEN** wird die Verbindung vom Service Worker mit `NetworkOnly` direkt durchgereicht
- **THEN** versucht Workbox NICHT, den `text/event-stream`-Response zu cachen oder bei Timeout aus dem Cache zu beantworten

#### Scenario: API-Aufruf mit Netzwerk (unverändert)

- **WHEN** der Nutzer online ist und Daten abruft (nicht SSE)
- **THEN** liefert der Service Worker frische Daten vom Server
- **THEN** speichert der Service Worker die Antwort im Cache

#### Scenario: API-Aufruf ohne Netzwerk (unverändert)

- **WHEN** der Nutzer offline ist und eine Seite aufruft (nicht SSE)
- **THEN** liefert der Service Worker die zuletzt gecachte Antwort (sofern vorhanden)

### Requirement: Service Worker meldet erkannte Updates an die App

Der Service Worker SHALL erkannte Updates (neues Bundle nach `make deploy`) via `onNeedRefresh`-Callback an die App melden. Die App SHALL bei Erhalt dieses Callbacks denselben Update-Banner anzeigen wie bei SSE-basierter Versionserkennung. Der Reload-Flow SHALL beim Klick auf „Jetzt laden" sicherstellen, dass die App den neuen Stand sieht — auch wenn der neue Service Worker zum Klick-Zeitpunkt noch nicht im `waiting`-State ist:

1. Wenn `registration.waiting` direkt verfügbar ist: `SKIP_WAITING` senden, auf `controllerchange` warten, dann `location.reload()` (bisheriges Verhalten).
2. Sonst: `registration.update()` aufrufen, bis ~5 s auf `registration.waiting` pollen.
3. Wenn nach dem Poll ein `waiting`-SW da ist: weiter wie in (1).
4. Wenn weiterhin keiner: `caches.delete('api-cache')` ausführen, dann `location.reload()`.

#### Scenario: SW erkennt neues Bundle und zeigt Banner (unverändert)

- **WHEN** nach einem Deployment ein neuer Service Worker heruntergeladen wurde
- **WHEN** der SW in den `waiting`-State übergeht
- **THEN** ruft Workbox `onNeedRefresh` auf
- **THEN** zeigt die App den Update-Banner „Neue Version verfügbar"

#### Scenario: Reload mit bereits wartendem SW (unverändert)

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
- **THEN** löscht die App den `api-cache`-Cache via `caches.delete('api-cache')`
- **THEN** ruft die App `location.reload()` auf
- **THEN** sieht die Nutzerin keinen veralteten `index.html`-Precache mehr für API-Daten

#### Scenario: PWA Standalone erkennt Update beim App-Start (unverändert)

- **WHEN** die installierte PWA nach einem Deployment neu gestartet wird
- **WHEN** Netzwerkzugang besteht
- **THEN** prüft der SW beim Start auf Updates
- **THEN** erscheint bei verfügbarem Update der Banner beim nächsten Seitenbesuch
