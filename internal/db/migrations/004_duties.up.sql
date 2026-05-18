CREATE TABLE duty_types (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,
    hours_value     REAL    NOT NULL DEFAULT 1.0,
    cash_substitute REAL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE duty_slots (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    event_name      TEXT     NOT NULL,
    event_date      DATE     NOT NULL,
    duty_type_id    INTEGER  NOT NULL REFERENCES duty_types(id) ON DELETE RESTRICT,
    role_desc       TEXT,
    slots_total     INTEGER  NOT NULL DEFAULT 1,
    slots_filled    INTEGER  NOT NULL DEFAULT 0,
    team_id         INTEGER  REFERENCES teams(id) ON DELETE SET NULL,
    season_id       INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE duty_assignments (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    duty_slot_id    INTEGER  NOT NULL REFERENCES duty_slots(id)  ON DELETE CASCADE,
    user_id         INTEGER  NOT NULL REFERENCES users(id)        ON DELETE CASCADE,
    status          TEXT     NOT NULL DEFAULT 'assigned' CHECK (status IN ('assigned','fulfilled','cash_substitute')),
    cash_amount     REAL,
    fulfilled_at    DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (duty_slot_id, user_id)
);

CREATE TABLE duty_accounts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    season_id   INTEGER NOT NULL REFERENCES seasons(id)  ON DELETE CASCADE,
    soll        REAL    NOT NULL DEFAULT 0,
    ist         REAL    NOT NULL DEFAULT 0,
    UNIQUE (user_id, season_id)
);

CREATE TABLE duty_season_targets (
    season_id       INTEGER NOT NULL REFERENCES seasons(id)    ON DELETE CASCADE,
    duty_type_id    INTEGER NOT NULL REFERENCES duty_types(id) ON DELETE CASCADE,
    target_hours    REAL    NOT NULL DEFAULT 0,
    PRIMARY KEY (season_id, duty_type_id)
);
