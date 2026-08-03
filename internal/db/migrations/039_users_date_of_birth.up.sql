-- Self-Service-Geburtsdatum für Nutzer mit direkt verknüpftem Mitgliederaccount
-- (Kontakt-Tab im Profil, analog zu street/zip/city). Solange NULL, zeigt das
-- Frontend als Default das Geburtsdatum aus dem Mitgliederbereich
-- (members.date_of_birth) an — kein Backfill hier nötig.
ALTER TABLE users ADD COLUMN date_of_birth TEXT;
