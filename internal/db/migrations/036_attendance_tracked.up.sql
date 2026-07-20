-- attendance_tracked-Flag: explizites Signal, ob die Anwesenheit einer
-- Trainings-Session bzw. eines Spiels bewertet wurde. Statistiken und
-- „offen zu erfassen"-Listen ignorieren Rows für Sessions/Spiele mit
-- attendance_tracked=0.
--
-- Backfill: Bestands-Termine mit mindestens einer attendance-Row bekommen
-- attendance_tracked=1 — damit ändert sich das UI-Verhalten historischer
-- Daten nicht.

ALTER TABLE training_sessions ADD COLUMN attendance_tracked INTEGER NOT NULL DEFAULT 0;
ALTER TABLE games             ADD COLUMN attendance_tracked INTEGER NOT NULL DEFAULT 0;

UPDATE training_sessions
   SET attendance_tracked = 1
 WHERE EXISTS (SELECT 1 FROM training_attendances ta WHERE ta.training_id = training_sessions.id);

UPDATE games
   SET attendance_tracked = 1
 WHERE EXISTS (SELECT 1 FROM game_attendances ga WHERE ga.game_id = games.id);
