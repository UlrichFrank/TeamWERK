## Context

Alle Seiten der App laden Daten einmalig beim Mount und zeigen danach veralteten Stand, bis der Nutzer manuell neu lädt. Ein 15-Sekunden-Polling wurde als Workaround eingebaut (und wieder zurückgerollt), weil Polling Latenz und unnötige Requests erzeugt. SSE ist der minimale, saubere Ansatz: eine offene HTTP-Verbindung, über die der Server typisierte Signale sendet, wenn Mutations stattfinden.

## Goals / Non-Goals

**Goals:**
- Sofortige Aktualisierung bei jeder Mutation (Mitfahrgelegenheiten, Mitglieder, Dienste, Spielplan, Einstellungen)
- Kein DB-Polling auf dem Server — Events entstehen direkt nach erfolgreichen DB-Writes
- Minimale Server-Last (Goroutinen sind günstig, in-memory Hub ohne externe Deps)
- Eine einzige SSE-Verbindung pro Browser-Session (nicht pro Seite)

**Non-Goals:**
- Kein Streaming von Daten über SSE (nur Signal "welcher Bereich hat sich geändert")
- Kein WebSocket (keine Bidirektionalität nötig)
- Kein Reconnect-Loop bei dauerhaftem Fehler (Browser auto-reconnect reicht)
- Kein Fan-out nach Rolle (alle verbundenen Clients erhalten alle Events)

## Decisions

### D1: Globaler EventHub im eigenen Package

```
internal/hub/
  hub.go      → EventHub struct + Subscribe/Unsubscribe/Broadcast
  handler.go  → SSE-Handler Events(w, r)
```

```
EventHub
  mu      sync.Mutex
  clients map[chan string]struct{}

Subscribe()  → chan string    // client registrieren
Unsubscribe(ch)               // client entfernen
Broadcast(event string)       // allen senden (non-blocking)
```

`Broadcast()` verwendet `select`+`default` damit ein langsamer Client den Hub nicht blockiert. Der Hub lebt als Zeiger im `main()` und wird per Dependency Injection in alle Handler übergeben.

**Warum `chan string` statt `chan struct{}`:** Das Frontend kann gezielt nur auf relevante Events reagieren — eine Mitfahrgelegenheiten-Seite ignoriert `"members"`-Events. Das vermeidet unnötige Re-Renders.

**Alternative:** Separates Event-Bus-Package mit Subscribe-by-Type — abgelehnt, zu komplex. Stattdessen: jede Page filtert selbst nach Event-Typ.

### D2: Handler-Erweiterung per Hub-Feld

Da alle Handler bereits `type Handler struct{ db *sql.DB }` verwenden, wird `hub *hub.EventHub` als weiteres Feld ergänzt. `NewHandler` erhält einen `*hub.EventHub`-Parameter. Das entspricht dem bestehenden Muster (kein DI-Framework, explizite Übergabe).

```go
// Beispiel
type Handler struct {
    db  *sql.DB
    hub *hub.EventHub
}
func NewHandler(db *sql.DB, h *hub.EventHub) *Handler { ... }
```

**Alternative:** Hub als Singleton-Global — abgelehnt, weil testfeindlich.

### D3: Auth via JWT-Query-Parameter

`EventSource` im Browser unterstützt keine Custom-Header. Da der SSE-Stream nur Event-Typen (keine Nutzerdaten) sendet, wäre ein öffentlicher Endpoint vertretbar — dennoch wird Auth beibehalten, um unbegrenzte offene Verbindungen von Nicht-Nutzern zu verhindern.

Lösung: `auth.Middleware` prüft neben dem `Authorization`-Header auch `?token=<jwt>`. Frontend exportiert `getAccessToken()` aus `lib/api.ts` und hängt den Token an die EventSource-URL.

**Alternative:** Separater kurzlebiger SSE-Token — abgelehnt, zu komplex.

### D4: `useLiveUpdates`-Hook im Frontend

Ein einziger Custom Hook verwaltet die EventSource-Verbindung. Er nimmt einen `onEvent(eventType: string)`-Callback entgegen und schließt die Verbindung beim Unmount.

```ts
useLiveUpdates((event) => {
  if (event === 'mitfahrgelegenheiten') load(true)
})
```

Jede Page entscheidet selbst, auf welche Events sie reagiert. Der Hook öffnet genau eine SSE-Verbindung pro Mount — bei Single-Page-Navigation mit globalem Mount (z.B. in AppShell) wäre es eine Verbindung pro Session.

**Alternative:** Globaler SSE-Context in `App.tsx` mit einem zentralen Event-Emitter — sauberer, aber mehr Infrastruktur. Für den aktuellen Anwendungsfall (wenige Seiten, einfache Callbacks) ist der Page-lokale Hook ausreichend.

### D5: 30-Sekunden Keepalive-Kommentar

Nginx und andere Proxies schließen idle HTTP-Verbindungen nach 60–90 Sekunden. Ein SSE-Kommentar (`: ping\n\n`) alle 30 Sekunden hält die Verbindung offen, ohne ein `onmessage`-Event im Client auszulösen. `proxy_buffering off` in nginx-intern.conf für den `/api/events`-Endpoint sicherstellen.

## Risks / Trade-offs

- **Token-Ablauf während offener SSE-Verbindung** → Der Go-Handler prüft das JWT nur beim Verbindungsaufbau. Nach Ablauf des Access-Tokens (15 min) bleibt die SSE-Verbindung offen. Akzeptabel: der Stream enthält keine sensiblen Daten. Bei Browser-Refresh wird eine neue Verbindung mit neuem Token aufgebaut.
- **Viele offene Verbindungen** → Bei 50 gleichzeitigen Nutzern entstehen 50 offene Goroutinen und Channels. Für den VPS (1 GB RAM) kein Problem — Go-Goroutinen kosten ~8 KB.
- **Nginx-Konfiguration** → `proxy_buffering off` muss für `/api/events` gesetzt sein, sonst werden SSE-Events gepuffert. Zu prüfen und ggf. anzupassen in `deploy/nginx-intern.conf`.
- **Keine Tests für den Hub** → Da der Hub in-memory ist und keine DB-Abhängigkeit hat, ist er gut testbar, aber im Scope dieser Änderung werden keine Unit-Tests ergänzt.
