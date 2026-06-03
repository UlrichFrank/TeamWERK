## 1. Backend — Hub-Anbindung

- [x] 1.1 `internal/trainings/handler.go`: `Handler`-Struct um `hub *hub.EventHub` ergänzen
- [x] 1.2 `NewHandler(db *sql.DB, hub *hub.EventHub) *Handler` Signatur aktualisieren
- [x] 1.3 `Respond` (POST /training-sessions/{id}/respond): `h.hub.Broadcast("trainings")` nach erfolgreichem Upsert
- [x] 1.4 `UpdateSession` (PUT /training-sessions/{id}): `h.hub.Broadcast("trainings")` nach erfolgreichem Update
- [x] 1.5 `CreateSession` (POST /training-sessions): `h.hub.Broadcast("trainings")` nach erfolgreichem Insert
- [x] 1.6 `SaveAttendances` (POST /training-sessions/{id}/attendances): `h.hub.Broadcast("trainings")` nach `tx.Commit()`
- [x] 1.7 `CreateSeries` (POST /training-series): `h.hub.Broadcast("trainings")` nach `tx.Commit()`
- [x] 1.8 `UpdateSeries` (PUT /training-series/{id}): `h.hub.Broadcast("trainings")` nach `tx.Commit()`
- [x] 1.9 `DeleteSeries` (DELETE /training-series/{id}): `h.hub.Broadcast("trainings")` nach erfolgreichem Delete

## 2. Backend — Wiring in main.go

- [x] 2.1 `cmd/teamwerk/main.go` Zeile 94: `trainings.NewHandler(database)` → `trainings.NewHandler(database, hubInstance)`

## 3. Frontend — Live-Updates

- [x] 3.1 `web/src/pages/TrainingsPage.tsx`: `useLiveUpdates` importieren und `useLiveUpdates((event) => { if (event === 'trainings') load() })` hinzufügen
- [x] 3.2 `web/src/pages/TrainingsDetailPage.tsx`: `useLiveUpdates` importieren und `useLiveUpdates((event) => { if (event === 'trainings') load() })` hinzufügen

## 4. Verifikation

- [x] 4.1 Go-Build erfolgreich: `go build ./cmd/teamwerk`
- [ ] 4.2 Manueller Test: RSVP als Spieler ändern → `TrainingsDetailPage` als Trainer aktualisiert sich sofort
- [ ] 4.3 Manueller Test: Session absagen als Trainer → `TrainingsPage` als Spieler zeigt Absage sofort
