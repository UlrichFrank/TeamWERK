package trainings_test

// rsvp-aenderung-trainer-push: eine bestehende Spieler-RSVP, deren Status sich
// innerhalb von 7 Tagen vor Trainingsbeginn ändert, erzeugt eine
// `operativ`-Meldung an die Trainer des Kaders der Einheit. Die Einheiten der
// Fixtures beginnen am 2026-06-15 um 18:00 Europe/Berlin.

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

type rsvpNotice struct {
	userIDs  []int
	category string
	body     string
	url      string
}

// captureRSVPNotices fängt alle notify.Send-Aufrufe ab. SendAsync ruft Send in
// einer Goroutine — deshalb der Kanal.
func captureRSVPNotices(t *testing.T) chan rsvpNotice {
	t.Helper()
	ch := make(chan rsvpNotice, 16)
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, uids []int, category, _, body, url string, _ ...notify.Option) {
		ch <- rsvpNotice{uids, category, body, url}
	}
	t.Cleanup(func() { notify.Send = orig })
	return ch
}

func expectRSVPNotice(t *testing.T, ch chan rsvpNotice) rsvpNotice {
	t.Helper()
	select {
	case n := <-ch:
		return n
	case <-time.After(2 * time.Second):
		t.Fatal("keine Trainer-Meldung erhalten")
	}
	return rsvpNotice{}
}

func expectNoRSVPNotice(t *testing.T, ch chan rsvpNotice) {
	t.Helper()
	select {
	case n := <-ch:
		t.Fatalf("unerwartete Meldung: %+v", n)
	case <-time.After(200 * time.Millisecond):
	}
}

type rsvpPushFixture struct {
	db                    *sql.DB
	sessionID, kaderID    int
	uPlayer, mPlayer      int
	uTrainer, mTrainer    int
	uTrainer2             int
	playerTok, trainerTok string
}

// newRSVPPushFixture baut eine Einheit mit einem Spieler und zwei Trainern im
// Kader der Einheit. sessionID/kaderID kommen vom Aufrufer, damit Mannschaft
// und Übungsgruppe dieselbe Fixture nutzen.
func newRSVPPushFixture(t *testing.T, db *sql.DB, sessionID int) rsvpPushFixture {
	t.Helper()
	f := rsvpPushFixture{db: db, sessionID: sessionID}
	if err := db.QueryRow(`SELECT kader_id FROM training_sessions WHERE id=?`, sessionID).Scan(&f.kaderID); err != nil {
		t.Fatalf("kader_id: %v", err)
	}
	f.uPlayer = testutil.CreateUser(t, db, "standard")
	f.mPlayer = testutil.CreateMember(t, db, f.uPlayer)
	addKaderMember(t, db, f.kaderID, f.mPlayer)
	f.uTrainer = testutil.CreateUser(t, db, "standard")
	f.mTrainer = testutil.CreateMember(t, db, f.uTrainer)
	f.uTrainer2 = testutil.CreateUser(t, db, "standard")
	for _, m := range []int{f.mTrainer, testutil.CreateMember(t, db, f.uTrainer2)} {
		if _, err := db.Exec(`INSERT INTO kader_trainers (kader_id, member_id) VALUES (?, ?)`, f.kaderID, m); err != nil {
			t.Fatalf("kader_trainers: %v", err)
		}
	}
	f.playerTok = testutil.Token(t, f.uPlayer, "standard", []string{"spieler"})
	f.trainerTok = testutil.Token(t, f.uTrainer, "standard", []string{"trainer"})
	return f
}

func newTeamRSVPPushFixture(t *testing.T) rsvpPushFixture {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	return newRSVPPushFixture(t, db, testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-06-15"))
}

func (f rsvpPushFixture) addResponse(t *testing.T, memberID, byUser int, status string) {
	t.Helper()
	if _, err := f.db.Exec(`INSERT INTO training_responses (training_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, ?, '', CURRENT_TIMESTAMP)`, f.sessionID, memberID, byUser, status); err != nil {
		t.Fatalf("training_responses: %v", err)
	}
}

func (f rsvpPushFixture) respond(t *testing.T, now, token string, body map[string]any) int {
	t.Helper()
	h := trainings.NewHandler(f.db, testutil.TestConfig(), hub.NewHub())
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", now)))
	srv := testServer(t, h)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", f.sessionID), token, body)
	res.Body.Close()
	return res.StatusCode
}

func sameUserIDs(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[int]int{}
	for _, id := range got {
		seen[id]++
	}
	for _, id := range want {
		if seen[id] == 0 {
			return false
		}
		seen[id]--
	}
	return true
}

func TestRespond_AenderungImFenster_BenachrichtigtTrainer(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	f.addResponse(t, f.mPlayer, f.uPlayer, "declined")
	ch := captureRSVPNotices(t)

	if st := f.respond(t, "2026-06-13 18:00", f.playerTok, map[string]any{"status": "confirmed"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	n := expectRSVPNotice(t, ch)
	if n.category != "operativ" {
		t.Errorf("category %q, want operativ", n.category)
	}
	if !sameUserIDs(n.userIDs, []int{f.uTrainer, f.uTrainer2}) {
		t.Errorf("Empfänger %v, want %v", n.userIDs, []int{f.uTrainer, f.uTrainer2})
	}
	if !strings.Contains(n.body, "von Absage auf Zusage") || !strings.Contains(n.body, "15.06.2026 um 18:00 Uhr") {
		t.Errorf("body %q", n.body)
	}
	if n.url != fmt.Sprintf("/termine?focus=training-%d", f.sessionID) {
		t.Errorf("url %q", n.url)
	}
}

func TestRespond_Uebungsgruppe_BenachrichtigtTrainer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	f := newRSVPPushFixture(t, db, createPracticeSession(t, db, groupID, seasonID, "2026-06-15"))
	f.addResponse(t, f.mPlayer, f.uPlayer, "confirmed")
	ch := captureRSVPNotices(t)

	if st := f.respond(t, "2026-06-10 12:00", f.playerTok, map[string]any{"status": "declined"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	n := expectRSVPNotice(t, ch)
	if !sameUserIDs(n.userIDs, []int{f.uTrainer, f.uTrainer2}) {
		t.Errorf("Empfänger %v, want %v", n.userIDs, []int{f.uTrainer, f.uTrainer2})
	}
}

func TestRespond_ErsteAntwort_KeineMeldung(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	ch := captureRSVPNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.playerTok, map[string]any{"status": "declined"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectNoRSVPNotice(t, ch)
}

func TestRespond_GleicherStatus_KeineMeldung(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	f.addResponse(t, f.mPlayer, f.uPlayer, "maybe")
	ch := captureRSVPNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.playerTok, map[string]any{"status": "maybe", "reason": "neu"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectNoRSVPNotice(t, ch)
}

func TestRespond_AchtTageVorher_KeineMeldung(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	f.addResponse(t, f.mPlayer, f.uPlayer, "confirmed")
	ch := captureRSVPNotices(t)
	if st := f.respond(t, "2026-06-07 18:00", f.playerTok, map[string]any{"status": "declined"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectNoRSVPNotice(t, ch)
}

func TestRespond_GenauSiebenTage_Meldung(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	f.addResponse(t, f.mPlayer, f.uPlayer, "confirmed")
	ch := captureRSVPNotices(t)
	if st := f.respond(t, "2026-06-08 18:00", f.playerTok, map[string]any{"status": "declined"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectRSVPNotice(t, ch)
}

func TestRespond_TrainerEigeneAntwort_KeineMeldung(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	f.addResponse(t, f.mTrainer, f.uTrainer, "confirmed")
	ch := captureRSVPNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.trainerTok, map[string]any{"status": "declined"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectNoRSVPNotice(t, ch)
}

func TestRespond_TrainerFuerSpieler_AusloeserNichtEmpfaenger(t *testing.T) {
	f := newTeamRSVPPushFixture(t)
	f.addResponse(t, f.mPlayer, f.uPlayer, "confirmed")
	ch := captureRSVPNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.trainerTok, map[string]any{"status": "declined", "member_id": f.mPlayer}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	n := expectRSVPNotice(t, ch)
	if !sameUserIDs(n.userIDs, []int{f.uTrainer2}) {
		t.Errorf("Empfänger %v, want nur %d", n.userIDs, f.uTrainer2)
	}
}

func TestRespond_SeriesUnavailable_KeineMeldung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	creator := testutil.CreateUser(t, db, "standard")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, creator)
	f := newRSVPPushFixture(t, db, testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, "2026-06-15"))
	f.addResponse(t, f.mPlayer, f.uPlayer, "confirmed")
	if _, err := db.Exec(`INSERT INTO member_series_unavailabilities (member_id, training_series_id, created_by)
		VALUES (?, ?, ?)`, f.mPlayer, seriesID, creator); err != nil {
		t.Fatal(err)
	}
	ch := captureRSVPNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.playerTok, map[string]any{"status": "declined"}); st != http.StatusForbidden {
		t.Fatalf("status %d, want 403", st)
	}
	expectNoRSVPNotice(t, ch)
}
