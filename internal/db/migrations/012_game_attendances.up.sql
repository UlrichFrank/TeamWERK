-- game-attendance: post-hoc Erfassung der Spiel-Anwesenheit durch den Trainer,
-- analog zu training_attendances (siehe openspec/changes/anwesenheits-statistik).

CREATE TABLE game_attendances (
    id        INTEGER  PRIMARY KEY AUTOINCREMENT,
    game_id   INTEGER  NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    member_id INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    present   INTEGER  NOT NULL CHECK (present IN (0, 1)),
    noted_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (game_id, member_id)
);

CREATE INDEX idx_game_attendances_game ON game_attendances(game_id);
