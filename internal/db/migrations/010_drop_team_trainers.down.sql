-- 010 down: Restore team_trainers table (empty — data not recoverable).
CREATE TABLE IF NOT EXISTS team_trainers (
    team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, user_id)
);
