-- Rückbau: media-Spalten width/height entfernen.
-- SQLite ≥ 3.35 unterstützt DROP COLUMN direkt; modernc.org/sqlite (v1.50.x)
-- bringt SQLite ≥ 3.47 mit, daher kein Tabellen-Rebuild nötig.

ALTER TABLE media DROP COLUMN width;
ALTER TABLE media DROP COLUMN height;
