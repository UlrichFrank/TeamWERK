## Context

Heutiger Ist-Zustand (`internal/attendance/classify.go`):

```
Row-Zustand              → Category    → UI
──────────────────────────────────────────────
kein Row                 → unknown     → "—"
Row present=1            → present     → anwesend
Row present=0            → missed      → abwesend
```

Bug-Kette:

1. `TermineDetailPage.toggleAttendance` (`web/src/pages/TermineDetailPage.tsx:295`) schickt bei jedem Einzel-Klick die **gesamte** Kader-Liste als Snapshot. Für Mitglieder ohne Eintrag in `attendanceMap` wird `false` gesetzt → Server schreibt `present=0`-Rows für alle.
2. `SaveAttendances` (`internal/trainings/handler.go:1795`, `internal/games/handler.go:2959`) upsertet unbedingt — es gibt keinen DELETE-Pfad.

Der Nutzer akzeptiert das Bulk-Verhalten bewusst („einmal geklickt = Session ist bewertet"), fordert aber (a) ein explizites Flag „Session wurde bewertet" statt Ableitung aus der Row-Existenz und (b) einen Reset, der eine versehentlich aktivierte Erfassung wieder als „nicht bewertet" markiert.

Betroffene Aggregationen leben in `internal/attendance/handler.go` (`loadCounts`, `loadMemberEvents`, `GetTeamOpen`) sowie im Erinnerungs-Job `internal/scheduler/attendance_reminders.go`.

## Goals / Non-Goals

**Goals:**
- Ein session-/spiel-lokales Flag entscheidet, ob die gespeicherten `attendance`-Rows in der Statistik gelten. Default `0`.
- Erst-Save eines Bulk-Pakets kippt das Flag auf `1` — atomar in derselben Transaktion, damit Statistik und Flag konsistent bleiben.
- Reset (neue DELETE-Route) kippt das Flag zurück auf `0`, ohne Rows anzufassen. Erneuter Save re-aktiviert.
- Aggregation ignoriert Rows für Sessions/Spiele mit `attendance_tracked=0` — die Statistik zeigt „—" statt „missed" für die betroffenen Kinder.
- Backfill ändert für Bestandsdaten nichts am sichtbaren Verhalten (alle Sessions mit ≥1 Row → tracked=1).

**Non-Goals:**
- **Kein** Umbau des UI zu tri-state Checkboxes und **kein** Delta-Payload-Fix (die separate Root-Cause-Diskussion ist bewusst aufgeschoben).
- **Keine** DELETE der `training_attendances`/`game_attendances`-Rows beim Reset — Zombie-Rows bleiben liegen (Undo-freundlich, per Design des Nutzers).
- **Kein** eigener „Reset audit trail" (kein Wer/Wann-Log für Resets — Broadcast + normales Behavior reicht).

## Decisions

### D1: Flag lebt an `training_sessions` / `games`, nicht an `training_attendances`

Alternativen:
- (a) Flag pro Row (`training_attendances.tracked`) — kollidiert semantisch mit `present IS NULL` und erfordert Fan-Out beim Reset.
- (b) Separate Tabelle `attendance_tracking_state(session_id, kind)` — zusätzliche Joins ohne Gewinn.
- (c) **Gewählt**: Spalte am Session-/Spiel-Row.

**Warum**: Ein Reset-Vorgang ist inhärent session-weit (der Trainer resetted die ganze Session, nicht einzelne Rows). Ein Boolean an der Session-Zeile ist ein einzelnes `UPDATE`, in allen Aggregations-SQLs ein einziges `AND …attendance_tracked=1`, und existiert schon in einem Query-Kontext, den die betroffenen Handler ohnehin selektieren.

### D2: Erst-Save setzt Flag als Teil derselben Transaktion

Alternativen:
- (a) DB-Trigger auf INSERT von `training_attendances` — zusätzlicher Migrations-Zustand, harder to reason about.
- (b) Anwendungs-seitig **nach** dem Loop ein `UPDATE`, in derselben Tx wie die Upserts. **Gewählt.**

**Warum**: Trigger sind SQLite-möglich, aber der Codepfad ist ohnehin transaktional (`SaveAttendances` öffnet bereits eine Tx). Ein zusätzliches `UPDATE … SET attendance_tracked=1 WHERE id=?` vor dem `Commit` ist ein triviales Statement, ohne Migrations-Trigger-Wartungslast. Zusätzlich gilt: wenn `SaveAttendances` alle Einträge überspringt (z.B. Trainer-only-Paket oder alle unavailable), soll das Flag **nicht** kippen — deshalb Bedingung „mindestens ein Upsert wurde ausgeführt". Anwendungs-seitig einfach ein `if wroteAny { UPDATE … }`.

### D3: Reset lässt Rows liegen (kein DELETE)

**Warum**: (i) Der Nutzer wählte explizit diese Semantik. (ii) Wenn der Trainer versehentlich resettet und danach ein Kind erneut ankreuzt, sind die alten Werte über Bulk-Save wieder aktiv (Undo-Pfad). (iii) Ein DELETE wäre unumkehrbar und ohne Backup nicht wiederherstellbar.

Trade-off: `training_attendances` sammelt „tote" Rows an, die nie wieder in der Statistik erscheinen, solange das Flag `0` bleibt. Bei einer typischen Vereinsgröße (< 500 Sessions/Saison × ~20 Mitglieder = 10k Rows) irrelevant für Performance und Storage. Aufräumen wäre späterer Refactor, nicht Teil dieses Change.

### D4: Aggregation filtert per SQL-`WHERE`, nicht per Post-Filter in Go

Konkret: In `loadCounts` und `loadMemberEvents` wird der `LEFT JOIN training_attendances` um `AND ts.attendance_tracked=1` ergänzt (bzw. per zusätzlichem `LEFT JOIN`-Prädikat auf `attendance_tracked`), so dass `present` in Go weiterhin als `NULL` ankommt, wenn die Session nicht tracked ist. Damit greift die bestehende `Classify(nil, …)`-Logik unverändert und liefert `CategoryUnknown` — genau das gewünschte Ergebnis („—").

**Alternative**: In Go nach dem Scan filtern. Verworfen: schleppt unnötige Daten mit und macht die drei SQL-Blöcke inkonsistent (Team-Stats vs. Member-Stats).

### D5: `GetTeamOpen` invertiert den Filter

Heute: „offen" = `NOT EXISTS (attendance-Row)`. Neu: „offen" = `attendance_tracked=0`. Damit taucht eine reset-e Session korrekt wieder als „offen zu erfassen" auf — das ist die einzige spürbare UI-Konsequenz des Resets neben der Kategorien-Änderung.

Der Reminder-Job `internal/scheduler/attendance_reminders.go` bekommt denselben Umbau (dasselbe SQL-Muster).

### D6: Beide DELETE-Routen unter derselben Auth wie SaveAttendances

`hasTeamAccess`-Check (Trainer des Teams + sportliche Leitung + admin). Kein neuer Auth-Weg.

### D7: Broadcast auf existierendem Kanal `attendance-changed`

Frontend abonniert bereits `attendance-changed` in `TermineDetailPage.tsx` (Zeile 291-292). Ein Reset erzeugt genau denselben Refresh-Trigger wie ein Save — keine neuen Event-Typen, keine Anpassung der Broadcast-Allowlist im Arch-Test.

## Risks / Trade-offs

- **Backfill-Reichweite**: Ein Bestand-Session, deren einzige Attendance-Rows historisch `present=0` sind (z.B. wegen desselben Bugs aus der Vergangenheit), wird durch den Backfill als `tracked=1` markiert und behält die „abwesend"-Anzeige. → Bewusst akzeptiert; ein pauschales Neu-Bewerten wäre destruktiv und lag außerhalb des Nutzerwunschs. Trainer kann pro betroffener Session manuell resetten.
- **Zombie-Rows** wachsen unbeschränkt bei häufigem Reset. → Vertretbar bei Vereinsgröße; falls nötig, späteres Cleanup-Job (nicht Teil dieses Change).
- **Reminder-Job**: Wenn `attendance_reminders.go` den Filter nicht mitzieht, versendet er keine Erinnerung mehr für zurückgesetzte Sessions (statt neuer Erinnerungen). → Explizit mit umbauen und via Test absichern.
- **Rennen zwischen Save und Reset**: SQLite serialisiert Writes; letzte gewinnt. Kein reales Problem, aber ein manueller Reset direkt nach einem Save wird das Flag auf 0 setzen (erwartet).

## Migration Plan

1. Migration `0NN_attendance_tracked.up.sql`:
   ```sql
   ALTER TABLE training_sessions ADD COLUMN attendance_tracked INTEGER NOT NULL DEFAULT 0;
   ALTER TABLE games             ADD COLUMN attendance_tracked INTEGER NOT NULL DEFAULT 0;
   UPDATE training_sessions SET attendance_tracked=1
     WHERE EXISTS (SELECT 1 FROM training_attendances ta WHERE ta.training_id = training_sessions.id);
   UPDATE games SET attendance_tracked=1
     WHERE EXISTS (SELECT 1 FROM game_attendances ga WHERE ga.game_id = games.id);
   ```
   `.down.sql` droppt beide Spalten (SQLite: `ALTER TABLE … DROP COLUMN` verfügbar seit 3.35, akzeptiert für Rollback).

2. Deploy-Reihenfolge: Migration läuft via `make deploy` automatisch vor dem Service-Restart — kein manueller Schritt.

3. Rollback: `make migrate-remote-down N=1` entfernt die Spalte; Backfill-Info geht verloren, aber die Legacy-Semantik („Row existiert → tracked") ist ein Superset der neuen — kein Datenverlust.

## Open Questions

_Keine offenen Punkte — Nutzerantworten deckten Scope (Trainings + Spiele), Reset-Semantik (Rows bleiben) und Auth (wie Save) ab._
