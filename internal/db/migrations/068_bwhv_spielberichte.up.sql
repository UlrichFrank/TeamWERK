-- 068_bwhv_spielberichte: Spielberichte, Tabellen und Spielerstatistik aus der
-- öffentlichen Handball4All-Schnittstelle des BWHV (bwhv-spielberichte).
--
-- Rein additiv. Die zentrale Grenze dieses Changes ist, was hier NICHT steht:
-- games bekommt keine Ergebnisspalten und keine neuen Zeilen. Der komplette
-- Staffel-Spielplan (~810 Begegnungen/Saison, überwiegend fremde Vereine) lebt
-- in bwhv_games; nur wo eine BWHV-Spielnummer auf ein bestehendes eigenes Spiel
-- zeigt, wird bwhv_games.game_id gesetzt. Dieselbe Figur wie
-- training_sessions.team_id IS NULL bei den Übungsgruppen: die Abwesenheit der
-- Verknüpfung ist das Gate, und Spielplan, Dienst-Regeneration, iCal-Feed, RSVP
-- und Anwesenheit bleiben davon strukturell unberührt (design.md §7).

-- Staffel je Kader ---------------------------------------------------------
-- kader ist bereits saison-gebunden, die Zuordnung gilt damit je Saison ohne
-- zusätzliche Verknüpfung. Bewusst OHNE CHECK auf kind='team': SQLite kann das
-- nur über einen Tabellen-Rebuild, und kader wurde dafür schon dreimal neu
-- gebaut (034, 041, 057). Die Beschränkung erzwingt der Handler mit HTTP 409.
ALTER TABLE kader ADD COLUMN staffel TEXT;

-- Staffel-Snapshot ---------------------------------------------------------
-- Gespeichert wird der Staffelcode (gClassSname, z.B. "mB-RL-BW"), NICHT die
-- Handball4All-Klassen-ID: die ist saisonabhängig, der Code ist es nicht. Die
-- ID wird bei jedem Lauf aus dem Katalog aufgelöst, so übersteht die Zuordnung
-- einen Saisonwechsel ohne Nachpflege (design.md §1.2).
--
-- org_id/sub_org_id bilden die og/o-Falle ab: Bezirks-Staffeln sind nur über
-- o=<bezirk> bei og=<verband> erreichbar; og=<bezirk> antwortet HTTP 401.
CREATE TABLE bwhv_staffeln (
    id         INTEGER  PRIMARY KEY,
    season_id  INTEGER  NOT NULL REFERENCES seasons(id) ON DELETE CASCADE,
    code       TEXT     NOT NULL,
    name       TEXT     NOT NULL DEFAULT '',
    org_id     INTEGER  NOT NULL,
    sub_org_id INTEGER,
    period_id  TEXT     NOT NULL DEFAULT '',
    table_json TEXT     NOT NULL DEFAULT '',
    polled_at  DATETIME,
    UNIQUE (season_id, code)
);

-- Begegnungen --------------------------------------------------------------
-- game_no ist die BWHV-Spielnummer (gNo) und damit derselbe Anker wie
-- games.external_id aus Migration 042. sgid ist leer, solange kein Bericht
-- freigegeben ist — die Schnittstelle liefert dort 0. Genau dieses Feld ist das
-- Bereitschaftssignal: das System muss den Zeitpunkt der Berichtsfreigabe nicht
-- schätzen (design.md §1.4).
CREATE TABLE bwhv_games (
    id             INTEGER  PRIMARY KEY,
    staffel_id     INTEGER  NOT NULL REFERENCES bwhv_staffeln(id) ON DELETE CASCADE,
    game_no        TEXT     NOT NULL,
    sgid           TEXT     NOT NULL DEFAULT '',
    game_id        INTEGER  REFERENCES games(id) ON DELETE SET NULL,
    date           DATE     NOT NULL,
    time           TEXT     NOT NULL DEFAULT '',
    home_team      TEXT     NOT NULL DEFAULT '',
    guest_team     TEXT     NOT NULL DEFAULT '',
    home_goals     INTEGER,
    guest_goals    INTEGER,
    home_goals_ht  INTEGER,
    guest_goals_ht INTEGER,
    hall_number    TEXT     NOT NULL DEFAULT '',
    UNIQUE (staffel_id, game_no)
);

-- Das Poll-Fenster wird aus dieser Tabelle abgeleitet (Datum + Zeit + offenes
-- sgid), deshalb trägt der Index genau diese Spalten. Ein eigenes Zustandsfeld
-- für die Poll-Steuerung gibt es bewusst nicht — es wäre eine zweite Wahrheit
-- neben den Begegnungen selbst (design.md §4).
CREATE INDEX idx_bwhv_games_staffel_date ON bwhv_games(staffel_id, date);
CREATE INDEX idx_bwhv_games_game_id ON bwhv_games(game_id) WHERE game_id IS NOT NULL;

-- Berichte -----------------------------------------------------------------
-- state trennt drei Fälle, die nicht zusammenfallen dürfen: pending = PDF noch
-- nicht verarbeitet (Abruf wiederholbar), parsed = ausgewertet, parse_failed =
-- die harte Kreuzprobe (Verlaufs-Summe gegen Kopf-Endstand) ist gescheitert.
-- Im letzten Fall entsteht KEINE Zeile in bwhv_player_games/bwhv_events, das
-- PDF bleibt aber liegen, damit ein Parser-Fix denselben Bericht ohne erneuten
-- Fremdabruf verarbeiten kann (design.md §5.3).
CREATE TABLE bwhv_reports (
    id             INTEGER  PRIMARY KEY,
    bwhv_game_id   INTEGER  NOT NULL UNIQUE REFERENCES bwhv_games(id) ON DELETE CASCADE,
    sgid           TEXT     NOT NULL,
    state          TEXT     NOT NULL DEFAULT 'pending'
                   CHECK (state IN ('pending','parsed','parse_failed')),
    pdf_path       TEXT     NOT NULL DEFAULT '',
    spectators     TEXT     NOT NULL DEFAULT '',
    referees       TEXT     NOT NULL DEFAULT '',
    warnings_json  TEXT     NOT NULL DEFAULT '',
    failure_reason TEXT     NOT NULL DEFAULT '',
    attempts       INTEGER  NOT NULL DEFAULT 0,
    fetched_at     DATETIME,
    parsed_at      DATETIME
);

-- Spieler ------------------------------------------------------------------
-- Eigene und fremde Spieler stehen in derselben Tabelle; member_id ist gesetzt
-- oder NULL, mehr unterscheidet sie nicht. Eine Anonymisierungsschicht wie in
-- dutyfairness gibt es hier bewusst nicht (design.md §2).
--
-- Identitätsraum ist (staffel, team_name, name): der NAME ist der Schlüssel,
-- Schreibvarianten führt der Import über Levenshtein zusammen, bevor er
-- schreibt. Die Trikotnummer bestätigt nur und bricht Gleichstand bei
-- Namensgleichheit — sie ist NICHT Teil des UNIQUE, weil ein geliehenes Trikot
-- sonst eine zweite Person erzeugte (design.md §6.1).
CREATE TABLE bwhv_players (
    id         INTEGER  PRIMARY KEY,
    staffel_id INTEGER  NOT NULL REFERENCES bwhv_staffeln(id) ON DELETE CASCADE,
    team_name  TEXT     NOT NULL,
    name       TEXT     NOT NULL,
    birth_year INTEGER,
    member_id  INTEGER  REFERENCES members(id) ON DELETE SET NULL,
    conflict   TEXT     NOT NULL DEFAULT '',
    UNIQUE (staffel_id, team_name, name)
);

CREATE INDEX idx_bwhv_players_member ON bwhv_players(member_id) WHERE member_id IS NOT NULL;

-- Spielerzeile je Bericht (Mannschaftsliste) -------------------------------
-- jersey_number hängt hier und nicht am Spieler: sie kann sich je Spiel ändern.
CREATE TABLE bwhv_player_games (
    report_id        INTEGER NOT NULL REFERENCES bwhv_reports(id) ON DELETE CASCADE,
    player_id        INTEGER NOT NULL REFERENCES bwhv_players(id) ON DELETE CASCADE,
    side             TEXT    NOT NULL DEFAULT '' CHECK (side IN ('home','guest','')),
    jersey_number    INTEGER,
    goals            INTEGER NOT NULL DEFAULT 0,
    seven_m_attempts INTEGER NOT NULL DEFAULT 0,
    seven_m_goals    INTEGER NOT NULL DEFAULT 0,
    two_min          INTEGER NOT NULL DEFAULT 0,
    yellow           INTEGER NOT NULL DEFAULT 0,
    red              INTEGER NOT NULL DEFAULT 0,
    blue             INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (report_id, player_id)
);

CREATE INDEX idx_bwhv_player_games_player ON bwhv_player_games(player_id);

-- Spielverlauf -------------------------------------------------------------
-- kind kennt 'other' als ausdrücklichen Auffangwert: eine Verlaufszeile
-- unbekannter Form wird mit ihrem Rohtext gespeichert statt verworfen, damit
-- keine Information verlorengeht und ein neuer Ereignistyp beim Nachsehen
-- auffällt (design.md §5.2).
--
-- game_second ist die Spielzeit in Sekunden (aus "12:27"), clock_time die
-- Uhrzeit aus dem Bericht. Beide getrennt, weil nur die erste vergleichbar ist.
CREATE TABLE bwhv_events (
    id            INTEGER PRIMARY KEY,
    report_id     INTEGER NOT NULL REFERENCES bwhv_reports(id) ON DELETE CASCADE,
    seq           INTEGER NOT NULL,
    clock_time    TEXT    NOT NULL DEFAULT '',
    game_second   INTEGER NOT NULL DEFAULT 0,
    score_home    INTEGER,
    score_guest   INTEGER,
    kind          TEXT    NOT NULL
                  CHECK (kind IN ('goal','seven_m_goal','seven_m_miss','two_min',
                                  'warning','disqualification','timeout','other')),
    side          TEXT    NOT NULL DEFAULT '' CHECK (side IN ('home','guest','')),
    player_id     INTEGER REFERENCES bwhv_players(id) ON DELETE SET NULL,
    jersey_number INTEGER,
    raw_text      TEXT    NOT NULL DEFAULT '',
    UNIQUE (report_id, seq)
);

CREATE INDEX idx_bwhv_events_player ON bwhv_events(player_id) WHERE player_id IS NOT NULL;
