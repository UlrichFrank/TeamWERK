-- 038_video_teams: Viele-zu-viele-Zuordnung Video ↔ Team (Sichtbarkeit).
-- video_teams ersetzt den bisherigen Ein-Team-Filter (videos.team_id bleibt
-- als Eigentümer-Team für Verwaltungsrechte erhalten).
CREATE TABLE video_teams (
    video_id INTEGER NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    team_id  INTEGER NOT NULL REFERENCES teams(id)  ON DELETE CASCADE,
    PRIMARY KEY (video_id, team_id)
);
CREATE INDEX idx_video_teams_video_id ON video_teams(video_id);

-- Bestehende Zuordnungen übernehmen.
INSERT INTO video_teams (video_id, team_id)
SELECT id, team_id FROM videos;
