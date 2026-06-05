CREATE TABLE game_lineup (
    game_id   INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    added_by  INTEGER REFERENCES users(id),
    added_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (game_id, member_id)
);
