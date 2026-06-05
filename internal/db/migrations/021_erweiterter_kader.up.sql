CREATE TABLE kader_extended_members (
    kader_id  INTEGER NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    added_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (kader_id, member_id)
);
