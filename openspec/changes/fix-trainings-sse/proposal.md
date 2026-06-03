## Why

Das Trainings-Modul wurde ohne SSE-Anbindung eingeführt. Wenn ein Spieler seinen RSVP-Status ändert, sehen Trainer und andere Spieler die Aktualisierung erst nach manuellem Neuladen — obwohl das SSE-Infrastruktur-Pattern (`hub.Broadcast` + `useLiveUpdates`) bereits für duties, games, members und carpooling funktioniert.

## What Changes

- `internal/trainings/handler.go`: `Handler`-Struct erhält `hub *hub.EventHub`; `NewHandler` nimmt zweiten Parameter; alle 7 Mutations-Handler rufen nach erfolgreichem Schreiben `h.hub.Broadcast("trainings")` auf
- `cmd/teamwerk/main.go`: `trainings.NewHandler(database, hubInstance)` (war: `trainings.NewHandler(database)`)
- `web/src/pages/TrainingsPage.tsx`: `useLiveUpdates` abonniert `"trainings"`-Events und ruft `load()` auf
- `web/src/pages/TrainingsDetailPage.tsx`: `useLiveUpdates` abonniert `"trainings"`-Events und ruft `load()` auf

## Capabilities

### New Capabilities

- `trainings-live-updates`: Echtzeit-SSE-Broadcasts für alle Trainings-Mutationen; Frontend-Seiten reagieren sofort auf Änderungen anderer Nutzer

### Modified Capabilities

*(keine Anforderungsänderungen an bestehenden Specs)*

## Impact

- **Betroffene Dateien:** `internal/trainings/handler.go`, `cmd/teamwerk/main.go`, `web/src/pages/TrainingsPage.tsx`, `web/src/pages/TrainingsDetailPage.tsx`
- **API:** keine Änderungen (gleiche Routen, gleiche Payloads)
- **Dependencies:** keine neuen Pakete
- **SSE-Hub:** bereits laufend, kein Mehraufwand auf dem VPS (1 GB RAM)
