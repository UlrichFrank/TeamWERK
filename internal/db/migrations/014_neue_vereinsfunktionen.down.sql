-- 014 down: Neue Vereinsfunktionen rückgängig machen
PRAGMA foreign_keys = OFF;

DELETE FROM member_club_functions WHERE function IN ('kassierer','sportliche_leitung');

CREATE TABLE member_club_functions_old (
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    function  TEXT    NOT NULL CHECK(function IN ('spieler','trainer','vorstand','vorstand_beisitzer')),
    PRIMARY KEY (member_id, function)
);

INSERT INTO member_club_functions_old SELECT * FROM member_club_functions;

DROP TABLE member_club_functions;
ALTER TABLE member_club_functions_old RENAME TO member_club_functions;

PRAGMA foreign_keys = ON;
