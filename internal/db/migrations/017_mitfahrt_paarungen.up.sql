-- Rebuild mitfahrgelegenheiten without the UNIQUE(game_id, user_id) constraint
-- so that a user can have multiple 'suche' entries per game.
CREATE TABLE mitfahrgelegenheiten_new (
  id         INTEGER  PRIMARY KEY AUTOINCREMENT,
  game_id    INTEGER  NOT NULL REFERENCES games(id) ON DELETE CASCADE,
  user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  typ        TEXT     NOT NULL CHECK(typ IN ('biete','suche')),
  plaetze    INTEGER,
  treffpunkt TEXT,
  notiz      TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO mitfahrgelegenheiten_new
  SELECT id, game_id, user_id, typ, plaetze, treffpunkt, notiz, created_at, updated_at
  FROM mitfahrgelegenheiten;

DROP TABLE mitfahrgelegenheiten;

ALTER TABLE mitfahrgelegenheiten_new RENAME TO mitfahrgelegenheiten;

-- Partial unique index: one 'biete' entry per user per game, but unlimited 'suche'
CREATE UNIQUE INDEX idx_mitfahr_biete_unique
  ON mitfahrgelegenheiten(game_id, user_id)
  WHERE typ = 'biete';

-- Pairing table
CREATE TABLE mitfahrt_paarungen (
  id            INTEGER  PRIMARY KEY AUTOINCREMENT,
  biete_id      INTEGER  NOT NULL REFERENCES mitfahrgelegenheiten(id) ON DELETE CASCADE,
  suche_id      INTEGER  NOT NULL REFERENCES mitfahrgelegenheiten(id) ON DELETE CASCADE,
  initiiert_von TEXT     NOT NULL CHECK(initiiert_von IN ('biete','suche')),
  status        TEXT     NOT NULL DEFAULT 'pending'
                         CHECK(status IN ('pending','confirmed','rejected')),
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(biete_id, suche_id)
);
