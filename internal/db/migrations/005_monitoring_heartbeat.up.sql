-- Single-Row-Tabelle für den Zeitstempel des letzten erfolgreichen Scheduler-Laufs.
-- Dient als Datenquelle für den Dead-Man-Switch (scheduler_age_sec in /api/healthz,
-- teamwerk_scheduler_age_seconds in /api/metrics). Der CHECK(id=1) erzwingt genau
-- eine Zeile; geschrieben wird per INSERT ... ON CONFLICT(id) DO UPDATE.
CREATE TABLE monitoring_heartbeat (
    id         INTEGER PRIMARY KEY CHECK (id = 1),
    updated_at TEXT NOT NULL
);
