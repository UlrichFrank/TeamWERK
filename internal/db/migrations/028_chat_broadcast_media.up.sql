-- Bilder in Chat-Nachrichten und Mitteilungen (chat-broadcast-bilder).
-- Neuer gemeinsamer Media-Store + optionale media_id auf messages/broadcasts.
-- SQLite kann den CHECK auf body nicht in-place lockern → Tabellen-Rebuild.
-- Der Migrationslauf (internal/db/db.go:Migrate) setzt PRAGMA foreign_keys=OFF,
-- daher löst DROP TABLE hier KEINE Cascade-Deletes auf message_reactions/
-- broadcast_reads aus (Bestandszeilen bleiben erhalten).

CREATE TABLE media (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    disk_name   TEXT    NOT NULL UNIQUE,
    mime_type   TEXT    NOT NULL,
    size        INTEGER NOT NULL,
    uploaded_by INTEGER NOT NULL REFERENCES users(id),
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- messages: neue Spalte media_id + gelockerter CHECK (Text ODER Bild).
CREATE TABLE messages_new (
    id              INTEGER PRIMARY KEY,
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       INTEGER NOT NULL REFERENCES users(id),
    body            TEXT    NOT NULL,
    sent_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reply_to_id     INTEGER REFERENCES messages(id),
    edited_at       DATETIME,
    deleted_at      DATETIME,
    is_system       BOOLEAN NOT NULL DEFAULT 0,
    media_id        INTEGER REFERENCES media(id),
    CHECK ((length(body) > 0 OR media_id IS NOT NULL) AND length(body) <= 2000)
);

INSERT INTO messages_new (id, conversation_id, sender_id, body, sent_at, reply_to_id, edited_at, deleted_at, is_system)
SELECT id, conversation_id, sender_id, body, sent_at, reply_to_id, edited_at, deleted_at, is_system FROM messages;

DROP TABLE messages;
ALTER TABLE messages_new RENAME TO messages;
CREATE INDEX idx_messages_conv ON messages(conversation_id, sent_at DESC);

-- broadcasts: neue Spalte media_id + gelockerter CHECK (Text ODER Bild).
CREATE TABLE broadcasts_new (
    id          INTEGER PRIMARY KEY,
    sender_id   INTEGER NOT NULL REFERENCES users(id),
    target_type TEXT    NOT NULL CHECK(target_type IN ('all', 'team', 'role')),
    target_id   INTEGER,
    target_role TEXT,
    body        TEXT    NOT NULL,
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at   DATETIME,
    media_id    INTEGER REFERENCES media(id),
    CHECK ((length(body) > 0 OR media_id IS NOT NULL) AND length(body) <= 2000)
);

INSERT INTO broadcasts_new (id, sender_id, target_type, target_id, target_role, body, sent_at, edited_at)
SELECT id, sender_id, target_type, target_id, target_role, body, sent_at, edited_at FROM broadcasts;

DROP TABLE broadcasts;
ALTER TABLE broadcasts_new RENAME TO broadcasts;
