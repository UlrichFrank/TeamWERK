CREATE TABLE IF NOT EXISTS push_subscriptions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint   TEXT    NOT NULL UNIQUE,
    p256dh     TEXT    NOT NULL,
    auth       TEXT    NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user_id ON push_subscriptions(user_id);
