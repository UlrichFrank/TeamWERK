-- Hinweis: der join_date-Backfill (014.up) wird bewusst NICHT rückgängig gemacht
-- (die implizit gesetzten Eintrittsdaten sind fachlich korrekt und nicht
-- unterscheidbar von echten Eingaben).
ALTER TABLE seasons DROP COLUMN is_inaugural;
ALTER TABLE members DROP COLUMN exit_date;
