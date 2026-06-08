CREATE TABLE member_absences (
    id         INTEGER PRIMARY KEY,
    member_id  INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    type       TEXT    NOT NULL CHECK(type IN ('vacation','injury')),
    start_date DATE    NOT NULL,
    end_date   DATE    NOT NULL,
    note       TEXT    NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_member_absences_member ON member_absences(member_id);
CREATE INDEX idx_member_absences_dates  ON member_absences(start_date, end_date);

ALTER TABLE members ADD COLUMN absences_public INTEGER NOT NULL DEFAULT 0;

ALTER TABLE training_responses ADD COLUMN absence_id INTEGER REFERENCES member_absences(id) ON DELETE CASCADE;

ALTER TABLE game_responses ADD COLUMN absence_id INTEGER REFERENCES member_absences(id) ON DELETE CASCADE;
