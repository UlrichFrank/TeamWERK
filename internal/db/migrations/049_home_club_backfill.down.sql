-- Rücknahme des Backfills: setzt home_club_id zurück auf NULL nur für
-- Mitglieder, deren Freitext-home_club von dieser Migration gemappt wurde.
-- Frontend-manuelle Zuweisungen (home_club ohne passenden Freitext im Mapping
-- oder bewusst NULL gewählt) sind nicht betroffen.
--
-- Asymmetrie: Die gelöschten 'TS'-Freitexte (Yonas Ogbamicael, Florian
-- Scheurle) werden bewusst NICHT wiederhergestellt — der Verlust ist
-- aufgehoben, weil 'TS' inhaltlich leer war. Down ist hier ein Schema-Rollback,
-- keine vollständige Daten-Rücknahme.

UPDATE members SET home_club_id = NULL
  WHERE home_club IN (
    -- aus Migration 047 (Bestand)
    'SKG Gablenberg', 'SkG',
    'SKG Max-Eyth-See',
    'SK Stuttgart',
    'Spvgg Canstatt', 'Spvgg Cannstatt',
    'TB Gaisburg',
    'TB Untertürkheim',
    'TSV Münster',
    'TV Cannstatt',
    -- aus Migration 048 (neue Stammvereine)
    'HSG Cannstatt', 'Ca-Mü-Max',
    'HSG Oberer Neckar', 'ON',
    'Hbi W/F', 'Hbi',
    'HsG GaGa', 'HSG GaGa', 'GaGa',
    'Sportvg Feuerbach',
    'HSV Zuffenhausen',
    'TuS Stuttgart',
    'TV Obertürkheim',
    'TSV Korntal',
    'SV Stgt Kickers',
    'Fellbach',
    'Deizisau',
    'Asperg',
    'HeLi'
  );
