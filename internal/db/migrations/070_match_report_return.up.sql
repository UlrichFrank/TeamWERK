-- Spielbericht-Rückgabe: ein Freigeber gibt einen eingereichten Bericht
-- (pending_review) mit Kommentar an den Autor zurück; der Bericht landet
-- wieder in 'draft'. Kein neuer State — „zurückgegeben" ist ein Draft mit
-- gesetztem review_comment. Der Kommentar bleibt beim erneuten Einreichen
-- stehen, damit der Freigeber sieht, worum er gebeten hatte; eine weitere
-- Rückgabe überschreibt ihn.
ALTER TABLE match_reports ADD COLUMN review_comment TEXT;
ALTER TABLE match_reports ADD COLUMN returned_at DATETIME;
