-- Treffzeit + Treffpunkt-Ort je Spiel (spiel-treffpunkt).
-- Die Treffzeit wird als Abstand in Minuten VOR dem Anwurf gespeichert, nicht als
-- Uhrzeit: verlegt der H4A-Import oder ein Bearbeiten den Anwurf (games.time),
-- wandert die Treffzeit mit, ohne dass einer dieser Schreibpfade davon wissen muss.
-- NULL = keine Treffzeit. Der Ort existiert nur zusammen mit einer Treffzeit
-- (erzwingt der Handler).
-- Additiv: das Vorgänger-Binary ignoriert Spalten und Tabelle.

ALTER TABLE games ADD COLUMN meet_offset_minutes INTEGER
    CHECK (meet_offset_minutes IS NULL OR meet_offset_minutes BETWEEN 0 AND 720);
ALTER TABLE games ADD COLUMN meet_place TEXT NOT NULL DEFAULT ''
    CHECK (length(meet_place) <= 100);

-- Debounce-Queue der Änderungs-Push: höchstens eine wartende Meldung je Spiel.
-- prev_* hält den Stand VOR der ersten Änderung des Fensters (wird beim Upsert
-- nicht überschrieben) — der Scheduler meldet nur den Netto-Unterschied.
CREATE TABLE pending_game_meeting_push (
    game_id      INTEGER  PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
    prev_offset  INTEGER,
    prev_place   TEXT     NOT NULL DEFAULT '',
    notify_after DATETIME NOT NULL,
    updated_by   INTEGER  REFERENCES users(id) ON DELETE SET NULL
);
