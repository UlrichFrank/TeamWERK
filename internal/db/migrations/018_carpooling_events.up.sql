CREATE TABLE carpooling_events (
  id         INTEGER  PRIMARY KEY AUTOINCREMENT,
  user_id    INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  game_id    INTEGER  NOT NULL REFERENCES games(id) ON DELETE CASCADE,
  type       TEXT     NOT NULL CHECK(type IN (
               'biete_created','suche_created',
               'pairing_requested','pairing_confirmed','pairing_rejected','pairing_cancelled',
               'biete_deleted','suche_deleted'
             )),
  actor_name TEXT     NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
