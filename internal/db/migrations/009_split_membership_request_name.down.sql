ALTER TABLE membership_requests ADD COLUMN name TEXT NOT NULL DEFAULT '';

UPDATE membership_requests SET
  name = CASE
    WHEN last_name != '' THEN trim(first_name || ' ' || last_name)
    ELSE first_name
  END;

ALTER TABLE membership_requests DROP COLUMN first_name;
ALTER TABLE membership_requests DROP COLUMN last_name;
