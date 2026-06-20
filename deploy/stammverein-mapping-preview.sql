-- Stammverein-Mapping VORSCHAU (read-only — verändert nichts).
--
-- Zweck: vor dem Backfill home_club -> home_club_id zeigen, welche Freitext-
-- Werte auf welchen Stammverein gemappt würden. Jede UNMATCHED-Zeile muss
-- vom Vorstand/Kassierer entschieden werden (manuell zuweisen oder bewusst
-- NULL = aktiv_ohne lassen).
--
-- Ausführen gegen die Produktiv-DB:
--   sqlite3 /var/lib/teamwerk/teamwerk.db < deploy/stammverein-mapping-preview.sql
--
-- Die Normalisierung bildet beitragslauf.NormalizeClubName ab (lowercase,
-- Punkte/Bindestriche/Schrägstriche entfernt). Der einzige Unterschied
-- (Zusammenfassen mehrfacher Innen-Leerzeichen) ist hier nicht abgebildet —
-- solche Fälle erscheinen konservativ als UNMATCHED zur manuellen Prüfung.

.mode column
.headers on

SELECT
    m.home_club                                             AS freitext,
    COUNT(*)                                                AS anzahl_mitglieder,
    s.id                                                    AS vorgeschlagene_id,
    s.name                                                  AS vorgeschlagener_verein,
    CASE WHEN s.id IS NULL THEN 'UNMATCHED' ELSE 'exakt' END AS status
FROM members m
LEFT JOIN stammvereine s
    ON lower(replace(replace(replace(s.name, '.', ''), '-', ''), '/', '')) =
       lower(replace(replace(replace(m.home_club, '.', ''), '-', ''), '/', ''))
WHERE TRIM(COALESCE(m.home_club, '')) <> ''
GROUP BY m.home_club, s.id, s.name
ORDER BY status, anzahl_mitglieder DESC;
