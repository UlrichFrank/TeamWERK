## Context

`users.role` hat bisher zwei unabhängige Konzepte in einem Enum vereint:

- **Systemzugriff**: Wer darf welche API-Routen aufrufen und welche UI-Bereiche sehen (admin vs. alle anderen)
- **Vereinsfunktion**: Welche Rolle hat jemand im Verein (Spieler, Trainer, Vorstand, ...)

Die Folge: `members.club_function` existiert parallel als zweite Quelle für dieselbe Information, und es ist strukturell unmöglich, dass ein Mitglied mehrere Funktionen hat (z.B. Spieler, der auch als Trainer tätig ist). Außerdem hat `elternteil` keine Entsprechung in `club_function`, weil Eltern keine Vereinsmitglieder sind — der Begriff bezeichnet eine Beziehung, keine Funktion.

**Nicht jeder Nutzer ist Mitglied.** Es gibt deutlich mehr Nutzerkonten als Mitglieder. Die Vereinsfunktion ist damit ein optionales Attribut des Mitgliedsatzes, nicht des Nutzerkontos.

## Goals / Non-Goals

**Goals:**
- `users.role` auf reine Zugriffsebene reduzieren: `admin | standard`
- Vereinsfunktion als Multi-Value-Attribut direkt am Mitglied speichern (Junction-Tabelle)
- JWT trägt `club_functions []string` und `is_parent bool` — Handler brauchen keine Extra-DB-Abfragen für Funktionschecks
- Dienstpflicht-Priorität bei Mehrfachfunktionen: Trainer > Spieler > Elternteil
- Einladungsflow vereinfacht: nur noch `admin | standard` als Ziel-Rolle

**Non-Goals:**
- Feinkörnigeres RBAC (Row-Level-Policies, Permission-Scopes)
- Änderung der Dienstpflicht-Berechnungslogik selbst (nur Priorität)
- Neue UI-Features jenseits der Formularumstellung für Mehrfachauswahl

## Decisions

### 1. Junction-Tabelle statt Single-Column oder Boolean-Flags

**Entschieden:** `member_club_functions(member_id, function)` mit PRIMARY KEY(member_id, function).

Alternativen:
- *Boolean-Flags* (`is_spieler`, `is_trainer`, …): einfach, aber Schema-Änderung bei jeder neuen Funktion
- *JSON-Array in Spalte*: kein FK, kein Index, keine CHECK-Constraint
- *Junction-Tabelle*: normalisiert, erweiterbar, direkt querybar mit JOIN oder EXISTS

### 2. Vereinsfunktion im JWT (Option A)

**Entschieden:** Login-Query befüllt `club_functions []string` aus `member_club_functions` und `is_parent bool` aus `family_links`. Handler-Code nutzt `claims.HasFunction("trainer")` ohne Extra-DB-Abfrage.

Alternativen:
- *Nur DB-Lookup pro Request*: sauberste Trennung, aber Extra-Abfrage bei jedem Middleware-Check
- *Effektive Rolle ableiten* (z.B. trainer → "trainer" im JWT): entspricht dem alten Zustand, löst das Mehrfachfunktions-Problem nicht

JWT-TTL ist 15 Minuten — eine kurzfristige Inkonsistenz nach Funktionsänderung ist akzeptabel.

### 3. Dienstpflicht-Priorität: Trainer > Spieler > Elternteil

**Entschieden:** Bei Mehrfachfunktionen bestimmt die Funktion höchster Priorität die `target_role` für Dienstpflichtberechnungen. Die Funktion `effectivePersona(clubFunctions []string, isParent bool) string` kapselt diese Logik einmalig.

`duty_season_targets.target_role` und `duty_types.target_role` behalten ihre bestehenden Werte — sie bezeichnen Dienstpflicht-Kategorien, nicht Nutzerrollen.

### 4. Einladung ohne Vereinsfunktion

**Entschieden:** `invitation_tokens.target_role` kennt nur noch `'admin' | 'standard'`. Die Vereinsfunktion entsteht unabhängig, wenn ein Mitglied mit dem Nutzerkonto verknüpft wird. Nur Admins können Admins einladen — die `roleRank`-Map entfällt.

### 5. SQLite-Tabellenmigration via Rekreation

Da SQLite `ALTER TABLE … DROP COLUMN` erst ab 3.35 stabil unterstützt und CHECK-Constraints nicht änderbar sind, werden `users` und `invitation_tokens` per Standard-SQLite-Pattern migriert: neue Tabelle anlegen, Daten kopieren, alte Tabelle droppen, umbenennen. golang-migrate führt alles in einer Transaktion aus.

## Risks / Trade-offs

| Risiko | Mitigation |
|---|---|
| Alle aktiven JWT-Sessions werden ungültig (Breaking JWT Change) | Deploy-Fenster kommunizieren; Nutzer müssen sich neu einloggen (15 min TTL, kaum Auswirkung) |
| SQLite-Tabellenmigration von `users` ist destruktiv | Down-Migration in `.down.sql` vollständig umkehren; pre-deploy Backup via `make backup-remote` |
| Bestehende Datensätze: `users.role='trainer'` → `standard`, aber Mitglied hat `club_function='trainer'` schon korrekt | Migration liest `members.club_function` für Junction-Tabelle; `users.role` wird nur zu `standard` gesetzt |
| `duty_season_targets.target_role` referenziert weiterhin alte Werte ('spieler', 'elternteil') | Werte bleiben semantisch gültig, werden zur Laufzeit gegen `club_functions`/`is_parent` gemappt |

## Migration Plan

```sql
-- 1. Junction-Tabelle anlegen
CREATE TABLE member_club_functions (
  member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  function  TEXT    NOT NULL CHECK(function IN ('spieler','trainer','vorstand','vorstand_beisitzer')),
  PRIMARY KEY (member_id, function)
);

-- 2. Bestehende club_function-Daten migrieren
INSERT INTO member_club_functions (member_id, function)
SELECT id, club_function FROM members WHERE club_function IS NOT NULL;

-- 3. users-Tabelle rekreieren (CHECK-Constraint ändern)
CREATE TABLE users_new (
  id       INTEGER  PRIMARY KEY AUTOINCREMENT,
  email    TEXT     NOT NULL UNIQUE,
  name     TEXT     NOT NULL,
  password TEXT     NOT NULL,
  role     TEXT     NOT NULL DEFAULT 'standard'
                   CHECK(role IN ('admin','standard')),
  team_id  INTEGER  REFERENCES teams(id)
);
INSERT INTO users_new SELECT id, email, name, password,
  CASE WHEN role = 'admin' THEN 'admin' ELSE 'standard' END,
  team_id FROM users;
DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

-- 4. invitation_tokens rekreieren
-- (analog: target_role CHECK auf 'admin'|'standard' reduzieren,
--  bestehende Tokens auf 'standard' setzen)

-- 5. members.club_function-Spalte entfernen (ebenfalls via Rekreation)
```

**Rollback:** `.down.sql` rekreiert Originaltabellen; Junction-Tabelle wird gedroppt. Datenverlust bei Mehrfachfunktionen, die nach der Migration hinzugefügt wurden.

**Deploy-Reihenfolge:**
1. `make deploy` führt `migrate up` automatisch aus
2. Alle aktiven JWT-Sessions verfallen (15 min max)
3. Kein manueller Eingriff nötig

## Open Questions

- Soll beim Deploy ein expliziter `DELETE FROM refresh_tokens` ausgeführt werden, um Refresh-Token-Sessions sofort zu beenden (statt auf natürlichen Verfall zu warten)?
