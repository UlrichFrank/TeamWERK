package scheduler

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type sentMsg struct {
	category, title, body, url string
	uids                       []int
}

// captureSend ersetzt notify.Send für die Dauer des Tests.
func captureSend(t *testing.T) *[]sentMsg {
	t.Helper()
	var sent []sentMsg
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, uids []int, category, title, body, url string, _ ...notify.Option) {
		sent = append(sent, sentMsg{category, title, body, url, uids})
	}
	t.Cleanup(func() { notify.Send = orig })
	return &sent
}

// meetingGame legt ein Spiel (Anwurf 18:00) mit einem Kader-Spieler als
// Empfänger an und eine fällige pending-Zeile mit dem gegebenen Ausgangsstand.
func meetingGame(t *testing.T, date string, prevOffset any, prevPlace string) (*Scheduler, *sql.DB, int) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Team A")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, date)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	if _, err := db.Exec(`INSERT INTO pending_game_meeting_push (game_id, prev_offset, prev_place, notify_after)
		VALUES (?, ?, ?, datetime('now','-1 minute'))`, gameID, prevOffset, prevPlace); err != nil {
		t.Fatalf("pending insert: %v", err)
	}
	return New(db, testutil.TestConfig(), nil), db, gameID
}

func pendingMeetingRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM pending_game_meeting_push`).Scan(&n)
	return n
}

func TestMeetingPush_MehrfachKorrekturEineMeldung(t *testing.T) {
	sent := captureSend(t)
	s, db, gameID := meetingGame(t, "2099-01-15", nil, "")
	// Endstand nach mehreren Korrekturen: 16:45 am Parkplatz.
	db.Exec(`UPDATE games SET meet_offset_minutes=75, meet_place='Parkplatz' WHERE id=?`, gameID)

	if n, err := s.processPendingGameMeetings(); err != nil || n != 1 {
		t.Fatalf("pushed=%d err=%v, want 1", n, err)
	}
	if len(*sent) != 1 {
		t.Fatalf("expected 1 message, got %d", len(*sent))
	}
	m := (*sent)[0]
	if m.category != "games" || m.title != "Treffzeit" || len(m.uids) == 0 {
		t.Errorf("unexpected message %+v", m)
	}
	if !strings.Contains(m.body, "Treffen 16:45 Uhr, Parkplatz") || !strings.Contains(m.body, "15.01.2099") {
		t.Errorf("body = %q", m.body)
	}
	if !strings.HasSuffix(m.url, "/termine?focus=game-"+itoa(gameID)) {
		t.Errorf("url = %q", m.url)
	}
	if pendingMeetingRows(t, db) != 0 {
		t.Error("pending row should be deleted")
	}
}

func TestMeetingPush_NettoUnveraendert(t *testing.T) {
	sent := captureSend(t)
	s, db, gameID := meetingGame(t, "2099-01-15", 90, "Halle")
	db.Exec(`UPDATE games SET meet_offset_minutes=90, meet_place='Halle' WHERE id=?`, gameID)

	if n, _ := s.processPendingGameMeetings(); n != 0 || len(*sent) != 0 {
		t.Fatalf("expected no message, got %d / %v", n, *sent)
	}
	if pendingMeetingRows(t, db) != 0 {
		t.Error("pending row should be deleted")
	}
}

func TestMeetingPush_Entfernt(t *testing.T) {
	sent := captureSend(t)
	s, _, _ := meetingGame(t, "2099-01-15", 90, "Halle")

	if n, _ := s.processPendingGameMeetings(); n != 1 {
		t.Fatalf("expected 1 message, got %d", n)
	}
	m := (*sent)[0]
	if m.title != "Treffzeit entfällt" || !strings.Contains(m.body, "entfällt") {
		t.Errorf("unexpected message %+v", m)
	}
}

func TestMeetingPush_VergangenesSpiel(t *testing.T) {
	sent := captureSend(t)
	s, db, gameID := meetingGame(t, "2000-01-15", nil, "")
	db.Exec(`UPDATE games SET meet_offset_minutes=90 WHERE id=?`, gameID)

	if n, _ := s.processPendingGameMeetings(); n != 0 || len(*sent) != 0 {
		t.Fatalf("expected no message for past game")
	}
	if pendingMeetingRows(t, db) != 0 {
		t.Error("pending row should be deleted even for past game")
	}
}

func TestMeetingPush_NochNichtFaellig(t *testing.T) {
	sent := captureSend(t)
	s, db, gameID := meetingGame(t, "2099-01-15", nil, "")
	db.Exec(`UPDATE games SET meet_offset_minutes=90 WHERE id=?`, gameID)
	db.Exec(`UPDATE pending_game_meeting_push SET notify_after=datetime('now','+5 minutes')`)

	if n, _ := s.processPendingGameMeetings(); n != 0 || len(*sent) != 0 {
		t.Fatalf("expected no message before notify_after")
	}
	if pendingMeetingRows(t, db) != 1 {
		t.Error("pending row must survive until due")
	}
}

func TestMeetingPhrase(t *testing.T) {
	cases := []struct {
		date, start string
		off         sql.NullInt64
		place, want string
	}{
		{"2026-10-11", "15:00", sql.NullInt64{Int64: 90, Valid: true}, "Bus", "Treffen 13:30 Uhr, Bus"},
		{"2026-10-11T00:00:00Z", "06:00", sql.NullInt64{Int64: 420, Valid: true}, "", "Treffen 23:00 Uhr (Vortag)"},
		{"2026-10-11", "15:00", sql.NullInt64{}, "Bus", ""},
	}
	for _, c := range cases {
		if got := meetingPhrase(c.date, c.start, c.off, c.place); got != c.want {
			t.Errorf("meetingPhrase(%s %s) = %q, want %q", c.date, c.start, got, c.want)
		}
	}
}

// Die Spielerinnerung nennt eine gesetzte Treffzeit samt Ort; ausgelöst wird
// weiterhin über den Anwurf (spiel-treffpunkt / push-reminders).
func TestGameReminder_NenntTreffzeit(t *testing.T) {
	sent := captureSend(t)
	db := testutil.NewDB(t)
	teamID, _ := setupTeamPlayer(t, db)
	gameID := createGameAt(t, db, teamID, 20*time.Hour)
	db.Exec(`UPDATE games SET meet_offset_minutes=90, meet_place='Parkplatz Vereinsheim' WHERE id=?`, gameID)
	var date, start string
	db.QueryRow(`SELECT date, time FROM games WHERE id=?`, gameID).Scan(&date, &start)
	want := meetingPhrase(date, start, sql.NullInt64{Int64: 90, Valid: true}, "Parkplatz Vereinsheim")

	New(db, testutil.TestConfig(), nil).sendGameReminders()

	if len(*sent) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(*sent))
	}
	body := (*sent)[0].body
	if !strings.Contains(body, start+" Uhr · "+want) {
		t.Errorf("body = %q, want it to contain %q", body, start+" Uhr · "+want)
	}
}

func TestGameReminder_OhneTreffzeitKeinTreffzeitTeil(t *testing.T) {
	sent := captureSend(t)
	db := testutil.NewDB(t)
	teamID, _ := setupTeamPlayer(t, db)
	createGameAt(t, db, teamID, 20*time.Hour)

	New(db, testutil.TestConfig(), nil).sendGameReminders()

	if len(*sent) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(*sent))
	}
	if strings.Contains((*sent)[0].body, "Treffen") {
		t.Errorf("body = %q, must not mention Treffen", (*sent)[0].body)
	}
}
