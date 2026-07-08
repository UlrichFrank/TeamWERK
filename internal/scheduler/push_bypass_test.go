package scheduler

import (
	"database/sql"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/push"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Diese Tests nageln bewusst das AKTUELLE Verhalten fest: Die folgenden
// Scheduler-Jobs rufen push.SendToUsers OHNE push.FilterByPushPref auf und
// stellen daher an Empfänger mit push_enabled=0 dennoch zu.
//
// OFFENE DESIGN-FRAGE (nicht in diesem Change entschieden): Ist dieser
// Preference-Bypass gewollt (operativ wichtige Erinnerungen an Verantwortliche)
// oder ein Bug? Diese Tests schützen nur vor UNBEABSICHTIGTER Änderung — wird
// der Bypass bewusst korrigiert, sind sie entsprechend anzupassen.

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

// TestAttendanceReminder_BypassesPushPref — Trainer mit push_enabled=0 erhält
// die Anwesenheits-Erinnerung trotzdem.
func TestAttendanceReminder_BypassesPushPref(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, _ := makeTrainerInSeason(t, db, teamID, seasonID)
	testutil.CreateTrainingSession(t, db, teamID, seasonID, past(30))
	testutil.CreateNotificationPreference(t, db, trainerUserID, "duty_reminders", false, false)

	pushes := capturePush(t)
	New(db, testutil.TestConfig(), nil).sendAttendanceRemindersAt(at19())

	waitPushContains(t, pushes, trainerUserID)
}

// TestMatchReportReviewReminder_BypassesPushPref — Freigeber mit push_enabled=0
// erhält die Review-Erinnerung trotzdem.
func TestMatchReportReviewReminder_BypassesPushPref(t *testing.T) {
	db := testutil.NewDB(t)
	reviewerID := testutil.CreateMedienUser(t, db)
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
	testutil.CreateNotificationPreference(t, db, reviewerID, "games", false, false)

	pushes := capturePush(t)
	New(db, testutil.TestConfig(), nil).sendMatchReportReviewReminders()

	waitPushContains(t, pushes, reviewerID)
}

// TestVideoRetentionWarning_BypassesPushPref — Trainer mit push_enabled=0 erhält
// die T-7-Löschwarnung trotzdem.
func TestVideoRetentionWarning_BypassesPushPref(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := retentionConfig(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	setSeasonEndDate(t, db, seasonID, -83)
	teamID := testutil.CreateTeam(t, db, "Team A")
	creator := testutil.CreateUser(t, db, "standard")
	trainerUserID, _ := addTrainer(t, db, teamID, seasonID)
	testutil.CreateVideo(t, db, teamID, seasonID, creator, "ready")
	testutil.CreateNotificationPreference(t, db, trainerUserID, "games", false, false)

	pushes := capturePush(t)
	New(db, cfg, nil).runVideoRetention()

	waitPushContains(t, pushes, trainerUserID)
}
