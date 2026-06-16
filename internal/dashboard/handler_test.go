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
