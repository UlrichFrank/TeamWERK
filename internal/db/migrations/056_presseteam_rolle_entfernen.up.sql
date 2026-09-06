-- Die System-Rolle 'presseteam' entfällt ersatzlos: alles, was sie erlaubte
-- (Spielbericht-Draft anlegen, Spielbericht-Dienst ziehen), darf künftig jeder
-- eingeloggte Nutzer. Wer den Bericht schreibt, bestimmt weiterhin der Besitz des
-- Spielbericht-Dienstes; veröffentlicht wird weiterhin nur über die
-- Vereinsfunktion medien/vorstand.
--
-- Reihenfolge zwingend: erst die Bestände umschreiben, dann den CHECK
-- verengen — sonst scheitert der INSERT SELECT des Rebuilds am neuen Constraint.
--
-- SQLite kennt kein ALTER CHECK → Tabellen-Rebuild (12-Schritt-Recipe wie in
-- 019_match_reports). FK-Enforcement ist während der Migration auf
-- Connection-Ebene deaktiviert (siehe internal/db/db.go Migrate()), daher kein
-- PRAGMA nötig.

UPDATE users             SET role = 'standard' WHERE role = 'presseteam';
UPDATE invitation_tokens SET role = 'standard' WHERE role = 'presseteam';

-- users: role-CHECK verengen via Rebuild.
CREATE TABLE users_new (
    id                 INTEGER  PRIMARY KEY AUTOINCREMENT,
    email              TEXT,
    password           TEXT     NOT NULL DEFAULT '',
    role               TEXT     NOT NULL DEFAULT 'standard'
                       CHECK (role IN ('admin','standard')),
    team_id            INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    first_name         TEXT     NOT NULL DEFAULT '',
    last_name          TEXT     NOT NULL DEFAULT '',
    street             TEXT,
    zip                TEXT,
    city               TEXT,
    photo_path         TEXT,
    duty_reminder_days INT      NULL DEFAULT NULL,
    last_login_at      DATETIME,
    can_login          INTEGER  NOT NULL DEFAULT 1,
    maps_provider      TEXT     NOT NULL DEFAULT 'auto'
                       CHECK (maps_provider IN ('auto','google','apple')),
    login_name         TEXT,
    recovery_email     TEXT,
    failed_login_count INTEGER  NOT NULL DEFAULT 0,
    locked_until       TEXT,
    date_of_birth      TEXT
);

INSERT INTO users_new (id, email, password, role, team_id, created_at,
    updated_at, first_name, last_name, street, zip, city, photo_path,
    duty_reminder_days, last_login_at, can_login, maps_provider, login_name,
    recovery_email, failed_login_count, locked_until, date_of_birth)
SELECT id, email, password, role, team_id, created_at, updated_at,
    first_name, last_name, street, zip, city, photo_path, duty_reminder_days,
    last_login_at, can_login, maps_provider, login_name, recovery_email,
    failed_login_count, locked_until, date_of_birth
FROM users;

DROP TABLE users;
ALTER TABLE users_new RENAME TO users;

CREATE UNIQUE INDEX users_email_login_unique ON users(email)
WHERE can_login = 1 AND email IS NOT NULL;
CREATE UNIQUE INDEX users_login_name_unique ON users(LOWER(login_name))
WHERE login_name IS NOT NULL;

-- invitation_tokens: role-CHECK verengen via Rebuild.
CREATE TABLE invitation_tokens_new (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    email      TEXT     NOT NULL,
    team_id    INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    role       TEXT     NOT NULL DEFAULT 'standard'
               CHECK (role IN ('admin','standard')),
    token      TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    comment    TEXT,
    first_name TEXT     NOT NULL DEFAULT '',
    last_name  TEXT     NOT NULL DEFAULT '',
    member_id  INTEGER  REFERENCES members(id) ON DELETE SET NULL
);

INSERT INTO invitation_tokens_new (id, email, team_id, role, token, expires_at,
    used_at, created_at, comment, first_name, last_name, member_id)
SELECT id, email, team_id, role, token, expires_at, used_at, created_at,
    comment, first_name, last_name, member_id
FROM invitation_tokens;

DROP TABLE invitation_tokens;
ALTER TABLE invitation_tokens_new RENAME TO invitation_tokens;
