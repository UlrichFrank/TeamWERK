-- spielbericht-medien-gate: Review-Gate für Match-Reports.
--
-- 1) match_reports.state CHECK erweitern um 'pending_review'
-- 2) match_reports.submitted_at (für 5-Tage-Reminder) + reviewer_user_id (Audit)
-- 3) member_club_functions.function CHECK erweitern um 'medien'
--
-- SQLite kennt kein ALTER CHECK → Tabellen-Rebuild (siehe 019_match_reports.up.sql).
-- FK-Enforcement ist während Migrate() auf Connection-Ebene aus.

-- match_reports: state-CHECK erweitern + zwei neue Spalten.
CREATE TABLE match_reports_new (
    id                INTEGER  PRIMARY KEY AUTOINCREMENT,
    game_id           INTEGER  NOT NULL UNIQUE REFERENCES games(id) ON DELETE CASCADE,
    duty_slot_id      INTEGER  REFERENCES duty_slots(id) ON DELETE SET NULL,
    author_user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reviewer_user_id  INTEGER  REFERENCES users(id) ON DELETE SET NULL,
    state             TEXT     NOT NULL DEFAULT 'draft'
                      CHECK (state IN ('draft','pending_review','publishing','published','publish_failed')),
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
    submitted_at      DATETIME,
    published_at      DATETIME
);

INSERT INTO match_reports_new (id, game_id, duty_slot_id, author_user_id,
    reviewer_user_id, state, home_goals, away_goals, home_goals_ht, away_goals_ht,
    tournament, abstract, body_md, published_url, typo3_page_uid, error_message,
    created_at, updated_at, submitted_at, published_at)
SELECT id, game_id, duty_slot_id, author_user_id,
    NULL, state, home_goals, away_goals, home_goals_ht, away_goals_ht,
    tournament, abstract, body_md, published_url, typo3_page_uid, error_message,
    created_at, updated_at, NULL, published_at
FROM match_reports;

DROP TABLE match_reports;
ALTER TABLE match_reports_new RENAME TO match_reports;

CREATE INDEX idx_match_reports_author ON match_reports(author_user_id);
CREATE INDEX idx_match_reports_state  ON match_reports(state);
CREATE INDEX idx_match_reports_slot   ON match_reports(duty_slot_id);
CREATE INDEX idx_match_reports_submitted ON match_reports(submitted_at)
    WHERE state = 'pending_review';

-- member_club_functions: function-CHECK erweitern um 'medien'.
CREATE TABLE member_club_functions_new (
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    function  TEXT    NOT NULL CHECK(function IN ('spieler','trainer','vorstand','vorstand_beisitzer','kassierer','sportliche_leitung','medien')),
    PRIMARY KEY (member_id, function)
);

INSERT INTO member_club_functions_new (member_id, function)
SELECT member_id, function FROM member_club_functions;

DROP TABLE member_club_functions;
ALTER TABLE member_club_functions_new RENAME TO member_club_functions;
