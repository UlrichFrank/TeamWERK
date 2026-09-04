-- Mitteilungs-Ziele als Zeilen (mitteilung-team-gruppen): broadcasts.target_type
-- (eine Spalte, ein Wert) weicht der Zeilentabelle broadcast_targets. Damit
-- trägt eine Mitteilung mehrere Ziele, und Ziele können teambezogen sein.
--
-- Genau der additive Weg, den mitteilung-zielgruppen/design.md §4 für diesen
-- Fall vorgezeichnet hat ("Zeilentabelle daneben, target_type als
-- Ein-Zeilen-Fall migrieren").
--
-- SQLite kann einen CHECK nicht in-place ändern und Spalten nicht zusammen mit
-- einem geänderten CHECK entfernen → Tabellen-Rebuild nach dem Muster aus 028
-- und 049. Der Migrationslauf (internal/db/db.go:Migrate) setzt
-- PRAGMA foreign_keys=OFF, daher löst DROP TABLE broadcasts KEINE
-- Cascade-Deletes auf broadcast_reads aus. Das ist hier die entscheidende
-- Invariante: an broadcast_reads hängt die gesamte Zustellung, target_type
-- wurde nach dem Senden nirgends mehr gelesen.

CREATE TABLE broadcast_targets (
    broadcast_id INTEGER NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    kind         TEXT    NOT NULL CHECK(kind IN (
                     'users', 'members', 'spieler', 'eltern',
                     'team_spieler', 'team_eltern', 'team_trainer', 'alle_trainer',
                     'legacy')),
    -- Historische Aufzeichnung, kein aktiver Verweis: ein gelöschtes Team soll
    -- die Mitteilung nicht mitnehmen, deshalb bewusst kein ON DELETE CASCADE
    -- (Teams werden ohnehin über is_active stillgelegt, nicht gelöscht).
    team_id      INTEGER REFERENCES teams(id),
    CHECK ((kind LIKE 'team\_%' ESCAPE '\') = (team_id IS NOT NULL))
);

-- Ein Ziel je Mitteilung nur einmal. Als PRIMARY KEY über die drei Spalten
-- griffe die Regel bei den vereinsweiten Zielen nicht: dort ist team_id NULL,
-- und in einem SQLite-Unique-Index sind NULLs paarweise verschieden — zwei
-- Zeilen (broadcast, 'users', NULL) wären erlaubt. COALESCE macht die Lücke zu.
CREATE UNIQUE INDEX idx_broadcast_targets_unique
    ON broadcast_targets(broadcast_id, kind, COALESCE(team_id, 0));
CREATE INDEX idx_broadcast_targets_broadcast ON broadcast_targets(broadcast_id);

-- Bestandszeilen übernehmen ihren bisherigen Wert als genau ein Ziel ohne Team.
INSERT INTO broadcast_targets (broadcast_id, kind, team_id)
SELECT id, target_type, NULL FROM broadcasts;

CREATE TABLE broadcasts_new (
    id        INTEGER PRIMARY KEY,
    sender_id INTEGER NOT NULL REFERENCES users(id),
    body      TEXT    NOT NULL,
    sent_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at DATETIME,
    media_id  INTEGER REFERENCES media(id),
    CHECK ((length(body) > 0 OR media_id IS NOT NULL) AND length(body) <= 2000)
);

INSERT INTO broadcasts_new (id, sender_id, body, sent_at, edited_at, media_id)
SELECT id, sender_id, body, sent_at, edited_at, media_id FROM broadcasts;

DROP TABLE broadcasts;
ALTER TABLE broadcasts_new RENAME TO broadcasts;
