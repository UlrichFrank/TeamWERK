package scheduler

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// at19 liefert einen Berliner Zeitpunkt, der das 19:00-Gate passiert (heute 20 Uhr).
func at19() time.Time {
	now := time.Now().In(timez.Berlin())
	return time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, timez.Berlin())
}

// at18 liefert einen Berliner Zeitpunkt vor dem Gate (heute 18 Uhr).
func at18() time.Time {
	now := time.Now().In(timez.Berlin())
	return time.Date(now.Year(), now.Month(), now.Day(), 18, 0, 0, 0, timez.Berlin())
}

// makeTrainerInSeason registriert einen Trainer für (teamID, seasonID) und
// gibt user_id + kader_id zurück.
func makeTrainerInSeason(t *testing.T, db *sql.DB, teamID, seasonID int) (userID, kaderID int) {
	t.Helper()
	userID = testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		memberID, "trainer"); err != nil {
		t.Fatalf("club function: %v", err)
	}
	kaderID = testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, memberID)
	return
}

// past gibt ein YYYY-MM-DD eindeutig vor heute zurück (60 Tage zurück).
func past(daysAgo int) string {
	return time.Now().AddDate(0, 0, -daysAgo).Format("2006-01-02")
}

func TestAttendanceReminders_TrainerWithOpenEvents_GetsLog(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, _ := makeTrainerInSeason(t, db, teamID, seasonID)
	testutil.CreateTrainingSession(t, db, teamID, seasonID, past(30))

	s := New(db, testutil.TestConfig(), nil)
	now := at19()
	s.sendAttendanceRemindersAt(now)

	if got := logCount(t, db, "attendance-reminder", hashDate(now.Format("2006-01-02"))); got != 1 {
		t.Errorf("expected 1 reminder log row for trainer, got %d", got)
	}
	_ = trainerUserID
}

func TestAttendanceReminders_NoOpenEvents_NoLog(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	makeTrainerInSeason(t, db, teamID, seasonID)
	// nur ein zukünftiges Training existiert — soll keinen Reminder triggern
	testutil.CreateTrainingSession(t, db, teamID, seasonID, "2027-01-15")

	s := New(db, testutil.TestConfig(), nil)
	now := at19()
	s.sendAttendanceRemindersAt(now)

	if got := logCount(t, db, "attendance-reminder", hashDate(now.Format("2006-01-02"))); got != 0 {
		t.Errorf("expected 0 reminders (no open events), got %d", got)
	}
}

func TestAttendanceReminders_Idempotent_TwoRunsOneLog(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	makeTrainerInSeason(t, db, teamID, seasonID)
	testutil.CreateTrainingSession(t, db, teamID, seasonID, past(30))

	s := New(db, testutil.TestConfig(), nil)
	now := at19()
	s.sendAttendanceRemindersAt(now)
	s.sendAttendanceRemindersAt(now)

	if got := logCount(t, db, "attendance-reminder", hashDate(now.Format("2006-01-02"))); got != 1 {
		t.Errorf("expected exactly 1 reminder log row after two runs, got %d", got)
	}
}

func TestAttendanceReminders_StopOnceAttendanceRecorded(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	makeTrainerInSeason(t, db, teamID, seasonID)
	tsID := testutil.CreateTrainingSession(t, db, teamID, seasonID, past(30))
	playerMemberID := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(`INSERT INTO training_attendances (training_id, member_id, present) VALUES (?, ?, 1)`,
		tsID, playerMemberID); err != nil {
		t.Fatalf("seed attendance: %v", err)
	}
	if _, err := db.Exec(`UPDATE training_sessions SET attendance_tracked=1 WHERE id=?`, tsID); err != nil {
		t.Fatalf("seed attendance_tracked: %v", err)
	}

	s := New(db, testutil.TestConfig(), nil)
	now := at19()
	s.sendAttendanceRemindersAt(now)

	if got := logCount(t, db, "attendance-reminder", hashDate(now.Format("2006-01-02"))); got != 0 {
		t.Errorf("expected 0 reminders (event already has attendance), got %d", got)
	}
}

func TestAttendanceReminders_NoActiveSeason_Silent(t *testing.T) {
	db := testutil.NewDB(t)
	// keine Saison angelegt
	s := New(db, testutil.TestConfig(), nil)
	now := at19()
	s.sendAttendanceRemindersAt(now) // darf nicht crashen

	var any int
	db.QueryRow(`SELECT COUNT(*) FROM notification_log WHERE ref_type='attendance-reminder'`).Scan(&any)
	if any != 0 {
		t.Errorf("no active season must not produce any reminder, got %d", any)
	}
}

func TestAttendanceReminders_BeforeHourGate_DoesNothing(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	makeTrainerInSeason(t, db, teamID, seasonID)
	testutil.CreateTrainingSession(t, db, teamID, seasonID, past(30))

	s := New(db, testutil.TestConfig(), nil)
	s.sendAttendanceRemindersAt(at18()) // 18:00 — vor dem Gate

	var any int
	db.QueryRow(`SELECT COUNT(*) FROM notification_log WHERE ref_type='attendance-reminder'`).Scan(&any)
	if any != 0 {
		t.Errorf("must not send before 19:00 gate, got %d", any)
	}
}

// ---------- Body-Formatierung ----------

func TestBuildAttendanceReminderBody_TwoEvents_NoSuffix(t *testing.T) {
	events := []openAttendanceEvent{
		{teamName: "D-Jugend", eventDate: "2026-04-14", eventType: "training"},
		{teamName: "D-Jugend", eventDate: "2026-04-18", eventType: "game"},
	}
	body := buildAttendanceReminderBody(events)
	if !strings.HasPrefix(body, "2 offene Erfassungen:") {
		t.Errorf("expected prefix '2 offene Erfassungen:', got %q", body)
	}
	if strings.Contains(body, "weitere") {
		t.Errorf("two events must not have a 'weitere' suffix: %q", body)
	}
	if !strings.Contains(body, "Training") || !strings.Contains(body, "Spiel") {
		t.Errorf("body must mention both labels, got %q", body)
	}
}

func TestBuildAttendanceReminderBody_FiveEvents_HasSuffix(t *testing.T) {
	events := []openAttendanceEvent{
		{teamName: "T1", eventDate: "2026-04-14", eventType: "training"},
		{teamName: "T1", eventDate: "2026-04-15", eventType: "training"},
		{teamName: "T1", eventDate: "2026-04-16", eventType: "training"},
		{teamName: "T1", eventDate: "2026-04-17", eventType: "training"},
		{teamName: "T1", eventDate: "2026-04-18", eventType: "training"},
	}
	body := buildAttendanceReminderBody(events)
	if !strings.Contains(body, "und 2 weitere") {
		t.Errorf("expected 'und 2 weitere' suffix, got %q", body)
	}
}

func TestBuildAttendanceReminderBody_ThreeEvents_ExactlyNoSuffix(t *testing.T) {
	events := []openAttendanceEvent{
		{teamName: "T1", eventDate: "2026-04-14", eventType: "training"},
		{teamName: "T1", eventDate: "2026-04-15", eventType: "training"},
		{teamName: "T1", eventDate: "2026-04-16", eventType: "training"},
	}
	body := buildAttendanceReminderBody(events)
	if strings.Contains(body, "weitere") {
		t.Errorf("three events: no suffix, got %q", body)
	}
}
