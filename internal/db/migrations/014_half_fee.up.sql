-- Halber Beitrag bei unterjährigem Ein-/Austritt und im ersten Abrechnungsjahr.

-- Austrittsdatum: ermöglicht die Halbierung (und Einbeziehung) unterjähriger
-- Austritte im Beitragslauf. Nullbar in der DB; Pflicht (bei status=ausgetreten)
-- nur in der App-Validierung.
ALTER TABLE members ADD COLUMN exit_date DATE;

-- Erstes Abrechnungsjahr des Vereins: einmalige Startkonzession (alle zahlen halb).
-- Pro-Saison-Flag, vom Admin in der Saisonverwaltung gesetzt.
ALTER TABLE seasons ADD COLUMN is_inaugural INTEGER NOT NULL DEFAULT 0;

-- Backfill Eintrittsdatum für Bestandsmitglieder: ein Tag VOR dem frühesten
-- Saisonstart, damit die Eintritts-Halbierung (join_date im Saisonfenster) für
-- Altbestand nie greift. NICHT created_at verwenden — die Datensätze entstanden
-- bei Systemeinführung (≈ jetzt) und lägen sonst im aktuellen Saisonfenster.
-- Fallback fixes frühes Datum, falls noch keine Saison existiert.
UPDATE members
SET join_date = COALESCE((SELECT date(MIN(start_date), '-1 day') FROM seasons), '2000-01-01')
WHERE join_date IS NULL OR join_date = '';
