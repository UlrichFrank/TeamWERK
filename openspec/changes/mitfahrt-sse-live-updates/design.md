## Context

Die Mitfahrgelegenheiten-Seite lädt Daten aktuell nur einmalig beim Mount. Das 15-Sekunden-Polling ist ein Workaround mit Latenz und unnötigen Requests. SSE ist der minimale, saubere Ansatz: eine offene HTTP-Verbindung, über die der Server ein Signal sendet, sobald eine Mutation stattfindet.

## Goals / Non-Goals

**Goals:**
- Sofortige Aktualisierung bei Mutations (Eintrag anlegen/löschen, Paarung anfragen/bestätigen/ablehnen)
- Minimale Server-Last (Goroutinen sind günstig, kein DB-Polling)
- Keine neuen Dependencies

**Non-Goals:**
- Kein Streaming von Daten über SSE (nur Signal "refresh", Daten kommen weiterhin per REST)
- Kein WebSocket (unnötige Bidirektionalität)
- Kein SSE für andere Seiten als Mitfahrgelegenheiten

## Decisions

### D1: EventHub als in-memory Pub/Sub

```
EventHub
  mu      sync.Mutex
  clients map[chan struct{}]struct{}

Subscribe()  → chan struct{}   // client registrieren
Unsubscribe(ch)               // client entfernen
Broadcast()                   // allen senden (non-blocking)
```

`Broadcast()` verwendet `select`+`default` damit ein langsamer Client den Hub nicht blockiert. Der Hub lebt als Feld im `carpooling.Handler` und wird bei `NewHandler` erzeugt.

**Alternative:** Redis Pub/Sub oder Channel-basierter Event-Bus mit separatem Package — abgelehnt, weil über-engineered für die Last (wenige simultane Nutzer, single-process Deployment).

### D2: SSE-Endpoint ohne eigenen Auth-Middleware, aber mit JWT als Query-Parameter

`EventSource` im Browser unterstützt keine Custom-Header. Da der SSE-Stream ausschließlich das Signal `data: refresh\n\n` sendet (keine Nutzerdaten), wäre ein öffentlicher Endpoint vertretbar — dennoch wird Auth beibehalten um unbegrenzte offene Verbindungen von Nicht-Nutzern zu verhindern.

Lösung: `auth.Middleware` prüft neben dem `Authorization`-Header auch den Query-Parameter `?token=<jwt>`. Frontend ruft `getAccessToken()` (neue Hilfsfunktion in `lib/api.ts`) auf und hängt den Token an die URL.

**Alternative:** Separater kurzlebiger SSE-Token — abgelehnt, zu komplex für den Nutzen.

### D3: 30-Sekunden Keepalive-Kommentar

Nginx und andere Proxies schließen idle HTTP-Verbindungen nach 60–90 Sekunden. Ein SSE-Kommentar (`: ping\n\n`) alle 30 Sekunden hält die Verbindung offen, ohne ein `onmessage`-Event im Client auszulösen.

### D4: Frontend-Fehlerbehandlung

`EventSource` reconnectet automatisch (Browser-Standard). Bei permanentem Fehler (`onerror` mit `readyState === CLOSED`) wird die Verbindung geschlossen — kein reconnect-Loop. Die Seite verliert dann nur die Live-Updates, bleibt aber funktionsfähig.

## Risks / Trade-offs

- **Token-Ablauf während offener SSE-Verbindung** → Der Go-Handler prüft das JWT nur beim Verbindungsaufbau. Nach Ablauf des Access-Tokens (15 min) bleibt die SSE-Verbindung offen bis der Client die Seite verlässt. Akzeptabel: der Stream enthält keine sensiblen Daten.
- **Viele offene Verbindungen** → Bei 100+ gleichzeitigen Nutzern auf der Seite entstehen 100+ offene Goroutinen. Für den VPS (1 GB RAM) kein Problem — Go-Goroutinen kosten ~8 KB.
- **Nginx-Konfiguration** → Nginx muss `proxy_buffering off` für den SSE-Endpoint haben, sonst werden Events gebuffert. Zu prüfen in `deploy/nginx-intern.conf`.
