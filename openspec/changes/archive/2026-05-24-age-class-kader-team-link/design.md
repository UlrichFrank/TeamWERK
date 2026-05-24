## Context

`age_class_game_rules` (Migration 010) wurde mit Kurzform-PKs ('A'–'D') angelegt, während `teams.age_class` und `kader.age_class` seit jeher Langformen ('A-Jugend' usw.) speichern. `effectiveEventDuration` in `internal/games` überbrückt das mit einem `[:1]`-Hack. Es gibt keinen FK-Constraint, der verhindert, dass Teams mit freiem Text angelegt werden, und die Admin-UI für Teams hat kein validiertes Dropdown.

Betroffene Dateien: `internal/db/migrations/010_*`, `internal/config/handler.go`, `internal/games/handler.go`, `web/src/pages/AdminAgeClassRulesPage.tsx`, `web/src/pages/AdminDutyTypesPage.tsx` (Teams-Sektion).

## Goals / Non-Goals

**Goals:**
- `age_class_game_rules.age_class` verwendet dieselben Werte wie `teams.age_class` ('A-Jugend' usw.)
- `teams.age_class` ist per FK auf `age_class_game_rules.age_class` beschränkt (nullable)
- Admin-UI für Teams bietet ein Dropdown aus der DB statt Freitext
- `effectiveEventDuration`-Workaround entfernt

**Non-Goals:**
- FK auf `kader.age_class` (Kader-Daten kommen aus Importen; Validierung dort separat)
- Neue Altersklassen jenseits A–D (E-/F-Jugend, Männer, Frauen) — diese Teams bleiben NULL
- Versionsverlauf oder Audit-Log der Regeländerungen

## Decisions

### 1. Langform-Keys in `age_class_game_rules` statt Anpassung der Teams-Daten

Bestehende Teams und Kader-Einträge nutzen bereits Langformen. Die Daten zu migrieren (z.B. 'B-Jugend' → 'B') würde alle Import-Logiken und UI-Labels brechen. Einfacher und risikoärmer ist es, `age_class_game_rules` anzupassen.

**Alternativen:**
- Teams auf Kurzform migrieren: bricht UI-Labels, Import-Code, Kader-Logik
- Mapping-Tabelle: unnötige Komplexität für vier feste Werte

### 2. Neue Migration statt Änderung von 010

Migration 010 ist bereits auf Produktion angewendet (auch wenn lokal unversioniert). Eine neue Migration 011 ist sicherer:
1. Existing rows in `age_class_game_rules` werden auf Langform umbenannt (`UPDATE … SET age_class='A-Jugend' WHERE age_class='A'`)
2. CHECK-Constraint kann nicht nachträglich via ALTER geändert werden (SQLite-Einschränkung) → Tabelle neu erstellen
3. FK auf `teams.age_class` via neues Constraint-Flag auf `teams`

**SQLite-Constraint-Änderung:** SQLite unterstützt kein `ALTER COLUMN`. Vorgehen: neue Tabelle anlegen, Daten kopieren, alte droppen, umbenennen.

### 3. FK nullable — Teams ohne Jugendklasse sind erlaubt

Erwachsene Teams (Männer, Frauen) haben keine Jugendklassen-Regel. `teams.age_class` bleibt nullable; ein NULL-Wert bei Heim/Auswärts-Event gibt HTTP 422 (bestehende Fehlermeldung bleibt).

### 4. Dropdown im Frontend aus `/api/admin/age-class-rules`

Der bestehende GET-Endpunkt liefert alle konfigurierten Klassen. Die Teams-Admin-UI (bisher Freitext) ruft diesen Endpoint ab und zeigt ein `<select>`. Kein neuer Backend-Endpoint nötig.

## Risks / Trade-offs

- **Risk**: SQLite-Tabellen-Rewrite in der Migration schlägt fehl bei großen Datenmengen.
  → Mitigation: Innerhalb einer Transaktion; `teams` hat typisch < 20 Einträge.

- **Risk**: FK-Constraint auf `teams.age_class` verhindert das Anlegen von Teams mit freiem Text für neue Klassen (z.B. 'E-Jugend') ohne gleichzeitigen Eintrag in `age_class_game_rules`.
  → Mitigation: Akzeptiert — neue Klassen müssen zuerst in den Regeln angelegt werden. Admin-UI macht das transparent.

- **Risk**: Migration auf Produktion ändert `age_class_game_rules`-PKs; falls der Backend-Code vor der Migration deployt wird, bricht der bisherige Workaround nicht, aber nach Migration ist der Workaround tot Code.
  → Mitigation: Code-Änderung (Workaround entfernen) und Migration in einem Deployment.

## Migration Plan

1. **Migration 011** (`011_age_class_canonical.up.sql`):
   - `age_class_game_rules` neu erstellen mit Langform-PKs und Langform-Check-Constraint
   - Daten mit Langform-Keys einfügen
   - `teams`-Tabelle mit FK-Constraint neu erstellen (SQLite-Rewrite-Pattern)
   - Bestehende Teamdaten kopieren

2. **Backend**: `validAgeClasses`-Map auf Langform; Workaround in `effectiveEventDuration` entfernen

3. **Frontend**: `AdminAgeClassRulesPage.tsx` — kein `-Jugend`-Anhängen; Teams-Admin — Dropdown statt `<input type="text">`

4. **Rollback** (`011_age_class_canonical.down.sql`): Zurück zu Kurzform-Keys; FK auf `teams` entfernen; Workaround muss manuell reaktiviert werden (daher: zuerst Code, dann Migration deployen, Rollback-Reihenfolge umgekehrt)

## Open Questions

- Soll die Admin-UI für Altersklassen-Regeln das Hinzufügen neuer Klassen ermöglichen (z.B. 'E-Jugend')? Aktuell: nur A–D editierbar. → Für diesen Change ausgeschlossen; separates Ticket.
