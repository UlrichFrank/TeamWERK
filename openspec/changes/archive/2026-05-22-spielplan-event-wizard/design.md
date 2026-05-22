## Context

Der Spielplan (`games`-Tabelle) hat aktuell ein `team_id NOT NULL`-Feld — ein Spiel gehört immer genau einem Team. Die Slot-Generierung und alle Queries bauen darauf auf. Das Frontend `SpielplanPage.tsx` hat einen einfachen Dialog "Heimspiel anlegen" (admin-only), der noch den alten Preview-Endpunkt `/api/admin/game-template/preview` aufruft (seit `dienstplan-vorlagen-multi` veraltet). Der neue Endpunkt heißt `/api/admin/duty-templates/{id}/preview`.

Generische Events (Turnier, Weihnachtsfeier) betreffen oft mehrere Mannschaften gleichzeitig — das aktuelle Datenmodell kann das nicht abbilden.

## Goals / Non-Goals

**Goals:**
- `game_teams`-Junction-Tabelle für 1..n Teams pro Event
- 4-Schritt-Wizard im Frontend (Typ → Details → Vorlage → Bestätigen)
- Trainer dürfen für eigene Teams anlegen (heim/auswärts), für alle Teams bei generisch
- Spielplan lesbar für alle eingeloggten User
- Bug-Fix Preview-URL

**Non-Goals:**
- Spielplan ohne Login sichtbar (öffentliche Ansicht)
- Historisierung welche Vorlage für welches Spiel verwendet wurde
- Vorlagen je Team oder je Saison
- Wiederkehrende Events / Serien

## Decisions

### 1. Junction-Tabelle `game_teams` statt `team_id NULL`

```sql
CREATE TABLE game_teams (
  game_id  INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
  team_id  INTEGER NOT NULL REFERENCES teams(id) ON DELETE RESTRICT,
  PRIMARY KEY (game_id, team_id)
);
```

`games.team_id` wird in der Migration auf `game_teams` übertragen und dann gedroppt (SQLite: Tabellen-Rebuild nötig).

**Alternative erwogen:** `team_id = NULL` für "alle Teams" — abgelehnt, weil unklar welche Teams das Event betrifft und alle Queries komplexer werden. Junction-Tabelle ist explizit und erweiterbar.

**Alternative erwogen:** Mehrfach-Insert (ein `games`-Record pro Team) — abgelehnt, weil dasselbe Event als N unabhängige Einträge erscheint und schwer synchron zu halten ist.

### 2. Slot-Generierung: pro Team ein Satz Slots

Wenn ein Event 3 Teams betrifft und die gewählte Vorlage 4 Slot-Typen hat, entstehen 12 Slots (`duty_slots.team_id` bleibt, zeigt auf das jeweilige Team). Jeder Slot-Satz ist identisch (gleiche Zeiten, gleiche Typen) — nur `team_id` unterscheidet sich.

**Alternative erwogen:** Ein Satz Slots für alle Teams geteilt (team_id=NULL) — abgelehnt, weil die Dienstbörse nach `team_id` filtert und Mitglieder nur ihre eigenen Slots sehen sollen.

### 3. Trainer-Check im Backend

`POST /api/admin/games`: Falls Caller `role=trainer`, wird geprüft ob alle `team_ids` in `SELECT team_id FROM team_trainers WHERE user_id=?` liegen. Ausnahme: `event_type=generisch` → kein Check.

**Alternative erwogen:** Check im Frontend — abgelehnt, da leicht umgehbar. Backend-Validierung ist autoritativ.

### 4. Vorlagenauswahl explizit (nicht automatisch)

Im Wizard Schritt 3 wird eine Liste passender Vorlagen gezeigt (gefiltert nach `template_type` des gewählten Event-Typs). Der User wählt explizit. Das Backend nimmt die `template_id` direkt entgegen — `findTemplateForGame()` wird nur noch als Fallback genutzt wenn keine `template_id` übergeben wird (für Rückwärtskompatibilität mit regenerate).

### 5. `event_type`-Feld in `games`-Tabelle

Neues Feld `event_type TEXT CHECK('heim','auswärts','generisch') NOT NULL DEFAULT 'heim'` in `games`. Bisher wurde `is_home BOOLEAN` für die Unterscheidung heim/auswärts genutzt; generisch ist neu. `is_home` bleibt für DB-Kompatibilität, wird aber aus `event_type` abgeleitet (`heim` → `is_home=1`, alles andere → `is_home=0`).

### 6. Berechtigungen via neuer Route-Gruppe

```
GET /api/games, /api/games/{id}      → alle authed (aus RequireRole heraus)
POST/PUT/DELETE /api/admin/games/*   → RequireRole("admin","vorstand","trainer")
```

## Risks / Trade-offs

- **Alle Game-Queries brechen**: Jede Query mit `games.team_id` muss auf `JOIN game_teams` umgestellt werden → Mitigation: vollständiger Grep vor Migration, explizite Tests
- **Migration auf VPS**: SQLite-Tabellen-Rebuild ist nicht transaktional in allen Versionen → Mitigation: Backup vor `make deploy`, `migrate down` als Rollback
- **Mehrere Slots pro generischem Event**: Bei 8 Teams × 5 Slot-Typen = 40 Slots für ein Event. Für die Dienstbörse korrekt (jeder sieht seine Team-Slots), aber Admin-Übersicht wird voller → Mitigation: In der Spieltag-Detailseite nach Team gruppieren

## Migration Plan

1. Migration `024_game_teams.up.sql`:
   - `ALTER TABLE games ADD COLUMN event_type TEXT NOT NULL DEFAULT 'heim'`
   - `UPDATE games SET event_type = CASE WHEN is_home=1 THEN 'heim' ELSE 'auswärts' END`
   - `CREATE TABLE game_teams (...)`
   - `INSERT INTO game_teams SELECT id, team_id FROM games`
   - Tabellen-Rebuild `games` ohne `team_id`
2. Backend: alle Handler und Queries auf `game_teams` JOIN umstellen
3. Frontend: Wizard implementieren, Preview-Bug fixen
4. `make deploy` (build + migrate up + restart)

**Rollback:** `migrate down` stellt `team_id` wieder her; altes Frontend deployen.

## Open Questions

- Soll die Spieltag-Detailseite bei Multi-Team-Events die Slots nach Team gruppieren?
- Soll es möglich sein, nachträglich Teams zu einem Event hinzuzufügen oder zu entfernen?
