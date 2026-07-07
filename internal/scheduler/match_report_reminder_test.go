package scheduler

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestMatchReportReviewReminder_5DayOldFiresOnce validiert die Kern-Invariante
// aus D-5 (spielbericht-medien-gate): Ein Bericht in 'pending_review' seit
// mehr als 5 Tagen löst *genau eine* Reminder-Notification pro Freigeber aus.
// Zweiter Job-Lauf schreibt nichts nach — notification_log verhindert das.
func TestMatchReportReviewReminder_5DayOldFiresOnce(t *testing.T) {
	db := testutil.NewDB(t)

	// Freigeber-User (medien) + Autor + Spiel + Bericht.
	reviewerID := testutil.CreateMedienUser(t, db)
	authorID := testutil.CreateUser(t, db, "standard")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-07-01")
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)

	// State auf pending_review, submitted_at 6 Tage in der Vergangenheit.
	if _, err := db.Exec(
		`UPDATE match_reports
		 SET state='pending_review',
		     submitted_at=datetime('now','-6 days')
		 WHERE id=?`, reportID); err != nil {
		t.Fatalf("prepare pending_review row: %v", err)
	}

	s := New(db, testutil.TestConfig(), nil)

	// Erster Lauf: Log-Eintrag wird geschrieben.
	s.sendMatchReportReviewReminders()
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM notification_log
		 WHERE ref_type='match_report_review_reminder' AND ref_id=? AND user_id=?`,
		reportID, reviewerID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 notification_log row after first run, got %d", count)
	}

	// Zweiter Lauf: darf nichts nachlegen (Idempotenz).
	s.sendMatchReportReviewReminders()
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM notification_log
		 WHERE ref_type='match_report_review_reminder' AND ref_id=? AND user_id=?`,
		reportID, reviewerID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 notification_log row after second run (idempotent), got %d", count)
	}
}

// TestMatchReportReviewReminder_YoungReportSkipped stellt sicher, dass Berichte
// unter 5 Tagen keinen Reminder auslösen.
func TestMatchReportReviewReminder_YoungReportSkipped(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateMedienUser(t, db)
	authorID := testutil.CreateUser(t, db, "standard")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-07-01")
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)

	// 3 Tage alt.
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=datetime('now','-3 days') WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}

	s := New(db, testutil.TestConfig(), nil)
	s.sendMatchReportReviewReminders()

	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM notification_log WHERE ref_type='match_report_review_reminder'`,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 reminders for 3-day-old report, got %d", count)
	}
}
