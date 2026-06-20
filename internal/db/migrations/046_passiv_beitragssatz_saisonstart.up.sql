-- Fix Datums-Bug in der Beitragsmatrix: Der Beitragslauf nutzt IMMER den
-- Stichtag 01.07. der Saison. Der Passiv-Satz aus 043 galt jedoch erst ab
-- 2027-01-01 (2027-01-01 > 2026-07-01), wodurch passive Mitglieder in der
-- Saison 2026/27 fälschlich mit `kein_beitragssatz` ausgeschlossen wurden.
-- Alle drei Kategorien stammen aus derselben Anlage 1 (beschlossen 22.04.2026)
-- und sollen ab Saisonstart 2026/27 (01.07.2026) gelten.
-- Betrag identisch (6000 ct), daher ändert sich am späteren Stichtag nichts.
INSERT OR IGNORE INTO beitrags_saetze (kategorie, betrag_eur, valid_from) VALUES
    ('passiv', 6000, '2026-07-01');
