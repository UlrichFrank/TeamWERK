## 1. Klassifikation: `internal/attendance/classify.go`

- [x] 1.1 `Classify()`-Signatur ändern: Parameter `hasAbsence` entfernen, `declined` allein entscheidet über `CategoryExcused`
- [x] 1.2 `classify_test.go`: bestehende Tests an neue Signatur anpassen; neuen Testfall ergänzen, der einen `declined`-Fall **ohne** `hasAbsence`/`absence_id`-Äquivalent als `excused` erwartet
- [x] 1.3 `TestClassify_AttendanceOverridesAutoDecline` (Regressionstest) bleibt grün: `present=true` schlägt weiterhin jede Absage

## 2. Klassifikation: `internal/attendance/handler.go`

- [x] 2.1 `classifyRow()` an neue `Classify`-Signatur anpassen (Parameter `hasAbsence` entfernen)
- [x] 2.2 `loadMemberEvents`: SQL-SELECTs für Trainings (~Zeile 456-478) und Spiele (~Zeile 522-541) um die Spalte `tr.absence_id IS NOT NULL`/`gr.absence_id IS NOT NULL` bereinigen (nicht mehr für Klassifikation nötig), Scan- und Aufrufstellen anpassen
- [x] 2.3 `loadCounts`: in beiden `CASE`-Ausdrücken für „excused" (Trainings ~Zeile 139-142, Spiele ~Zeile 191-193) die Bedingung `AND tr.absence_id IS NOT NULL`/`AND gr.absence_id IS NOT NULL` entfernen
- [x] 2.4 Test: `GET /api/members/{id}/attendance-stats` liefert für eine manuelle Absage ohne `absence_id` `category: "excused"` in der Termin-Liste
- [x] 2.5 Test: `GET /api/teams/{id}/attendance-stats` zählt dieselbe manuelle Absage in `training_excused`/`game_excused`
- [x] 2.6 Vergleichstest: für ein Mitglied mit gemischten Absage-Formen (mit und ohne `absence_id`) stimmt die Summe der Team-Aggregat-Zähler mit der Anzahl der `excused`-Termine in der Mitglieds-Detailliste überein (Divergenz-Schutz zwischen den zwei parallelen Implementierungen)

## 3. Spiele: `internal/games/handler.go`

- [x] 3.1 Helper-Funktion (analog `unavailableMembersForSession` in `internal/trainings/handler.go`) ergänzen, die pro `gameID` eine Map der Mitglieder mit `game_responses.status='declined'` liefert (ohne `absence_id`-Bedingung)
- [x] 3.2 In `SaveAttendances` die Map einmal pro Request laden und in der Persist-Schleife: eingehenden Eintrag mit `present=false` für ein Mitglied in dieser Map überspringen (kein Insert/Update), `present=true` weiterhin immer schreiben
- [x] 3.3 Test: `present=false` für Mitglied mit genehmigter Abwesenheit (`absence_id` gesetzt) erzeugt keine `game_attendances`-Zeile, übrige Einträge im selben Paket werden gespeichert, HTTP 200
- [x] 3.4 Test: `present=false` für Mitglied mit manueller Absage ohne Abwesenheit (`absence_id IS NULL`) erzeugt ebenfalls keine `game_attendances`-Zeile
- [x] 3.5 Test: `present=true` für abgesagtes Mitglied (mit oder ohne `absence_id`) wird weiterhin normal persistiert (Regressionstest für die bestehende D1-Override-Regel)
- [x] 3.6 Test: wiederholter Bulk-Save mit `present=false` für dasselbe abgesagte Mitglied bleibt idempotent ohne `game_attendances`-Zeile

## 4. Trainings: `internal/trainings/handler.go`

- [x] 4.1 Analoge Helper-Funktion für Trainings ergänzen (`training_responses.status='declined'`, ohne `absence_id`-Bedingung), Stil an `unavailableMembersForSession` angelehnt
- [x] 4.2 In `SaveAttendances` dieselbe Skip-Logik wie bei Spielen ergänzen (`present=false` überspringen, `present=true` immer schreiben), Reihenfolge mit bestehendem `unavailable`- und `isTrainerOnly`-Skip beachten
- [x] 4.3 Test: `present=false` für Mitglied mit genehmigter Abwesenheit erzeugt keine `training_attendances`-Zeile, übrige Einträge im selben Paket werden gespeichert, HTTP 204
- [x] 4.4 Test: `present=false` für Mitglied mit manueller Absage ohne Abwesenheit erzeugt ebenfalls keine `training_attendances`-Zeile
- [x] 4.5 Test: `present=true` für abgesagtes Mitglied (mit oder ohne `absence_id`) wird weiterhin normal persistiert
- [x] 4.6 Test: Zusammenspiel mit bestehendem `unavailable`-Skip bleibt korrekt (beide Skips dürfen sich nicht gegenseitig blockieren)

## 5. Verifikation

- [x] 5.1 `make test` (inkl. Architektur- und Broadcast-Gate) grün
- [x] 5.2 `make lint` grün
- [x] 5.3 Szenario aus dem Bug-Report nachgestellt — als automatisierter End-to-End-Regressionstest `TestBugReport_BulkSaveDoesNotDowngradeDeclinedToMissed` (beide Absage-Formen) statt als einmalige manuelle Prüfung
- [x] 5.4 „dauerhaft abgemeldet" zeigt weiterhin das eigene Label „abgemeldet" — Label-Mapping in `AttendanceStatsView.tsx` unverändert (`unavailable`→„abgemeldet", `excused`→„entschuldigt"), Kategorie-Vorrang durch `TestSaveAttendances_UnavailableAndDeclinedSkipsCoexist` abgesichert
- [x] 5.5 `openspec validate fix-attendance-excused-override --strict`

## 6. Bestandsdaten (manuell, außerhalb des automatisierten Deploys)

- [x] 6.1 Betroffene Bestandszeilen identifiziert: `present=0`-Zeilen für Mitglieder mit `status='declined'` (mit oder ohne `absence_id`), für Spiele und Trainings. Lokaler Stand: **184 Zeilen / 35 Mitglieder** (168 Trainings — davon 167 manuelle Absagen — und 16 Spiele). Query siehe design.md, Risiko-Abschnitt.
- [x] 6.2 Lokale Dev-DB (`./teamwerk.db`) bereinigt: 184 Zeilen gelöscht, Backup unter `teamwerk.db.bak`. Verifiziert: 0 widersprüchliche Zeilen übrig, `present=1` (518) und legitime `present=0` ohne Absage (319) sowie alle 261 Absagen unverändert.
- [x] 6.3 **Prod-Bereinigung durchgeführt** (04.08.2026, nach Deploy von `c94fbfe`). Backup: `/var/lib/teamwerk/teamwerk.db.prefix-attendance.bak`. Gelöscht: **224 Zeilen bei 50 Mitgliedern** (168 Trainings — davon 167 manuelle Absagen — und 56 Spiele). Verifiziert: 0 widersprüchliche Zeilen übrig; `present=1` unverändert (518 Trainings / 125 Spiele), `present=0` exakt um die gelöschten Zeilen reduziert (487→319 / 196→140), alle Absagen unverändert (268 Trainings / 416 Spiele).
