## Context

Das Mitfahrgelegenheiten-Board (`internal/carpooling/`, Migration 012) zeigt Fahrangebote und Gesuche, aber koordiniert keine Verbindlichkeit. Nutzer kennen sich in einem kleinen Handballverein — das System braucht keinen komplexen Verhandlungsmechanismus, sondern eine klare Zusage mit Push-Feedback.

Bestehende Infrastruktur, die genutzt wird:
- `mitfahrgelegenheiten`-Tabelle mit `plaetze`-Feld (bisher nur für `biete`)
- `internal/notifications/` für Push-Subscriptions
- `internal/carpooling/handler.go` mit List/Upsert/Delete

## Goals / Non-Goals

**Goals:**
- Paarungen zwischen Angeboten und Gesuchen mit Status `pending → confirmed / rejected`
- Beidseitige Initiierung (Bieter oder Sucher kann anfragen)
- Kapazitätsprüfung beim Bestätigen, Auto-Reject überschüssiger Anfragen
- Push-Benachrichtigungen bei Bestätigung, Ablehnung und Stornierung
- Mehrere Gesuche pro User/Spiel erlauben (manuelles Aufteilen)
- Bestätigte Paarungen für alle sichtbar im Board

**Non-Goals:**
- Automatisches Matching (Nutzer entscheidet selbst)
- Chat/Kommentar-Funktion zwischen Fahrtpartnern
- Teilbestätigungen innerhalb eines einzelnen Gesuchs (1 Gesuch = 1 Bieter)

## Decisions

### D1: Neue Tabelle `mitfahrt_paarungen` statt Status-Feld in `mitfahrgelegenheiten`

Ein Paarungs-Status auf dem Eintrag selbst würde die n:m-Beziehung (ein Bieter kann mehrere Sucher bestätigen) nicht abbilden. Eine eigene Tabelle hält Bieter- und Sucher-FK, `anzahl`, `initiiert_von` und `status`.

```sql
CREATE TABLE mitfahrt_paarungen (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  biete_id      INTEGER NOT NULL REFERENCES mitfahrgelegenheiten(id) ON DELETE CASCADE,
  suche_id      INTEGER NOT NULL REFERENCES mitfahrgelegenheiten(id) ON DELETE CASCADE,
  initiiert_von TEXT    NOT NULL CHECK(initiiert_von IN ('biete','suche')),
  status        TEXT    NOT NULL DEFAULT 'pending'
                        CHECK(status IN ('pending','confirmed','rejected')),
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(biete_id, suche_id)
);
```

`anzahl` wird nicht gespeichert — es ist immer `suche.plaetze` (ein Gesuch deckt sich vollständig mit einem Bieter).

### D2: UNIQUE(game_id, user_id) nur noch per-typ enforced via Trigger oder Application-Level

SQLite erlaubt kein partielles UNIQUE-Index mit `WHERE`-Klausel über mehrere Werte auf diese Weise... tatsächlich erlaubt SQLite partielle Indizes:

```sql
CREATE UNIQUE INDEX idx_mitfahr_biete_unique
  ON mitfahrgelegenheiten(game_id, user_id)
  WHERE typ = 'biete';
```

Damit kann ein User pro Spiel nur ein Angebot, aber beliebig viele Gesuche eintragen. Der alte `UNIQUE(game_id, user_id)`-Constraint in der Tabellendefinition wird in der Migration entfernt und durch diesen partiellen Index ersetzt.

### D3: Kapazitätsprüfung beim Anfragen im Handler (nicht per DB-Constraint)

Beim `POST /api/mitfahrt-paarungen` prüft der Handler vor dem Anlegen der Paarung:

1. Bieter hat noch genug freie Plätze: `biete.plaetze - sum(confirmed + pending paarungen.suche.plaetze) >= suche.plaetze`
2. Sucher hat noch keine andere pending/confirmed Paarung für dasselbe Gesuch

Ist die Kapazität nicht ausreichend, antwortet die API sofort mit 409 Conflict — es wird keine Paarung angelegt. Dadurch entfällt jede Auto-Reject-Logik beim Bestätigen.

Beim `POST /api/mitfahrt-paarungen/{id}/confirm` wird zur Sicherheit erneut geprüft (Race Condition), aber im Normalfall ist die Kapazität bereits beim Anfragen reserviert.

### D4: `plaetze` für Gesuche im Formular explizit abfragen

Das Feld existiert bereits in der Tabelle (nullable). Beim Anlegen eines Gesuchs wird es nun als Pflichtfeld behandelt (Validierung im Handler: `plaetze` muss ≥ 1 sein wenn `typ='suche'`).

### D5: Paarungen im List-Endpunkt mitliefern

`GET /api/mitfahrgelegenheiten` gibt künftig für jedes Spiel auch `paarungen: []` mit. Jeder Eintrag enthält Bieter-Name, Sucher-Name, Anzahl, Status. Das Frontend entscheidet die Darstellung. Kein separater Endpunkt nötig.

## Risks / Trade-offs

**Race Condition bei gleichzeitiger Bestätigung** → Kapazitätsprüfung in SQLite-Transaktion mit `BEGIN IMMEDIATE` schützt ausreichend bei Single-Writer-SQLite.

**Mehrere Gesuche pro User/Spiel** → UI muss deutlich machen, dass man mehrere Einträge hat (z.B. Badge-Zähler). Verwirrungsrisiko bei unklarer Darstellung.

**Kapazitätsreservierung durch pending** → Pending-Paarungen werden auf die Kapazität angerechnet. Dadurch kann ein Bieter mit 2 Plätzen keine dritte Anfrage mehr annehmen, auch wenn noch nichts bestätigt ist. Risiko: Bieter bestätigt nie → Plätze dauerhaft blockiert. Mitigation: Bieter kann Anfragen ablehnen; kein automatischer Timeout in dieser Version.

## Migration Plan

1. Migration `013_mitfahrt_paarungen.up.sql`:
   - `ALTER TABLE mitfahrgelegenheiten DROP ... UNIQUE` — in SQLite nicht direkt möglich. Stattdessen: Tabelle neu erstellen ohne UNIQUE-Constraint, Daten kopieren, alten Drop, partiellen Index anlegen.
   - Neue Tabelle `mitfahrt_paarungen` anlegen.
2. Down-Migration: Tabellen-Swap rückgängig, `mitfahrt_paarungen` droppen.
3. Rollback-Strategie: Bei Fehler `migrate down 1`, Binary-Rollback via `make deploy` mit vorherigem Tag.
