## 1. Backend — Trainings

- [x] 1.1 `internal/trainings/handler.go`: `attendanceItem`-Struct um `IsTrainer bool` (JSON: `is_trainer`) erweitern
- [x] 1.2 `GetAttendances`-Query: dritten UNION-Zweig ergänzen (SELECT aus `kader_trainers` JOIN `members` + `training_responses`), `is_trainer=1`; im äußeren `NOT EXISTS`-Filter der Spieler-/Erweiterten-Zweige Trainer ausschließen
- [x] 1.3 Result-Loop: für Trainer-Zeilen ohne `training_responses`-Row virtuellen `rsvp_status='confirmed'` setzen (Analog zum `rsvpOptOut`-Zweig, aber unabhängig vom Session-Setting)
- [x] 1.4 Result-Loop: für Trainer-Zeilen `Present = nil` erzwingen (auch für Trainer/Admin-Aufrufer)
- [x] 1.5 Header-Zähler-Query in `GetSession` bzw. der aggregierten Session-Liste: `WHERE member_id NOT IN (SELECT member_id FROM kader_trainers WHERE kader_id = ?)` ergänzen für `confirmed_count`/`declined_count`/`pending_count`
- [x] 1.6 `Respond`-Route: bereits kompatibel — `default`-Branch akzeptiert Trainer-`member_id` für Selbst- und Fremd-Antwort (kein Code-Change)
- [x] 1.7 `SaveAttendances`-Route: wenn Ziel-`member_id` in `kader_trainers` steht (und nicht als Spieler des Kaders) → HTTP 400 (kein Attendance für Trainer)

## 2. Backend — Games

- [x] 2.1 `internal/games/handler.go`: `attendanceItem`-Struct + `participantItem` um `IsTrainer bool` erweitern
- [x] 2.2 `GetAttendances`- und `GetParticipants`-Query: UNION-Zweig für Trainer über `kader_trainers` JOIN `kader` JOIN `game_teams`; NOT-EXISTS-Filter symmetrisch zu Trainings
- [x] 2.3 Result-Loop: virtueller `confirmed`-Default und `present=nil` analog zu Trainings; Dedup-Priorität Trainer > Stammkader > Erweiterter Kader
- [x] 2.4 Header-Zähler-Query in `ListGames`/`GetGame`/`ListMyGames`: Trainer aus `confirmed_count`/`declined_count`/`maybe_count` ausschließen
- [x] 2.5 `RespondToGame`-Route: bereits kompatibel (kein Code-Change)
- [x] 2.6 `SaveAttendances`-Route: Trainer-Ziel → HTTP 400 (analog Trainings)

## 3. Backend-Tests

- [x] 3.1 `internal/trainings/trainer_rsvp_test.go`: Happy-Path — Trainer erscheint mit `is_trainer=true` und `rsvp_status='confirmed'` ohne existierende Row
- [x] 3.2 Trainings: Explizite Absage überschreibt Default (`TestGetAttendances_Trainer_ExplicitDeclineOverrides`)
- [x] 3.3 Trainings: Header-Zähler ignoriert Trainer (`TestGetSession_ConfirmedCount_ExcludesTrainer`)
- [x] 3.4 Trainings: `SaveAttendances` mit Trainer-`member_id` → 400 (`TestSaveAttendances_TrainerRejected`); `training_attendances` bleibt leer
- [x] 3.5 `internal/games/trainer_rsvp_test.go`: Trainer default confirmed, no attendance
- [x] 3.6 Games: Explizite Absage überschreibt Default
- [x] 3.7 Games: Header-Zähler ignoriert Trainer
- [x] 3.8 Games: `SaveAttendances` mit Trainer-`member_id` → 400
- [x] 3.9 Games: Bestandstest `TestGetGameAttendances_HappyPath` auf 3 Items (Trainer + Regular + Extended) angepasst

## 4. Frontend — TermineDetailPage

- [x] 4.1 `web/src/pages/TermineDetailPage.tsx`: `AttendanceItem`-Typ und `ParticipantItem`-Typ um `is_trainer` erweitern
- [x] 4.2 `TableRow`-Typ und `toRow` erweitern; `is_trainer` durchreichen
- [x] 4.3 Default-Sektions-Erstellung (`effectiveSections`) neu ordnen: Trainer / Spieler / Erweiterter Kader mit Titeln; leere Sektionen weglassen
- [x] 4.4 `ParticipantRow`: Aufstellung- und Anwesend-`<td>` bleiben für Trainer-Zeilen leer (kein Rendering von Checkbox/Strich)
- [x] 4.5 `groupByTeam`-Zweig: Trainer innerhalb jeder Team-Sektion ans Ende oben sortieren (Trainer-first innerhalb der Team-Gruppe)
- [x] 4.6 Zusagen-Zähler im Header (Games-Zweig) ignorieren Trainer

## 5. Verifikation

- [x] 5.1 `go build ./...` grün
- [x] 5.2 `go test ./...` grün (1156 Tests)
- [x] 5.3 `pnpm -C web build` grün
- [x] 5.4 `pnpm -C web test` grün (477 Tests)
- [x] 5.5 `pnpm -C web lint` grün (0 errors, 3 pre-existing warnings)
- [x] 5.6 `openspec validate termine-trainer-rsvp` grün
- [ ] 5.7 Manuelle UI-Verifikation nach Deploy auf Produktion
