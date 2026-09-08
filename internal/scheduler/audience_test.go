package scheduler

import (
	"database/sql"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// ── Empfängermenge der Scheduler-Meldungen ───────────────────────────────────
//
// Erinnerungen (24h/3h) und Termin-Hinweise liefen vor dem Change
// `terminmeldungen-erweiterter-kader` über eine eigene Kopie der
// Empfängerauflösung, die nur `player_memberships` kannte: der erweiterte Kader
// bekam keine einzige Erinnerung. Jetzt teilen sie sich `notify.TeamAudience`
// mit games und trainings.

// extendedTeam legt Team, aktive Saison und einen Kader an, in dem ein Spieler
// NUR im erweiterten Kader steht, plus dessen Elternteil.
func extendedTeam(t *testing.T) (db *sql.DB, teamID, extUser, extParent, trainerUser int) {
	t.Helper()
	db = testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID = testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	extUser = testutil.CreateUser(t, db, "standard")
	extMember := testutil.CreateMember(t, db, extUser)
	if _, err := db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?,?)`,
		kaderID, extMember); err != nil {
		t.Fatalf("kader_extended_members: %v", err)
	}

	extParent = testutil.CreateUser(t, db, "standard")
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`,
		extParent, extMember); err != nil {
		t.Fatalf("family_links: %v", err)
	}

	trainerUser = testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMember)

	return db, teamID, extUser, extParent, trainerUser
}

// logCountForUser zählt die Idempotenz-Zeilen eines einzelnen Nutzers. Der Slot
// ist pro Nutzer geclaimt — genau darüber lässt sich prüfen, WER den Reminder
// bekommen hat.
func logCountForUser(t *testing.T, db *sql.DB, userID int, refType string, refID int) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM notification_log WHERE user_id=? AND ref_type=? AND ref_id=?`,
		userID, refType, refID).Scan(&n); err != nil {
		t.Fatalf("count notification_log: %v", err)
	}
	return n
}

func TestGameReminder_ErwKaderErhaeltReminder(t *testing.T) {
	db, teamID, extUser, extParent, trainerUser := extendedTeam(t)
	gameID := createGameAt(t, db, teamID, 2*time.Hour)

	s := New(db, testutil.TestConfig(), nil)
	s.sendGameReminders()
	s.sendGameReminders() // zweiter Lauf darf nicht doppeln

	for _, u := range []struct {
		id    int
		rolle string
	}{{extUser, "erweiterter Kader"}, {extParent, "dessen Elternteil"}, {trainerUser, "Trainer"}} {
		for _, slot := range []string{"game_reminder_24h", "game_reminder_3h"} {
			if got := logCountForUser(t, db, u.id, slot, gameID); got != 1 {
				t.Errorf("%s (user %d): %s = %d Zeilen, want genau 1", u.rolle, u.id, slot, got)
			}
		}
	}
}

func TestTrainingReminder_ErwKaderErhaeltReminder(t *testing.T) {
	db, teamID, extUser, extParent, trainerUser := extendedTeam(t)
	var seasonID int
	db.QueryRow(`SELECT id FROM seasons WHERE is_active=1`).Scan(&seasonID)

	at := time.Now().In(timez.Berlin()).Add(2 * time.Hour)
	res, err := db.Exec(
		`INSERT INTO training_sessions (kader_id, team_id, season_id, date, start_time, end_time, title, status)
		 VALUES ((SELECT id FROM kader WHERE team_id = ? AND season_id = ?), ?, ?, ?, ?, '23:59', 'Einheit', 'active')`,
		teamID, seasonID, teamID, seasonID, at.Format("2006-01-02"), at.Format("15:04"))
	if err != nil {
		t.Fatalf("insert training_session: %v", err)
	}
	id, _ := res.LastInsertId()
	sessionID := int(id)

	s := New(db, testutil.TestConfig(), nil)
	s.sendTrainingReminders()
	s.sendTrainingReminders()

	for _, u := range []struct {
		id    int
		rolle string
	}{{extUser, "erweiterter Kader"}, {extParent, "dessen Elternteil"}, {trainerUser, "Trainer"}} {
		for _, slot := range []string{"training_reminder_24h", "training_reminder_3h"} {
			if got := logCountForUser(t, db, u.id, slot, sessionID); got != 1 {
				t.Errorf("%s (user %d): %s = %d Zeilen, want genau 1", u.rolle, u.id, slot, got)
			}
		}
	}
}

func TestEventNotePush_ErwKaderErhaeltHinweis(t *testing.T) {
	db, teamID, extUser, extParent, trainerUser := extendedTeam(t)
	var seasonID int
	db.QueryRow(`SELECT id FROM seasons WHERE is_active=1`).Scan(&seasonID)
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2099-01-15")

	if _, err := db.Exec(`INSERT INTO pending_event_notes_push
		(ref_type, ref_id, note_text, notify_after, updated_by)
		VALUES ('training', ?, 'Halle gesperrt', datetime('now','-1 minute'), ?)`,
		sessionID, extUser); err != nil {
		t.Fatalf("pending insert: %v", err)
	}

	var recipients []int
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, userIDs []int, _, _, _, _ string, _ ...notify.Option) {
		recipients = userIDs
	}
	t.Cleanup(func() { notify.Send = orig })

	s := New(db, testutil.TestConfig(), nil)
	pushed, err := s.processPendingEventNotes()
	if err != nil {
		t.Fatalf("processPendingEventNotes: %v", err)
	}
	if pushed != 1 {
		t.Fatalf("erwartet 1 Push, got %d", pushed)
	}

	got := map[int]bool{}
	for _, id := range recipients {
		got[id] = true
	}
	if !got[extUser] {
		t.Errorf("erweiterter Kader (user %d) fehlt in %v", extUser, recipients)
	}
	if !got[extParent] {
		t.Errorf("Elternteil des erweiterten Mitglieds (user %d) fehlt in %v", extParent, recipients)
	}
	if !got[trainerUser] {
		t.Errorf("Trainer (user %d) fehlt in %v", trainerUser, recipients)
	}
	if pendingCount(t, s, "training", sessionID) != 0 {
		t.Errorf("pending row muss gelöscht sein")
	}
}
