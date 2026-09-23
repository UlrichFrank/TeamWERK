package games_test

// rsvp-aenderung-trainer-push: eine bestehende Spieler-RSVP, deren Status sich
// innerhalb von 7 Tagen vor Spielbeginn ändert, erzeugt eine `operativ`-Meldung
// an die Trainer der Spiel-Kader. Das Spiel aus setupCutoffGame beginnt am
// 2026-06-15 um 18:00 Europe/Berlin.

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type sentNotice struct {
	userIDs  []int
	category string
	title    string
	body     string
	url      string
}

// captureNotices fängt alle notify.Send-Aufrufe ab. SendAsync ruft Send in
// einer Goroutine — deshalb der Kanal statt eines Slices.
func captureNotices(t *testing.T) chan sentNotice {
	t.Helper()
	ch := make(chan sentNotice, 16)
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, uids []int, category, title, body, url string, _ ...notify.Option) {
		ch <- sentNotice{uids, category, title, body, url}
	}
	t.Cleanup(func() { notify.Send = orig })
	return ch
}

func expectNotice(t *testing.T, ch chan sentNotice) sentNotice {
	t.Helper()
	select {
	case n := <-ch:
		return n
	case <-time.After(2 * time.Second):
		t.Fatal("keine Trainer-Meldung erhalten")
	}
	return sentNotice{}
}

func expectNoNotice(t *testing.T, ch chan sentNotice) {
	t.Helper()
	select {
	case n := <-ch:
		t.Fatalf("unerwartete Meldung: %+v", n)
	case <-time.After(200 * time.Millisecond):
	}
}

func addGameResponse(t *testing.T, db *sql.DB, gameID, memberID, byUser int, status string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at)
		VALUES (?, ?, ?, ?, '', datetime('now'))`, gameID, memberID, byUser, status); err != nil {
		t.Fatalf("game_responses: %v", err)
	}
}

func addKaderTrainerRow(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO kader_trainers (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("kader_trainers: %v", err)
	}
}

type rsvpPushFixture struct {
	db                    *sql.DB
	gameID, kaderID       int
	uPlayer, mPlayer      int
	uTrainer, mTrainer    int
	uTrainer2             int
	playerTok, trainerTok string
	respondURL            string
}

func newRSVPPushFixture(t *testing.T) rsvpPushFixture {
	t.Helper()
	db, gameID, _, _, kaderID := setupCutoffGame(t)
	f := rsvpPushFixture{db: db, gameID: gameID, kaderID: kaderID}
	f.uPlayer = testutil.CreateUser(t, db, "standard")
	f.mPlayer = testutil.CreateMember(t, db, f.uPlayer)
	addKaderMember(t, db, kaderID, f.mPlayer)
	f.uTrainer = testutil.CreateUser(t, db, "standard")
	f.mTrainer = testutil.CreateMember(t, db, f.uTrainer)
	addKaderTrainerRow(t, db, kaderID, f.mTrainer)
	addClubFunction(t, db, f.mTrainer, "trainer") // Termin-Sichtbarkeit (UserCanSeeGame)
	f.uTrainer2 = testutil.CreateUser(t, db, "standard")
	addKaderTrainerRow(t, db, kaderID, testutil.CreateMember(t, db, f.uTrainer2))
	f.playerTok = testutil.Token(t, f.uPlayer, "standard", []string{"spieler"})
	f.trainerTok = testutil.Token(t, f.uTrainer, "standard", []string{"trainer"})
	f.respondURL = fmt.Sprintf("/api/games/%d/respond", gameID)
	return f
}

func (f rsvpPushFixture) respond(t *testing.T, now, token string, body map[string]any) int {
	t.Helper()
	srv := cutoffServer(t, newGamesHandler(t, f.db, berlinTime(t, now)))
	res := testutil.Post(t, srv, f.respondURL, token, body)
	res.Body.Close()
	return res.StatusCode
}

func TestRespondToGame_AenderungImFenster_BenachrichtigtTrainer(t *testing.T) {
	f := newRSVPPushFixture(t)
	addGameResponse(t, f.db, f.gameID, f.mPlayer, f.uPlayer, "confirmed")
	ch := captureNotices(t)

	if st := f.respond(t, "2026-06-12 18:00", f.playerTok, map[string]any{"status": "declined", "reason": "krank"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	n := expectNotice(t, ch)
	if n.category != "operativ" {
		t.Errorf("category %q, want operativ", n.category)
	}
	if !sameIDs(n.userIDs, []int{f.uTrainer, f.uTrainer2}) {
		t.Errorf("Empfänger %v, want Trainer %v", n.userIDs, []int{f.uTrainer, f.uTrainer2})
	}
	for _, want := range []string{"von Zusage auf Absage", "15.06.2026", "Grund: krank"} {
		if !strings.Contains(n.body, want) {
			t.Errorf("body %q enthält %q nicht", n.body, want)
		}
	}
	if n.url != fmt.Sprintf("/termine?focus=game-%d", f.gameID) {
		t.Errorf("url %q", n.url)
	}
	expectNoNotice(t, ch)
}

func TestRespondToGame_ErsteAntwort_KeineMeldung(t *testing.T) {
	f := newRSVPPushFixture(t)
	ch := captureNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.playerTok, map[string]any{"status": "declined"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectNoNotice(t, ch)
}

func TestRespondToGame_GleicherStatus_KeineMeldung(t *testing.T) {
	f := newRSVPPushFixture(t)
	addGameResponse(t, f.db, f.gameID, f.mPlayer, f.uPlayer, "declined")
	ch := captureNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.playerTok, map[string]any{"status": "declined", "reason": "doch Urlaub"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectNoNotice(t, ch)
}

func TestRespondToGame_AchtTageVorher_KeineMeldung(t *testing.T) {
	f := newRSVPPushFixture(t)
	addGameResponse(t, f.db, f.gameID, f.mPlayer, f.uPlayer, "confirmed")
	ch := captureNotices(t)
	if st := f.respond(t, "2026-06-07 18:00", f.playerTok, map[string]any{"status": "declined"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectNoNotice(t, ch)
}

func TestRespondToGame_GenauSiebenTage_Meldung(t *testing.T) {
	f := newRSVPPushFixture(t)
	addGameResponse(t, f.db, f.gameID, f.mPlayer, f.uPlayer, "confirmed")
	ch := captureNotices(t)
	if st := f.respond(t, "2026-06-08 18:00", f.playerTok, map[string]any{"status": "maybe"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectNotice(t, ch)
}

func TestRespondToGame_TrainerEigeneAntwort_KeineMeldung(t *testing.T) {
	f := newRSVPPushFixture(t)
	addGameResponse(t, f.db, f.gameID, f.mTrainer, f.uTrainer, "confirmed")
	ch := captureNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.trainerTok, map[string]any{"status": "declined"}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	expectNoNotice(t, ch)
}

func TestRespondToGame_TrainerFuerSpieler_AusloeserNichtEmpfaenger(t *testing.T) {
	f := newRSVPPushFixture(t)
	addGameResponse(t, f.db, f.gameID, f.mPlayer, f.uPlayer, "confirmed")
	ch := captureNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.trainerTok, map[string]any{"status": "declined", "member_id": f.mPlayer}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	n := expectNotice(t, ch)
	if !sameIDs(n.userIDs, []int{f.uTrainer2}) {
		t.Errorf("Empfänger %v, want nur den anderen Trainer %d", n.userIDs, f.uTrainer2)
	}
}

func TestRespondToGame_Elternteil_NameDesKindes(t *testing.T) {
	f := newRSVPPushFixture(t)
	uParent := testutil.CreateUser(t, f.db, "standard")
	if _, err := f.db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, uParent, f.mPlayer); err != nil {
		t.Fatal(err)
	}
	addGameResponse(t, f.db, f.gameID, f.mPlayer, uParent, "confirmed")
	var childName string
	f.db.QueryRow(`SELECT first_name || ' ' || last_name FROM members WHERE id=?`, f.mPlayer).Scan(&childName)
	ch := captureNotices(t)

	tok := testutil.TokenWithIsParent(t, uParent, "standard", nil, true)
	if st := f.respond(t, "2026-06-13 18:00", tok, map[string]any{"status": "declined", "member_id": f.mPlayer}); st != http.StatusNoContent {
		t.Fatalf("status %d, want 204", st)
	}
	n := expectNotice(t, ch)
	if !strings.HasPrefix(n.body, childName+" hat für") || !strings.Contains(n.title, childName) {
		t.Errorf("Meldung nennt das Kind nicht: title %q, body %q", n.title, n.body)
	}
}

func TestRespondToGame_AbsenceLock_KeineMeldung(t *testing.T) {
	f := newRSVPPushFixture(t)
	res, err := f.db.Exec(`INSERT INTO member_absences (member_id, type, start_date, end_date, created_by)
		VALUES (?, 'vacation', '2026-06-14', '2026-06-16', ?)`, f.mPlayer, f.uPlayer)
	if err != nil {
		t.Fatal(err)
	}
	absID, _ := res.LastInsertId()
	if _, err := f.db.Exec(`INSERT INTO game_responses (game_id, member_id, responded_by, status, reason, responded_at, absence_id)
		VALUES (?, ?, ?, 'declined', '', datetime('now'), ?)`, f.gameID, f.mPlayer, f.uPlayer, absID); err != nil {
		t.Fatal(err)
	}
	ch := captureNotices(t)
	if st := f.respond(t, "2026-06-13 18:00", f.playerTok, map[string]any{"status": "confirmed"}); st != http.StatusForbidden {
		t.Fatalf("status %d, want 403", st)
	}
	expectNoNotice(t, ch)
}

func sameIDs(got, want []int) bool {
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
