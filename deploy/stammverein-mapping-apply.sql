-- Stammverein-Mapping APPLY — erst nach Review der Vorschau ausführen!
--
-- Setzt home_club_id NUR für exakte (normalisierte) Treffer. UNMATCHED-Werte
-- bleiben bewusst NULL und werden manuell im Mitglied zugewiesen — es findet
-- KEIN Fuzzy-Matching statt, um keine falschen Zuordnungen festzuschreiben.
--
-- NICHT Teil der automatischen Migration. Manuell ausführen:
--   sqlite3 /var/lib/teamwerk/teamwerk.db < deploy/stammverein-mapping-apply.sql
--
-- Vorher die Vorschau prüfen: deploy/stammverein-mapping-preview.sql

UPDATE members
   SET home_club_id = (
       SELECT s.id FROM stammvereine s
       WHERE lower(replace(replace(replace(s.name, '.', ''), '-', ''), '/', '')) =
             lower(replace(replace(replace(members.home_club, '.', ''), '-', ''), '/', ''))
   )
 WHERE TRIM(COALESCE(home_club, '')) <> ''
   AND home_club_id IS NULL;

-- Kontrolle nach dem Apply: gesetzte vs. noch offene (UNMATCHED) Zuordnungen.
SELECT
    COUNT(home_club_id)                                              AS gesetzt,
    SUM(CASE WHEN TRIM(COALESCE(home_club,'')) <> '' AND home_club_id IS NULL
             THEN 1 ELSE 0 END)                                      AS unmatched_offen
FROM members;
