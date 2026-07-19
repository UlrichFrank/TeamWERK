package trainings_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

// unavailScenario baut ein Team mit eigenem Trainer, einer Serie und einem
// serien-gebundenen Termin. Rückgabe: die zentralen IDs für die Tests.
type unavailScenario struct {
	db           *sql.DB
	srv          *httptest.Server
	teamID       int
	seasonID     int
	seriesID     int
	sessionID    int // gehört zur Serie
	kaderID      int
	trainerUser  int
	trainerToken string
}

func setupUnavail(t *testing.T) (*trainings.Handler, unavailScenario) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	adminUser := testutil.CreateUser(t, db, "admin")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminUser)
	sessionID := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, "2026-03-15")

	// Trainer des Teams.
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`, trainerMember, "trainer")
	testutil.AddKaderTrainer(t, db, kaderID, trainerMember)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUser, "standard", []string{"trainer"})
	return h, unavailScenario{
		db: db, srv: srv, teamID: teamID, seasonID: seasonID, seriesID: seriesID,
		sessionID: sessionID, kaderID: kaderID, trainerUser: trainerUser, trainerToken: token,
	}
}

// --- CRUD (8.1) ---------------------------------------------------------

func TestCreateSeriesUnavailability_OwnTeamTrainer_201(t *testing.T) {
	_, sc := setupUnavail(t)
	member := testutil.CreateMember(t, sc.db, 0)
	res := testutil.Post(t, sc.srv, fmt.Sprintf("/api/training-series/%d/unavailabilities", sc.seriesID),
		sc.trainerToken, map[string]any{"member_id": member, "reason": "A-Jugend"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var n int
	sc.db.QueryRow(`SELECT COUNT(*) FROM member_series_unavailabilities WHERE training_series_id=? AND member_id=? AND end_date IS NULL`,
		sc.seriesID, member).Scan(&n)
	if n != 1 {
		t.Errorf("expected 1 permanent row, got %d", n)
	}
}

func TestCreateSeriesUnavailability_ForeignTeamTrainer_403(t *testing.T) {
	_, sc := setupUnavail(t)
	// Fremder Trainer eines anderen Teams.
	otherTeam := testutil.CreateTeam(t, sc.db, "Team B")
	otherKader := testutil.CreateKader(t, sc.db, otherTeam, sc.seasonID)
	foreignUser := testutil.CreateUser(t, sc.db, "standard")
	foreignMember := testutil.CreateMember(t, sc.db, foreignUser)
	sc.db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`, foreignMember, "trainer")
	testutil.AddKaderTrainer(t, sc.db, otherKader, foreignMember)
	token := testutil.Token(t, foreignUser, "standard", []string{"trainer"})

	member := testutil.CreateMember(t, sc.db, 0)
	res := testutil.Post(t, sc.srv, fmt.Sprintf("/api/training-series/%d/unavailabilities", sc.seriesID),
		token, map[string]any{"member_id": member})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	var n int
	sc.db.QueryRow(`SELECT COUNT(*) FROM member_series_unavailabilities WHERE training_series_id=?`, sc.seriesID).Scan(&n)
	if n != 0 {
		t.Errorf("no row must be created, got %d", n)
	}
}

func TestCreateSeriesUnavailability_Player_403(t *testing.T) {
	_, sc := setupUnavail(t)
	// Reiner Spieler (keine Vereinsfunktion) — Router-Gate lehnt ab.
	playerUser := testutil.CreateUser(t, sc.db, "standard")
	testutil.CreateMember(t, sc.db, playerUser)
	token := testutil.Token(t, playerUser, "standard", nil)
	member := testutil.CreateMember(t, sc.db, 0)
	res := testutil.Post(t, sc.srv, fmt.Sprintf("/api/training-series/%d/unavailabilities", sc.seriesID),
		token, map[string]any{"member_id": member})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestListSeriesUnavailabilities_Trainer_200(t *testing.T) {
	_, sc := setupUnavail(t)
	member := testutil.CreateMember(t, sc.db, 0)
	testutil.CreateSeriesUnavailability(t, sc.db, member, sc.seriesID, "", "", "grund", sc.trainerUser)
	res := testutil.Get(t, sc.srv, fmt.Sprintf("/api/training-series/%d/unavailabilities", sc.seriesID), sc.trainerToken)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var out struct {
		Items []struct {
			MemberID  int     `json:"member_id"`
			StartDate *string `json:"start_date"`
			EndDate   *string `json:"end_date"`
			Reason    string  `json:"reason"`
		} `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&out)
	if len(out.Items) != 1 || out.Items[0].MemberID != member {
		t.Fatalf("expected one item for member %d, got %+v", member, out.Items)
	}
	if out.Items[0].StartDate != nil || out.Items[0].EndDate != nil {
		t.Errorf("open window must yield null start/end, got %+v", out.Items[0])
	}
}

func TestListSeriesUnavailabilities_ForeignTrainer_403(t *testing.T) {
	_, sc := setupUnavail(t)
	otherTeam := testutil.CreateTeam(t, sc.db, "Team B")
	otherKader := testutil.CreateKader(t, sc.db, otherTeam, sc.seasonID)
	foreignUser := testutil.CreateUser(t, sc.db, "standard")
	foreignMember := testutil.CreateMember(t, sc.db, foreignUser)
	sc.db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`, foreignMember, "trainer")
	testutil.AddKaderTrainer(t, sc.db, otherKader, foreignMember)
	token := testutil.Token(t, foreignUser, "standard", []string{"trainer"})
	res := testutil.Get(t, sc.srv, fmt.Sprintf("/api/training-series/%d/unavailabilities", sc.seriesID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestDeleteSeriesUnavailability_Trainer_204(t *testing.T) {
	_, sc := setupUnavail(t)
	member := testutil.CreateMember(t, sc.db, 0)
	uid := testutil.CreateSeriesUnavailability(t, sc.db, member, sc.seriesID, "", "", "", sc.trainerUser)
	res := testutil.Delete(t, sc.srv, fmt.Sprintf("/api/training-series/%d/unavailabilities/%d", sc.seriesID, uid), sc.trainerToken)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var n int
	sc.db.QueryRow(`SELECT COUNT(*) FROM member_series_unavailabilities WHERE id=?`, uid).Scan(&n)
	if n != 0 {
		t.Errorf("row must be deleted, got %d", n)
	}
}

func TestDeleteSeriesUnavailability_WrongSeries_404(t *testing.T) {
	_, sc := setupUnavail(t)
	// Abmeldung gehört zu einer ANDEREN Serie desselben Teams.
	otherSeries := testutil.CreateTrainingSeries(t, sc.db, sc.teamID, sc.seasonID, sc.trainerUser)
	member := testutil.CreateMember(t, sc.db, 0)
	uid := testutil.CreateSeriesUnavailability(t, sc.db, member, otherSeries, "", "", "", sc.trainerUser)
	res := testutil.Delete(t, sc.srv, fmt.Sprintf("/api/training-series/%d/unavailabilities/%d", sc.seriesID, uid), sc.trainerToken)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
	var n int
	sc.db.QueryRow(`SELECT COUNT(*) FROM member_series_unavailabilities WHERE id=?`, uid).Scan(&n)
	if n != 1 {
		t.Errorf("row of other series must survive, got %d", n)
	}
}

// --- RSVP-Sperre (8.2) --------------------------------------------------

func TestRespond_UnavailablePlayer_403(t *testing.T) {
	_, sc := setupUnavail(t)
	playerUser := testutil.CreateUser(t, sc.db, "standard")
	player := testutil.CreateMember(t, sc.db, playerUser)
	testutil.CreateSeriesUnavailability(t, sc.db, player, sc.seriesID, "", "", "", sc.trainerUser)
	token := testutil.Token(t, playerUser, "standard", nil)

	res := testutil.Post(t, sc.srv, fmt.Sprintf("/api/training-sessions/%d/respond", sc.sessionID),
		token, map[string]any{"status": "confirmed"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	var n int
	sc.db.QueryRow(`SELECT COUNT(*) FROM training_responses WHERE training_id=? AND member_id=?`, sc.sessionID, player).Scan(&n)
	if n != 0 {
		t.Errorf("no response row must be written, got %d", n)
	}
}

func TestRespond_UnavailableParentForChild_403(t *testing.T) {
	_, sc := setupUnavail(t)
	parentUser := testutil.CreateUser(t, sc.db, "standard")
	child := testutil.CreateMember(t, sc.db, 0)
	testutil.AddFamilyLink(t, sc.db, parentUser, child)
	testutil.CreateSeriesUnavailability(t, sc.db, child, sc.seriesID, "", "", "", sc.trainerUser)
	token := testutil.TokenWithIsParent(t, parentUser, "standard", nil, true)

	res := testutil.Post(t, sc.srv, fmt.Sprintf("/api/training-sessions/%d/respond", sc.sessionID),
		token, map[string]any{"member_id": child, "status": "declined"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestRespond_NotAffected_Succeeds(t *testing.T) {
	_, sc := setupUnavail(t)
	playerUser := testutil.CreateUser(t, sc.db, "standard")
	player := testutil.CreateMember(t, sc.db, playerUser)
	// Zukünftiger Termin (kein RSVP-Cutoff) außerhalb des Abmelde-Fensters.
	futureSession := testutil.CreateTrainingSessionForSeries(t, sc.db, sc.teamID, sc.seasonID, sc.seriesID, "2026-09-10")
	testutil.CreateSeriesUnavailability(t, sc.db, player, sc.seriesID, "2026-05-01", "2026-06-30", "", sc.trainerUser)
	token := testutil.Token(t, playerUser, "standard", nil)

	res := testutil.Post(t, sc.srv, fmt.Sprintf("/api/training-sessions/%d/respond", futureSession),
		token, map[string]any{"status": "confirmed"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for unaffected session, got %d", res.StatusCode)
	}
}

// --- Attendance-Skip (8.2) ----------------------------------------------

func TestSaveAttendances_UnavailableSkipped_RestSaved(t *testing.T) {
	_, sc := setupUnavail(t)
	// Session in Vergangenheit, damit die Erfassung erlaubt ist.
	pastSession := testutil.CreateTrainingSessionForSeries(t, sc.db, sc.teamID, sc.seasonID, sc.seriesID, "2026-03-15")
	// Beide als Spieler im Kader.
	unavailPlayer := testutil.CreateMember(t, sc.db, 0)
	normalPlayer := testutil.CreateMember(t, sc.db, 0)
	sc.db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, sc.kaderID, unavailPlayer)
	sc.db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, sc.kaderID, normalPlayer)
	testutil.CreateSeriesUnavailability(t, sc.db, unavailPlayer, sc.seriesID, "", "", "", sc.trainerUser)

	body := []map[string]any{
		{"member_id": unavailPlayer, "present": true},
		{"member_id": normalPlayer, "present": true},
	}
	res := testutil.Post(t, sc.srv, fmt.Sprintf("/api/training-sessions/%d/attendances", pastSession), sc.trainerToken, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var nUnavail, nNormal int
	sc.db.QueryRow(`SELECT COUNT(*) FROM training_attendances WHERE training_id=? AND member_id=?`, pastSession, unavailPlayer).Scan(&nUnavail)
	sc.db.QueryRow(`SELECT COUNT(*) FROM training_attendances WHERE training_id=? AND member_id=?`, pastSession, normalPlayer).Scan(&nNormal)
	if nUnavail != 0 {
		t.Errorf("unavailable player must have no attendance row, got %d", nUnavail)
	}
	if nNormal != 1 {
		t.Errorf("normal player attendance must be saved, got %d", nNormal)
	}
}

// --- Session-Roster liefert unavailable-Feld (8.2 / 7.1) -----------------

func TestGetAttendances_UnavailableField(t *testing.T) {
	_, sc := setupUnavail(t)
	unavailPlayer := testutil.CreateMember(t, sc.db, 0)
	normalPlayer := testutil.CreateMember(t, sc.db, 0)
	sc.db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, sc.kaderID, unavailPlayer)
	sc.db.Exec(`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`, sc.kaderID, normalPlayer)
	testutil.CreateSeriesUnavailability(t, sc.db, unavailPlayer, sc.seriesID, "", "", "A-Jugend", sc.trainerUser)

	res := testutil.Get(t, sc.srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sc.sessionID), sc.trainerToken)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []struct {
		MemberID    int `json:"member_id"`
		Unavailable *struct {
			Reason    string `json:"reason"`
			Permanent bool   `json:"permanent"`
		} `json:"unavailable"`
	}
	json.NewDecoder(res.Body).Decode(&items)
	var sawUnavail, sawNormal bool
	for _, it := range items {
		switch it.MemberID {
		case unavailPlayer:
			sawUnavail = true
			if it.Unavailable == nil || it.Unavailable.Reason != "A-Jugend" || !it.Unavailable.Permanent {
				t.Errorf("unavailable player must carry {A-Jugend, permanent:true}, got %+v", it.Unavailable)
			}
		case normalPlayer:
			sawNormal = true
			if it.Unavailable != nil {
				t.Errorf("normal player must have unavailable=null, got %+v", it.Unavailable)
			}
		}
	}
	if !sawUnavail || !sawNormal {
		t.Fatalf("both players must be present in roster (visible), sawUnavail=%v sawNormal=%v", sawUnavail, sawNormal)
	}
}
