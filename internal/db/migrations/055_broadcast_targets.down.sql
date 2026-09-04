-- Rückbau von mitteilung-team-gruppen: broadcasts.target_type kehrt zurück,
-- broadcast_targets entfällt.
--
-- BEWUSST VERLUSTBEHAFTET: eine Mitteilung mit mehreren Zielen passt nicht in
-- eine Spalte. Solche Zeilen fallen auf 'legacy' zurück — ein Wert, den der
-- CHECK erlaubt und den der Handler beim Schreiben ablehnt. Mitteilungen mit
-- genau einem Ziel behalten ihren Wert, sofern er im alten Vokabular existiert;
-- die neuen teambezogenen Kinds haben dort kein Gegenstück und werden ebenfalls
-- zu 'legacy'.
--
-- Tragbar aus demselben Grund wie beim Rückbau von 049: target_type wird nach
-- dem Senden nirgends gelesen, die Empfängermenge steht vollständig in
-- broadcast_reads — und die fasst diese Migration nicht an.
--
-- Läuft mit foreign_keys=OFF (siehe up-Migration) → keine Cascade-Deletes auf
-- broadcast_reads.

CREATE TABLE broadcasts_old (
    id          INTEGER PRIMARY KEY,
    sender_id   INTEGER NOT NULL REFERENCES users(id),
    target_type TEXT    NOT NULL CHECK(target_type IN ('users', 'members', 'spieler', 'eltern', 'legacy')),
    body        TEXT    NOT NULL,
    sent_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    edited_at   DATETIME,
    media_id    INTEGER REFERENCES media(id),
    CHECK ((length(body) > 0 OR media_id IS NOT NULL) AND length(body) <= 2000)
);

INSERT INTO broadcasts_old (id, sender_id, target_type, body, sent_at, edited_at, media_id)
SELECT b.id, b.sender_id,
       COALESCE((
           SELECT CASE
                    WHEN COUNT(*) = 1
                     AND MIN(t.kind) IN ('users', 'members', 'spieler', 'eltern')
                    THEN MIN(t.kind)
                    ELSE 'legacy'
                  END
           FROM broadcast_targets t
           WHERE t.broadcast_id = b.id
       ), 'legacy'),
       b.body, b.sent_at, b.edited_at, b.media_id
FROM broadcasts b;

DROP TABLE broadcasts;
ALTER TABLE broadcasts_old RENAME TO broadcasts;

DROP TABLE broadcast_targets;
