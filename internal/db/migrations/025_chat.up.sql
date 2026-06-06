CREATE TABLE conversations (
    id         INTEGER PRIMARY KEY,
    type       TEXT    NOT NULL CHECK(type IN ('direct', 'group')),
    name       TEXT,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE conversation_members (
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    joined_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    left_at         DATETIME,
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_conv_members_user ON conversation_members(user_id);

CREATE TABLE messages (
    id              INTEGER PRIMARY KEY,
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       INTEGER NOT NULL REFERENCES users(id),
    body            TEXT    NOT NULL CHECK(length(body) > 0 AND length(body) <= 2000),
    sent_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_conv ON messages(conversation_id, sent_at DESC);

CREATE TABLE message_reads (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    read_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (message_id, user_id)
);

CREATE INDEX idx_message_reads_user ON message_reads(user_id);

CREATE TABLE broadcasts (
    id          INTEGER PRIMARY KEY,
    sender_id   INTEGER NOT NULL REFERENCES users(id),
    target_type TEXT    NOT NULL CHECK(target_type IN ('all', 'team', 'role')),
    target_id   INTEGER,
    target_role TEXT,
    body        TEXT    NOT NULL CHECK(length(body) > 0 AND length(body) <= 2000),
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE broadcast_reads (
    broadcast_id INTEGER NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    user_id      INTEGER NOT NULL REFERENCES users(id),
    read_at      DATETIME,
    PRIMARY KEY (broadcast_id, user_id)
);

CREATE INDEX idx_broadcast_reads_user ON broadcast_reads(user_id);
