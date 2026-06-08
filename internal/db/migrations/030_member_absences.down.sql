DROP TABLE IF EXISTS member_absences;

CREATE TABLE game_responses_new (
    game_id      INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    member_id    INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    responded_by INTEGER NOT NULL REFERENCES users(id),
    status       TEXT    NOT NULL CHECK(status IN ('confirmed','declined','maybe')),
    reason       TEXT    NOT NULL DEFAULT '',
    responded_at TEXT    NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (game_id, member_id)
);
INSERT INTO game_responses_new SELECT game_id, member_id, responded_by, status, reason, responded_at FROM game_responses;
DROP TABLE game_responses;
ALTER TABLE game_responses_new RENAME TO game_responses;

CREATE TABLE training_responses_new (
    training_id   INTEGER  NOT NULL REFERENCES training_sessions(id) ON DELETE CASCADE,
    member_id     INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    responded_by  INTEGER  NOT NULL REFERENCES users(id),
    status        TEXT     NOT NULL CHECK (status IN ('confirmed','declined','maybe')),
    reason        TEXT     NOT NULL DEFAULT '',
    responded_at  TEXT     NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (training_id, member_id)
);
INSERT INTO training_responses_new SELECT training_id, member_id, responded_by, status, reason, responded_at FROM training_responses;
DROP TABLE training_responses;
ALTER TABLE training_responses_new RENAME TO training_responses;
