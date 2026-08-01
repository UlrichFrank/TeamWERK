-- Erweitert die Principal-Typen von folder_permissions um 'team' und 'team_parents'.
-- principal_ref trägt für beide eine teams.id als Text; die Zugehörigkeit wird bei
-- jedem Zugriff gegen den Kader der aktiven Saison aufgelöst (siehe internal/policy).
--
-- SQLite kann einen CHECK-Constraint nicht per ALTER TABLE ändern, deshalb ein
-- Table-Rebuild. folder_permissions hat einen ausgehenden FK auf file_folders und
-- KEINE eingehenden Fremdschlüssel; es existieren auch keine Indizes auf der
-- Tabelle. Der Rebuild ist damit unkritisch.
--
-- ACHTUNG: Der down-Pfad löscht 'team'/'team_parents'-Zeilen ersatzlos (Datenverlust
-- in Rückrichtung) — siehe 038_folder_permissions_team_principals.down.sql.

CREATE TABLE folder_permissions_new (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id      INTEGER NOT NULL REFERENCES file_folders(id) ON DELETE CASCADE,
    principal_type TEXT    NOT NULL CHECK (principal_type IN
                     ('everyone','role','club_function','user','team','team_parents')),
    principal_ref  TEXT,
    can_read       INTEGER NOT NULL DEFAULT 0,
    can_write      INTEGER NOT NULL DEFAULT 0
);

INSERT INTO folder_permissions_new (id, folder_id, principal_type, principal_ref, can_read, can_write)
SELECT id, folder_id, principal_type, principal_ref, can_read, can_write FROM folder_permissions;

DROP TABLE folder_permissions;

ALTER TABLE folder_permissions_new RENAME TO folder_permissions;
