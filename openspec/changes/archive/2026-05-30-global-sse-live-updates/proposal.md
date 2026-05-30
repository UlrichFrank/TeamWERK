## Why

Änderungen in der App (neue Mitfahrgelegenheiten, Paarungen, Mitglieder, Dienste, Spielplan) sind für andere eingeloggte Nutzer erst nach einem manuellen Seiten-Reload sichtbar. Server-Sent Events (SSE) ersetzen diesen Workaround: der Server pusht ein typisiertes Signal exakt dann, wenn eine Mutation stattfindet — ohne Polling, ohne Latenz.

## What Changes

- **Neues `internal/hub`-Package** — globaler `EventHub` als in-memory Pub/Sub für alle SSE-Clients
- **Neuer SSE-Endpoint** `GET /api/events` — hält Verbindungen offen, sendet typisierte Event-Signale bei Mutations
- **Auth-Middleware erweitert** — akzeptiert `?token=<jwt>` als Query-Parameter (EventSource unterstützt keine Custom-Header)
- **Alle Mutation-Handler angepasst** — rufen `hub.Broadcast("<event-typ>")` nach erfolgreichen DB-Writes auf (carpooling, members, duties, games, config)
- **Neuer `useLiveUpdates`-Hook** — zentrale EventSource-Verbindung im Frontend, verteilt Events an subscribte Pages
- **Alle relevanten Pages angepasst** — laden Daten still neu wenn ein passendes SSE-Event ankommt

## Capabilities

### New Capabilities

- `sse-live-updates`: Globaler SSE-Stream sendet typisierte Refresh-Signale an alle verbundenen Clients wenn sich Daten in einem Bereich ändern (Mitfahrgelegenheiten, Mitglieder, Dienste, Spielplan, Einstellungen)

### Modified Capabilities

*(keine bestehenden Specs betroffen)*

## Impact

- **Neue Dateien:** `internal/hub/hub.go`, `internal/hub/handler.go`, `web/src/hooks/useLiveUpdates.ts`
- **Geänderte Dateien:**
  - `internal/auth/middleware.go` — Query-Parameter-Auth
  - `cmd/teamwerk/main.go` — Hub-Initialisierung, Route, DI in alle Handler
  - `internal/carpooling/handler.go` — Broadcast nach Upsert/Delete
  - `internal/carpooling/paarungen_handler.go` — Broadcast nach Pairing-Mutations
  - `internal/members/handler.go` — Broadcast nach Member-Mutations
  - `internal/duties/handler.go` — Broadcast nach Slot/Assignment-Mutations
  - `internal/games/handler.go` — Broadcast nach Game-Mutations
  - `internal/config/handler.go` — Broadcast nach Settings-Mutations
  - `web/src/lib/api.ts` — `getAccessToken()` exportieren
  - `web/src/pages/MitfahrgelegenheitenPage.tsx`, `MembersPage.tsx`, `DutyBoardPage.tsx`, `DutySlotsPage.tsx`, `GameSchedulePage.tsx` u.a. — `useLiveUpdates` integrieren
- **Keine DB-Migrationen**, keine neuen externen Dependencies
- **Keine Breaking Changes** an bestehenden API-Endpoints
- Offene Goroutinen pro eingeloggtem Nutzer auf einer Live-Seite (~8 KB je Goroutine, für VPS vernachlässigbar)
