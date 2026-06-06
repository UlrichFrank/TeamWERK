## Why

Wenn ein Spiel oder Sonstiger Termin gelöscht wird, bleiben die verknüpften Dienste (`duty_slots`) in der Datenbank erhalten, weil `duty_slots.game_id` mit `ON DELETE SET NULL` definiert ist. Das führt zu verwaisten Diensten, die nicht mehr zu einem Termin gehören und für Admins unsichtbar werden. Nutzererwartung ist: Event weg → Dienste weg.

## What Changes

- **BREAKING** (Daten): Migration ändert `duty_slots.game_id` FK von `ON DELETE SET NULL` auf `ON DELETE CASCADE` — beim Löschen eines Spiels/Termins werden alle zugehörigen Dienste automatisch mitgelöscht.
- `GameEditModal.tsx` wird abgesichert: Delete-Aufruf erhält `?delete_slots=true` als Defense-in-depth.
- `SpieltagDetailPage.tsx`: Die "Verknüpfte Dienste ebenfalls löschen"-Checkbox wird entfernt, da das Verhalten nun immer kaskadiert (kein opt-out mehr).
- Backend `DeleteGame`-Handler: Der `delete_slots`-Branch bleibt als Fallback erhalten, ist aber für neue Deployments redundant.

## Capabilities

### New Capabilities
- `game-deletion-cascade`: Automatisches Löschen aller `duty_slots` (und deren Assignments via vorhandenem CASCADE) beim Löschen eines Spiels/Termins — auf DB-Ebene und als Frontend-Absicherung.

### Modified Capabilities

## Impact

- `internal/db/migrations/027_game_deletion_cascade.up.sql` — neue Migration (ALTER TABLE Workaround via SQLite-Rewrite)
- `internal/db/migrations/027_game_deletion_cascade.down.sql`
- `internal/games/handler.go` — `DeleteGame` bleibt unverändert (redundanter Pfad schadet nicht)
- `web/src/components/GameEditModal.tsx` — Delete-URL bekommt `?delete_slots=true`
- `web/src/pages/SpieltagDetailPage.tsx` — Checkbox und `deleteWithSlots`-State entfernen
- Bestehende Daten: Verwaiste Dienste (`game_id IS NULL`) sind von der Migration nicht betroffen
