-- 014: Neue Vereinsfunktionen
-- Erweitert member_club_functions um 'kassierer' und 'sportliche_leitung'.
PRAGMA foreign_keys = OFF;

CREATE TABLE member_club_functions_new (
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    function  TEXT    NOT NULL CHECK(function IN ('spieler','trainer','vorstand','vorstand_beisitzer','kassierer','sportliche_leitung')),
    PRIMARY KEY (member_id, function)
);

INSERT INTO member_club_functions_new SELECT * FROM member_club_functions;

DROP TABLE member_club_functions;
ALTER TABLE member_club_functions_new RENAME TO member_club_functions;

PRAGMA foreign_keys = ON;
