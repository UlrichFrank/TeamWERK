-- Erweitert den Seed der Stammvereine aus Migration 047 um die im Mapping
-- deploy/stammverein-mapping-049.yaml als "neu: true" geprüften Vereine.
-- Reine Datenmigration auf stammvereine — der Backfill nach
-- members.home_club_id erfolgt getrennt in Migration 049.
--
-- Idempotent über UNIQUE-Constraint auf name (INSERT OR IGNORE).

INSERT OR IGNORE INTO stammvereine (name, sort_order) VALUES
    ('HSG Cannstatt/Münster/Max-Eyth-See',  9),
    ('HSG Oberer Neckar',                  10),
    ('Hbi Weilimdorf/Feuerbach',           11),
    ('HSG Gablenberg-Gaisburg',            12),
    ('Sportvg Feuerbach',                  13),
    ('HSV Zuffenhausen',                   14),
    ('TuS Stuttgart',                      15),
    ('TV Obertürkheim',                    16),
    ('TSV Korntal',                        17),
    ('SV Stuttgarter Kickers',             18),
    ('TV Fellbach',                        19),
    ('TV Deizisau',                        20),
    ('SG Asperg',                          21),
    ('SG Hegensberg-Liebersbronn',         22);
