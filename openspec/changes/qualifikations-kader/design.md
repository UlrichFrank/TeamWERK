## Context

Die `kader`-Tabelle hat aktuell einen UNIQUE-Constraint auf `(season_id, age_class, gender, team_number)`. Das verhindert, dass für dieselbe Altersklasse/Geschlecht in einer Saison zwei parallele Kader existieren können. Im Handball-Betrieb gibt es jedoch Qualifikationsphasen, in denen ein temporärer Kader in veränderter Zusammensetzung (ggf. hochgezogene Jugendspieler, anderer Trainer) parallel zum Saisonkader spielberechtigt ist.

Bestehende Kader-Infrastruktur (kader_members, kader_trainers, Spielzuordnung, Dienste) soll unverändert bleiben — nur die Constraint-Logik und das Aktivierungskonzept ändern sich.

## Goals / Non-Goals

**Goals:**
- Pro Altersklasse/Geschlecht/Saison darf maximal ein aktiver regulärer + ein aktiver Qualifikationskader gleichzeitig existieren
- Aktivierung wird explizit gesetzt (nicht automatisch anhand von Daten)
- Inaktive Kader bleiben als History erhalten (keine harten Löschungen)
- Admin-UI im Saisons-Tab zeigt pro Altersklasse/Geschlecht die Kader-Auswahl

**Non-Goals:**
- Zeitbasierte automatische Umschaltung zwischen Kadern
- Mehr als zwei parallele Kader pro Altersklasse/Geschlecht
- Separate Dienst- oder Spielabrechnung für den Qualifikationskader

## Decisions

### 1. `type` + `is_active` auf der `kader`-Tabelle

Statt eines neuen `qualification_periods`-Konzepts werden zwei Felder zur bestehenden `kader`-Tabelle hinzugefügt:

```sql
ALTER TABLE kader ADD COLUMN type      TEXT    NOT NULL DEFAULT 'regular'
    CHECK(type IN ('regular','qualification'));
ALTER TABLE kader ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1;
```

**Begründung:** Alles Bestehende (kader_members, kader_trainers, Spiel-FK, Dienst-Audiences) bleibt unverändert. Ein separates Konzept würde Duplikation der gesamten Kader-Infrastruktur bedeuten.

**Alternative verworfen:** `team_number=2` als Workaround — falsche Semantik, führt zu Fehlern im `teamLabel()`-Display.

### 2. Zwei partielle UNIQUE-Indizes statt eines globalen

```sql
-- Bestehenden Index droppen:
DROP INDEX kader_unique;

-- Neue partielle Indizes:
CREATE UNIQUE INDEX kader_unique_active_regular
    ON kader(season_id, age_class, gender, team_number)
    WHERE type='regular' AND is_active=1;

CREATE UNIQUE INDEX kader_unique_active_quali
    ON kader(season_id, age_class, gender)
    WHERE type='qualification' AND is_active=1;
```

**Begründung:** SQLite unterstützt partielle Unique-Indizes. Damit kann es beliebig viele inaktive Kader historisch geben, aber maximal einen aktiven pro Typ/Altersklasse/Geschlecht.

### 3. Aktivierung als expliziter API-Endpunkt

```
PUT /api/admin/kader/:id/activate
```

Setzt den Kader auf `is_active=1` und alle anderen Kader desselben `(season_id, age_class, gender, type)` auf `is_active=0` — analog zu `PUT /api/admin/seasons/:id/activate`.

**Begründung:** Atomare Transaktion, klar definiertes Verhalten, kein Race Condition-Risiko.

### 4. Kader-Listing filtert standardmäßig auf `is_active=1`

Die bestehenden Listing-Abfragen erhalten `WHERE is_active=1`. Ein optionaler Query-Parameter `?include_inactive=true` kann für die History-Ansicht ergänzt werden (vorerst nicht priorisiert).

## Risks / Trade-offs

- **Bestehende Queries ohne `is_active`-Filter** könnten nach Migration inaktive Kader zurückgeben → alle relevanten Queries in `handler.go` und `copy.go` prüfen und filtern
- **Migration muss `is_active=1` setzen**, bevor der neue Unique-Index greift → Reihenfolge in der Migration-SQL kritisch (erst ALTER, dann DROP INDEX, dann CREATE INDEX)
- **Kein automatischer Quasi-Zeitraum**: Trainer müssen den Qualikader manuell aktivieren und nach Ende wieder deaktivieren — bewusste Entscheidung, da Automatisierung fehleranfällig wäre

## Migration Plan

Migration `015_qualifikations_kader.up.sql`:

1. `ALTER TABLE kader ADD COLUMN type TEXT NOT NULL DEFAULT 'regular' CHECK(...)`
2. `ALTER TABLE kader ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1`
3. `DROP INDEX kader_unique`
4. `CREATE UNIQUE INDEX kader_unique_active_regular ... WHERE type='regular' AND is_active=1`
5. `CREATE UNIQUE INDEX kader_unique_active_quali ... WHERE type='qualification' AND is_active=1`

Rollback (`015_...down.sql`): Inverse Reihenfolge, `ALTER TABLE` via Tabellen-Rebuild (SQLite).

## Open Questions

- Soll ein deaktivierter Qualikader in der UI sichtbar sein (z.B. als „Archiv"-Eintrag unter dem aktiven Kader)? → Erstversion: nein, nur aktive Kader sichtbar.
