-- Index zuerst droppen, sonst scheitert DROP COLUMN login_name.
DROP INDEX IF EXISTS users_login_name_unique;
ALTER TABLE users DROP COLUMN login_name;

ALTER TABLE membership_requests DROP COLUMN parent_email;
ALTER TABLE membership_requests DROP COLUMN is_child;
