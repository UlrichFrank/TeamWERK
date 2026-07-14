-- Profilbild-Konsolidierung: users.photo_path wird die einzige Foto-Quelle.
-- Bislang liefen drei Upload-Pfade parallel:
--   * Kind pflegt eigenes Profil  → users.photo_path
--   * Elternteil pflegt Kind      → members.photo_path
--   * Admin pflegt Mitglied       → members.photo_path
-- Ergebnis: Eltern und Kind sahen unterschiedliche Fotos. Ab jetzt schreiben
-- alle drei Pfade in users.photo_path (via members.user_id-Lookup). Members
-- ohne User-Account haben kein Foto (bewusste Entscheidung, siehe
-- openspec/changes/unified-user-photo/design.md).
--
-- Migrationsstrategie:
--   1) Fotos von Members-mit-User in users.photo_path übernehmen, falls dort
--      noch keins liegt (users-Foto gewinnt bei Konflikt).
--   2) photo_visible aus members auf user_visibility spiegeln.
--   3) Spalten members.photo_path und members.photo_visible entfernen.
-- Verwaiste Dateien im uploadDir werden vom Server-Backfill beim Start
-- aufgeräumt (internal/upload/backfill.go).

UPDATE users
SET photo_path = (
    SELECT m.photo_path FROM members m
    WHERE m.user_id = users.id AND m.photo_path IS NOT NULL AND m.photo_path != ''
    LIMIT 1
)
WHERE (photo_path IS NULL OR photo_path = '')
  AND EXISTS (
    SELECT 1 FROM members m
    WHERE m.user_id = users.id AND m.photo_path IS NOT NULL AND m.photo_path != ''
  );

-- photo_visible=1 aus members ins user_visibility spiegeln.
-- INSERT für User ohne Visibility-Zeile.
INSERT INTO user_visibility (user_id, photo_visible)
SELECT m.user_id, 1
FROM members m
WHERE m.user_id IS NOT NULL
  AND m.photo_visible = 1
  AND NOT EXISTS (
    SELECT 1 FROM user_visibility uv WHERE uv.user_id = m.user_id
  );

-- UPDATE für User mit Visibility-Zeile, deren photo_visible noch 0 ist.
UPDATE user_visibility
SET photo_visible = 1
WHERE photo_visible = 0
  AND user_id IN (
    SELECT user_id FROM members WHERE user_id IS NOT NULL AND photo_visible = 1
  );

-- Spalten aus members entfernen (SQLite 3.35+ unterstützt DROP COLUMN;
-- Pattern bereits in 002/003 verwendet).
ALTER TABLE members DROP COLUMN photo_path;
ALTER TABLE members DROP COLUMN photo_visible;
