-- Herkunft einer Chat-Gruppe (chat-gruppe-aktualisieren): die Standard-Gruppen-
-- Kacheln, aus denen die Gruppe besteht. Grundlage für den Abgleich „Aktualisieren".
-- Bewusst KEIN Backfill: Bestandsgruppen bleiben ohne Herkunft, bis ihr Ersteller
-- sie im Modal festlegt — eine abgeleitete Zuordnung wäre ein unsichtbarer Fehler.
-- ref_id ohne FK: zeigt je nach group_type auf teams.id oder kader.id (0 bei
-- alle_trainer); ein verschwundener Kader soll im Modal als leer auffallen,
-- nicht still aus der Herkunft fallen.
-- Additiv: das Vorgänger-Binary ignoriert die Tabelle.

CREATE TABLE conversation_sources (
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    group_type      TEXT    NOT NULL CHECK (group_type IN ('team', 'practice')),
    ref_id          INTEGER NOT NULL,
    kind            TEXT    NOT NULL CHECK (kind IN ('trainer', 'spieler', 'eltern', 'alle_trainer')),
    PRIMARY KEY (conversation_id, group_type, ref_id, kind)
);
