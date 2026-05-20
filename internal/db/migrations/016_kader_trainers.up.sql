CREATE TABLE kader_trainers (
    kader_id  INTEGER NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    member_id INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    PRIMARY KEY (kader_id, member_id)
);
