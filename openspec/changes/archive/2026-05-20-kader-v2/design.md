## Context

Die erste Kader-Implementation (Season-based Kader Management) legte die Grundstruktur: `kader`-Tabelle mit `(season_id, age_class, gender)`, Mitgliederzuweisung via `kader_members`, Content-Assist-Suche mit Bracket-Filterung. Drei Probleme wurden nachträglich identifiziert:

1. **Falsche Basisjahre** in `ageBracketRef2025`: überlappende Werte (A=2006/07, B=2007/08 usw.) statt DHB-konformer 2-Jahres-Lücken zwischen Klassen.
2. **Kein Modus-Konzept**: Pro Altersklasse/Geschlecht gibt es genau einen Kader. Ein Verein kann aber zwei Mannschaften pro Altersklasse haben — je eine pro Jahrgang (dediziert) — oder eine gemischte Mannschaft.
3. **Kein Lifecycle**: Es gibt keinen Endpoint zum Anlegen oder Löschen einzelner Kader nach der Initialisierung.

## Goals / Non-Goals

**Goals:**
- DHB-konforme Jahrgangskalkulation: A-Jugend=2007/08, B-Jugend=2009/10, C-Jugend=2011/12, D-Jugend=2013/14 für Saison 2025/26; Offset +1 pro Saison bleibt
- `dedicated_birth_year` pro Kader: NULL → gemischt (voller Bracket), Jahreszahl → nur dieser Jahrgang im Filter/Auto-Assign
- Mehrere Teams pro Altersklasse/Geschlecht via `team_number` (1 oder 2)
- Kader anlegen (einzeln) und löschen (nur wenn leer)
- Kachel zeigt zugeordnete Jahrgänge; Modus-Umschalter in der UI

**Non-Goals:**
- Mehr als zwei Teams pro Altersklasse/Geschlecht (team_number > 2)
- Automatische Migration bestehender Kader-Daten in dedizierte Jahrgänge
- Copy-Workflow mit Jahrgangsmodus (bleibt wie bisher: member_source-Optionen)

## Decisions

### 1. `dedicated_birth_year INTEGER` statt `year_mode` Enum

**Entschieden**: Nullable Integer — `NULL` = gemischt, `2011` = nur Jahrgang 2011.

**Alternative**: `year_mode TEXT CHECK('mixed','dedicated')` + `birth_year INTEGER`. Erfordert zwei Spalten und eine CHECK-Constraint-Kombination (year_mode='dedicated' → birth_year NOT NULL).

**Rationale**: Der Nullable Integer ist selbstdokumentierend, erfordert keine JOIN-Logik und lässt sich direkt in der Filterabfrage nutzen (`WHERE birth_year = ?` vs. `BETWEEN ? AND ?`).

### 2. `team_number` als Integer (1 oder 2)

**Entschieden**: `INTEGER NOT NULL DEFAULT 1`, UNIQUE auf `(season_id, age_class, gender, team_number)`.

**Alternative**: Freier Name (`team_name TEXT`, z.B. „C-Jugend 1 Stuttgart"). Zu viel Freiheit, schwer zu sortieren und in der Bracket-Logik zu referenzieren.

**Rationale**: Nummer 1 und 2 sind im deutschen Handball Standard. Keine weiteren Teams pro Altersklasse geplant. Die Anzeige im Frontend bleibt sprechend: „C-Jugend 1 (Jg. 2011)".

### 3. Löschen nur wenn leer (kein Force-Delete)

**Entschieden**: `DELETE /api/admin/kader/{id}` gibt HTTP 409 zurück wenn `kader_members` Einträge existiert, mit Body `{"error": "...", "member_count": N}`.

**Alternative**: Force-Delete mit Kaskade. Zu riskant für versehentliches Datenverlust.

**Rationale**: Der Admin muss Mitglieder erst manuell entfernen oder umbuchen. Das ist eine bewusste Entscheidung, keine technische Limitierung.

### 4. Backend berechnet `birth_years` im Response

`ListKader` und `GetKader` geben ein zusätzliches Feld `birth_years: []int` zurück (aus `dedicated_birth_year` oder aus `ComputeAgeBrackets`). Das Frontend zeigt diesen Wert direkt an — keine clientseitige Berechnung.

### 5. Modus-Umschalter im Frontend via PUT

Wenn der Nutzer von „gemischt" auf „dediziert" umschaltet (oder umgekehrt), ruft das Frontend `PUT /api/admin/kader/{id}` mit `{"dedicated_birth_year": 2011}` oder `{"dedicated_birth_year": null}` auf. Kein eigener Endpoint nötig.

## Risks / Trade-offs

- **Bestehende Kader nach Migration**: Nach der DB-Migration haben alle existierenden Kader `team_number=1` und `dedicated_birth_year=NULL` (Default). Das ist korrekt (Jahrgangsmischung, eine Mannschaft) und erfordert keine Datenmigration.
- **age_brackets_test.go**: Die Tests wurden mit den alten (falschen) Referenzwerten geschrieben und müssen aktualisiert werden. Das ist ein bewusster Breaking Change in den Testwerten.
- **Copy-Workflow**: `copyKader` und `autoAssignMembers` kennen noch keinen `dedicated_birth_year`. Beim Kopieren wird `dedicated_birth_year=NULL` gesetzt (Jahrgangsmischung), was sicher ist. Die Assignments-Logik nutzt weiterhin den vollen Bracket für `auto-assign`.

## Migration Plan

Migration `014_kader_team_number.up.sql`:
```sql
ALTER TABLE kader ADD COLUMN team_number INTEGER NOT NULL DEFAULT 1;
ALTER TABLE kader ADD COLUMN dedicated_birth_year INTEGER;
-- SQLite unterstützt kein ALTER CONSTRAINT — neuer Unique-Index stattdessen:
DROP INDEX IF EXISTS idx_kader_season; -- vorhandener Index auf season_id
CREATE UNIQUE INDEX kader_unique ON kader(season_id, age_class, gender, team_number);
```

Rollback via `014_kader_team_number.down.sql`:
```sql
DROP INDEX IF EXISTS kader_unique;
CREATE INDEX idx_kader_season ON kader(season_id);
-- SQLite unterstützt kein DROP COLUMN vor 3.35 → Tabelle neu aufbauen (nicht nötig für Rollback im Test)
```

Deployment: `make deploy` führt `migrate up` automatisch aus. Kein Downtime erforderlich (additive Migration).

## Open Questions

- Soll `team_number` im Frontend-Label immer angezeigt werden (auch wenn es nur 1 Team gibt), oder nur wenn > 1 Team pro Klasse existiert? → Empfehlung: nur wenn team_number > 1 oder ein zweites Team für die Klasse existiert.
