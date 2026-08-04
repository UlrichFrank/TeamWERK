## Why

Auf `/profil/anwesenheit?member=191` wird ein Termin als „fehlt" (missed) angezeigt, obwohl das Mitglied für diesen Termin abgesagt hatte und das als entschuldigtes Fehlen gelten soll. Nach Rücksprache mit dem Vereinsverantwortlichen gilt fachlich: **jede Form der Absage ist ein entschuldigtes Fehlen** — egal ob automatisch durch eine erfasste Abwesenheit (Urlaub/Verletzung, `member_absences` → `absence_id` auf der RSVP-Antwort gesetzt) oder durch eine manuelle Absage des Spielers selbst (RSVP „Absagen" ohne hinterlegte Abwesenheit, `absence_id` bleibt `NULL`). „Dauerhaft abmelden" durch den Trainer (Serien-Abmeldung) bleibt bewusst eine eigene, bereits vollständig aus der Statistik ausgeschlossene Kategorie „abgemeldet" und wird nicht in „entschuldigt" umbenannt.

Zwei Fehler liegen vor:

1. **Klassifikations-Lücke**: `internal/attendance/classify.go` (`Classify()`) und die parallele SQL-Aggregation in `internal/attendance/handler.go` (`loadCounts`) werten eine `declined`-Antwort nur dann als `excused` („entschuldigt"), wenn zusätzlich `absence_id IS NOT NULL` gesetzt ist. Eine manuelle Absage ohne hinterlegte Abwesenheit fällt dadurch auf `unknown` (weder entschuldigt noch fehlt) statt auf `excused` — nicht dem gewünschten Verhalten entsprechend.
2. **Persistenz-Bug** (ursprünglich gemeldet): `TermineDetailPage.tsx` sendet bei jedem Checkbox-Klick eines Trainers die komplette Teilnehmer-Liste als Bulk-`POST`, wobei jedes nicht angeklickte, noch nie erfasste Mitglied auf `present=false` defaultet — auch entschuldigt abgesagte Mitglieder. Der Server persistiert das unbesehen, was `Classify()` gemäß der bestehenden, bewusst so spezifizierten Priorität „explizite Erfassung schlägt Auto-Decline" zu `missed` statt `excused` macht.

## What Changes

- **`internal/attendance/classify.go`**: `Classify()` verliert den Parameter `hasAbsence` — jede `declined`-Antwort ohne persistierte `attendance`-Zeile zählt als `excused`, unabhängig von `absence_id`.
- **`internal/attendance/handler.go`**: `classifyRow()` und die SQL-Queries in `loadMemberEvents` sowie die parallele SQL-native Aggregation in `loadCounts` (Team-Statistik) verlieren die Bedingung `absence_id IS NOT NULL` bei der Ermittlung von „entschuldigt" — nur noch `status='declined'` entscheidet.
- **`internal/games/handler.go` / `internal/trainings/handler.go` `SaveAttendances`**: der bestehende Persistenz-Fix wird auf **jede** `declined`-Antwort ausgeweitet (nicht nur absence-verknüpfte): ein eingehender Eintrag mit `present=false` wird für ein Mitglied mit `status='declined'` **nicht** persistiert. `present=true` wird weiterhin immer geschrieben (bestehende, getestete Override-Regel „war trotz Absage doch da" bleibt erhalten).
- **Unverändert**: `CategoryUnavailable` („dauerhaft abgemeldet", Serien-Abmeldung) bleibt eine eigene Kategorie mit eigenem Label „abgemeldet", weiterhin mit Vorrang vor `excused` und vollständig aus allen drei Statistik-Säulen ausgeschlossen — wird **nicht** zu „entschuldigt" vereinheitlicht.

## Capabilities

### New Capabilities
(keine)

### Modified Capabilities
- `attendance-statistics`: Klassifikationsregel „entschuldigt" erfordert keine `absence_id` mehr, jede `declined`-Antwort zählt.
- `game-attendance`: `POST /api/games/{id}/attendances` überspringt `present=false`-Einträge für **jedes** Mitglied mit `status='declined'` (nicht mehr nur absence-verknüpfte).
- `training-attendance`: analog für `POST /api/training-sessions/{id}/attendances`.

## Impact

- `internal/attendance/classify.go` (`Classify`, Signatur-Änderung)
- `internal/attendance/handler.go` (`classifyRow`, `loadMemberEvents`, `loadCounts` — SQL und Scan-Anpassungen an zwei parallelen Klassifikations-Implementierungen)
- `internal/games/handler.go` (`SaveAttendances`)
- `internal/trainings/handler.go` (`SaveAttendances`)
- Tests: `internal/attendance/classify_test.go`, `internal/attendance/handler_test.go`, `internal/games/*_test.go`, `internal/trainings/*_test.go`
- Kein Frontend-Change (Label „entschuldigt" für `CategoryExcused` existiert bereits, `CategoryUnavailable`/„abgemeldet" bleibt unverändert)
- Keine Migration
- Optional/außerhalb der Tasks: manuelle Bereinigung bereits fälschlich gespeicherter `present=0`-Bestandsdaten (siehe design.md)
