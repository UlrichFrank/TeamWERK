-- notification_preferences: zwei neue Kategorien 'operativ' (Vereins-/Funktionärs-
-- Erinnerungen) und 'sonstiges' ("Sonstige Events", z.B. Video fertig). SQLite kann
-- den CHECK nicht in-place ändern → Tabellen-Rebuild, Bestandszeilen 1:1 kopiert.

CREATE TABLE notification_preferences_new (
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category      TEXT    NOT NULL,
    push_enabled  INTEGER NOT NULL DEFAULT 1,
    email_enabled INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, category),
    CHECK (category IN ('games','trainings','duties','duty_reminders','carpooling','membership','chat','operativ','sonstiges'))
);

INSERT INTO notification_preferences_new (user_id, category, push_enabled, email_enabled)
SELECT user_id, category, push_enabled, email_enabled FROM notification_preferences;

DROP TABLE notification_preferences;
ALTER TABLE notification_preferences_new RENAME TO notification_preferences;
