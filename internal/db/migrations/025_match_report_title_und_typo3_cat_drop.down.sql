-- Rollback zu 025_match_report_title_und_typo3_cat_drop.up.sql:
--   * teams.typo3_category_uid als nullable INTEGER wieder anlegen (Bestand
--     der Spalte vor dem Rollback ist verloren — Werte waren in Produktion
--     ohnehin NULL, siehe design.md/D-3).
--   * match_reports.title entfernen.

ALTER TABLE teams ADD COLUMN typo3_category_uid INTEGER;

ALTER TABLE match_reports DROP COLUMN title;
