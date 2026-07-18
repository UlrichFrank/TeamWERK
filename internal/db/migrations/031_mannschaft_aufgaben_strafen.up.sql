-- Mannschafts-Aufgaben (Responsibilities) + Mannschafts-Strafen (Penalties).
-- Alles kader-scoped (Kader = Team + Saison), konsistent mit kader_members/
-- kader_trainers. label/reason/amount_cent werden als Snapshot gespeichert, damit
-- ein späterer Catalog-Edit bereits vergebene Zuweisungen/Strafen nicht rückwirkend
-- ändert. Der Strafenwart ist ein per-Kader-Appointment (kein globaler
-- member_club_functions-Wert) — Sibling von kader_trainers.

-- Aufgaben-Catalog pro Kader (Trainer pflegt). Reines Vorschlags-Vokabular, keine Semantik.
CREATE TABLE responsibility_types (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id   INTEGER  NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    label      TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kader_id, label)
);
CREATE INDEX idx_responsibility_types_kader ON responsibility_types(kader_id);

-- Zuweisung einer Aufgabe an einen Spieler. label = Snapshot (Catalog ODER Freitext).
CREATE TABLE member_responsibilities (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id   INTEGER  NOT NULL REFERENCES kader(id)   ON DELETE CASCADE,
    member_id  INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    label      TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kader_id, member_id, label)
);
CREATE INDEX idx_member_responsibilities_kader  ON member_responsibilities(kader_id);
CREATE INDEX idx_member_responsibilities_member ON member_responsibilities(member_id);

-- Strafen-Catalog pro Kader (Trainer pflegt) mit Default-Betrag in Cent.
CREATE TABLE penalty_types (
    id                  INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id            INTEGER  NOT NULL REFERENCES kader(id) ON DELETE CASCADE,
    reason              TEXT     NOT NULL,
    default_amount_cent INTEGER  NOT NULL DEFAULT 0 CHECK (default_amount_cent >= 0),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kader_id, reason)
);
CREATE INDEX idx_penalty_types_kader ON penalty_types(kader_id);

-- Strafenwart-Appointment pro Kader (Sibling von kader_trainers). Trainer ernennt.
CREATE TABLE kader_strafenwarte (
    kader_id   INTEGER  NOT NULL REFERENCES kader(id)   ON DELETE CASCADE,
    member_id  INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (kader_id, member_id)
);

-- Vergebene Strafe. reason/amount_cent = Snapshot. Nur INSERT + DELETE, kein Status
-- (Storno = DELETE einer Row, Zurücksetzen je Spieler = DELETE aller Rows des Members).
CREATE TABLE team_penalties (
    id                   INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id             INTEGER  NOT NULL REFERENCES kader(id)   ON DELETE CASCADE,
    member_id            INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    amount_cent          INTEGER  NOT NULL CHECK (amount_cent > 0),
    reason               TEXT     NOT NULL,
    created_by_member_id INTEGER  REFERENCES members(id) ON DELETE SET NULL,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_team_penalties_kader  ON team_penalties(kader_id);
CREATE INDEX idx_team_penalties_member ON team_penalties(member_id);
