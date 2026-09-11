-- Impersonation-Audit (security-haertung-welle-1, Entscheidung 7): die neue
-- Kategorie 'admin' hält den Impersonations-Hinweis für den Admin selbst im
-- Event-Log fest — drei Tage im Dashboard nachlesbar, nicht nur im slog.
--
-- SQLite kann einen CHECK-Constraint nicht per ALTER TABLE ändern →
-- Tabellen-Rebuild nach dem Muster aus 049_broadcast_zielgruppen.up.sql. Der
-- Migrationslauf (internal/db/db.go:Migrate) setzt PRAGMA foreign_keys=OFF,
-- DROP TABLE user_events löst deshalb keine Cascade-Deletes aus; Daten werden
-- 1:1 per INSERT … SELECT übernommen, kein Zeilenverlust.

CREATE TABLE user_events_new (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category   TEXT     NOT NULL
               CHECK (category IN ('games','trainings','duties','duty_reminders',
                                   'carpooling','membership','operativ','sonstiges',
                                   'admin')),
    title      TEXT     NOT NULL,
    body       TEXT     NOT NULL DEFAULT '',
    url        TEXT     NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    seen_at    DATETIME
);

INSERT INTO user_events_new (id, user_id, category, title, body, url, created_at, seen_at)
SELECT id, user_id, category, title, body, url, created_at, seen_at FROM user_events;

DROP TABLE user_events;
ALTER TABLE user_events_new RENAME TO user_events;

-- Indizes aus 050_user_events.up.sql unverändert neu anlegen (gehen beim
-- DROP TABLE mit verloren).
CREATE INDEX idx_user_events_user_created ON user_events(user_id, created_at DESC);
CREATE INDEX idx_user_events_retention ON user_events(seen_at, created_at);
