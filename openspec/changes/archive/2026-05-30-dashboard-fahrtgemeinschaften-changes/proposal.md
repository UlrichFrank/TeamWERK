## Why

Das Dashboard-Widget „Fahrtgemeinschaften" zeigt bisher nur statische Zähler (Angebote / Gesuche) für das nächste Auswärtsspiel. User erfahren Paarungsanfragen, Zusagen, Absagen und Löschungen ausschließlich über Push Notifications — im Dashboard fehlt jede Nachvollziehbarkeit, besonders wenn die Push-Notification verpasst wurde.

## What Changes

- `queryCarpoolingHint` im Dashboard-Handler wird um personalisierten Snapshot erweitert: eigener Eintragsstatus, aktive Paarungen, Ereignisse der letzten 48 h
- Neue DB-Tabelle `carpooling_events` speichert Lösch-Ereignisse, die sonst durch `ON DELETE CASCADE` spurlos verschwinden
- Delete-Handler in `internal/carpooling/handler.go` schreibt vor dem DELETE Ereignisse in die neue Tabelle
- `CarpoolingHintCard` im Frontend wird zu einer vollständigen Ereignis- und Statusanzeige ausgebaut

## Capabilities

### New Capabilities

- `carpooling-event-log`: Persistenter Log von Lösch-Ereignissen im Mitfahrgelegenheiten-System (`biete_deleted`, `suche_deleted`) für User, die eine aktive Paarung hatten

### Modified Capabilities

- `dashboard-carpooling-hint`: Die bisherige Hint-Anzeige (nur Zähler) wird zu einem personalisierten Status-Widget: eigener Eintragsstatus, Paarungsereignisse und Lösch-Log

## Impact

- **Backend:** Neue Migration `018_carpooling_events`, Erweiterung von `internal/dashboard/handler.go` (`queryCarpoolingHint`) und `internal/carpooling/handler.go` (Delete-Handler)
- **Frontend:** `CarpoolingHintCard` in `web/src/pages/DashboardPage.tsx` — neue Interfaces, erweitertes Rendering
- **Datenbank:** Neue Tabelle `carpooling_events` (id, user_id, game_id, type, actor_name, created_at)
- **Rollen:** Alle authentifizierten User (spieler, elternteil, trainer, admin)
- **Keine neuen Dependencies**
