-- notification_preferences: 'chat' zur erlaubten Kategorie-Menge hinzufügen.
-- SQLite kann einen CHECK-Constraint nicht in-place ändern → Tabellen-Rebuild.
-- Bestandszeilen werden 1:1 kopiert. Die frühere CHECK-Menge ließ 'chat' nicht
-- zu, obwohl Profil-UI (ProfileMiscTab) und push.GetAllPreferences 'chat' als
-- vollwertige Kategorie führen → PUT /notification-preferences scheiterte mit 500.

CREATE TABLE notification_preferences_new (
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category      TEXT    NOT NULL,
    push_enabled  INTEGER NOT NULL DEFAULT 1,
    email_enabled INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, category),
    CHECK (category IN ('games','trainings','duties','duty_reminders','carpooling','membership','chat'))
);

INSERT INTO notification_preferences_new (user_id, category, push_enabled, email_enabled)
SELECT user_id, category, push_enabled, email_enabled FROM notification_preferences;

DROP TABLE notification_preferences;
ALTER TABLE notification_preferences_new RENAME TO notification_preferences;
