package scheduler

import (
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// walCheckpointSettingKey ist der system_settings-Key, unter dem das Datum
// (Berlin, "YYYY-MM-DD") des zuletzt ausgeführten WAL-Checkpoints steht.
// Bewusst kein eigenes Idempotenz-Table wie notification_log: der Job läuft
// höchstens einmal pro Kalendertag, ein einzelner Zustandswert reicht
// (design.md Decision 6, tasks.md 3.3).
const walCheckpointSettingKey = "last_wal_checkpoint"

// Wartungsfenster: Sonntag 04:00 Uhr Europe/Berlin — Nebenzeit mit dem
// geringsten Nutzeraufkommen. Run() (und damit dieser Job) läuft minütlich
// über den Scheduler-Cron (setup-vps.sh), trifft das Fenster also in genau
// einer Minute pro Woche.
const (
	walCheckpointWeekday = time.Sunday
	walCheckpointHour    = 4
)

// walCheckpoint führt wöchentlich `PRAGMA wal_checkpoint(TRUNCATE)` aus, damit
// die WAL-Datei nicht unbegrenzt wächst (specs/db-maintenance/spec.md:
// "Wöchentlicher WAL-Checkpoint"). Dünner Wrapper um walCheckpointAt mit der
// echten Uhrzeit — der Scheduler selbst hat keine injizierbare Clock, die
// testbare Kernlogik nimmt `now` deshalb als Parameter (siehe *_test.go).
func (s *Scheduler) walCheckpoint() {
	s.walCheckpointAt(time.Now())
}

func (s *Scheduler) walCheckpointAt(now time.Time) {
	local := now.In(timez.Berlin())
	if local.Weekday() != walCheckpointWeekday || local.Hour() != walCheckpointHour {
		return
	}

	today := local.Format("2006-01-02")

	var last string
	err := s.db.QueryRow(
		`SELECT value FROM system_settings WHERE key = ?`, walCheckpointSettingKey,
	).Scan(&last)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logIfBusy(err, "walCheckpoint.read")
		slog.Error("scheduler wal checkpoint: marker read failed", "error", err)
		return
	}
	if last == today {
		return // im laufenden Wartungsfenster bereits ausgeführt
	}

	start := time.Now()
	var busy, walPages, checkpointedPages int
	row := s.db.QueryRow(`PRAGMA wal_checkpoint(TRUNCATE)`)
	if err := row.Scan(&busy, &walPages, &checkpointedPages); err != nil {
		logIfBusy(err, "walCheckpoint.pragma")
		slog.Error("scheduler wal checkpoint failed", "error", err)
		return
	}
	duration := time.Since(start)

	// Marker NACH erfolgreichem Checkpoint schreiben — ein fehlgeschlagener
	// Checkpoint darf nicht als "erledigt" markiert werden, sonst überspringt
	// der nächste Minutentakt im selben Fenster den nötigen Retry.
	if _, err := s.db.Exec(
		`INSERT INTO system_settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		walCheckpointSettingKey, today,
	); err != nil {
		logIfBusy(err, "walCheckpoint.markDone")
		slog.Error("scheduler wal checkpoint: marker write failed", "error", err)
		return
	}

	// busy=1 heißt: nicht alle Frames konnten checkpointed werden (z. B. ein
	// noch offener Reader) — TRUNCATE bricht dann nicht ab, verkürzt die WAL
	// aber nur soweit möglich. Wird mitgeloggt, ist aber kein Fehler.
	slog.Info("scheduler wal checkpoint done",
		"busy", busy == 1,
		"wal_pages", walPages,
		"checkpointed_pages", checkpointedPages,
		"duration_ms", duration.Milliseconds(),
	)
}
