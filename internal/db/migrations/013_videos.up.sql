-- 013_videos: Tabelle für selbst gehostete Spielvideos (HLS, self-hosted auf VPS).
-- Jedes Video gehört zu genau einem Team und einer Saison; Auslieferung nur an
-- berechtigte Nutzer (siehe internal/videos/access.go). Hinweis: design.md spricht von
-- `saisons(id)`, die reale Tabelle heißt jedoch `seasons` (siehe 001_initial.up.sql).
CREATE TABLE videos (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    title           TEXT NOT NULL,
    description     TEXT,
    team_id         INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    season_id       INTEGER NOT NULL REFERENCES seasons(id),
    game_id         INTEGER REFERENCES games(id) ON DELETE SET NULL,
    status          TEXT NOT NULL CHECK (status IN
                      ('uploading','queued','processing','ready','failed')),
    upload_id       TEXT,                    -- tus session
    size_bytes      INTEGER,                 -- finale Originalgröße
    duration_sec    INTEGER,                 -- aus ffprobe nach Upload
    failure_reason  TEXT,
    created_by      INTEGER NOT NULL REFERENCES users(id),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ready_at        DATETIME
);
CREATE INDEX idx_videos_team_status ON videos(team_id, status);
CREATE INDEX idx_videos_season ON videos(season_id);
CREATE INDEX idx_videos_status_created ON videos(status, created_at);
