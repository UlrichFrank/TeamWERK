CREATE TABLE file_folders (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    parent_id  INTEGER REFERENCES file_folders(id) ON DELETE CASCADE,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE folder_permissions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id      INTEGER NOT NULL REFERENCES file_folders(id) ON DELETE CASCADE,
    principal_type TEXT    NOT NULL CHECK (principal_type IN ('everyone','role','club_function','user')),
    principal_ref  TEXT,
    can_read       INTEGER NOT NULL DEFAULT 0,
    can_write      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE files (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id     INTEGER NOT NULL REFERENCES file_folders(id) ON DELETE CASCADE,
    original_name TEXT    NOT NULL,
    disk_name     TEXT    NOT NULL UNIQUE,
    size          INTEGER NOT NULL,
    mime_type     TEXT    NOT NULL DEFAULT '',
    uploaded_by   INTEGER NOT NULL REFERENCES users(id),
    created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
