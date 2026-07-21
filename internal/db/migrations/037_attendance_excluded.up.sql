-- attendance_excluded: Trainer können vergangene Termine explizit aus den
-- Anwesenheits-Statistiken ausschließen, ohne sie zu erfassen.
-- attendance_tracked bleibt 0; excluded=1 blendet den Termin aus den
-- "offenen Erfassungen" aus und lässt ihn in keiner Säule zählen.
ALTER TABLE training_sessions ADD COLUMN attendance_excluded INTEGER NOT NULL DEFAULT 0;
ALTER TABLE games             ADD COLUMN attendance_excluded INTEGER NOT NULL DEFAULT 0;
