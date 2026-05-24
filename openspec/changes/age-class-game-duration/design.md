## Context

TeamWERK verwaltet Handball-Teams verschiedener Altersklassen (A–D Jugend). Die Spieldauer variiert je Altersklasse (Halbzeit-Länge und Pausendauer). Aktuell ist `game_duration_minutes` global in der Spielplan-Vorlage (`game_templates`) gespeichert — für Jugendteams führt das zu manueller Nachpflege aller Vorlagen, wenn sich Verbandsvorgaben ändern.

Das `teams`-Schema hat bereits ein `age_class`-Feld. Die Logik zur Slot-Zeitberechnung liegt in `internal/games`.

## Goals / Non-Goals

**Goals:**
- Halbzeit-Dauer und Pause je Altersklasse (A/B/C/D) zentral konfigurierbar machen
- Einfache Admin-UI zum Bearbeiten dieser Werte (kein Freiformtext, nur Zahlen)
- Slot-Zeitberechnung nutzt bei bekannter Altersklasse automatisch die hinterlegte Regel
- Standardwerte als Seed-Data in der Migration (A: 30/15, B: 25/10, C: 25/10, D: 20/10)

**Non-Goals:**
- Kein Versionsverlauf der Regeländerungen
- Kein Unterschied zwischen Liga-Spielen und Turnieren innerhalb einer Altersklasse
- Kein Unterschied zwischen Liga-Spielen und Turnieren für Heim/Auswärts (immer Altersklassen-Regel)

## Decisions

### 1. Eigene Tabelle `age_class_game_rules` statt Erweiterung von `teams`

`teams` beschreibt eine Mannschaft; Spielregeln sind Verband-/Verbandsvorgaben, keine Team-Eigenschaften. Eine separate Lookup-Tabelle mit `age_class` als Primary Key (TEXT CHECK 'A'/'B'/'C'/'D') ist klarer trennbar und in der Admin-UI separat editierbar.

Alternativen: Spalten direkt auf `teams` — abgelehnt, weil jede Mannschaft denselben Wert tragen würde (Redundanz, Inkonsistenzrisiko).

### 2. Dauern-Quelle ist strikt durch `event_type` bestimmt — kein cascading Override

- **`heim` / `auswärts`**: Dauer kommt ausschließlich aus `age_class_game_rules` (2 × `half_duration_minutes` + `break_minutes`). `game_duration_minutes` der Vorlage wird ignoriert.
- **`generisch`**: Dauer kommt aus dem Feld `duration_minutes` der Vorlage (in der UI als „Dauer", nicht „Spieldauer" beschriftet). Keine Altersklassen-Regel wird konsultiert.

Das Feld in `game_templates` wird von `game_duration_minutes` zu `duration_minutes` umbenannt, um den template-type-neutralen Charakter zu betonen. Eine Migration (`ALTER TABLE … RENAME COLUMN`) ist nötig.

Fehlerfälle:
- Heim/Auswärts + Team ohne `age_class`: HTTP 422
- Generisch + Vorlage ohne `duration_minutes`: HTTP 422

### 3. Nur Admin darf Altersklassen-Regeln bearbeiten

Verband-Vorgaben ändern sich selten; `trainer` und `vorstand` können lesen, aber nur `admin` schreibt.

### 4. Backend in `internal/config` statt eigenem Package

Die Regeln sind Vereins-/Verbandskonfiguration, kein eigener Domänen-Kontext mit komplexer Logik. Das `config`-Package verwaltet bereits `clubs`, `seasons`, `teams` — das passt.

## Risks / Trade-offs

- **Risk**: Heim/Auswärts-Event mit Team ohne `age_class` (NULL) → Slot-Generierung schlägt fehl.
  → Mitigation: Backend gibt HTTP 422 mit klarer Fehlermeldung zurück; Validierung beim Event-Anlegen.

- **Risk**: Umbenennung `game_duration_minutes` → `duration_minutes` bricht bestehende API-Clients.
  → Mitigation: Frontend wird synchron angepasst; kein externer API-Konsument.

- **Risk**: Bestehende Spiele mit bereits generierten Slots haben keine Retro-Korrektur.
  → Mitigation: Nur neue Slots übernehmen die Regel; `regenerate`-Endpunkt existiert bereits für manuelle Korrekturen.

## Migration Plan

1. Migration `011_age_class_game_rules.up.sql`:
   - Tabelle `age_class_game_rules` anlegen mit Seed A(30,15), B(25,10), C(25,10), D(20,10)
   - `ALTER TABLE game_templates RENAME COLUMN game_duration_minutes TO duration_minutes`
2. `internal/games` — Slot-Zeitberechnung: branch nach `event_type` statt cascading lookup
3. Rollback: `011_age_class_game_rules.down.sql` — Tabelle droppen, Spalte zurückbenennen

## Open Questions

- Soll `age_class` auf `teams` ein NOT-NULL-Constraint werden, um NULL-Fälle zu verhindern? Aktuell nullable — ein separates Ticket könnte das nachziehen.
