CREATE TABLE notification_preferences (
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category      TEXT    NOT NULL,
    push_enabled  INTEGER NOT NULL DEFAULT 1,
    email_enabled INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, category),
    CHECK (category IN ('games','trainings','duties','duty_reminders','carpooling','membership'))
);

CREATE TABLE notification_log (
    user_id  INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ref_type TEXT    NOT NULL,
    ref_id   INTEGER NOT NULL,
    sent_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, ref_type, ref_id)
);
