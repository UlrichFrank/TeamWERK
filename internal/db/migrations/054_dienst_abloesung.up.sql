-- Ablösung: ein Dienst endet, wenn der nächste gleichartige beginnt.
--
-- Bewirtung ist ein Dienst am Spieltag, nicht am Spiel: die Rotation gibt jeder
-- gezogenen Mannschaft genau EINEN Slot an ihrem ersten Heimspiel, dessen Dauer aber
-- bisher nur dieses eine Spiel kennt. Bei eng aufeinander folgenden Spielen überlappt
-- deshalb jeder Bewirtungsdienst mit seinem Nachfolger — und die erfundene Zeit wird
-- über duty_slots.hours_value als geleistete Dienststunde gutgeschrieben.
--
-- Das Kennzeichen macht das aufgelöste Ende (End-Anker + End-Versatz) zum DECKEL:
--
--   Ende = MIN( Start des nächsten gleichartigen Dienstes am selben Tag , Deckel )
--
-- Es ist bewusst KEIN dritter duration_mode: die Regel ist eine Kappung des
-- dynamischen Modus, kein eigenes Ende — ein dritter Modus müsste End-Anker und
-- End-Versatz trotzdem tragen und hätte in der Maske dieselben Felder wie Modus 2.
-- Im Modus 'absolut' bleibt der Wert bedeutungslos, wird aber gespeichert, damit ein
-- Moduswechsel hin und zurück ihn nicht verliert.
--
-- Rein additiv, kein Backfill: Default 0 IST das bisherige Verhalten.
ALTER TABLE duty_types ADD COLUMN end_at_next_duty INTEGER NOT NULL DEFAULT 0
    CHECK(end_at_next_duty IN (0,1));

ALTER TABLE game_template_items ADD COLUMN end_at_next_duty INTEGER NOT NULL DEFAULT 0
    CHECK(end_at_next_duty IN (0,1));
