package scheduler

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)

// runTrainingDiaryRetention löscht Nachweis-Dateien des Trainingstagebuchs,
// deren Saison vor mehr als 90 Tagen endete.
//
// Bewusst anders als die Video-Retention: **keine T-7-Vorwarnung**. Bei einem
// Video ist die Warnung sinnvoll (großer, unwiederbringlicher Inhalt), bei
// einem Nachweis-Screenshot wäre der Push nur Lärm — das Original liegt
// ohnehin auf dem Handy des Spielers. Stattdessen ein statischer Hinweis im
// UI. Damit entfällt auch die notification_log-Idempotenz.
//
// Gelöscht wird ausschließlich die **Datei**: proof_purged_at wird gesetzt und
// proof_disk_name geNULLt, während Datum, Art, Dauer und RPE dauerhaft
// erhalten bleiben. Der Serve-Endpoint unterscheidet danach 410 („war da,
// wurde bereinigt") von 404 („hatte nie einen Nachweis").
//
// Einträge mit season_id IS NULL fallen durch den JOIN heraus und werden nie
// bereinigt — dieselbe fail-safe-Richtung wie bei den Videos.
//
// Der Vorgang ist von Natur aus idempotent: nach dem UPDATE erfüllt die Zeile
// die WHERE-Bedingung nicht mehr.
//
// Inline-SQL, weil der Scheduler als Foundation das Domain-Package
// trainingdiary nicht importieren darf (Architektur-Test). Das Pfadschema
// <dir>/<disk_name> ist trivial nachgebaut.
func (s *Scheduler) runTrainingDiaryRetention() {
	rows, err := s.db.Query(`
		SELECT e.id, e.proof_disk_name
		FROM training_diary_entries e
		JOIN seasons se ON se.id = e.season_id
		WHERE e.proof_disk_name IS NOT NULL
		  AND e.proof_purged_at IS NULL
		  AND se.end_date IS NOT NULL
		  AND date(se.end_date) < date('now','-90 days')`)
	if err != nil {
		logIfBusy(err, "runTrainingDiaryRetention.query")
		slog.Error("scheduler training diary retention query failed", "error", err)
		return
	}
	type row struct {
		id       int
		diskName string
	}
	var due []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.diskName); err != nil {
			continue
		}
		due = append(due, r)
	}
	rows.Close()

	purged := 0
	for _, r := range due {
		// Erst die Datei, dann der Marker: bricht der Lauf dazwischen ab,
		// zieht der nächste die Zeile sauber nach (die Datei fehlt dann
		// bereits, was hier ausdrücklich kein Fehler ist).
		if r.diskName != "" {
			path := filepath.Join(s.cfg.TrainingDiaryDir, r.diskName)
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				slog.Warn("training diary retention: remove proof failed",
					"entry_id", r.id, "error", err)
			}
		}
		if _, err := s.db.Exec(`
			UPDATE training_diary_entries
			   SET proof_purged_at = CURRENT_TIMESTAMP,
			       proof_disk_name = NULL
			 WHERE id = ?`, r.id); err != nil {
			logIfBusy(err, "runTrainingDiaryRetention.update")
			slog.Error("scheduler training diary retention update failed",
				"entry_id", r.id, "error", err)
			continue
		}
		purged++
	}
	if purged > 0 {
		slog.Info("scheduler training diary retention: proofs purged", "count", purged)
	}
}
