## Context

`duty_slots.game_id` ist aktuell mit `ON DELETE SET NULL` referenziert. Das bedeutet: wird ein Spiel/Termin gelöscht, bleibt der Dienst erhalten und `game_id` wird auf `NULL` gesetzt. Diese "verwaisten" Dienste tauchen in der normalen Verwaltung nicht mehr auf, belegen aber Datenbankeinträge und verwirren Nutzer.

Der Delete-Pfad läuft über zwei Stellen:
1. `GameEditModal.tsx` → `DELETE /api/admin/kalender/{id}` (kein `?delete_slots=true`)
2. `SpieltagDetailPage.tsx` → `DELETE /api/admin/kalender/{id}?delete_slots=true` (mit Checkbox, default true)

## Goals / Non-Goals

**Goals:**
- Löschen eines Spiels/Termins löscht immer alle verknüpften Dienste (und deren Assignments via vorhandenem CASCADE)
- Kein opt-out mehr: Die Entscheidung ist immer "alles löschen"
- Keine Code-Pfade, die zu verwaisten Diensten führen können

**Non-Goals:**
- Verwaiste Dienste aufräumen, die bereits in der DB existieren (`game_id IS NULL`) — separates Thema
- Änderung am Backend-Handler selbst (bleibt als Fallback erhalten)

## Decisions

### D1: ON DELETE CASCADE via SQLite-Rewrite-Migration

SQLite unterstützt kein `ALTER TABLE ... DROP CONSTRAINT`. Der FK muss über eine Neu-Erstellung der Tabelle geändert werden:

1. Neue Tabelle `duty_slots_new` mit `game_id REFERENCES games(id) ON DELETE CASCADE`
2. Daten kopieren
3. Alte Tabelle droppen, neue umbenennen
4. Indexes neu anlegen

**Warum nicht nur Backend-Fix?** Der Backend-Pfad `GameEditModal` hat den Parameter vergessen — das könnte in Zukunft wieder passieren. DB-Level ist die robustere Garantie.

**Warum nicht nur `?delete_slots=true` überall?** Code kann sich ändern. Ein zweiter Löschpfad ohne Parameter würde wieder verwaiste Dienste erzeugen. DB-Constraint ist idiotsicher.

### D2: GameEditModal bekommt ?delete_slots=true als Defense-in-depth

Auch nach der Migration schadet es nicht, den Parameter mitzuschicken — der Backend-Handler löscht Slots explizit, bevor SQLite kaskadieren kann. Redundanz ist hier kein Problem.

### D3: Checkbox in SpieltagDetailPage entfernen

Die Checkbox "Verknüpfte Dienste ebenfalls löschen" wird entfernt. Da die DB jetzt immer kaskadiert, ist ein opt-out nicht mehr möglich und die Checkbox wäre irreführend (sie steuert nur noch den expliziten Backend-DELETE, der sowieso vor dem CASCADE greift).

## Risks / Trade-offs

- [Migration irreversibel für Daten] Dienste die beim Löschen mitgelöscht werden, sind weg → Mitigation: Das ist das gewünschte Verhalten; Down-Migration stellt SET NULL zurück
- [SQLite Rewrite bei laufender DB] Tabelle wird neu erstellt; auf dem VPS muss die Migration sauber durchlaufen → Mitigation: `make migrate-remote-up` mit WAL-Mode; bei Fehler rollt golang-migrate zurück
- [Verwaiste Dienste (game_id IS NULL) unberührt] Bestehende verwaiste Einträge werden nicht aufgeräumt → Mitigation: Separates Cleanup-Script falls nötig, kein Blocking-Issue

## Migration Plan

1. `027_game_deletion_cascade.up.sql` — Tabellen-Rewrite mit neuem FK
2. `027_game_deletion_cascade.down.sql` — zurück auf SET NULL
3. Deploy via `make deploy` (führt automatisch `migrate up` aus)
4. Rollback: `make migrate-remote-down` setzt FK zurück auf SET NULL
