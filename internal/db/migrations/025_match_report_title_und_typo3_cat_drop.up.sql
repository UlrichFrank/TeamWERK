-- spielbericht-titel-und-publisher-fix:
--   1) match_reports.title als editierbares Titel-Feld (NOT NULL DEFAULT '',
--      damit Bestandsdrafts nicht kollidieren). Handler befüllt neue Drafts
--      via BuildTitle(matchDate, opponent); Autor:in überschreibt im Formular.
--   2) teams.typo3_category_uid entfällt — die TYPO3-Middleware löst die
--      sys_category ab jetzt anhand des Team-Kürzels (team_category_name) auf.
--      Harter Cut, kein Fallback.
--
-- SQLite 3.35+ unterstützt ALTER TABLE ... DROP COLUMN direkt; wird im Repo
-- bereits mehrfach genutzt (z.B. 009_drop_legacy_bank_columns.up.sql,
-- 021_teams_typo3_category.down.sql).

ALTER TABLE match_reports ADD COLUMN title TEXT NOT NULL DEFAULT '';

ALTER TABLE teams DROP COLUMN typo3_category_uid;
