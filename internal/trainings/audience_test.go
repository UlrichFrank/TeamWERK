package trainings_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

// ── Empfängermenge der Trainings-Meldungen ───────────────────────────────────
//
// Trainings waren vor dem Change `terminmeldungen-erweiterter-kader` halb
// repariert: der erweiterte Kader stand in der Empfängerauflösung, seine Eltern
// nicht. Bei Minderjährigen ist das die falsche Hälfte — gefahren wird von zu
// Hause. Diese Tests halten beide Hälften fest.
//
// captureTrainingNotifications (cancellation_test.go) verwirft die
// Empfängerliste; hier wird sie gebraucht, deshalb ein eigener Recorder.

type trainingAudience struct {
	db           *sql.DB
	srv          *httptest.Server
	season       int
	team         int
	player       int
	playerParent int
	ext          int
	extParent    int
	trainerUser  int
	outsider     int
	admin        int
	recv         chan []int
}

func newTrainingAudience(t *testing.T) *trainingAudience {
	t.Helper()
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)

	player := testutil.CreateUser(t, db, "standard")
	playerMember := testutil.CreateMember(t, db, player)
	mustExec(t, db, `INSERT INTO kader_members (kader_id, member_id) VALUES (?,?)`, kader, playerMember)
	playerParent := testutil.CreateUser(t, db, "standard")
	mustExec(t, db, `INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, playerParent, playerMember)

	ext := testutil.CreateUser(t, db, "standard")
	extMember := testutil.CreateMember(t, db, ext)
	mustExec(t, db, `INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?,?)`, kader, extMember)
	extParent := testutil.CreateUser(t, db, "standard")
	mustExec(t, db, `INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, extParent, extMember)

	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddClubFunction(t, db, trainerMember, "trainer")
	testutil.AddKaderTrainer(t, db, kader, trainerMember)

	outsider := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, outsider)

	admin := testutil.CreateUser(t, db, "admin")
	setUserName(t, db, admin, "Tim", "Meier")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	recv := make(chan []int, 16)
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, userIDs []int, _, _, _, _ string, _ ...notify.Option) {
		recv <- userIDs
	}
	t.Cleanup(func() { notify.Send = orig })

	return &trainingAudience{
		db: db, srv: srv, season: season, team: team,
		player: player, playerParent: playerParent,
		ext: ext, extParent: extParent, trainerUser: trainerUser, outsider: outsider,
		admin: admin, recv: recv,
	}
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// assertAudience zieht die einzige erwartete Meldung ab und prüft ihre
// Empfängermenge.
func (f *trainingAudience) assertAudience(t *testing.T) {
	t.Helper()
	var recipients []int
	select {
	case recipients = <-f.recv:
	default:
		t.Fatal("keine Benachrichtigung versendet")
	}

	got := map[int]bool{}
	for _, id := range recipients {
		got[id] = true
	}
	for _, want := range []struct {
		id    int
		rolle string
	}{
		{f.player, "Stammkader"},
		{f.playerParent, "Elternteil des Stammspielers"},
		{f.ext, "erweiterter Kader"},
		{f.extParent, "Elternteil des erweiterten Spielers"},
		{f.trainerUser, "Trainer des Kaders"},
	} {
		if !got[want.id] {
			t.Errorf("%s (user %d) fehlt in der Empfängermenge %v", want.rolle, want.id, recipients)
		}
	}
	if got[f.outsider] {
		t.Errorf("Mitglied ohne Kader-Zugehörigkeit (user %d) darf keine Meldung bekommen", f.outsider)
	}
}

func (f *trainingAudience) token(t *testing.T) string {
	t.Helper()
	return testutil.Token(t, f.admin, "admin", nil)
}

// Der Fall, der vorher fehlte: das Elternteil eines nur erweiterten Mitglieds.
func TestUpdateSession_ElternDesErwKadersWerdenBenachrichtigt(t *testing.T) {
	f := newTrainingAudience(t)
	sessionID := testutil.CreateTrainingSession(t, f.db, f.team, f.season, "2026-03-15")

	res := testutil.Do(t, f.srv, http.MethodPut,
		"/api/training-sessions/"+strconv.Itoa(sessionID), f.token(t),
		updateSessionBody("cancelled", "Halle gesperrt"))
	res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 200/204", res.StatusCode)
	}

	f.assertAudience(t)
}

func TestDeleteSeries_ErwKaderUndElternWerdenBenachrichtigt(t *testing.T) {
	f := newTrainingAudience(t)
	seriesID := testutil.CreateTrainingSeries(t, f.db, f.team, f.season, f.admin)

	res := testutil.Do(t, f.srv, http.MethodDelete,
		"/api/training-series/"+strconv.Itoa(seriesID)+"?scope=all", f.token(t), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	f.assertAudience(t)
}

func TestDeleteSession_ErwKaderUndElternWerdenBenachrichtigt(t *testing.T) {
	f := newTrainingAudience(t)
	sessionID := testutil.CreateTrainingSession(t, f.db, f.team, f.season, "2026-03-15")

	res := testutil.Do(t, f.srv, http.MethodDelete,
		"/api/training-sessions/"+strconv.Itoa(sessionID), f.token(t), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	f.assertAudience(t)
}
