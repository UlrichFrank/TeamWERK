## Why

Die Mitfahrgelegenheiten-Seite zeigt Änderungen (neue Angebote, Anfragen, Bestätigungen) nicht live — der Nutzer muss die Seite manuell neu laden. Ein 15-Sekunden-Polling wurde als Übergangslösung eingebaut, verursacht aber unnötige Requests und hat eine Verzögerung. Server-Sent Events (SSE) ersetzen das Polling: der Server pusht ein Signal exakt dann, wenn sich etwas ändert.

## What Changes

- **Polling entfernen** aus `MitfahrgelegenheitenPage.tsx` (`setInterval` + `visibilitychange`)
- **Neues `EventHub`** in `internal/carpooling/events.go` — in-memory Pub/Sub für SSE-Clients
- **SSE-Handler** `Events(w, r)` in `internal/carpooling/handler.go` — hält Verbindungen offen, sendet `data: refresh` bei Mutations
- **`Broadcast()`-Aufrufe** nach jeder Mutation in `handler.go` und `paarungen_handler.go`
- **Route** `GET /api/mitfahrgelegenheiten/events` in `cmd/teamwerk/main.go` (authenticated)
- **`EventSource`-Integration** im Frontend statt Polling; Auth via Query-Parameter `?token=<jwt>` mit neuer Hilfsfunktion `getAccessToken()` in `lib/api.ts`

## Capabilities

### New Capabilities

- `mitfahrt-sse`: Server-Sent Events Stream für die Mitfahrgelegenheiten-Seite — sendet ein "refresh"-Signal an alle verbundenen Clients wenn sich Einträge oder Paarungen ändern

### Modified Capabilities

*(keine bestehenden Specs betroffen)*

## Impact

- **Neue Datei:** `internal/carpooling/events.go`
- **Geänderte Dateien:**
  - `internal/carpooling/handler.go`
  - `internal/carpooling/paarungen_handler.go`
  - `cmd/teamwerk/main.go`
  - `web/src/pages/MitfahrgelegenheitenPage.tsx`
  - `web/src/lib/api.ts`
- **Keine API-Breaking-Changes**, keine DB-Migrationen, keine neuen Dependencies
- Offene SSE-Verbindungen pro eingeloggtem User auf der Mitfahrgelegenheiten-Seite (minimale Server-Last, Go-Goroutinen sind günstig)
