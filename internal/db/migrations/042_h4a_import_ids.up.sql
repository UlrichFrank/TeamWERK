-- H4A-Import: externe IDs + Hallennummer + gelerntes Staffel→Mannschaft-Mapping.
--
-- venues.hall_number  = BWHV-Hallennummer (aus der Hallenliste). Nullable, weil
--   manuelle Nicht-BWHV-Orte und mehrdeutige Adressen keine eindeutige Nummer
--   haben. Partial-Unique-Index erlaubt beliebig viele NULLs, erzwingt aber
--   Eindeutigkeit der gesetzten Nummern.
-- games.external_id   = BWHV-Spielnummer (H4A `Nr.`). Idempotenz-Anker für den
--   Spielimport. Kein globaler UNIQUE — Bestandsspiele ohne external_id sowie
--   manuell angelegte Spiele koexistieren; Eindeutigkeit wird fachlich im
--   Import geprüft, nicht per Constraint erzwungen.
-- h4a_staffel_team_map = gelernte Zuordnung H4A-Staffel (+ eigener Vereinsname)
--   → TeamWERK-Mannschaft. Beim ersten Import manuell gesetzt, danach vorbelegt.

ALTER TABLE venues ADD COLUMN hall_number INTEGER;
CREATE UNIQUE INDEX idx_venues_hall_number ON venues(hall_number) WHERE hall_number IS NOT NULL;

ALTER TABLE games ADD COLUMN external_id TEXT;
CREATE INDEX idx_games_external_id ON games(external_id) WHERE external_id IS NOT NULL;

CREATE TABLE h4a_staffel_team_map (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    staffel    TEXT     NOT NULL,
    club_alias TEXT     NOT NULL,
    team_id    INTEGER  NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (staffel, club_alias)
);
CREATE INDEX idx_h4a_staffel_team_map_team ON h4a_staffel_team_map(team_id);
