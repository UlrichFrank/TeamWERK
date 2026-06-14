-- Remove duplicate suche entries, keeping only the most recent per (game_id, user_id)
DELETE FROM mitfahrgelegenheiten
WHERE typ = 'suche'
  AND id NOT IN (
    SELECT MAX(id)
    FROM mitfahrgelegenheiten
    WHERE typ = 'suche'
    GROUP BY game_id, user_id
  );

CREATE UNIQUE INDEX idx_mitfahr_suche_unique
    ON mitfahrgelegenheiten(game_id, user_id) WHERE typ = 'suche';
