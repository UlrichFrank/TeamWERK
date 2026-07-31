-- Rollback auf die vier ursprünglichen Principal-Typen.
--
-- ACHTUNG — DATENVERLUST: Alle Berechtigungen vom Typ 'team' und 'team_parents'
-- werden ersatzlos gelöscht. Sie MÜSSEN vor dem Rebuild verschwinden, sonst
-- scheitert der INSERT am wiederhergestellten CHECK-Constraint. Ordner, deren
-- Zugriff ausschließlich über Team-Berechtigungen geregelt war, sind danach nur
-- noch für Admins und Ordner-Eigentümer erreichbar.

DELETE FROM folder_permissions WHERE principal_type IN ('team','team_parents');

CREATE TABLE folder_permissions_old (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id      INTEGER NOT NULL REFERENCES file_folders(id) ON DELETE CASCADE,
    principal_type TEXT    NOT NULL CHECK (principal_type IN ('everyone','role','club_function','user')),
    principal_ref  TEXT,
    can_read       INTEGER NOT NULL DEFAULT 0,
    can_write      INTEGER NOT NULL DEFAULT 0
);

INSERT INTO folder_permissions_old (id, folder_id, principal_type, principal_ref, can_read, can_write)
SELECT id, folder_id, principal_type, principal_ref, can_read, can_write FROM folder_permissions;

DROP TABLE folder_permissions;

ALTER TABLE folder_permissions_old RENAME TO folder_permissions;
