-- Rollback: CHECK zurück ohne 'operativ'/'sonstiges'; entsprechende Zeilen werden
-- verworfen (WHERE category NOT IN (...)), sonst würde der alte CHECK sie ablehnen.

CREATE TABLE notification_preferences_new (
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category      TEXT    NOT NULL,
    push_enabled  INTEGER NOT NULL DEFAULT 1,
    email_enabled INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, category),
    CHECK (category IN ('games','trainings','duties','duty_reminders','carpooling','membership','chat'))
);

INSERT INTO notification_preferences_new (user_id, category, push_enabled, email_enabled)
SELECT user_id, category, push_enabled, email_enabled FROM notification_preferences
WHERE category NOT IN ('operativ','sonstiges');

DROP TABLE notification_preferences;
ALTER TABLE notification_preferences_new RENAME TO notification_preferences;
