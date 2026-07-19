-- Trainer-gepflegte, serien-gebundene Dauer-Abmeldung eines Spielers.
-- Orthogonal zu member_absences (member-global, selbst-gepflegt, zählt entschuldigt):
-- diese ist team-/serien-gebunden, trainer-gepflegt und schließt Session×Member
-- vollständig aus dem Statistik-Nenner aus. team_id wird NICHT redundant gespeichert,
-- sondern über training_series.team_id abgeleitet.
CREATE TABLE member_series_unavailabilities (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    member_id           INTEGER NOT NULL REFERENCES members(id)         ON DELETE CASCADE,
    training_series_id  INTEGER NOT NULL REFERENCES training_series(id) ON DELETE CASCADE,
    start_date          DATE,           -- NULL = ab Serien-Beginn
    end_date            DATE,           -- NULL = permanent (bis Serien-Ende)
    reason              TEXT NOT NULL DEFAULT '',
    created_by          INTEGER NOT NULL REFERENCES users(id)           ON DELETE RESTRICT,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (member_id, training_series_id, start_date)
);
CREATE INDEX idx_msu_series ON member_series_unavailabilities(training_series_id);
CREATE INDEX idx_msu_member ON member_series_unavailabilities(member_id);
