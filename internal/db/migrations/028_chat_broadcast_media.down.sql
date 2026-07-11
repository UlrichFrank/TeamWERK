-- Rückbau von chat-broadcast-bilder: media_id entfällt, strikter body-CHECK zurück,
-- media-Tabelle entfernen. Zeilen mit leerem Body (reine Bildbeiträge) bekommen
-- beim Rückbau einen Platzhalter, damit der strikte CHECK(length(body) > 0) hält.
-- Läuft mit foreign_keys=OFF (siehe up-Migration) → keine Cascade-Deletes.

CREATE TABLE messages_old (
    id              INTEGER PRIMARY KEY,
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       INTEGER NOT NULL REFERENCES users(id),
    body            TEXT    NOT NULL CHECK(length(body) > 0 AND length(body) <= 2000),
    sent_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reply_to_id     INTEGER REFERENCES messages(id),
    edited_at       DATETIME,
    deleted_at      DATETIME,
    is_system       BOOLEAN NOT NULL DEFAULT 0
);

INSERT INTO messages_old (id, conversation_id, sender_id, body, sent_at, reply_to_id, edited_at, deleted_at, is_system)
SELECT id, conversation_id, sender_id,
       CASE WHEN length(body) > 0 THEN body ELSE '[Bild]' END,
       sent_at, reply_to_id, edited_at, deleted_at, is_system FROM messages;

DROP TABLE messages;
ALTER TABLE messages_old RENAME TO messages;
CREATE INDEX idx_messages_conv ON messages(conversation_id, sent_at DESC);

CREATE TABLE broadcasts_old (
    id          INTEGER PRIMARY KEY,
    sender_id   INTEGER NOT NULL REFERENCES users(id),
    target_type TEXT    NOT NULL CHECK(target_type IN ('all', 'team', 'role')),
    target_id   INTEGER,
    target_role TEXT,
    body        TEXT    NOT NULL CHECK(length(body) > 0 AND length(body) <= 2000),
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at   DATETIME
);

INSERT INTO broadcasts_old (id, sender_id, target_type, target_id, target_role, body, sent_at, edited_at)
SELECT id, sender_id, target_type, target_id, target_role,
       CASE WHEN length(body) > 0 THEN body ELSE '[Bild]' END,
       sent_at, edited_at FROM broadcasts;

DROP TABLE broadcasts;
ALTER TABLE broadcasts_old RENAME TO broadcasts;

DROP TABLE media;
