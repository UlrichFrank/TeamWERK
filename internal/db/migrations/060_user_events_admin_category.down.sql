-- Rückbau von user_events_admin_category: Kategorie 'admin' entfällt wieder.
--
-- BEWUSST VERLUSTBEHAFTET: Zeilen mit category='admin' (Impersonations-Hinweise)
-- werden gelöscht, bevor der alte, engere CHECK wiederhergestellt wird — sie
-- würden sonst den Rebuild mit einem Constraint-Verstoß scheitern lassen.
-- Das ist tragbar, weil der Hinweis im slog-Warn ("impersonation", …) ohnehin
-- separat protokolliert wird (design.md Entscheidung 7).

DELETE FROM user_events WHERE category = 'admin';

CREATE TABLE user_events_old (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category   TEXT     NOT NULL
               CHECK (category IN ('games','trainings','duties','duty_reminders',
                                   'carpooling','membership','operativ','sonstiges')),
    title      TEXT     NOT NULL,
    body       TEXT     NOT NULL DEFAULT '',
    url        TEXT     NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    seen_at    DATETIME
);

INSERT INTO user_events_old (id, user_id, category, title, body, url, created_at, seen_at)
SELECT id, user_id, category, title, body, url, created_at, seen_at FROM user_events;

DROP TABLE user_events;
ALTER TABLE user_events_old RENAME TO user_events;

CREATE INDEX idx_user_events_user_created ON user_events(user_id, created_at DESC);
CREATE INDEX idx_user_events_retention ON user_events(seen_at, created_at);
