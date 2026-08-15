-- Mitteilungs-Zielgruppen (mitteilung-zielgruppen): das alte Trio
-- target_type/target_id/target_role weicht vier vereinsweiten Zielgruppen.
--
-- Vorher war 'role' funktionslos: der Resolver löste target_role gegen
-- users.role auf (nur 'admin'/'standard'), die Werte im CHECK waren aber
-- Vereinsfunktionen. Der Fan-out traf null Empfänger, ohne Fehler.
--
-- SQLite kann einen CHECK nicht in-place ändern und Spalten nicht zusammen mit
-- einem geänderten CHECK entfernen → Tabellen-Rebuild nach dem Muster aus 028.
-- Der Migrationslauf (internal/db/db.go:Migrate) setzt PRAGMA foreign_keys=OFF,
-- daher löst DROP TABLE broadcasts KEINE Cascade-Deletes auf broadcast_reads
-- aus. Das ist hier die entscheidende Invariante: an broadcast_reads hängt die
-- gesamte Zustellung, target_type wird nach dem Senden nirgends mehr gelesen.
--
-- Bestandswerte: 'all' ist semantisch identisch zu 'users'. 'team' und 'role'
-- haben kein Gegenstück und werden auf 'legacy' abgebildet — ein Wert, der
-- persistierbar und lesbar, über die API aber nicht setzbar ist. Die frühere
-- Team-Zuordnung geht dabei verloren (target_id entfällt); sie wurde nie
-- angezeigt, und die Empfängermenge bleibt in broadcast_reads vollständig.

CREATE TABLE broadcasts_new (
    id          INTEGER PRIMARY KEY,
    sender_id   INTEGER NOT NULL REFERENCES users(id),
    target_type TEXT    NOT NULL CHECK(target_type IN ('users', 'members', 'spieler', 'eltern', 'legacy')),
    body        TEXT    NOT NULL,
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at   DATETIME,
    media_id    INTEGER REFERENCES media(id),
    CHECK ((length(body) > 0 OR media_id IS NOT NULL) AND length(body) <= 2000)
);

INSERT INTO broadcasts_new (id, sender_id, target_type, body, sent_at, edited_at, media_id)
SELECT id, sender_id,
       CASE target_type WHEN 'all' THEN 'users' ELSE 'legacy' END,
       body, sent_at, edited_at, media_id FROM broadcasts;

DROP TABLE broadcasts;
ALTER TABLE broadcasts_new RENAME TO broadcasts;
