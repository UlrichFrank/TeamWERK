DROP TABLE IF EXISTS mitfahrt_paarungen;

DROP INDEX IF EXISTS idx_mitfahr_biete_unique;

CREATE TABLE mitfahrgelegenheiten_old (
  id         INTEGER  PRIMARY KEY AUTOINCREMENT,
  game_id    INTEGER  NOT NULL REFERENCES games(id) ON DELETE CASCADE,
  user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  typ        TEXT     NOT NULL CHECK(typ IN ('biete','suche')),
  plaetze    INTEGER,
  treffpunkt TEXT,
  notiz      TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(game_id, user_id)
);

-- Copy only the first entry per (game_id, user_id) to restore the unique constraint
INSERT OR IGNORE INTO mitfahrgelegenheiten_old
  SELECT id, game_id, user_id, typ, plaetze, treffpunkt, notiz, created_at, updated_at
  FROM mitfahrgelegenheiten;

DROP TABLE mitfahrgelegenheiten;

ALTER TABLE mitfahrgelegenheiten_old RENAME TO mitfahrgelegenheiten;
