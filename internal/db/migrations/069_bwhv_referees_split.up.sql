-- 069_bwhv_referees_split: Schiedsrichter eines Spielberichts als einzelne
-- Personen (staffel-statistiken).
--
-- Rein additiv. bwhv_reports.referees BLEIBT unangetastet: die ungetrennte
-- Zeile des Dokuments ist der Beleg und erlaubt eine später verbesserte
-- Trennung ohne erneuten Fremdabruf — dieselbe Begründung, aus der das PDF
-- liegen bleibt (design.md §7).
--
-- referees_json trägt die getrennten Namen als JSON-Array; leer heißt "noch
-- nicht getrennt" (Bestandsberichte) oder "keine Namen im Dokument". Beide
-- Fälle erzeugen keine Zeile in der Schiedsrichter-Rangliste, und ein Backfill
-- über die Namensregel ist bewusst NICHT Teil dieses Changes: er machte für
-- den gesamten Bestand den unsicheren Pfad zum Hauptpfad.
ALTER TABLE bwhv_reports ADD COLUMN referees_json TEXT NOT NULL DEFAULT '';

-- referees_uncertain=1 heißt: die Namen standen in EINER Spalte der Textebene
-- und wurden nach der Namensregel geschnitten. Die Rangliste weist das aus,
-- statt den Schnitt als gesichert durchlaufen zu lassen.
ALTER TABLE bwhv_reports ADD COLUMN referees_uncertain INTEGER NOT NULL DEFAULT 0;
