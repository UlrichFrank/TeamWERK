CREATE TABLE mitfahrgelegenheiten (
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
