## 1. EventHub implementieren

- [x] 1.1 `internal/hub/hub.go` anlegen mit `EventHub`-Struct (`sync.Mutex`, `map[chan string]struct{}`), Methoden `Subscribe() chan string`, `Unsubscribe(ch chan string)`, `Broadcast(event string)` (non-blocking via `select`+`default`)
- [x] 1.2 `internal/hub/handler.go` anlegen mit `Handler`-Struct (`hub *EventHub`), `NewHandler(h *EventHub) *Handler`, Methode `Events(w, r)` — SSE-Header setzen, Hub subscriben, Loop mit `select` auf Hub-Signal (→ `data: <event>\n\n` + Flush), 30s-Ticker (→ `: ping\n\n` + Flush), `r.Context().Done()`

## 2. Auth-Middleware erweitern

- [x] 2.1 `internal/auth/middleware.go`: Token-Extraktion so erweitern, dass neben `Authorization: Bearer <token>` auch `?token=<jwt>` als Query-Parameter akzeptiert wird (nur wenn kein Header vorhanden)

## 3. Hub in alle Mutation-Handler einbauen

- [x] 3.1 `internal/carpooling/handler.go`: `hub *hub.EventHub` zum `Handler`-Struct hinzufügen, `NewHandler` erweitern, `h.hub.Broadcast("mitfahrgelegenheiten")` nach `Upsert` und `Delete` aufrufen
- [x] 3.2 `internal/carpooling/paarungen_handler.go`: `h.hub.Broadcast("mitfahrgelegenheiten")` nach `RequestPairing`, `ConfirmPairing` und `RejectPairing` aufrufen (jeweils nach erfolgreichem DB-Write, vor `w.WriteHeader`)
- [x] 3.3 `internal/members/handler.go`: `hub *hub.EventHub` zum `Handler`-Struct hinzufügen, `NewHandler` erweitern, `h.hub.Broadcast("members")` nach Create, Update und Status-Change aufrufen
- [x] 3.4 `internal/duties/handler.go`: `hub *hub.EventHub` zum `Handler`-Struct hinzufügen, `NewHandler` erweitern, `h.hub.Broadcast("duties")` nach Slot-Create, Slot-Update, Slot-Delete, Assignment-Fulfill und Assignment-CashSubstitute aufrufen
- [x] 3.5 `internal/games/handler.go`: `hub *hub.EventHub` zum `Handler`-Struct hinzufügen, `NewHandler` erweitern, `h.hub.Broadcast("games")` nach Game-Create, Game-Update und Game-Delete aufrufen
- [x] 3.6 `internal/config/handler.go`: `hub *hub.EventHub` zum `Handler`-Struct hinzufügen, `NewHandler` erweitern, `h.hub.Broadcast("settings")` nach Settings-Mutations aufrufen (Altersklassen, Teams, Club-Daten)

## 4. Route und DI in main.go

- [x] 4.1 `cmd/teamwerk/main.go`: `hubInstance := hub.NewHub()` anlegen, `hub.NewHandler(hubInstance)` als `hubH` anlegen
- [x] 4.2 `cmd/teamwerk/main.go`: `hubInstance` an alle Handler übergeben die einen Hub erhalten (`carpoolingH`, `membersH`, `dutiesH`, `gamesH`, `configH`)
- [x] 4.3 `cmd/teamwerk/main.go`: `r.Get("/api/events", hubH.Events)` im authenticated-Block registrieren

## 5. Frontend — Hilfsfunktion und Hook

- [x] 5.1 `web/src/lib/api.ts`: `getAccessToken(): string | null` exportieren, die den aktuellen in-memory Access-Token zurückgibt
- [x] 5.2 `web/src/hooks/useLiveUpdates.ts` anlegen: Custom Hook mit `EventSource`-Verbindung zu `/api/events?token=${getAccessToken()}`, Callback `onEvent(eventType: string)`, `es.onerror` bei `CLOSED`-State schließen, Cleanup `return () => es.close()`

## 6. Frontend — Pages auf useLiveUpdates umstellen

- [x] 6.1 `web/src/pages/MitfahrgelegenheitenPage.tsx`: `useLiveUpdates` integrieren — bei `"mitfahrgelegenheiten"` → `load(true)` (silent reload)
- [x] 6.2 `web/src/pages/MembersPage.tsx`: `useLiveUpdates` integrieren — bei `"members"` → `load()` (oder äquivalenter Reload-Aufruf)
- [x] 6.3 `web/src/pages/DutyBoardPage.tsx`: `useLiveUpdates` integrieren — bei `"duties"` → `load()` (silent reload) [implemented in DutyPage.tsx]
- [x] 6.4 `web/src/pages/DutySlotsPage.tsx`: `useLiveUpdates` integrieren — bei `"duties"` → `load()` (silent reload) [implemented in DutyPage.tsx]
- [x] 6.5 `web/src/pages/GameSchedulePage.tsx`: `useLiveUpdates` integrieren — bei `"games"` → `loadGames()` [implemented in KalenderPage.tsx]: `useLiveUpdates` integrieren — bei `"games"` → `load()` (silent reload)

## 7. Nginx prüfen

- [x] 7.1 `deploy/nginx-intern.conf`: Sicherstellen dass für `/api/events` `proxy_buffering off` gesetzt ist (sonst werden SSE-Events gebuffert)
