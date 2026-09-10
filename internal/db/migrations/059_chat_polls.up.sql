-- 059_chat_polls: Signal-artige Umfragen in Gruppenkonversationen
-- (chat-umfragen).
--
-- Die Frage steht in der bestehenden messages.body (design.md §1) — eine
-- Umfrage ist eine messages-Zeile + drei Seitentabellen, kein neuer
-- messages.kind. "Ist diese Nachricht eine Umfrage?" = existiert eine
-- chat_polls-Zeile. Rein additiv, keine Bestandsdaten betroffen.
CREATE TABLE chat_polls (
    message_id     INTEGER PRIMARY KEY REFERENCES messages(id) ON DELETE CASCADE,
    allow_multiple INTEGER NOT NULL DEFAULT 0 CHECK (allow_multiple IN (0,1)),
    closed_at      DATETIME
);

CREATE TABLE chat_poll_options (
    id         INTEGER PRIMARY KEY,
    message_id INTEGER NOT NULL REFERENCES chat_polls(message_id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    label      TEXT    NOT NULL CHECK (length(label) BETWEEN 1 AND 100),
    UNIQUE (message_id, position)
);

-- Stimme = vollständige Auswahl des Nutzers je Umfrage (design.md §2): PUT
-- .../poll/vote ersetzt die Zeilen dieses Nutzers innerhalb einer Umfrage
-- atomar (DELETE + INSERT in derselben Transaktion).
CREATE TABLE chat_poll_votes (
    option_id  INTEGER NOT NULL REFERENCES chat_poll_options(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (option_id, user_id)
);

CREATE INDEX idx_chat_poll_votes_user ON chat_poll_votes(user_id);
