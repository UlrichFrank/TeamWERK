-- Trainingstagebuch: Spieler erfassen eigenständig Trainingseinheiten außerhalb
-- des Vereinstrainings (Kraft/Ausdauer/etc.) mit optionalem Nachweis-Upload.
--
-- season_id ist bewusst NULLBAR und wird bei der Erfassung aus der zu diesem
-- Zeitpunkt AKTIVEN Saison übernommen — nicht per Datumsvergleich gegen
-- trained_on ermittelt. Saisons werden als freie Zeiträume gepflegt und stoßen
-- nicht lückenlos aneinander; eine datumsbasierte Zuordnung würde alle Einträge
-- aus der Sommerpause verlieren. `season_id IS NULL` bedeutet für den
-- Retention-Job "nie automatisch löschen".
--
-- proof_purged_at unterscheidet "Nachweis von der Retention gelöscht" von "hatte
-- nie einen Nachweis": der Retention-Job löscht die Datei, setzt proof_purged_at
-- und NULLt proof_disk_name; der Eintrag selbst (Datum/Art/Dauer/RPE) bleibt
-- dauerhaft erhalten.
CREATE TABLE training_diary_entries (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    member_id           INTEGER NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    season_id           INTEGER REFERENCES seasons(id) ON DELETE SET NULL,
    trained_on          DATE NOT NULL,
    kind                TEXT NOT NULL CHECK (kind IN
                          ('kraft','ausdauer','athletik','technik','beweglichkeit','reha','sonstiges')),
    kind_custom         TEXT,
    duration_min        INTEGER NOT NULL CHECK (duration_min > 0 AND duration_min <= 600),
    rpe                 INTEGER NOT NULL CHECK (rpe BETWEEN 1 AND 10),
    note                TEXT,
    proof_disk_name     TEXT,
    proof_mime          TEXT,
    proof_size          INTEGER,
    proof_uploaded_at   DATETIME,
    proof_purged_at     DATETIME,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_training_diary_member_date ON training_diary_entries(member_id, trained_on DESC);
CREATE INDEX idx_training_diary_season ON training_diary_entries(season_id);
