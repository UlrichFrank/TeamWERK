ALTER TABLE users ADD COLUMN duty_reminder_days INT NULL DEFAULT NULL;

CREATE TABLE duty_reminder_log (
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_date DATE     NOT NULL,
    sent_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, event_date)
);
