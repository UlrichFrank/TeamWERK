-- Rollback des Medien-Gate: alten Zustand von match_reports und
-- member_club_functions wiederherstellen. Datenverlust ist beabsichtigt
-- und dokumentiert:
--   - Zeilen mit state='pending_review' werden verworfen (nicht in Ziel-CHECK).
--   - Einträge mit function='medien' werden verworfen (nicht in Ziel-CHECK).
--   - Spalten submitted_at, reviewer_user_id fallen weg.

-- match_reports: zurück auf 019er-Schema.
CREATE TABLE match_reports_old (
    id                INTEGER  PRIMARY KEY AUTOINCREMENT,
    game_id           INTEGER  NOT NULL UNIQUE REFERENCES games(id) ON DELETE CASCADE,
    duty_slot_id      INTEGER  REFERENCES duty_slots(id) ON DELETE SET NULL,
    author_user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    state             TEXT     NOT NULL DEFAULT 'draft'
                      CHECK (state IN ('draft','publishing','published','publish_failed')),
    home_goals        INTEGER,
    away_goals        INTEGER,
    home_goals_ht     INTEGER,
    away_goals_ht     INTEGER,
    tournament        INTEGER  NOT NULL DEFAULT 0,
    abstract          TEXT     NOT NULL DEFAULT '' CHECK (length(abstract) <= 500),
    body_md           TEXT     NOT NULL DEFAULT '',
    published_url     TEXT,
    typo3_page_uid    INTEGER,
    error_message     TEXT,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at      DATETIME
);

INSERT INTO match_reports_old (id, game_id, duty_slot_id, author_user_id,
    state, home_goals, away_goals, home_goals_ht, away_goals_ht,
    tournament, abstract, body_md, published_url, typo3_page_uid, error_message,
    created_at, updated_at, published_at)
SELECT id, game_id, duty_slot_id, author_user_id,
    state, home_goals, away_goals, home_goals_ht, away_goals_ht,
    tournament, abstract, body_md, published_url, typo3_page_uid, error_message,
    created_at, updated_at, published_at
FROM match_reports
WHERE state IN ('draft','publishing','published','publish_failed');

DROP TABLE match_reports;
ALTER TABLE match_reports_old RENAME TO match_reports;

CREATE INDEX idx_match_reports_author ON match_reports(author_user_id);
CREATE INDEX idx_match_reports_state  ON match_reports(state);
CREATE INDEX idx_match_reports_slot   ON match_reports(duty_slot_id);

-- member_club_functions: zurück auf 001er-Set.
CREATE TABLE member_club_functions_old (
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    function  TEXT    NOT NULL CHECK(function IN ('spieler','trainer','vorstand','vorstand_beisitzer','kassierer','sportliche_leitung')),
    PRIMARY KEY (member_id, function)
);

INSERT INTO member_club_functions_old (member_id, function)
SELECT member_id, function FROM member_club_functions
WHERE function IN ('spieler','trainer','vorstand','vorstand_beisitzer','kassierer','sportliche_leitung');

DROP TABLE member_club_functions;
ALTER TABLE member_club_functions_old RENAME TO member_club_functions;
