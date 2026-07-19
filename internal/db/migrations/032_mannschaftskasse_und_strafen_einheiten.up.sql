-- Mannschaftskasse (Cashbook) + wählbare Einheit für Strafen (Euro | Striche).
-- Baut auf 031 (Mannschafts-Strafen) auf, bleibt rein additiv. Alles kader-scoped
-- (Kader = Team + Saison), konsistent mit penalty_types/team_penalties/
-- kader_strafenwarte. Der Kassenwart ist — wie der Strafenwart — ein per-Kader-
-- Appointment (kein globaler member_club_functions-Wert), Sibling von
-- kader_strafenwarte.

-- Einheit für Strafen pro Kader. Gilt für Katalog UND alle vergebenen Strafen des
-- Kaders (keine gemischten Einheiten). Default 'euro'; ein Strich wird intern als
-- 100 Cent gespeichert (Rate 1 € = 1 Strich), damit Summen einheitlich in Cent
-- aggregieren.
CREATE TABLE penalty_settings (
    kader_id INTEGER PRIMARY KEY REFERENCES kader(id) ON DELETE CASCADE,
    unit     TEXT NOT NULL DEFAULT 'euro' CHECK (unit IN ('euro', 'striche'))
);

-- Backfill: für jeden bestehenden Kader eine Default-Row (unit='euro'), damit
-- GET /penalty-settings nie auf eine fehlende Row läuft. Idempotent.
INSERT OR IGNORE INTO penalty_settings (kader_id, unit)
SELECT id, 'euro' FROM kader;

-- Kassenbuch pro Kader. amount_cent ist SIGNED: Einzahlung positiv, Ausgabe
-- negativ. Saldo = SUM(amount_cent). Nur INSERT + DELETE (kein Status/Storno),
-- konsistent mit team_penalties.
CREATE TABLE team_cashbook_entries (
    id                  INTEGER  PRIMARY KEY AUTOINCREMENT,
    kader_id            INTEGER  NOT NULL REFERENCES kader(id)   ON DELETE CASCADE,
    amount_cent         INTEGER  NOT NULL CHECK (amount_cent <> 0),
    note                TEXT     NOT NULL,
    entered_by_member_id INTEGER REFERENCES members(id) ON DELETE SET NULL,
    entered_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_team_cashbook_entries_kader ON team_cashbook_entries(kader_id);

-- Kassenwart-Appointment pro Kader (Sibling von kader_strafenwarte). Trainer
-- ernennt; Trainer ODER Kassenwart dürfen buchen.
CREATE TABLE kader_kassenwarte (
    kader_id   INTEGER  NOT NULL REFERENCES kader(id)   ON DELETE CASCADE,
    member_id  INTEGER  NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (kader_id, member_id)
);
