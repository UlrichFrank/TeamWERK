-- SQLite does not support DROP COLUMN in older versions; recreate table without gender column
CREATE TABLE members_backup AS SELECT id, first_name, last_name, date_of_birth, pass_number, jersey_number, position, status, user_id, created_at, updated_at, member_number FROM members;
DROP TABLE members;
ALTER TABLE members_backup RENAME TO members;
