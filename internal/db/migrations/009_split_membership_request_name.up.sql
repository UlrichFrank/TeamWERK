ALTER TABLE membership_requests ADD COLUMN first_name TEXT NOT NULL DEFAULT '';
ALTER TABLE membership_requests ADD COLUMN last_name  TEXT NOT NULL DEFAULT '';

UPDATE membership_requests SET
  first_name = CASE
    WHEN instr(name, ' ') > 0 THEN substr(name, 1, instr(name, ' ') - 1)
    ELSE name
  END,
  last_name = CASE
    WHEN instr(name, ' ') > 0 THEN substr(name, instr(name, ' ') + 1)
    ELSE ''
  END;

ALTER TABLE membership_requests DROP COLUMN name;
