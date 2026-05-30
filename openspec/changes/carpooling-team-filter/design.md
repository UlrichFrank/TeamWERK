## Context

`GET /api/mitfahrgelegenheiten` gibt derzeit alle zukünftigen Spiele an alle eingeloggten Nutzer zurück — ohne Prüfung der Team-Zugehörigkeit. Das ist ein Datenschutz-Problem: ein Elternteil sieht Spiele fremder Mannschaften. Die Dashboard-Logik löst dasselbe Problem bereits, aber mit rollenspezifischen Subquery-Strings direkt im Go-Code — schwer wiederverwendbar.

## Goals / Non-Goals

**Goals:**
- `GET /api/mitfahrgelegenheiten` filtert nach Team-Zugehörigkeit des Nutzers
- Optionaler `?team_id=X`-Parameter für Elternteile mit Kindern in mehreren Teams
- Zugriffskontrolle in einer DB-View kapseln, die von mehreren Handlern nutzbar ist
- Frontend-Toggle „Alle" → „Team" (kein Logik-Change)

**Non-Goals:**
- Dashboard-Handler noch nicht umstellen (eigener Folge-Task wenn nötig)
- Keine Änderung an anderen Endpoints in diesem Change

## Decisions

### DB-View `user_accessible_teams` statt Go-Subquery-Strings

Eine neue SQLite-View kapselt die gesamte rollenbasierte Zugriffslogik:

```sql
CREATE VIEW user_accessible_teams AS
-- spieler: direkt im Kader
SELECT m.user_id, k.team_id, k.season_id
FROM kader_members km
JOIN kader k ON k.id = km.kader_id
JOIN members m ON m.id = km.member_id
WHERE k.team_id IS NOT NULL

UNION ALL

-- elternteil: via family_links
SELECT fl.parent_user_id AS user_id, k.team_id, k.season_id
FROM family_links fl
JOIN kader_members km ON km.member_id = fl.member_id
JOIN kader k ON k.id = km.kader_id
WHERE k.team_id IS NOT NULL

UNION ALL

-- trainer: via kader_trainers
SELECT m.user_id, k.team_id, k.season_id
FROM kader_trainers kt
JOIN kader k ON k.id = kt.kader_id
JOIN members m ON m.id = kt.member_id
WHERE k.team_id IS NOT NULL;
```

Jeder Handler filtert dann per:
```sql
AND gt.team_id IN (
  SELECT team_id FROM user_accessible_teams
  WHERE user_id = ? AND season_id = ?
)
```

**Rationale:** Rollenlogik an einem Ort, keine Duplizierung zwischen Handlern, kein neues Go-Package nötig. Wiederverwendbar für zukünftige Endpoints. SQLite-Views sind kostenfrei (keine gespeicherten Daten).

**Alternative verworfen:** Go-Funktion die Subquery-String zurückgibt — Logik bleibt in Go verteilt, kann nicht von DB-seitigen Tools geprüft werden.

### Admin/Vorstand-Bypass im Handler, nicht in der View

`admin` und `vorstand` sehen alle Spiele. Der Handler prüft die Rolle und überspringt den View-JOIN komplett — die View enthält keine admin-Zeilen, das ist korrekt so.

**Rationale:** Admin-Bypass explizit im Code lesbar; View bleibt auf echte Team-Zugehörigkeit beschränkt.

### Optionaler `?team_id=X` Query-Parameter

Der Endpunkt akzeptiert einen optionalen `team_id`-Parameter. Wenn gesetzt, wird zusätzlich auf diese team_id gefiltert — aber nur wenn sie in `user_accessible_teams` für diesen Nutzer liegt. Ungültige team_ids (nicht zugänglich) ergeben eine leere Liste, keinen Fehler.

**Rationale:** Ein Elternteil mit zwei Kindern in verschiedenen Teams kann gezielt eine Mannschaft auswählen. Sichert gegen unsanktionierte Team-ID-Enumeration ab.

### Migration statt inline-SQL im Handler

Die View wird als DB-Migration angelegt (`020_user_accessible_teams.up.sql`), nicht als CREATE OR REPLACE im Startup-Code.

**Rationale:** Konsistenz mit bestehendem Migrations-Pattern; View ist in der DB-History nachvollziehbar.

## Risks / Trade-offs

- **family_links oder kader_members fehlen** → leere Liste für Nutzer ohne Zuordnung, kein Fehler. Frontend zeigt Leerstate.
- **View deckt keine „vorstand"-Rolle ab** → vorstand hat keine eigenen Kader-Einträge; wird korrekt wie admin behandelt (kein Filter).
- **Dashboard-Handler** nutzt noch die alte `teamQueryForUser()`-Logik — Inkonsistenz bleibt bis zur Umstellung. Kein Sicherheitsproblem, da Dashboard-Daten ohnehin rollenspezifisch sind.
