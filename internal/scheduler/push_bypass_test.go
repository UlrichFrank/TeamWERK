package scheduler

import (
	"database/sql"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/push"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// capturePush ersetzt den push.SendToUsers-Seam durch einen Recorder der
// Empfänger-Listen und stellt das Original per Cleanup wieder her.
func capturePush(t *testing.T) chan []int {
	t.Helper()
	ch := make(chan []int, 16)
	orig := push.SendToUsers
	push.SendToUsers = func(_ *sql.DB, _ *appconfig.Config, uids []int, _, _, _ string) {
		ch <- uids
	}
	t.Cleanup(func() { push.SendToUsers = orig })
	return ch
}

func waitPushContains(t *testing.T, ch chan []int, uid int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case uids := <-ch:
			for _, u := range uids {
				if u == uid {
					return
				}
			}
		case <-deadline:
			t.Fatalf("push an user %d nicht erhalten", uid)
		}
	}
}

func assertNoPush(t *testing.T, ch chan []int, uid int, within time.Duration) {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case uids := <-ch:
			for _, u := range uids {
				if u == uid {
					t.Fatalf("unerwarteter Push an user %d (Opt-out sollte greifen)", uid)
				}
			}
		case <-deadline:
			return
		}
	}
}

// --- attendance-reminder: respektiert jetzt 'operativ' ---

func TestAttendanceReminder_RespectsOperativOptOut(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, _ := makeTrainerInSeason(t, db, teamID, seasonID)
	testutil.CreateTrainingSession(t, db, teamID, seasonID, past(30))
	testutil.CreateNotificationPreference(t, db, trainerUserID, "operativ", false, false)

	pushes := capturePush(t)
	New(db, testutil.TestConfig(), nil).sendAttendanceRemindersAt(at19())

	assertNoPush(t, pushes, trainerUserID, 300*time.Millisecond)
}

func TestAttendanceReminder_DefaultSends(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, _ := makeTrainerInSeason(t, db, teamID, seasonID)
	testutil.CreateTrainingSession(t, db, teamID, seasonID, past(30))

	pushes := capturePush(t)
	New(db, testutil.TestConfig(), nil).sendAttendanceRemindersAt(at19())

	waitPushContains(t, pushes, trainerUserID)
}

// --- match-report-review-reminder: respektiert jetzt 'operativ' ---

func setupPendingReview(t *testing.T, db *sql.DB) (reviewerID int) {
	t.Helper()
	reviewerID = testutil.CreateMedienUser(t, db)
	authorID := testutil.CreateUser(t, db, "standard")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-07-01")
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=datetime('now','-6 days') WHERE id=?`,
		reportID); err != nil {
		t.Fatalf("prepare pending_review: %v", err)
	}
	return reviewerID
}

func TestMatchReportReviewReminder_RespectsOperativOptOut(t *testing.T) {
	db := testutil.NewDB(t)
	reviewerID := setupPendingReview(t, db)
	testutil.CreateNotificationPreference(t, db, reviewerID, "operativ", false, false)

	pushes := capturePush(t)
	New(db, testutil.TestConfig(), nil).sendMatchReportReviewReminders()

	assertNoPush(t, pushes, reviewerID, 300*time.Millisecond)
}

func TestMatchReportReviewReminder_DefaultSends(t *testing.T) {
	db := testutil.NewDB(t)
	reviewerID := setupPendingReview(t, db)

	pushes := capturePush(t)
	New(db, testutil.TestConfig(), nil).sendMatchReportReviewReminders()

	waitPushContains(t, pushes, reviewerID)
}

// --- video-retention-warning: BEWUSST harter Bypass (Datenverlust-Warnung) ---

// TestVideoRetentionWarning_BypassesPushPref nagelt fest, dass die T-7-Lösch-
// warnung UNABHÄNGIG von Push-Präferenzen zustellt — auch wenn der Trainer
// 'sonstiges' (oder alles andere) deaktiviert hat. Das ist bewusst (Datenverlust).
func TestVideoRetentionWarning_BypassesPushPref(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := retentionConfig(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	setSeasonEndDate(t, db, seasonID, -83)
	teamID := testutil.CreateTeam(t, db, "Team A")
	creator := testutil.CreateUser(t, db, "standard")
	trainerUserID, _ := addTrainer(t, db, teamID, seasonID)
	testutil.CreateVideo(t, db, teamID, seasonID, creator, "ready")
	testutil.CreateNotificationPreference(t, db, trainerUserID, "sonstiges", false, false)

	pushes := capturePush(t)
	New(db, cfg, nil).runVideoRetention()

	waitPushContains(t, pushes, trainerUserID)
}
