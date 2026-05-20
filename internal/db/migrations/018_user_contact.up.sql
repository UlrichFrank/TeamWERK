ALTER TABLE users ADD COLUMN street TEXT;
ALTER TABLE users ADD COLUMN zip TEXT;
ALTER TABLE users ADD COLUMN city TEXT;
ALTER TABLE users ADD COLUMN photo_path TEXT;

CREATE TABLE user_phones (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label       TEXT    NOT NULL,
    number      TEXT    NOT NULL,
    sort_order  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE user_visibility (
    user_id         INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    phones_visible  INTEGER NOT NULL DEFAULT 0,
    address_visible INTEGER NOT NULL DEFAULT 0,
    photo_visible   INTEGER NOT NULL DEFAULT 0
);
