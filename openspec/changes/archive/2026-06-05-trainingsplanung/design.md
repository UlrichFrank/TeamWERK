## Context

TeamWERK verwaltet bereits Spieltermine (`games`), Dienstslots (`duty_slots`) und Mitgliederdaten. Trainings fehlen als eigenständige Entität. Das neue Modul baut auf denselben Mustern auf: materialisierte Termine, direkte DB-Queries ohne ORM, Handler-Struct-Pattern im neuen Package `internal/trainings/`.

Relevante bestehende Strukturen:
- `teams` + `seasons` als FK-Basis für alle teambasierten Daten
- `family_links` für Elternteil-Kind-Beziehungen
- `members.user_id` verknüpft Spieler-Account mit Mitglied
- `duty_slots` / `duty_assignments` als Vorbild für Event + Response Pattern

## Goals / Non-Goals

**Goals:**
- Trainer können Trainingsserien (fester Wochentag) und Einzeltermine anlegen
- Spieler/Eltern können Zu-/Absagen mit optionaler Begründung geben
- Trainer können nach dem Training die tatsächliche Anwesenheit erfassen
- Trainings erscheinen im bestehenden Kalender neben Spielen
- Privacy: Absage-Begründungen nur für Trainer/Admin + betroffenen Spieler/Elternteil sichtbar

**Non-Goals:**
- Trainingsstatistiken und Auswertungen (Ebene 3 — späteres Feature)
- Chat/Nachrichten-System
- Push-Notifications für neue Trainingstermine
- Komplexe Recurrence-Rules (iCal RRULE) — nur feste Wochentage

## Decisions

### 1. Materialisierte Sessions statt virtueller Berechnung

**Entscheidung:** Beim Anlegen einer `training_series` werden alle `training_sessions` sofort als Rows in der DB generiert (vom `valid_from`-Datum bis `valid_until`, für jeden passenden Wochentag).

**Alternativen:** Sessions on-the-fly berechnen (nur Regel speichern).

**Begründung:** Materialisierte Sessions ermöglichen einfache Ausnahmen (einzelne Session absagen = ein UPDATE), direkte JOINs für Responses/Attendances, und vermeiden komplexe Recurrence-Logik im Query-Layer. Die Datenmenge ist überschaubar (ca. 2 Sessions/Woche × 30 Wochen = 60 Rows pro Serie und Saison).

### 2. Responses sind member-bezogen, nicht user-bezogen

**Entscheidung:** `training_responses` hat `member_id` als primären Schlüssel (UNIQUE pro training + member), plus `responded_by` (user_id) für den Audit-Trail.

**Alternativen:** Responses rein user-bezogen wie `duty_assignments`.

**Begründung:** Anwesenheit ist physisch (das Mitglied trainiert), nicht account-bezogen. Elternteile haben keinen eigenen `member`-Eintrag, aber ein `family_links`-Paar. Mit member-basiertem Key kann ein Elternteil für sein Kind antworten, ohne die Eindeutigkeit zu verletzen. Trainer sieht: „Leon M. — Abgesagt (eingetragen von Mutter)".

### 3. Zwei getrennte API-Endpunkte, Frontend merged im Kalender

**Entscheidung:** `/api/training-sessions` und `/api/games` bleiben getrennt. KalenderPage holt beide parallel und mergt sie im Frontend.

**Alternativen:** Neuer unified `/api/calendar`-Endpoint.

**Begründung:** Vermeidet Refactoring des bestehenden Games-Handlers. Beide Domains bleiben eigenständig. Der Kalender ist bereits eine React-Komponente, die Daten mergen kann. Bei 1 GB RAM auf dem VPS ist ein zusätzlicher kleiner paralleler Fetch unkritisch.

### 4. Series-Edit-Scope: `this_and_following` | `all`

**Entscheidung:** PUT `/api/training-series/{id}` akzeptiert einen `scope`-Parameter. `this_and_following` aktualisiert alle Sessions ab einem Datum (DELETE + Neugenerierung). `all` aktualisiert alle Sessions der Serie.

**Begründung:** Standard-Muster aus Kalender-Apps (Google Calendar). Individuelle Session-Overrides werden durch direktes PUT auf `/api/training-sessions/{id}` gehandhabt — die Session verliert damit ihren `series_id`-Bezug nicht, aber die Felder weichen von der Serie ab.

### 5. Privacy-Enforcement im Backend

**Entscheidung:** Der `reason`-Feld in der Response wird im Handler nulled, wenn der anfragende User weder Trainer/Admin ist noch der Spieler selbst oder ein Elternteil des Spielers.

**Umsetzung:** Handler prüft `claims.Role`, und falls spieler/elternteil: prüft ob `member.user_id == claims.UserID` (Spieler) oder ob ein `family_links`-Row existiert mit `parent_user_id = claims.UserID` AND `member_id = response.member_id`.

## Risks / Trade-offs

- **Bulk-Insert bei großen Serien** → Mitigation: SQLite-Transaktion, max. ~100 Sessions pro Serie (30 Wochen × 3 Tage), kein Performance-Problem
- **Series-Edit löscht und regeneriert Sessions** → Mitigation: Responses/Attendances an `session_id` gebunden; bei DELETE werden sie kaskadiert gelöscht. Trainer wird im UI gewarnt.
- **Elternteil-Autorisierung erfordert DB-Lookup bei jedem Response-Read** → Mitigation: Einfacher JOIN auf `family_links`, SQLite-Index auf `parent_user_id`

## Migration Plan

1. Migration `009_trainings.up.sql` anlegen (4 neue Tabellen + Indizes)
2. `internal/trainings/` Package anlegen, in `main.go` registrieren
3. Frontend-Seiten und Kalender-Integration deployen
4. `make deploy` führt `migrate up` automatisch aus — kein manueller Eingriff nötig
5. Rollback: `migrate down` entfernt alle 4 Tabellen; keine bestehenden Tabellen werden verändert

## Open Questions

- Sollen Trainer bei neuen RSVP-Einträgen benachrichtigt werden (z.B. „5 neue Absagen seit gestern")? → Für Ebene 3 / späteren Notification-Pass aufgehoben.
