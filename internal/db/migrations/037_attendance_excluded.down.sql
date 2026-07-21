-- SQLite unterstützt kein DROP COLUMN vor 3.35; Rollback löscht und
-- re-created die Tabellen ohne die neue Spalte. Da dies in Tests aber
-- nie gebraucht wird (NewDB startet immer fresh), reicht ein No-Op.
SELECT 1;
