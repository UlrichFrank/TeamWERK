-- Rückbau von mitteilung-zielgruppen: altes CHECK-Trio zurück, target_id und
-- target_role kehren als NULL-Spalten wieder.
--
-- BEWUSST VERLUSTBEHAFTET: die ursprüngliche Team-Zuordnung ('team' + target_id)
-- ist nach der up-Migration nicht mehr rekonstruierbar — target_id existiert
-- dort nicht. Alle Zeilen fallen deshalb auf 'all' zurück, unabhängig davon,
-- welche Zielgruppe sie ursprünglich hatten. Das ist tragbar, weil target_type
-- nach dem Senden nirgends gelesen wird und die Empfängermenge ausschließlich
-- in broadcast_reads steht — die diese Migration nicht anfasst.
--
-- Läuft mit foreign_keys=OFF (siehe up-Migration) → keine Cascade-Deletes auf
-- broadcast_reads.

CREATE TABLE broadcasts_old (
    id          INTEGER PRIMARY KEY,
    sender_id   INTEGER NOT NULL REFERENCES users(id),
    target_type TEXT    NOT NULL CHECK(target_type IN ('all', 'team', 'role')),
    target_id   INTEGER,
    target_role TEXT,
    body        TEXT    NOT NULL,
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at   DATETIME,
    media_id    INTEGER REFERENCES media(id),
    CHECK ((length(body) > 0 OR media_id IS NOT NULL) AND length(body) <= 2000)
);

INSERT INTO broadcasts_old (id, sender_id, target_type, target_id, target_role, body, sent_at, edited_at, media_id)
SELECT id, sender_id, 'all', NULL, NULL, body, sent_at, edited_at, media_id FROM broadcasts;

DROP TABLE broadcasts;
ALTER TABLE broadcasts_old RENAME TO broadcasts;
