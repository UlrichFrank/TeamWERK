CREATE TABLE message_reactions (
  message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  user_id    INTEGER NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
  emoji      TEXT    NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (message_id, user_id, emoji)
);
CREATE INDEX idx_message_reactions_msg ON message_reactions(message_id);
