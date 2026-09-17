DROP TABLE duty_board_reminder_log;

CREATE TABLE duty_reminder_log_old (
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_date DATE     NOT NULL,
    sent_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, event_date)
);

-- Nur die frueheren 2-Tage-Reminder ueberleben den Rollback; die 7/3/1-Tage-Zeilen
-- existierten vor der Up-Migration nicht.
INSERT INTO duty_reminder_log_old (user_id, event_date, sent_at)
SELECT user_id, event_date, sent_at FROM duty_reminder_log WHERE days_before = 2;

DROP TABLE duty_reminder_log;
ALTER TABLE duty_reminder_log_old RENAME TO duty_reminder_log;
