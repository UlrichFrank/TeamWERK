## Why

Bei Spieltagen mit mehreren Heimspielen (z.B. B-Jugend 11:00 + A-Jugend 15:00) werden Dienste heute pro Spiel einzeln regeneriert — dabei sieht jedes Spiel seinen eigenen Kontext, aber nicht den vollständigen Tageskontext. Das führt dazu, dass `same_day_behavior`-Optimierungen (skip/reduce) inkonsistent greifen: Spiel #4 ist optimiert, Spiel #3 noch nicht, bis es ebenfalls manuell regeneriert wird.

## What Changes

- Neuer Backend-Endpoint `POST /api/admin/games/regenerate-day` mit Query-Parameter `date=YYYY-MM-DD` berechnet und erzeugt Dienste für **alle Spiele eines Tages** in einem Schritt
- Alle bestehenden leeren Slots des Tages werden gelöscht, dann neu generiert — mit vollständigem Tageskontext für jedes Spiel
- Im Frontend: Neuer Button „Dienste für diesen Tag generieren" in der Spielplan-Kalenderansicht, sichtbar beim Klick auf einen Tag mit mindestens einem Heimspiel
- Die bestehende Einzel-Regenerierung (`POST /api/admin/games/{id}/regenerate`) bleibt erhalten

## Capabilities

### New Capabilities

- `tagesweise-dienstgenerierung`: Batch-Endpoint und UI-Einstiegspunkt zum Generieren aller Dienste eines Spieltags in einem Schritt, mit korrekter tagesübergreifender Optimierungslogik (same_day_behavior, adjacent_day_behavior)

### Modified Capabilities

<!-- keine bestehenden Specs betroffen -->

## Impact

- **Backend:** Neuer Handler in `internal/games/handler.go`, Route in `cmd/teamwerk/main.go`
- **Frontend:** `SpielplanPage.tsx` — Tages-Klick-Bereich erhält neuen Button und Dialog (Template-Auswahl + Bestätigung)
- **Keine Migration** erforderlich — nutzt bestehende Tabellen und Logik (`loadSameDayContext`, `applyBehavior`, `classifySlotPosition`)
- **Rolle:** Admin + Trainer (bestehende Middleware `RequireRole("admin","trainer")`)
