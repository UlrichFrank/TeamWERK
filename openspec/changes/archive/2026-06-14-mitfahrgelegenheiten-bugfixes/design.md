## Context

Die Mitfahrgelegenheiten-Seite hat vier zusammenhängende Probleme im Backend-Handler `internal/carpooling/handler.go` und im Frontend `MitfahrgelegenheitenPage.tsx`.

Aktueller Stand:
- `Upsert()`: `biete` → UPSERT-Logik; `suche` → blindes INSERT (kein Duplikatschutz)
- `List()`: `team_id`-Parameter ist implementiert, aber das Frontend schickt ihn nie
- `List()`-Query: `JOIN game_teams … JOIN teams` → ein Row pro Team-Verknüpfung → generische Events mit N Teams erscheinen N-mal
- `FormModal`: Typ-Wechsel resettet nur `plaetze`, nicht `treffpunkt`/`notiz`

## Goals / Non-Goals

**Goals:**
- `suche` erhält dieselbe UPSERT-Logik wie `biete` (SELECT → UPDATE oder INSERT)
- UNIQUE INDEX `(game_id, user_id) WHERE typ='suche'` als Datenbank-Safety-Net
- Bestehende `suche`-Duplikate werden vor der Migration dedupliziert (nur neuesten Row behalten)
- Typ-Wechsel im Modal resettet alle Formularfelder auf Defaults
- Team-Dropdown auf der Page; Nutzer in genau einem Team sehen ihn nicht
- Generische Events mit mehreren Teams erscheinen genau einmal; Team-Namen komma-separiert

**Non-Goals:**
- Kein Redesign der Seite
- Keine Änderung der Paarungs-Logik

## Decisions

### 1. UPSERT für `suche`: analog zu `biete`

```go
// Wie biete: SELECT existingID → UPDATE oder INSERT
scanErr := h.db.QueryRowContext(ctx,
    `SELECT id FROM mitfahrgelegenheiten WHERE game_id=? AND user_id=? AND typ='suche'`,
    body.GameID, userID).Scan(&existingID)
if scanErr == sql.ErrNoRows {
    INSERT …
} else {
    UPDATE … WHERE id = existingID
}
```

Der UNIQUE INDEX ist zusätzliche Absicherung für Race Conditions.

### 2. Duplikate bereinigen vor Migration

Down-Migration muss Duplikate entfernen, bevor der Index greift:
```sql
DELETE FROM mitfahrgelegenheiten
WHERE typ = 'suche'
  AND id NOT IN (
    SELECT MAX(id) FROM mitfahrgelegenheiten
    WHERE typ = 'suche'
    GROUP BY game_id, user_id
  );
CREATE UNIQUE INDEX idx_mitfahr_suche_unique
    ON mitfahrgelegenheiten(game_id, user_id) WHERE typ = 'suche';
```

### 3. Generische Events: GROUP BY + GROUP_CONCAT

Statt `JOIN teams` → `GROUP_CONCAT(t.name, ', ')` + `GROUP BY g.id`:

```sql
SELECT g.id, g.date, g.opponent,
       GROUP_CONCAT(t.name, ', ') AS team_names,
       g.event_type
FROM games g
JOIN game_teams gt ON g.id = gt.game_id
JOIN teams t ON t.id = gt.team_id
WHERE …
GROUP BY g.id
ORDER BY g.date ASC
```

Das Backend gibt `team` als komma-separierter String zurück — kein Frontend-Interface-Änderung nötig (war schon `string`).

### 4. Team-Dropdown: nur anzeigen wenn Nutzer Zugang zu >1 Team hat

Der Response von `GET /mitfahrgelegenheiten` kann bereits um `accessibleTeams []{ id, name }` erweitert werden, sodass das Frontend kein separates `GET /teams`-Call braucht.

Alternative: Frontend lädt `GET /teams/my`. Einfacher, da `teams/my` schon existiert.

**Gewählt:** Separater `GET /teams/my`-Call beim Load. Einfacher, keine Backend-Response-Änderung.

Dropdown nur rendern wenn `myTeams.length > 1`. Initial-Filter: kein Filter (alle Teams) oder automatisch das erste Team vorselektieren wenn exakt ein Team.

### 5. Modal zeigt bestehende Einträge

Beim Öffnen des Modals wird der passende eigene Eintrag aus `response.games` (bereits im Speicher) herausgesucht und als `initialBiete` / `initialSuche` übergeben. `FormModal` initialisiert die Felder daraus.

Beim Typ-Wechsel innerhalb des Modals:
```tsx
const switchTyp = (next: 'biete' | 'suche') => {
  setTyp(next)
  const existing = next === 'biete' ? initialBiete : initialSuche
  setTreffpunkt(existing?.treffpunkt ?? '')
  setNotiz(existing?.notiz ?? '')
  setPlaetze(existing ? String(existing.plaetze ?? 1) : next === 'biete' ? String(vehicleSeats ?? 1) : '1')
}
```

Kein Extra-API-Call nötig — die Daten liegen bereits in `response.games[x].biete/suche` mit `isOwn: true`.

## Risks / Trade-offs

- **Duplikat-Bereinigung**: `MAX(id)` behält den neuesten `suche`-Eintrag. Paarungen, die auf gelöschte Duplikate zeigen, werden via CASCADE automatisch bereinigt (FK auf mitfahrgelegenheiten mit `ON DELETE CASCADE`).
- **GROUP_CONCAT**: SQLite-Standard, kein Kompatibilitätsproblem.
- **Team-Filter Initialzustand**: Kein Auto-Select, User muss explizit filtern. Einfacher als Auto-Select-Logik.
