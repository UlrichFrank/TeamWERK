package scheduler

import (
	"log/slog"

	"github.com/teamstuttgart/teamwerk/internal/eventlog"
)

// purgeEventLog löscht abgelaufene Event-Log-Zeilen (design.md Decision 5):
// gesehen + 3 Tage (die fachliche Regel) und ungesehen + 90 Tage (Betriebs-
// Sicherung gegen unbegrenztes Wachstum durch Accounts, die nie wieder
// einloggen). Die eigentliche DELETE-Logik lebt in eventlog.Purge — der
// Scheduler ruft sie nur auf und loggt die Trefferzahl.
//
// Ohne Idempotenzschutz: ein DELETE ist von Natur aus wiederholbar, ein
// doppelter Lauf löscht beim zweiten Mal schlicht nichts mehr.
func (s *Scheduler) purgeEventLog() {
	n, err := eventlog.Purge(s.db)
	if err != nil {
		logIfBusy(err, "purgeEventLog")
		slog.Error("scheduler event log purge failed", "error", err)
		return
	}
	if n > 0 {
		slog.Info("scheduler event log purged", "count", n)
	}
}
