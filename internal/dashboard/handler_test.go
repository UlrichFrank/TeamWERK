package dashboard_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/dashboard"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func testServer(t *testing.T, h *dashboard.Handler) *httptest.Server {
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/dashboard", h.Get)
	})
}

// TestDashboard_MeineTermine_IsExtended verifies that a training event for a team
// the user only belongs to via kader_extended_members has isExtended=true.
func TestDashboard_MeineTermine_IsExtended(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Damen 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	// Training tomorrow so it shows as upcoming
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	testutil.CreateTrainingSession(t, db, teamID, seasonID, tomorrow)

	h := dashboard.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	var events []map[string]any
	json.Unmarshal(body["meineTermine"], &events)

	if len(events) == 0 {
		t.Fatal("expected at least one event in meineTermine")
	}
	if events[0]["isExtended"] != true {
		t.Errorf("expected isExtended=true for extended kader training event, got %v", events[0]["isExtended"])
	}
}

// TestDashboard_CarpoolingConfirmed_KindPaarung verifies that a parent sees their child's
// confirmed carpooling pairing in the dashboard carpoolingConfirmed section.
func TestDashboard_CarpoolingConfirmed_KindPaarung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	parentID := testutil.CreateUser(t, db, "standard")
	childUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, childUserID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentID, childMemberID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)

	bieterID := testutil.CreateUser(t, db, "standard")

	// Auswärtsspiel in der Zukunft
	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, futureDate)
	db.Exec(`UPDATE games SET is_home=0 WHERE id=?`, gameID)

	// Kind sucht, Bieter bietet, confirmed Paarung
	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()
	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, childUserID)
	sucheID, _ := sucheRes.LastInsertId()
	db.Exec(`INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von, status) VALUES (?, ?, 'suche', 'confirmed')`, bieteID, sucheID)

	h := dashboard.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, parentID, "standard", nil)
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	var confirmed []map[string]any
	json.Unmarshal(body["carpoolingConfirmed"], &confirmed)

	if len(confirmed) == 0 {
		t.Error("expected parent to see child's confirmed carpooling pairing in dashboard")
	}
}

// decodeConfirmedPaarung fetches the dashboard for `userID` and returns the
// first paarung of the first confirmed group, or fails.
func decodeConfirmedPaarung(t *testing.T, srv *httptest.Server, userID int) map[string]any {
	t.Helper()
	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	var confirmed []map[string]any
	json.Unmarshal(body["carpoolingConfirmed"], &confirmed)
	if len(confirmed) == 0 {
		t.Fatal("expected at least one confirmed group")
	}
	paarungen, _ := confirmed[0]["paarungen"].([]any)
	if len(paarungen) == 0 {
		t.Fatal("expected at least one paarung in confirmed group")
	}
	return paarungen[0].(map[string]any)
}

// setupConfirmedPaarung creates a season, team, away game, two carpool entries
// (biete + suche) and a confirmed paarung between them. Returns userID for the
// caller-controlled side. `myRole` is "biete" or "suche" — the test takes that
// role and the partner takes the other. `myTreffpunkt` and `partnerTreffpunkt`
// are written into the respective entries (empty string = NULL).
func setupConfirmedPaarung(t *testing.T, myRole, myTreffpunkt, partnerTreffpunkt string) (srv *httptest.Server, myUserID int) {
	t.Helper()
	d := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, d, "2025/26")
	d.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, d, "Herren 1")
	kaderID := testutil.CreateKader(t, d, teamID, seasonID)

	me := testutil.CreateUser(t, d, "standard")
	meMember := testutil.CreateMember(t, d, me)
	d.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, meMember)

	partner := testutil.CreateUser(t, d, "standard")

	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	gameID := testutil.CreateGame(t, d, seasonID, teamID, futureDate)
	d.Exec(`UPDATE games SET is_home=0 WHERE id=?`, gameID)

	insertEntry := func(userID int, typ, tp string) int64 {
		if tp == "" {
			r, _ := d.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, ?, 1)`, gameID, userID, typ)
			id, _ := r.LastInsertId()
			return id
		}
		r, _ := d.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze, treffpunkt) VALUES (?, ?, ?, 1, ?)`, gameID, userID, typ, tp)
		id, _ := r.LastInsertId()
		return id
	}

	var bieteID, sucheID int64
	if myRole == "biete" {
		bieteID = insertEntry(me, "biete", myTreffpunkt)
		sucheID = insertEntry(partner, "suche", partnerTreffpunkt)
	} else {
		bieteID = insertEntry(partner, "biete", partnerTreffpunkt)
		sucheID = insertEntry(me, "suche", myTreffpunkt)
	}
	d.Exec(`INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von, status) VALUES (?, ?, 'suche', 'confirmed')`, bieteID, sucheID)

	srv = testServer(t, dashboard.NewHandler(d))
	return srv, me
}

// TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_AsBieter verifies that the
// user, being on the bieter side of a confirmed paarung, sees the sucher's
// treffpunkt as partnerTreffpunkt.
func TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_AsBieter(t *testing.T) {
	srv, me := setupConfirmedPaarung(t, "biete", "MyTreff", "Bahnhof Mitte")
	p := decodeConfirmedPaarung(t, srv, me)
	if got := p["partnerTreffpunkt"]; got != "Bahnhof Mitte" {
		t.Errorf("expected partnerTreffpunkt='Bahnhof Mitte', got %v", got)
	}
}

// TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_AsSucher verifies that the
// user, being on the sucher side of a confirmed paarung, sees the bieter's
// treffpunkt as partnerTreffpunkt.
func TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_AsSucher(t *testing.T) {
	srv, me := setupConfirmedPaarung(t, "suche", "MyTreff", "Marktplatz")
	p := decodeConfirmedPaarung(t, srv, me)
	if got := p["partnerTreffpunkt"]; got != "Marktplatz" {
		t.Errorf("expected partnerTreffpunkt='Marktplatz', got %v", got)
	}
}

// TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_Empty verifies that when
// the partner has no treffpunkt set, partnerTreffpunkt is the empty string.
func TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_Empty(t *testing.T) {
	srv, me := setupConfirmedPaarung(t, "biete", "MyTreff", "")
	p := decodeConfirmedPaarung(t, srv, me)
	if got := p["partnerTreffpunkt"]; got != "" {
		t.Errorf("expected partnerTreffpunkt='', got %v", got)
	}
}

// TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_KindAsBieter verifies
// that a parent sees the sucher's treffpunkt when their child is the bieter.
func TestDashboard_CarpoolingConfirmed_PartnerTreffpunkt_KindAsBieter(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	parentID := testutil.CreateUser(t, db, "standard")
	childUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, childUserID)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentID, childMemberID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, childMemberID)

	sucherID := testutil.CreateUser(t, db, "standard")
	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, futureDate)
	db.Exec(`UPDATE games SET is_home=0 WHERE id=?`, gameID)

	// Kind=bieter (no treffpunkt), sucher has treffpunkt "Schule"
	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, childUserID)
	bieteID, _ := bieteRes.LastInsertId()
	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze, treffpunkt) VALUES (?, ?, 'suche', 1, ?)`, gameID, sucherID, "Schule")
	sucheID, _ := sucheRes.LastInsertId()
	db.Exec(`INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von, status) VALUES (?, ?, 'suche', 'confirmed')`, bieteID, sucheID)

	srv := testServer(t, dashboard.NewHandler(db))
	p := decodeConfirmedPaarung(t, srv, parentID)
	if got := p["partnerTreffpunkt"]; got != "Schule" {
		t.Errorf("expected partnerTreffpunkt='Schule', got %v", got)
	}
}

// decodeOpenGroups fetches the dashboard as the given user and returns the
// carpoolingOpenGroups array.
func decodeOpenGroups(t *testing.T, srv *httptest.Server, userID int, funcs []string) []map[string]any {
	t.Helper()
	token := testutil.Token(t, userID, "standard", funcs)
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	var groups []map[string]any
	json.Unmarshal(body["carpoolingOpenGroups"], &groups)
	return groups
}

// TestDashboard_OffeneGesuche_OwnTeam verifies that an open ride request (suche
// without a confirmed pairing) on an upcoming game of the user's team appears
// in carpoolingOpenGroups.
func TestDashboard_OffeneGesuche_OwnTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	requesterID := testutil.CreateUser(t, db, "standard")
	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, futureDate)

	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 2)`, gameID, requesterID)
	sucheID, _ := sucheRes.LastInsertId()

	srv := testServer(t, dashboard.NewHandler(db))
	groups := decodeOpenGroups(t, srv, userID, nil)

	if len(groups) != 1 {
		t.Fatalf("expected 1 open-request group, got %d", len(groups))
	}
	reqs, _ := groups[0]["requests"].([]any)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 open request, got %d", len(reqs))
	}
	r0 := reqs[0].(map[string]any)
	if got, _ := r0["sucheId"].(float64); int64(got) != sucheID {
		t.Errorf("expected sucheId=%d, got %v", sucheID, r0["sucheId"])
	}
	if got, _ := r0["plaetze"].(float64); got != 2 {
		t.Errorf("expected plaetze=2, got %v", r0["plaetze"])
	}
}

// TestDashboard_OffeneGesuche_ConfirmedExcluded verifies that a suche with a
// confirmed pairing is NOT listed as open, but still appears in
// carpoolingConfirmed.
func TestDashboard_OffeneGesuche_ConfirmedExcluded(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	bieterID := testutil.CreateUser(t, db, "standard")

	// Away game so it can also show in carpoolingConfirmed.
	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, futureDate)
	db.Exec(`UPDATE games SET is_home=0 WHERE id=?`, gameID)

	// User owns the suche; another user offers; pairing is confirmed.
	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()
	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, userID)
	sucheID, _ := sucheRes.LastInsertId()
	db.Exec(`INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von, status) VALUES (?, ?, 'suche', 'confirmed')`, bieteID, sucheID)

	srv := testServer(t, dashboard.NewHandler(db))

	// Not in open groups.
	groups := decodeOpenGroups(t, srv, userID, nil)
	for _, g := range groups {
		reqs, _ := g["requests"].([]any)
		if len(reqs) > 0 {
			t.Fatalf("expected no open requests once pairing is confirmed, got %d", len(reqs))
		}
	}

	// Still in carpoolingConfirmed.
	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/dashboard", token)
	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	var confirmed []map[string]any
	json.Unmarshal(body["carpoolingConfirmed"], &confirmed)
	if len(confirmed) == 0 {
		t.Error("expected confirmed pairing to remain in carpoolingConfirmed")
	}
}

// TestDashboard_OffeneGesuche_PendingStillOpen verifies that a suche with only a
// pending pairing still counts as open.
func TestDashboard_OffeneGesuche_PendingStillOpen(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	bieterID := testutil.CreateUser(t, db, "standard")
	requesterID := testutil.CreateUser(t, db, "standard")
	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, futureDate)

	bieteRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`, gameID, bieterID)
	bieteID, _ := bieteRes.LastInsertId()
	sucheRes, _ := db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, gameID, requesterID)
	sucheID, _ := sucheRes.LastInsertId()
	db.Exec(`INSERT INTO mitfahrt_paarungen (biete_id, suche_id, initiiert_von, status) VALUES (?, ?, 'biete', 'pending')`, bieteID, sucheID)

	srv := testServer(t, dashboard.NewHandler(db))
	groups := decodeOpenGroups(t, srv, userID, nil)

	if len(groups) != 1 {
		t.Fatalf("expected suche with only a pending pairing to stay open, got %d groups", len(groups))
	}
	reqs, _ := groups[0]["requests"].([]any)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 open request (pending counts as open), got %d", len(reqs))
	}
}

// TestDashboard_OffeneGesuche_OtherTeamExcluded verifies that an open request on
// a game of a team the user does not belong to is not shown (cross-team is a
// follow-up proposal).
func TestDashboard_OffeneGesuche_OtherTeamExcluded(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)

	myTeamID := testutil.CreateTeam(t, db, "Herren 1")
	myKaderID := testutil.CreateKader(t, db, myTeamID, seasonID)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, myKaderID, memberID)

	// Foreign team + game the user has no access to.
	otherTeamID := testutil.CreateTeam(t, db, "Damen 1")
	testutil.CreateKader(t, db, otherTeamID, seasonID)
	requesterID := testutil.CreateUser(t, db, "standard")
	futureDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	otherGameID := testutil.CreateGame(t, db, seasonID, otherTeamID, futureDate)
	db.Exec(`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`, otherGameID, requesterID)

	srv := testServer(t, dashboard.NewHandler(db))
	groups := decodeOpenGroups(t, srv, userID, nil)

	if len(groups) != 0 {
		t.Fatalf("expected no open-request groups for a foreign team's game, got %d", len(groups))
	}
}

// TestDashboard_MeineTermine_IsNotExtended verifies that a training event for a team
// the user belongs to via kader_members has isExtended=false.
func TestDashboard_MeineTermine_IsNotExtended(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Herren 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	testutil.CreateTrainingSession(t, db, teamID, seasonID, tomorrow)

	h := dashboard.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	var events []map[string]any
	json.Unmarshal(body["meineTermine"], &events)

	if len(events) == 0 {
		t.Fatal("expected at least one event in meineTermine")
	}
	if events[0]["isExtended"] != false {
		t.Errorf("expected isExtended=false for primary kader training event, got %v", events[0]["isExtended"])
	}
}

// TestDashboard_MeineDienste_AudienceFiltersOpenSlots verifies that a trainer
// does not see player-only audience slots in the "Meine Dienste" widget. The
// game is on the trainer's team and has both a trainer-audience and a
// spieler-audience open slot — only the trainer-audience slot must count.
func TestDashboard_MeineDienste_AudienceFiltersOpenSlots(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Damen 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	// Trainer user with member_club_functions=trainer, attached to the team's kader as trainer.
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'trainer')`, memberID)
	db.Exec(`INSERT INTO kader_trainers (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, tomorrow)

	dtSpieler := testutil.CreateDutyType(t, db, "Kampfgericht", 1.0)
	dtTrainer := testutil.CreateDutyType(t, db, "Coachen", 1.0)
	spielerSlot := testutil.CreateDutySlot(t, db, dtSpieler, seasonID, teamID, gameID, tomorrow)
	trainerSlot := testutil.CreateDutySlot(t, db, dtTrainer, seasonID, teamID, gameID, tomorrow)
	db.Exec(`UPDATE duty_slots SET audiences='["spieler"]' WHERE id=?`, spielerSlot)
	db.Exec(`UPDATE duty_slots SET audiences='["trainer"]' WHERE id=?`, trainerSlot)

	h := dashboard.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	var md map[string]any
	json.Unmarshal(body["meineDienste"], &md)
	if md["nextGame"] == nil {
		t.Fatalf("expected nextGame to be present for trainer with trainer-audience slot, got nil")
	}
	// CreateDutySlot creates 2 total / 0 filled — so 2 open slots per slot.
	// Only the trainer-audience slot should count → 2, not 4.
	if got, _ := md["openSlotsCount"].(float64); got != 2 {
		t.Errorf("expected openSlotsCount=2 (only trainer-audience slot), got %v", got)
	}
}

// TestDashboard_MeineDienste_AudienceHidesGameWithoutMatchingSlots verifies that
// a trainer whose team has an upcoming game with only player-audience duty slots
// does NOT see that game as nextGame on their dashboard.
func TestDashboard_MeineDienste_AudienceHidesGameWithoutMatchingSlots(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)
	teamID := testutil.CreateTeam(t, db, "Damen 1")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'trainer')`, memberID)
	db.Exec(`INSERT INTO kader_trainers (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)

	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, tomorrow)

	dtSpieler := testutil.CreateDutyType(t, db, "Kampfgericht", 1.0)
	spielerSlot := testutil.CreateDutySlot(t, db, dtSpieler, seasonID, teamID, gameID, tomorrow)
	db.Exec(`UPDATE duty_slots SET audiences='["spieler"]' WHERE id=?`, spielerSlot)

	h := dashboard.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	var md map[string]any
	json.Unmarshal(body["meineDienste"], &md)
	if md["nextGame"] != nil {
		t.Errorf("expected nextGame=nil when only spieler-audience slots exist for trainer, got %v", md["nextGame"])
	}
}

// TestDashboard_Doppelheimspiel_ListsBothTeams is a regression test for the MIN→GROUP_CONCAT fix:
// previously, a game referencing two teams of the same age_class+gender showed only one team's
// name in the dashboard. Now both teams must appear, comma-separated, in the teamName field.
func TestDashboard_Doppelheimspiel_ListsBothTeams(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID)

	// Zwei B-Jugend-m-Teams, beide im Kader des Users
	t1Res, _ := db.Exec(`INSERT INTO teams (name, age_class, gender) VALUES (?, ?, ?)`, "TS B1", "B-Jugend", "m")
	t1ID64, _ := t1Res.LastInsertId()
	t1ID := int(t1ID64)
	t2Res, _ := db.Exec(`INSERT INTO teams (name, age_class, gender) VALUES (?, ?, ?)`, "TS B2", "B-Jugend", "m")
	t2ID64, _ := t2Res.LastInsertId()
	t2ID := int(t2ID64)

	k1Res, _ := db.Exec(`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "B-Jugend", "m", t1ID, 1)
	k1ID, _ := k1Res.LastInsertId()
	k2Res, _ := db.Exec(`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "B-Jugend", "m", t2ID, 2)
	k2ID, _ := k2Res.LastInsertId()

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, k1ID, memberID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, k2ID, memberID)

	// Doppelheimspiel morgen
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, t1ID, tomorrow)
	db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, t2ID)

	h := dashboard.NewHandler(db)
	srv := testServer(t, h)

	token := testutil.Token(t, userID, "standard", nil)
	res := testutil.Get(t, srv, "/api/dashboard", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]json.RawMessage
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	var events []map[string]any
	json.Unmarshal(body["meineTermine"], &events)
	if len(events) == 0 {
		t.Fatal("expected the Doppelheimspiel to appear in meineTermine")
	}
	var gameEvent map[string]any
	for _, e := range events {
		if e["eventType"] == "spiel" {
			gameEvent = e
			break
		}
	}
	if gameEvent == nil {
		t.Fatalf("expected a spiel event, got: %v", events)
	}
	teamName, _ := gameEvent["teamName"].(string)
	if teamName != "mB1, mB2" {
		t.Errorf("expected teamName 'mB1, mB2' (both teams short-form, sorted), got %q", teamName)
	}
}
