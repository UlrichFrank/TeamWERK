package attendance_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type matrixBody struct {
	Events []struct {
		Kind      string `json:"kind"`
		ID        int    `json:"id"`
		Date      string `json:"date"`
		EventType string `json:"event_type"`
		Cancelled bool   `json:"cancelled"`
	} `json:"events"`
	Members []struct {
		MemberID int  `json:"member_id"`
		Extended bool `json:"extended"`
		Cells    []struct {
			Status      *string `json:"status"`
			IsDefault   bool    `json:"is_default"`
			Unavailable bool    `json:"unavailable"`
			Present     *bool   `json:"present"`
		} `json:"cells"`
	} `json:"members"`
}

const matrixRange = "?from=2026-04-01&to=2026-06-30"

func getMatrix(t *testing.T, db *sql.DB, teamID int, query, token string) (int, matrixBody) {
	t.Helper()
	srv := testServer(t, db)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/rsvp-matrix%s", teamID, query), token)
	defer res.Body.Close()
	var body matrixBody
	if res.StatusCode == http.StatusOK {
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
	}
	return res.StatusCode, body
}

// makePlayer legt einen Spieler mit eigenem Login an und hängt ihn in den Kader.
func makePlayer(t *testing.T, db *sql.DB, kaderID int) (userID, memberID int) {
	t.Helper()
	userID = testutil.CreateUser(t, db, "standard")
	memberID = testutil.CreateMember(t, db, userID)
	addKaderMember(t, db, kaderID, memberID)
	return userID, memberID
}

func setDefaults(t *testing.T, db *sql.DB, table string, id int, players, extended string) {
	t.Helper()
	if _, err := db.Exec(
		`UPDATE `+table+` SET rsvp_default_players=?, rsvp_default_extended=? WHERE id=?`,
		players, extended, id); err != nil {
		t.Fatalf("set defaults: %v", err)
	}
}

func statusOf(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func TestRSVPMatrix_HappyPath_StatusAusAntwortUndVoreinstellung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUser, kaderID := makeTrainer(t, db, teamID, seasonID)
	playerUser, player := makePlayer(t, db, kaderID)

	ts := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-02")
	game := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")
	setDefaults(t, db, "games", game, "confirmed", "none")
	setDefaults(t, db, "training_sessions", ts, "none", "none")
	recordManualDeclineTrainingResponse(t, db, ts, player, trainerUser)

	code, body := getMatrix(t, db, teamID, matrixRange,
		testutil.Token(t, playerUser, "standard", []string{"spieler"}))
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if len(body.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(body.Events))
	}
	// Chronologisch: Spiel (01.05.) vor Training (02.05.).
	if body.Events[0].Kind != "game" || body.Events[1].Kind != "training" {
		t.Fatalf("unexpected order: %+v", body.Events)
	}
	if body.Events[0].EventType != "heim" || body.Events[1].EventType != "training" {
		t.Errorf("unexpected event types: %+v", body.Events)
	}
	if len(body.Members) != 1 || body.Members[0].MemberID != player {
		t.Fatalf("expected exactly the player as row (no trainer), got %+v", body.Members)
	}
	cells := body.Members[0].Cells
	if statusOf(cells[0].Status) != "confirmed" || !cells[0].IsDefault {
		t.Errorf("game cell: want confirmed/default, got %s/%v", statusOf(cells[0].Status), cells[0].IsDefault)
	}
	if statusOf(cells[1].Status) != "declined" || cells[1].IsDefault {
		t.Errorf("training cell: want declined/explicit, got %s/%v", statusOf(cells[1].Status), cells[1].IsDefault)
	}
}

func TestRSVPMatrix_ErweiterterKaderNutztEigeneVoreinstellung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUser, kaderID := makeTrainer(t, db, teamID, seasonID)
	regular := testutil.CreateMember(t, db, 0)
	extended := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, regular)
	addExtendedMember(t, db, kaderID, extended)

	game := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")
	setDefaults(t, db, "games", game, "confirmed", "none")

	code, body := getMatrix(t, db, teamID, matrixRange,
		testutil.Token(t, trainerUser, "standard", []string{clubFnTrainr}))
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if len(body.Members) != 2 || body.Members[0].Extended || !body.Members[1].Extended {
		t.Fatalf("expected regular then extended, got %+v", body.Members)
	}
	if statusOf(body.Members[0].Cells[0].Status) != "confirmed" {
		t.Errorf("regular: want confirmed default, got %s", statusOf(body.Members[0].Cells[0].Status))
	}
	if body.Members[1].Cells[0].Status != nil {
		t.Errorf("extended with default 'none': want nil, got %s", statusOf(body.Members[1].Cells[0].Status))
	}
}

func TestRSVPMatrix_SerienAbmeldungUndAbgesagtesTraining(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUser, kaderID := makeTrainer(t, db, teamID, seasonID)
	player := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, player)

	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, trainerUser)
	ts := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, "2026-05-04")
	testutil.CreateSeriesUnavailability(t, db, player, seriesID, "", "", "Schule", trainerUser)
	cancelled := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-05")
	setSessionCancelled(t, db, cancelled)

	code, body := getMatrix(t, db, teamID, matrixRange,
		testutil.Token(t, trainerUser, "standard", []string{clubFnTrainr}))
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if len(body.Events) != 2 || body.Events[0].ID != ts || !body.Events[1].Cancelled {
		t.Fatalf("expected series session + cancelled session, got %+v", body.Events)
	}
	if !body.Members[0].Cells[0].Unavailable {
		t.Errorf("series unavailability must be flagged")
	}
	if body.Members[0].Cells[1].Unavailable {
		t.Errorf("session without series must not be flagged unavailable")
	}
}

func TestRSVPMatrix_AnwesenheitNurFuerTrainer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUser, kaderID := makeTrainer(t, db, teamID, seasonID)
	playerUser, player := makePlayer(t, db, kaderID)
	ts := testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate1)
	recordTrainingAttendance(t, db, ts, player, 1)

	_, trainerView := getMatrix(t, db, teamID, matrixRange,
		testutil.Token(t, trainerUser, "standard", []string{clubFnTrainr}))
	p := trainerView.Members[0].Cells[0].Present
	if p == nil || !*p {
		t.Errorf("trainer must see present=true, got %v", p)
	}

	_, playerView := getMatrix(t, db, teamID, matrixRange,
		testutil.Token(t, playerUser, "standard", []string{"spieler"}))
	if playerView.Members[0].Cells[0].Present != nil {
		t.Errorf("player must not see present")
	}
}

func TestRSVPMatrix_FremdesSpielErscheintNicht(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	otherTeam := testutil.CreateTeam(t, db, "Team B")
	adminUser := testutil.CreateUser(t, db, "admin")
	testutil.CreateGame(t, db, seasonID, otherTeam, "2026-05-01")
	testutil.CreateTrainingSession(t, db, otherTeam, seasonID, "2026-05-02")
	own := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-03")
	// Außerhalb des Zeitraums.
	testutil.CreateGame(t, db, seasonID, teamID, "2026-07-10")

	code, body := getMatrix(t, db, teamID, matrixRange, testutil.Token(t, adminUser, "admin", nil))
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if len(body.Events) != 1 || body.Events[0].ID != own {
		t.Fatalf("expected only own game in range, got %+v", body.Events)
	}
}

func TestRSVPMatrix_ElternteilDarfSehen(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	_, kaderID := makeTrainer(t, db, teamID, seasonID)
	child := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, child)
	parent := testutil.CreateUser(t, db, "standard")
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parent, child); err != nil {
		t.Fatalf("family_links: %v", err)
	}

	code, _ := getMatrix(t, db, teamID, matrixRange,
		testutil.TokenWithIsParent(t, parent, "standard", nil, true))
	if code != http.StatusOK {
		t.Fatalf("parent: expected 200, got %d", code)
	}
}

func TestRSVPMatrix_UnbeteiligterNutzer403(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	user := testutil.CreateUser(t, db, "standard")
	code, _ := getMatrix(t, db, teamID, matrixRange, testutil.Token(t, user, "standard", []string{"spieler"}))
	if code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", code)
	}
}

func TestRSVPMatrix_UnbekannteMannschaft404(t *testing.T) {
	db := testutil.NewDB(t)
	adminUser := testutil.CreateUser(t, db, "admin")
	code, _ := getMatrix(t, db, 9999, matrixRange, testutil.Token(t, adminUser, "admin", nil))
	if code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", code)
	}
}

func TestRSVPMatrix_UngueltigerZeitraum400(t *testing.T) {
	db := testutil.NewDB(t)
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminUser := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminUser, "admin", nil)
	for _, q := range []string{
		"",
		"?from=2026-04-01",
		"?from=kaputt&to=2026-06-30",
		"?from=2026-06-30&to=2026-04-01",
		"?from=2025-01-01&to=2026-06-30", // > 400 Tage
	} {
		if code, _ := getMatrix(t, db, teamID, q, token); code != http.StatusBadRequest {
			t.Errorf("query %q: expected 400, got %d", q, code)
		}
	}
}

func TestRSVPMatrix_OhneAktiveSaisonLeereZeilen(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	_, kaderID := makeTrainer(t, db, teamID, seasonID)
	addKaderMember(t, db, kaderID, testutil.CreateMember(t, db, 0))
	db.Exec(`UPDATE seasons SET is_active=0`)
	adminUser := testutil.CreateUser(t, db, "admin")

	code, body := getMatrix(t, db, teamID, matrixRange, testutil.Token(t, adminUser, "admin", nil))
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if len(body.Members) != 0 {
		t.Errorf("expected no rows without active season, got %d", len(body.Members))
	}
}

func TestRSVPMatrix_Unauthenticated401(t *testing.T) {
	db := testutil.NewDB(t)
	teamID := testutil.CreateTeam(t, db, "Team A")
	code, _ := getMatrix(t, db, teamID, matrixRange, "")
	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", code)
	}
}
