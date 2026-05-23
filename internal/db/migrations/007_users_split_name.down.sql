ALTER TABLE users ADD COLUMN name TEXT NOT NULL DEFAULT '';

UPDATE users SET
  name = CASE
    WHEN last_name != '' THEN first_name || ' ' || last_name
    ELSE first_name
  END;

ALTER TABLE users DROP COLUMN first_name;
ALTER TABLE users DROP COLUMN last_name;
