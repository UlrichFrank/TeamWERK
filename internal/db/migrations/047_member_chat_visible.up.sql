-- Opt-In pro Member: erlaubt gezielte Auswahl des Members im "Neue Nachricht"-
-- Dialog für Nutzer ohne gemeinsames Team. Default 0 (privat), analog zu
-- cross_team_visible (Migration 003), aber ein eigenes Feld, da beide
-- Sichtbarkeiten unabhängig gesteuert werden.
ALTER TABLE members ADD COLUMN chat_visible INTEGER NOT NULL DEFAULT 0;

-- Bestandsmitglieder mit einer Funktion, die typischerweise teamübergreifend
-- kontaktiert wird, starten mit an (einmaliger Default, kein laufender Trigger
-- auf spätere Funktionsänderungen).
UPDATE members SET chat_visible = 1
WHERE id IN (
    SELECT member_id FROM member_club_functions
    WHERE function IN ('trainer', 'sportliche_leitung', 'vorstand', 'vorstand_beisitzer', 'kassierer')
);
