## Why

Die RSVP-Konfiguration eines Termins (`rsvp_opt_out` und `rsvp_require_reason`) ist aktuell nach dem Anlegen nicht mehr änderbar — bei Trainings wird sie im UI gesperrt (`disabled={!isNewSeries}`), bei Spielen fehlt sie sowohl im Edit-Modal als auch im Backend-Endpoint `UpdateGame` komplett. Folge: Wenn ein Trainer beim Anlegen den Modus nicht bewusst gesetzt hat, bleibt das Spiel/Training für immer mit dem DB-Default (`rsvp_opt_out=0`, `rsvp_require_reason=1`) konfiguriert — und der aktuelle Wert ist im UI nirgends sichtbar. Das macht Konfigurationsfehler unsichtbar und unbehebbar und ist eine Voraussetzung, um Zähl-Inkonsistenzen zwischen Kalender und Detail-Ansicht sauber zu diagnostizieren.

## What Changes

- `PUT /api/games/{id}` akzeptiert neu die optionalen Felder `rsvp_opt_out` und `rsvp_require_reason` und schreibt sie ins UPDATE.
- `PUT /api/training-sessions/{id}` akzeptiert die beiden Felder ebenfalls (aktuell nur in `CreateSession` und `UpdateSeries` umgesetzt).
- `GameEditModal` zeigt zwei Checkboxen für `rsvp_opt_out` und `rsvp_require_reason` — sowohl beim Anlegen als auch beim Bearbeiten — und sendet sie im PUT-Payload mit. Default für `event_type='generisch'`: `rsvp_require_reason=0`.
- `AdminTrainingsPage`: die `disabled={!isNewSeries}`-Sperre an beiden Checkboxen wird entfernt; die Werte werden im PUT-Payload für Series **und** Session mitgesendet.
- Die aktuell konfigurierten Werte werden als Badges in der Spiel- bzw. Trainings-Detailansicht angezeigt („Opt-Out aktiv" / „Begründung bei Absage Pflicht"), damit der Status auch ohne Edit-Modal sichtbar ist.
- **BREAKING:** Das bisherige Spec-Verbot „Flag beim Bearbeiten eingefroren" (Scenario in `rsvp-event-config`) entfällt — sowohl Server-seitig als auch UI-seitig.
- **NICHT in diesem Change**: Der Counter-Bug zwischen `/api/games/my` (Kalender, `ListMyGames`) und `/api/games/{id}/participants` (Detail) wird hier nur sichtbarer, aber nicht behoben. Folge-Change.

## Capabilities

### New Capabilities
*(keine — die Funktionalität ist eine Ergänzung an einer bestehenden Capability)*

### Modified Capabilities
- `rsvp-event-config`: Das bisherige Verbot „nach dem Anlegen darf das Flag nicht mehr geändert werden" wird ersetzt durch eine Anforderung, die nachträgliches Ändern explizit erlaubt. Neue Anforderung: aktueller RSVP-Modus muss in der Detailansicht sichtbar sein.
- `game-edit-modal`: Die zwei RSVP-Felder werden Teil der editierbaren Felder; die Liste der dargestellten Felder wird erweitert.

## Impact

**Backend:**
- `internal/games/handler.go` — `UpdateGame` (Request-Struct und UPDATE-SQL erweitern)
- `internal/trainings/handler.go` — `UpdateSession` (Request-Struct und UPDATE-SQL erweitern)
- Tests: jeweils Happy-Path (Wert wird gespeichert), Partial-Update (fehlendes Feld lässt Wert unverändert), Permission (Spieler bekommt 403)

**Frontend:**
- `web/src/components/GameEditModal.tsx` — zwei Checkboxen ergänzen, im PUT-Payload mitsenden, Default-Logik für `event_type='generisch'` (`rsvp_require_reason=0`)
- `web/src/pages/AdminTrainingsPage.tsx` — `disabled={!isNewSeries}` entfernen, PUT-Payload für Series und Session ergänzen
- `web/src/pages/TermineDetailPage.tsx` — Badge-Anzeige für RSVP-Konfiguration ergänzen (Spiel-Pfad und Training-Pfad)

**Datenbank:** Keine neue Migration nötig — Spalten existieren seit Migration 015 (`rsvp_event_config.up.sql`) auf `games`, `training_series` und `training_sessions`.

**API-Kompatibilität:** Additiv. Bestehende Clients, die die neuen Felder nicht senden, lassen den bisherigen Wert unverändert (Partial-Update-Semantik).

**Live-Updates:** `PUT /api/games/{id}` und `PUT /api/training-sessions/{id}` broadcasten bereits `games`- bzw. `trainings`-Events via Hub — keine Änderung nötig.

**RAM-Footprint / Dependencies:** Kein Einfluss — keine neuen externen Bibliotheken, kein zusätzlicher RAM-Bedarf.
