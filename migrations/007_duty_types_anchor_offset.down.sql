-- SQLite does not support DROP COLUMN in older versions; recreate table without the added columns
CREATE TABLE duty_types_backup AS SELECT id, name, hours_value, cash_substitute, created_at FROM duty_types;
DROP TABLE duty_types;
ALTER TABLE duty_types_backup RENAME TO duty_types;
