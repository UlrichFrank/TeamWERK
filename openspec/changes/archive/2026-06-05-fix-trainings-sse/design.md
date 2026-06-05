## Context

TeamWERK nutzt ein etabliertes SSE-Muster für Echtzeit-Updates: Der Backend-`EventHub` (`internal/hub/`) hält offene SSE-Verbindungen zu allen Clients; Mutation-Handler rufen `h.hub.Broadcast("domain")` auf; Frontendseiten abonnieren via `useLiveUpdates(event => { if (event === 'domain') reload() })`. Dieses Muster ist in duties, games, members, carpooling und config im Einsatz.

`internal/trainings/handler.go` wurde ohne Hub-Anbindung eingeführt. Der `Handler`-Struct kennt den Hub nicht, `NewHandler` akzeptiert ihn nicht, und keiner der 7 Mutations-Handler sendet ein Broadcast. Die beiden Frontend-Seiten abonnieren keine SSE-Events.

## Goals / Non-Goals

**Goals:**
- Alle 7 Trainings-Mutations-Handler broadcasten `"trainings"` nach erfolgreichem Schreiben
- `TrainingsPage` und `TrainingsDetailPage` reagieren sofort auf `"trainings"`-Events
- Konsistenz mit dem bestehenden Hub-Pattern im gesamten Projekt

**Non-Goals:**
- Granulare Events pro Session-ID (alle Trainings-Clients laden neu — ausreichend für die Datenmenge)
- Push-Notifications oder WebSockets
- Änderungen am Hub selbst

## Decisions

**D1 — Event-Name `"trainings"` (ein String für das gesamte Modul)**
Alternatives considered: sessionspezifische Events wie `"trainings:105934"`. Abgelehnt, weil die Session-Liste immer aus denselben Queries kommt und die Clientanzahl klein ist. Ein einzelner String hält den Hub-Code minimal — exakt wie duties, games etc.

**D2 — Broadcast nach `tx.Commit()` / `ExecContext`, nicht davor**
Sicherstellt, dass Clients nur dann neu laden, wenn die DB-Änderung committed ist. Kein Broadcast bei Fehler im Handler (Fehlerpath gibt vorher zurück).

**D3 — `useLiveUpdates` in beiden Seiten, identischer Callback**
`TrainingsPage.load()` und `TrainingsDetailPage.load()` existieren bereits — sie werden einfach als Reaktion auf das Event aufgerufen. Kein zusätzlicher State, kein debounce nötig.

## Risks / Trade-offs

- **Race condition (minimal):** Sehr seltener Fall: Broadcast kommt an bevor der neue Client das initiale Load abgeschlossen hat — zweites `load()` überschreibt das erste. Akzeptabel, da idempotent.
- **Alle Trainings-Clients laden beim Broadcast neu**, nicht nur betroffene Session. Bei kleinem Team (< 50 User) kein Problem; bei Wachstum ggf. granulare Events nachrüsten.

## Migration Plan

Kein Datenbankschema-Änderung. Kein Deploy-Risiko. Rollback = Revert des Commits.
