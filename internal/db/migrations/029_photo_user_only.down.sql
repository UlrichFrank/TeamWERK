-- Rollback der Foto-Konsolidierung.
-- ACHTUNG: Datenverlust bewusst akzeptiert. Die Down-Migration stellt die
-- Spalten wieder her, kopiert Foto-Referenzen aber NICHT aus users zurück.
-- Betriebliche Absicherung: DB-Backup vor `make migrate-remote-up`.

ALTER TABLE members ADD COLUMN photo_path TEXT;
ALTER TABLE members ADD COLUMN photo_visible INTEGER NOT NULL DEFAULT 0;
