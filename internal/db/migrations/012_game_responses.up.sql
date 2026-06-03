CREATE TABLE IF NOT EXISTS game_responses (
    game_id      INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    member_id    INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    responded_by INTEGER NOT NULL REFERENCES users(id),
    status       TEXT    NOT NULL CHECK(status IN ('confirmed','declined','maybe')),
    reason       TEXT    NOT NULL DEFAULT '',
    responded_at TEXT    NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (game_id, member_id)
);
