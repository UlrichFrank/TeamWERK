## Context

Fachlich soll jede Form der Absage als entschuldigtes Fehlen zählen:

1. **Automatische Absage** durch eine erfasste Abwesenheit (`member_absences`, Urlaub/Verletzung): beim Anlegen setzt `internal/absences/handler.go` (`Create`, ~Zeile 381-420) für alle betroffenen Termine `training_responses`/`game_responses` auf `status='declined', absence_id=<id>`. Ebenso befüllen `CreateGame`/`CreateSession` (`internal/games/handler.go:1145-1156`, `internal/trainings/handler.go:766-780`) beim Anlegen eines neuen Termins automatisch entsprechende `declined`-Antworten, falls bereits eine überlappende Abwesenheit existiert.
2. **Manuelle Absage** durch den Spieler selbst über die reguläre RSVP-Route (`internal/games/handler.go:2452-2460` `RespondToGame`, `internal/trainings/handler.go:1485-1493` `Respond`) — setzt `status='declined'`, aber **nie** `absence_id` (die Spalte wird in diesem Insert/Upsert gar nicht befüllt; eine Antwort mit gesetzter `absence_id` ist zusätzlich vor Überschreiben gesperrt, siehe `internal/games/handler.go:2424-2431`).
3. **„Dauerhaft abmelden"** (Serien-Abmeldung, `training_series/{id}/unavailabilities`) ist strukturell etwas anderes: keine RSVP-Antwort, sondern ein eigener Tabellen-Eintrag (`member_series_unavailabilities`), der eine Session für ein Mitglied vollständig aus der Bezugsmenge nimmt. Bleibt laut Klärung mit dem Auftraggeber eine eigene Kategorie `CategoryUnavailable` mit Label „abgemeldet" — nicht Teil dieses Changes.

Heute verlangt sowohl `internal/attendance/classify.go` (`Classify`) als auch die dazu parallele, aus Performance-Gründen SQL-native Aggregation in `internal/attendance/handler.go` (`loadCounts`, für die Team-Statistik) zusätzlich zu `status='declined'` noch `absence_id IS NOT NULL`. Das ist eine **doppelte Implementierung derselben Klassifikationsregel** (einmal als Go-Funktion für die Mitglieds-Detailansicht via `loadMemberEvents`/`classifyRow`, einmal als SQL-`CASE`-Aggregation für die Team-Aggregat-Statistik) — beide Stellen müssen konsistent geändert werden, sonst driften Team- und Mitgliedsansicht auseinander.

Der ursprünglich gemeldete Persistenz-Bug (Bulk-Save überschreibt unberührte Mitglieder mit `present=false`) betrifft nach der Klassifikations-Änderung nicht mehr nur absence-verknüpfte, sondern jede `declined`-Antwort — der bereits geplante Skip in `SaveAttendances` wird entsprechend verallgemeinert.

## Goals / Non-Goals

**Goals:**
- Jede `declined`-Antwort (mit oder ohne `absence_id`) wird als `excused` klassifiziert, sofern keine explizite `attendance`-Zeile (`present`) vorliegt — konsistent in **beiden** Implementierungen (`classify.go`/`loadMemberEvents` und `loadCounts`).
- `present=true` überschreibt weiterhin jede Absage (bestehende, getestete D1-Regel „explizite Erfassung schlägt Auto-Decline" bleibt für den bewussten Einzel-Fall erhalten).
- `SaveAttendances` (Spiele + Trainings) überspringt `present=false`-Einträge für **jedes** `declined`-Mitglied, nicht nur absence-verknüpfte — schließt die Persistenz-Lücke für den erweiterten Kreis vollständig.
- `CategoryUnavailable` („dauerhaft abgemeldet") bleibt unverändert eine eigene Kategorie mit eigenem Label und weiterhin Vorrang vor `excused` (Nenner-Ausschluss-Regel aus der Capability `attendance-statistics` bleibt bestehen).

**Non-Goals:**
- Keine Umbenennung/Zusammenlegung von `CategoryUnavailable` mit `CategoryExcused` — explizit vom Auftraggeber ausgeschlossen.
- Kein Frontend-Change — das Label „entschuldigt" für `CategoryExcused` existiert bereits (`AttendanceStatsView.tsx`), es ändert sich nur, welche Fälle diese Kategorie erreichen.
- Keine Änderung an `absence_id` selbst, an `member_absences` oder an der RSVP-Sperr-Logik für absence-verknüpfte Antworten (`internal/games/handler.go:2424-2431` u.ä.) — diese Sperre ist ein unabhängiges Feature (verhindert, dass ein Mitglied seine eigene, durch eine Abwesenheit gesetzte Absage einfach zurücknimmt) und bleibt bestehen.
- Keine automatisierte rückwirkende Korrektur bereits fälschlich gespeicherter `present=0`-Zeilen (Bestandsdaten) — siehe Risiko unten.

## Decisions

**D1 — `absence_id` fällt aus der Klassifikationsregel, nicht aus dem Schema/den Queries insgesamt:** `Classify()` verliert den `hasAbsence`-Parameter vollständig (Signatur `Classify(present *bool, declined bool) Category`). In `loadMemberEvents` wird die SQL-Spalte `tr.absence_id IS NOT NULL`/`gr.absence_id IS NOT NULL` aus dem SELECT entfernt (wird für die Klassifikation nicht mehr gebraucht), `classifyRow` verliert ebenfalls den Parameter. In `loadCounts` wird `AND tr.absence_id IS NOT NULL`/`AND gr.absence_id IS NOT NULL` aus den beiden `CASE`-Ausdrücken für „excused" entfernt — übrig bleibt `present IS NULL AND status='declined'`. `absence_id` bleibt als Spalte und in der RSVP-Sperr-Logik an anderer Stelle unverändert bestehen, wird nur aus der Statistik-Klassifikation entfernt.

**D2 — `SaveAttendances`-Skip wird auf `status='declined'` verallgemeinert (kein `absence_id`-Check mehr nötig):** Die bereits geplante Helper-Map (analog `unavailableMembersForSession`) wird einfacher: `SELECT member_id FROM {training,game}_responses WHERE {training_id,game_id}=? AND status='declined'` — ohne `absence_id`-Bedingung. Der wertabhängige Skip (nur `present=false`, `present=true` immer schreiben) bleibt wie ursprünglich entworfen.

**D3 — Zwei parallele Klassifikations-Implementierungen bewusst beide anfassen, keine Vereinheitlichung im Rahmen dieses Changes:** `loadMemberEvents`/`classifyRow` (Go-Funktion, Mitglieds-Detailansicht) und `loadCounts` (SQL-natives Aggregat, Team-Statistik) dupliziert dieselbe Regel aus Performance-Gründen (Team-Aggregation über SQL `SUM/CASE` vermeidet, alle Rows nach Go zu laden). Eine Konsolidierung beider Implementierungen in eine gemeinsame Quelle wäre eine größere, hier nicht notwendige Refaktorierung — es genügt, beide synchron zu ändern und mit Tests abzusichern, dass Team- und Mitgliedsansicht dieselben Zahlen liefern.

## Risks / Trade-offs

- **[Risiko] Bereits fälschlich gespeicherte `present=0`-Zeilen (u.a. Mitglied 191) und bereits als „unbekannt" gezählte manuelle Absagen ohne `absence_id` bleiben nach diesem Fix in ihrem aktuellen (falschen) Zustand bestehen**, soweit sie bereits eine `attendance`-Zeile haben. → Mitigation: nach Deploy einmalig identifizieren (`SELECT ... FROM game_attendances ga JOIN game_responses gr ON ... WHERE ga.present=0 AND gr.status='declined'` — jetzt ohne `absence_id`-Bedingung, also größere Treffermenge als ursprünglich geplant — analog für Trainings) und betroffene Zeilen manuell löschen, damit die Statistik sie wieder korrekt als „entschuldigt" klassifiziert.
- **[Trade-off] Ein Trainer kann eine Absage nicht mehr über die Anwesenheits-Checkbox als „unentschuldigt gefehlt" markieren, egal ob die Absage eine hinterlegte Abwesenheit hat oder rein manuell war.** → Bewusst gewollt laut Klärung: jede Absage ist entschuldigt. Der einzige Weg, ein tatsächlich unentschuldigtes Fehlen zu erfassen, bleibt: das Mitglied hat gar nicht abgesagt (kein `declined`), und der Trainer erfasst `present=false` explizit.
- **[Risiko] Divergenz zwischen `loadMemberEvents` und `loadCounts`, falls nur eine der beiden Stellen geändert wird.** → Mitigation: Tasks verlangen explizit einen Vergleichstest (Team-Aggregat-Zähler eines Mitglieds stimmen mit der Summe seiner Termin-Kategorien aus der Detailansicht überein) für mindestens einen Fall mit manueller Absage ohne `absence_id`.

## Migration Plan

1. Backend-Fix deployen (additive Persist- und Klassifikationslogik, kein Schema-Change, kein Breaking Change an Response-Formaten).
2. Optional, außerhalb der Tasks dieses Changes: manuelle Bestandskorrektur der bereits falsch klassifizierten Zeilen (siehe Risiko oben) nach Rücksprache.

## Open Questions

Keine offenen Fragen — Scope wurde mit dem Auftraggeber geklärt (manuelle Absage zählt als entschuldigt, „dauerhaft abmelden" bleibt separat). Die Bestandskorrektur ist bewusst ein manueller Schritt außerhalb der Tasks dieses Changes.
