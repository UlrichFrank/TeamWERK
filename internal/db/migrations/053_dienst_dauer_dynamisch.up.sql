-- Dynamische Dienst-Dauer: bisher ist die Dauer eine feste Stundenzahl, die nichts
-- vom Spiel weiß. Ein Zeitnehmer ist aber so lange gebunden, wie das Spiel dauert —
-- und die schwankt je Altersklasse (age_class_game_rules) und je Vorlage
-- (game_templates.duration_minutes). Der Modus 'dynamisch' bestimmt das Ende über
-- Anker + Versatz, genau wie der Start es schon kann.
--
-- Die Stundenzahl bleibt in beiden Modi gepflegt: sie ist im dynamischen Modus der
-- Rückfall, falls die aufgelöste Endzeit vor der Startzeit läge. Ein Dienst fällt
-- damit nie aus dem Plan, weil jemand einen Versatz vertippt hat.
--
-- Kein Backfill: 'absolut' IST das bisherige Verhalten. Die End-Felder tragen
-- sinnvolle Startwerte ('end' + 0 = "bis Spielende"), damit ein Umschalten auf
-- 'dynamisch' sofort etwas Vernünftiges ergibt.
ALTER TABLE duty_types ADD COLUMN duration_mode TEXT NOT NULL DEFAULT 'absolut'
    CHECK(duration_mode IN ('absolut','dynamisch'));
ALTER TABLE duty_types ADD COLUMN end_anchor TEXT NOT NULL DEFAULT 'end'
    CHECK(end_anchor IN ('start','end'));
ALTER TABLE duty_types ADD COLUMN end_offset_minutes INTEGER NOT NULL DEFAULT 0;

ALTER TABLE game_template_items ADD COLUMN duration_mode TEXT NOT NULL DEFAULT 'absolut'
    CHECK(duration_mode IN ('absolut','dynamisch'));
ALTER TABLE game_template_items ADD COLUMN end_anchor TEXT NOT NULL DEFAULT 'end'
    CHECK(end_anchor IN ('start','end'));
ALTER TABLE game_template_items ADD COLUMN end_offset_minutes INTEGER NOT NULL DEFAULT 0;
