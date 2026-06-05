## Context

TeamWERK hat ein vollständig implementiertes RSVP-System für Trainings (`training_responses`) und
Spiele (`game_responses`). Bisher gibt es kein konfigurierbares Verhalten: Spieler müssen immer
aktiv zusagen, und eine Begründungspflicht ist nicht implementiert (der `rsvp-reason-modal`-Change
ist noch offen). Dieser Change ergänzt zwei Flags pro Termin und integriert die Modal-Logik.

Betroffene Tabellen: `training_series`, `training_sessions`, `games`.
Betroffene Handler: `internal/trainings/handler.go`, `internal/games/handler.go`.
Betroffene Frontend-Seite: `TerminePage.tsx`, Kalender-Wizard in `KalenderPage.tsx`,
Serie-Formular in `AdminTrainingsPage.tsx`.

## Goals / Non-Goals

**Goals:**
- `rsvp_opt_out` und `rsvp_require_reason` pro Termin konfigurierbar
- Sessions erben Flags einmalig von der Serie beim Anlegen
- Confirmed-Count bei Opt-Out berücksichtigt Spieler ohne Response-Eintrag
- Modal erscheint nur wenn `rsvp_require_reason = 1`
- Ersetzt `rsvp-reason-modal`-Change vollständig

**Non-Goals:**
- Live-Vererbung (Session zieht keine Änderungen der Serie nach)
- Bulk-Update bestehender Sessions bei Serien-Änderung
- Push-Benachrichtigung wenn ein Opt-Out-Termin angelegt wird

## Decisions

### D1: Flags als INTEGER (0/1) statt BOOLEAN

**Entscheidung**: `rsvp_opt_out INTEGER NOT NULL DEFAULT 0 CHECK(rsvp_opt_out IN (0,1))`,
`rsvp_require_reason INTEGER NOT NULL DEFAULT 1 CHECK(rsvp_require_reason IN (0,1))`.

**Rationale**: SQLite hat keinen BOOLEAN-Typ; INTEGER 0/1 ist der etablierte Projektstandard
(vgl. `is_home`, `present` in `training_attendances`).

### D2: Einmalige Vererbung beim Anlegen (Option A)

**Entscheidung**: Beim Erstellen einer Session via `POST /api/training-sessions` werden die
aktuellen Flags der Serie auf die Session kopiert. Kein nullable-Feld, keine COALESCE-Logik
zur Laufzeit.

**Rationale**: Einfacheres Schema (NOT NULL, kein nullable), keine JOIN-Komplexität in
Query-Pfaden. Der Nachteil (Serien-Änderung wirkt nicht rückwirkend) ist akzeptiert.

**Alternative**: Nullable override mit `COALESCE(session.flag, series.flag)` — abgelehnt
wegen erhöhter Query-Komplexität ohne praktischen Mehrwert für den Use-Case.

### D3: Flags beim Bearbeiten einer Session nicht editierbar

**Entscheidung**: Die Edit-API (`PUT /api/training-sessions/{id}`) nimmt `rsvp_opt_out` und
`rsvp_require_reason` nicht entgegen. Das Frontend blendet die Felder im Edit-Modal aus.

**Rationale**: Verhindert inkonsistente Zustände (z.B. nachträgliches Opt-Out wenn Spieler
bereits abgesagt haben). Wer die Policy ändert, legt neue Sessions an.

### D4: Confirmed-Count als SQL-Berechnung, kein denormalisiertes Feld

**Entscheidung**: Wenn `rsvp_opt_out = 1`, wird `confirmed_count` berechnet als:
```sql
(Anzahl explizit confirmed) + (Gesamtzahl Team-Mitglieder ohne Response-Eintrag)
```
Dies geschieht in der bestehenden ListSessions/ListMyGames-Query via Subquery.

**Rationale**: Kein denormalisiertes Zählfeld das bei jedem Insert/Delete aktualisiert werden
müsste. Die Query lädt sowieso bereits Response-Counts; ein zusätzlicher Subquery auf
`team_memberships` ist bei < 30 Spielern pro Team trivial.

Konkret (für training_sessions):
```sql
CASE WHEN ts.rsvp_opt_out = 1
  THEN (
    SELECT COUNT(*) FROM team_memberships tm2
    JOIN members m2 ON m2.id = tm2.member_id
    WHERE tm2.team_id = ts.team_id
    AND NOT EXISTS (
      SELECT 1 FROM training_responses tr2
      WHERE tr2.training_id = ts.id AND tr2.member_id = tm2.member_id
    )
  ) + COALESCE(SUM(CASE WHEN tr.status='confirmed' THEN 1 END), 0)
  ELSE COALESCE(SUM(CASE WHEN tr.status='confirmed' THEN 1 END), 0)
END AS confirmed_count
```

### D5: my_rsvp gibt 'confirmed' zurück wenn opt-out und kein Eintrag

**Entscheidung**: In `ListSessions` und `ListMyGames` gilt:
```go
if session.RsvpOptOut && myRSVP == nil {
    confirmed := "confirmed"
    session.MyRSVP = &confirmed
}
```

**Rationale**: Frontend braucht keinen opt_out-Flag selbst zu interpretieren —
`my_rsvp = "confirmed"` reicht, um den Zusagen-Button vorausgewählt darzustellen.
Konsistent mit dem bisherigen Verhalten (Frontend schaut nur auf `my_rsvp`).

### D6: Default rsvp_require_reason für generische Events = 0

**Entscheidung**: Beim Anlegen eines Spiels mit `event_type = 'generisch'` setzt der
Backend-Handler `rsvp_require_reason = 0` als Default (statt 1).

**Rationale**: Generische Events (Sommerfest, Jahreshauptversammlung) brauchen keine
Begründungspflicht. Das Frontend kann den Checkbox-Default entsprechend vorbelegen,
aber die Entscheidung liegt beim Anlegen.

## Risks / Trade-offs

- **[Trade-off] Kein rückwirkendes Opt-Out** → Bestehende Sessions mit vielen
  "kein Eintrag"-Spielern ändern ihr Verhalten nicht wenn die Serie nachträglich
  auf opt-out gestellt wird. Akzeptiert: Trainer legt neue Sessions an.

- **[Risiko] confirmed_count Über-/Unterschätzung bei Opt-Out** → Wenn ein Spieler
  das Team verlässt (team_membership endet), zählt er nicht mehr ohne Response-Eintrag.
  Das ist korrekt. Wenn ein Spieler neu ins Team kommt, zählt er sofort als confirmed
  (kein Eintrag = confirmed). Das ist die erwartete Semantik.

- **[Risiko] game_responses hat kein season_id-Feld** → Die Gesamtzahl aktiver Team-Mitglieder
  muss via `team_memberships JOIN seasons WHERE is_active = 1` ermittelt werden.
  Für Spiele bleibt die Season via `games.season_id` abrufbar.

## Migration Plan

1. Migration 015 ausführen (`rsvp_opt_out` + `rsvp_require_reason` auf alle 3 Tabellen;
   `training_sessions` DEFAULT 0 / DEFAULT 1 für bestehende Zeilen — kein Verhalten ändert sich)
2. Backend deployen (neue Flags in Create-Handlern, neue confirmed_count-Logik)
3. Frontend deployen (Modal-Logik, Opt-Out-UI, Konfigurations-Checkboxen im Wizard)

Rollback: Migration 015 down (Spalten droppen via Tabellen-Neuerstellung, da SQLite
kein DROP COLUMN vor Version 3.35 unterstützt).
