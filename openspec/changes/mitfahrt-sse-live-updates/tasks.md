## 1. Polling zurückrollen

- [ ] 1.1 `web/src/pages/MitfahrgelegenheitenPage.tsx`: `setInterval`, `visibilitychange`-Listener und das `silent`-Argument in `load()` entfernen — `useEffect` zurück auf `useEffect(() => { load() }, [])`

## 2. Auth-Middleware erweitern

- [ ] 2.1 `internal/auth/middleware.go`: Token-Extraktion so erweitern, dass neben `Authorization: Bearer <token>` auch `?token=<jwt>` als Query-Parameter akzeptiert wird (nur wenn kein Header vorhanden)

## 3. EventHub implementieren

- [ ] 3.1 `internal/carpooling/events.go` anlegen mit `EventHub`-Struct (`sync.Mutex`, `map[chan struct{}]struct{}`), Methoden `Subscribe() chan struct{}`, `Unsubscribe(ch chan struct{})`, `Broadcast()` (non-blocking via `select`+`default`)

## 4. SSE-Handler und Broadcast verdrahten

- [ ] 4.1 `internal/carpooling/handler.go`: `hub *EventHub` zum `Handler`-Struct hinzufügen, in `NewHandler` initialisieren
- [ ] 4.2 `internal/carpooling/handler.go`: Methode `Events(w, r)` implementieren — SSE-Header setzen, Hub subscriben, Loop mit `select` auf Hub-Signal (→ `data: refresh\n\n` + Flush), 30s-Ticker (→ `: ping\n\n` + Flush), `r.Context().Done()`
- [ ] 4.3 `internal/carpooling/handler.go`: `h.hub.Broadcast()` nach `Upsert` und nach `Delete` aufrufen
- [ ] 4.4 `internal/carpooling/paarungen_handler.go`: `h.hub.Broadcast()` nach `RequestPairing`, `ConfirmPairing` und `RejectPairing` aufrufen (jeweils nach dem erfolgreichen DB-Write, vor dem `w.WriteHeader`)

## 5. Route registrieren

- [ ] 5.1 `cmd/teamwerk/main.go`: `r.Get("/api/mitfahrgelegenheiten/events", carpoolH.Events)` im authenticated-Block hinzufügen (bei den anderen Mitfahrgelegenheiten-Routen)

## 6. Frontend auf EventSource umstellen

- [ ] 6.1 `web/src/lib/api.ts`: Hilfsfunktion `getAccessToken(): string` exportieren, die den aktuellen in-memory Access-Token zurückgibt
- [ ] 6.2 `web/src/pages/MitfahrgelegenheitenPage.tsx`: `useEffect` auf `EventSource` umstellen — URL mit `?token=${getAccessToken()}`, `es.onmessage = () => load(true)`, `es.onerror = () => es.close()`, Cleanup `return () => es.close()`

## 7. Nginx prüfen

- [ ] 7.1 `deploy/nginx-intern.conf`: Sicherstellen dass für `/api/mitfahrgelegenheiten/events` `proxy_buffering off` gesetzt ist (sonst werden SSE-Events gebuffert und kommen verzögert an)
