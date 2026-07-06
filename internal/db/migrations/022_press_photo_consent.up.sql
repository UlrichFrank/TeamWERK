-- members.foto_veroeffentlichung: dedizierte DSGVO-Einwilligung, dass Fotos der
-- Person auf öffentlichen Kanälen des Vereins (Homepage team-stuttgart.org,
-- Spielberichte) veröffentlicht werden dürfen. Abgegrenzt von photo_visible
-- (nur interne Profilbild-Sichtbarkeit im Portal). foto_veroeffentlichung_date
-- dokumentiert den Zeitpunkt der Einwilligung.
--
-- Neuanlage-Default: 0 (opt-in). Bestandsmitglieder werden auf 1 gesetzt
-- (bewusste Betreiber-Entscheidung: der Bestand wurde de facto bereits über
-- photo_visible publiziert) mit dem Migrationsdatum als _date.

ALTER TABLE members ADD COLUMN foto_veroeffentlichung INTEGER NOT NULL DEFAULT 0;
ALTER TABLE members ADD COLUMN foto_veroeffentlichung_date DATE;

UPDATE members
SET foto_veroeffentlichung = 1,
    foto_veroeffentlichung_date = date('now');
