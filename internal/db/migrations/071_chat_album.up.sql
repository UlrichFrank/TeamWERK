-- Chat-Alben (chat-mehrere-bilder): eine Nachricht bzw. Mitteilung trägt bis zu
-- 10 Bilder. Quelle der Wahrheit sind die Zuordnungstabellen; die Spalten
-- messages.media_id / broadcasts.media_id bleiben bestehen und tragen weiter das
-- Bild auf Position 0 — für den CHECK (Text ODER Bild) und für ältere Clients.
-- Additiv: das Vorgänger-Binary ignoriert die neuen Tabellen.

CREATE TABLE message_media (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    media_id   INTEGER NOT NULL REFERENCES media(id),
    position   INTEGER NOT NULL CHECK (position BETWEEN 0 AND 9),
    PRIMARY KEY (message_id, position)
);
-- Eine media-ID hängt an höchstens einer Nachricht; zugleich Lookup-Index für
-- media.canSee.
CREATE UNIQUE INDEX idx_message_media_media ON message_media(media_id);

CREATE TABLE broadcast_media (
    broadcast_id INTEGER NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    media_id     INTEGER NOT NULL REFERENCES media(id),
    position     INTEGER NOT NULL CHECK (position BETWEEN 0 AND 9),
    PRIMARY KEY (broadcast_id, position)
);
CREATE UNIQUE INDEX idx_broadcast_media_media ON broadcast_media(media_id);

-- Bestandsbilder als Ein-Element-Album übernehmen.
INSERT INTO message_media (message_id, media_id, position)
    SELECT id, media_id, 0 FROM messages WHERE media_id IS NOT NULL;
INSERT INTO broadcast_media (broadcast_id, media_id, position)
    SELECT id, media_id, 0 FROM broadcasts WHERE media_id IS NOT NULL;
