## 1. DB-Migration

- [x] 1.1 `027_game_deletion_cascade.up.sql` erstellen: duty_slots-Tabelle via SQLite-Rewrite neu anlegen mit `game_id REFERENCES games(id) ON DELETE CASCADE`
- [x] 1.2 `027_game_deletion_cascade.down.sql` erstellen: Tabelle zurück auf `ON DELETE SET NULL`
- [x] 1.3 Migration lokal testen: `make migrate-up` und prüfen ob Schema korrekt ist

## 2. Frontend Fix

- [x] 2.1 `GameEditModal.tsx`: Delete-URL von `/admin/kalender/${game.id}` auf `/admin/kalender/${game.id}?delete_slots=true` ändern
- [x] 2.2 `SpieltagDetailPage.tsx`: `deleteWithSlots`-State und Checkbox entfernen; Delete-URL immer mit `?delete_slots=true`; Delete-Dialog zeigt Anzahl Dienste als Info (kein opt-out)
