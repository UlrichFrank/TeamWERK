package scheduler

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func pendingCount(t *testing.T, s *Scheduler, refType string, refID int) int {
	t.Helper()
	var n int
	s.db.QueryRow(`SELECT COUNT(*) FROM pending_event_notes_push WHERE ref_type=? AND ref_id=?`,
		refType, refID).Scan(&n)
	return n
}

func TestEventNotesPush_FutureEvent_SendsPush_DeletesRow(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2099-01-15")

	// Empfänger: Spieler im aktiven Kader des Teams.
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("kader_members: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO pending_event_notes_push
		(ref_type, ref_id, note_text, notify_after, updated_by)
		VALUES ('training', ?, 'Halle gesperrt', datetime('now','-1 minute'), ?)`,
		sessionID, userID); err != nil {
		t.Fatalf("pending insert: %v", err)
	}

	s := New(db, testutil.TestConfig(), nil)
	pushed, err := s.processPendingEventNotes()
	if err != nil {
		t.Fatalf("processPendingEventNotes: %v", err)
	}
	if pushed != 1 {
		t.Errorf("expected 1 push for future event with recipient, got %d", pushed)
	}
	if pendingCount(t, s, "training", sessionID) != 0 {
		t.Errorf("pending row should be deleted")
	}
}

func TestEventNotesPush_PastEvent_SkipsPush_DeletesRow(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2000-01-15")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("kader_members: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO pending_event_notes_push
		(ref_type, ref_id, note_text, notify_after, updated_by)
		VALUES ('training', ?, 'zu spät', datetime('now','-1 minute'), ?)`,
		sessionID, userID); err != nil {
		t.Fatalf("pending insert: %v", err)
	}

	s := New(db, testutil.TestConfig(), nil)
	pushed, err := s.processPendingEventNotes()
	if err != nil {
		t.Fatalf("processPendingEventNotes: %v", err)
	}
	if pushed != 0 {
		t.Errorf("expected 0 push for past event, got %d", pushed)
	}
	if pendingCount(t, s, "training", sessionID) != 0 {
		t.Errorf("pending row should be deleted even for past event")
	}
}

func TestEventNotesPush_NotYetDue_KeepsRow(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2099-01-15")

	if _, err := db.Exec(`INSERT INTO pending_event_notes_push
		(ref_type, ref_id, note_text, notify_after, updated_by)
		VALUES ('training', ?, 'noch nicht fällig', datetime('now','+5 minutes'), NULL)`,
		sessionID); err != nil {
		t.Fatalf("pending insert: %v", err)
	}

	s := New(db, testutil.TestConfig(), nil)
	pushed, err := s.processPendingEventNotes()
	if err != nil {
		t.Fatalf("processPendingEventNotes: %v", err)
	}
	if pushed != 0 {
		t.Errorf("expected 0 push when not due, got %d", pushed)
	}
	if pendingCount(t, s, "training", sessionID) != 1 {
		t.Errorf("not-yet-due pending row must remain")
	}
}

func TestEventNotesPush_DeletedEvent_DeletesRow_NoPush(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	// ref_id verweist auf ein nicht (mehr) existierendes Training.
	if _, err := db.Exec(`INSERT INTO pending_event_notes_push
		(ref_type, ref_id, note_text, notify_after, updated_by)
		VALUES ('training', 99999, 'verwaist', datetime('now','-1 minute'), NULL)`); err != nil {
		t.Fatalf("pending insert: %v", err)
	}

	s := New(db, testutil.TestConfig(), nil)
	pushed, err := s.processPendingEventNotes()
	if err != nil {
		t.Fatalf("processPendingEventNotes: %v", err)
	}
	if pushed != 0 {
		t.Errorf("expected 0 push for deleted event, got %d", pushed)
	}
	if pendingCount(t, s, "training", 99999) != 0 {
		t.Errorf("orphan pending row should be deleted")
	}
}
