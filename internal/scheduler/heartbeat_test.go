package scheduler

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Ein erfolgreicher Scheduler-Lauf MUSS den Monitoring-Heartbeat aktualisieren —
// die Datenquelle für den externen Dead-Man-Switch.
func TestScheduler_HeartbeatRecorded(t *testing.T) {
	db := testutil.NewDB(t)

	// Vorher existiert kein Heartbeat.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM monitoring_heartbeat`).Scan(&count); err != nil {
		t.Fatalf("count before: %v", err)
	}
	if count != 0 {
		t.Fatalf("heartbeat row count before run = %d, want 0", count)
	}

	// Leeres DB ⇒ keine Reminder ⇒ Mailer (nil) wird nicht aufgerufen.
	New(db, testutil.TestConfig(), nil).Run()

	var updatedAt string
	if err := db.QueryRow(`SELECT updated_at FROM monitoring_heartbeat WHERE id = 1`).Scan(&updatedAt); err != nil {
		t.Fatalf("heartbeat not written: %v", err)
	}
	if updatedAt == "" {
		t.Fatal("heartbeat updated_at is empty after run")
	}
}
