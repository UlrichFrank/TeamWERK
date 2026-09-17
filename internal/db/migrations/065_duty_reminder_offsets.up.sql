-- Mehrere Erinnerungs-Zeitpunkte (7/3/2/1 Tage) statt nur einem fixen 2-Tage-Reminder.
-- duty_reminder_log bekommt den Offset als Teil des Eindeutigkeits-Schlüssels, sonst
-- blockiert ein bereits verschickter 7-Tage-Reminder den 3-/2-/1-Tage-Reminder für
-- denselben event_date. SQLite kennt kein ADD COLUMN mit neuem PK -> Tabellen-Rebuild
-- (Muster wie Migration 018). Bestandszeilen waren ausschließlich der alte 2-Tage-
-- Reminder, daher Backfill mit days_before=2.
CREATE TABLE duty_reminder_log_new (
    user_id     INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_date  DATE     NOT NULL,
    days_before INTEGER  NOT NULL DEFAULT 2,
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, event_date, days_before)
);

INSERT INTO duty_reminder_log_new (user_id, event_date, days_before, sent_at)
SELECT user_id, event_date, 2, sent_at FROM duty_reminder_log;

DROP TABLE duty_reminder_log;
ALTER TABLE duty_reminder_log_new RENAME TO duty_reminder_log;

-- Eigenständige Idempotenz für die vereinsweite Vorstands-Übersicht (3/1 Tage vorher),
-- bewusst getrennt von duty_reminder_log: andere Empfängergruppe, anderer Zweck.
CREATE TABLE duty_board_reminder_log (
    user_id     INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_date  DATE     NOT NULL,
    days_before INTEGER  NOT NULL,
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, event_date, days_before)
);
