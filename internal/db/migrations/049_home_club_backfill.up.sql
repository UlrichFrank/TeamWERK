-- Backfill members.home_club_id aus dem geprüften Freitext-Mapping in
-- deploy/stammverein-mapping-049.yaml. Voraussetzung: Migration 048 hat
-- die neuen Stammvereine bereits geseedet.
--
-- Invarianten:
--   - Nur exakt geprüfte Zuordnungen (kein Fuzzy-Matching).
--   - Idempotent über `home_club_id IS NULL` — Mitglieder, deren Zuordnung
--     im Frontend bereits gesetzt wurde, werden nicht überschrieben.
--   - Robust gegen abweichende AUTOINCREMENT-Reihenfolge (Name-Subquery
--     statt roher ID).
--   - Freitext 'TS' wird gelöscht (home_club -> NULL), da Bedeutung nicht
--     aufklärbar — siehe YAML.
--   - Freitext 'Flüchtling' bleibt bestehen, home_club_id NULL (kein Verein).

-- --- Bestehende 8 Stammvereine (aus Migration 047) -------------------------

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'SKG Gablenberg 1884')
  WHERE home_club_id IS NULL AND home_club IN ('SKG Gablenberg', 'SkG');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'SKG Stuttgart Max-Eyth-See 1898')
  WHERE home_club_id IS NULL AND home_club IN ('SKG Max-Eyth-See');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'SportKultur Stuttgart')
  WHERE home_club_id IS NULL AND home_club IN ('SK Stuttgart');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'Spvgg 1897 Cannstatt')
  WHERE home_club_id IS NULL AND home_club IN ('Spvgg Canstatt', 'Spvgg Cannstatt');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'TB Gaisburg 1886')
  WHERE home_club_id IS NULL AND home_club IN ('TB Gaisburg');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'TB Untertürkheim 1888')
  WHERE home_club_id IS NULL AND home_club IN ('TB Untertürkheim');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'TSV Stuttgart-Münster 1875/99')
  WHERE home_club_id IS NULL AND home_club IN ('TSV Münster');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'TV Cannstatt 1846')
  WHERE home_club_id IS NULL AND home_club IN ('TV Cannstatt');

-- --- Neue Stammvereine (aus Migration 048) --------------------------------

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'HSG Cannstatt/Münster/Max-Eyth-See')
  WHERE home_club_id IS NULL AND home_club IN ('HSG Cannstatt', 'Ca-Mü-Max');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'HSG Oberer Neckar')
  WHERE home_club_id IS NULL AND home_club IN ('HSG Oberer Neckar', 'ON');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'Hbi Weilimdorf/Feuerbach')
  WHERE home_club_id IS NULL AND home_club IN ('Hbi W/F', 'Hbi');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'HSG Gablenberg-Gaisburg')
  WHERE home_club_id IS NULL AND home_club IN ('HsG GaGa', 'HSG GaGa', 'GaGa');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'Sportvg Feuerbach')
  WHERE home_club_id IS NULL AND home_club IN ('Sportvg Feuerbach');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'HSV Zuffenhausen')
  WHERE home_club_id IS NULL AND home_club IN ('HSV Zuffenhausen');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'TuS Stuttgart')
  WHERE home_club_id IS NULL AND home_club IN ('TuS Stuttgart');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'TV Obertürkheim')
  WHERE home_club_id IS NULL AND home_club IN ('TV Obertürkheim');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'TSV Korntal')
  WHERE home_club_id IS NULL AND home_club IN ('TSV Korntal');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'SV Stuttgarter Kickers')
  WHERE home_club_id IS NULL AND home_club IN ('SV Stgt Kickers');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'TV Fellbach')
  WHERE home_club_id IS NULL AND home_club IN ('Fellbach');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'TV Deizisau')
  WHERE home_club_id IS NULL AND home_club IN ('Deizisau');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'SG Asperg')
  WHERE home_club_id IS NULL AND home_club IN ('Asperg');

UPDATE members SET home_club_id = (SELECT id FROM stammvereine WHERE name = 'SG Hegensberg-Liebersbronn')
  WHERE home_club_id IS NULL AND home_club IN ('HeLi');

-- --- Freitext-Bereinigung -------------------------------------------------
-- 'TS' ist nicht aufklärbar (siehe YAML); Freitext entfernen, damit das
-- Mitglied im Frontend als "ohne Stammverein" erscheint und ggf. neu erfasst
-- werden kann.

UPDATE members SET home_club = NULL WHERE home_club = 'TS';
