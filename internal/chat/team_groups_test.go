package chat_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type tgFixture struct {
	db                       *sql.DB
	team1, team2             int
	season                   int
	kader1, kader2           int
	trainerU1, trainerU2     int
	playerU1, playerU2       int
	parentU1, parentU2       int
	extPlayerU1, extPlayerU2 int
	extParentU1, extParentU2 int
}

// setupTwoTeams creates two teams (team1, team2) in the active season with:
//   - one trainer linked to each team
//   - one regular player linked to each team
//   - one parent linked to that player on each team
//   - one extended kader player linked to each team
//   - one parent of that extended player on each team
func setupTwoTeams(t *testing.T) (*tgFixture, *chi.Mux) {
	t.Helper()
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team1 := testutil.CreateTeam(t, db, "T1")
	team2 := testutil.CreateTeam(t, db, "T2")
	kader1 := testutil.CreateKader(t, db, team1, season)
	kader2 := testutil.CreateKader(t, db, team2, season)

	mkTrainer := func(kaderID int) int {
		u := testutil.CreateUser(t, db, "standard")
		m := testutil.CreateMember(t, db, u)
		testutil.AddKaderTrainer(t, db, kaderID, m)
		return u
	}
	mkPlayer := func(kaderID int) (playerUser int, parentUser int) {
		pu := testutil.CreateUser(t, db, "standard")
		pm := testutil.CreateMember(t, db, pu)
		if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, pm); err != nil {
			t.Fatalf("insert kader_members: %v", err)
		}
		parU := testutil.CreateUser(t, db, "standard")
		if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parU, pm); err != nil {
			t.Fatalf("insert family_links: %v", err)
		}
		return pu, parU
	}
	mkExtPlayer := func(kaderID int) (playerUser int, parentUser int) {
		pu := testutil.CreateUser(t, db, "standard")
		pm := testutil.CreateMember(t, db, pu)
		testutil.AddExtendedKaderMember(t, db, kaderID, pm)
		parU := testutil.CreateUser(t, db, "standard")
		if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parU, pm); err != nil {
			t.Fatalf("insert family_links: %v", err)
		}
		return pu, parU
	}

	f := &tgFixture{
		db:    db,
		team1: team1, team2: team2, season: season,
		kader1: kader1, kader2: kader2,
	}
	f.trainerU1 = mkTrainer(kader1)
	f.trainerU2 = mkTrainer(kader2)
	f.playerU1, f.parentU1 = mkPlayer(kader1)
	f.playerU2, f.parentU2 = mkPlayer(kader2)
	f.extPlayerU1, f.extParentU1 = mkExtPlayer(kader1)
	f.extPlayerU2, f.extParentU2 = mkExtPlayer(kader2)

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	r := chi.NewRouter()
	r.Get("/api/chat/team-groups", h.ListTeamGroups)
	r.Get("/api/chat/team-groups/{teamId}/{kind}/members", h.ResolveTeamGroup)
	return f, r
}

func decodeJSON[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	defer res.Body.Close()
	var v T
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return v
}

func TestListTeamGroups_PlayerSeesOwnTeam(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, f.playerU1, "standard", nil))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	gotKinds := map[int]map[string]int{}
	for _, g := range groups {
		if gotKinds[g.TeamID] == nil {
			gotKinds[g.TeamID] = map[string]int{}
		}
		gotKinds[g.TeamID][g.Kind] = g.Count
	}
	if _, ok := gotKinds[f.team2]; ok {
		t.Errorf("player should not see foreign team T2, got %+v", gotKinds)
	}
	t1 := gotKinds[f.team1]
	if t1 == nil {
		t.Fatalf("player should see own team T1, got %+v", gotKinds)
	}
	if t1["trainer"] != 1 {
		t.Errorf("expected 1 trainer in T1, got %d", t1["trainer"])
	}
	// Players: extended player included; caller (regular player) excluded
	if t1["spieler"] != 1 {
		t.Errorf("expected 1 spieler in T1 (extended only, caller excluded), got %d", t1["spieler"])
	}
	// Parents: own parent + extended parent
	if t1["eltern"] != 2 {
		t.Errorf("expected 2 eltern in T1, got %d", t1["eltern"])
	}
}

func TestListTeamGroups_VorstandSeesAllTeams(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	outsider := testutil.CreateUser(t, f.db, "standard")

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, outsider, "standard", []string{"vorstand"}))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	seen := map[int]int{}
	for _, g := range groups {
		seen[g.TeamID]++
	}
	if seen[f.team1] != 3 || seen[f.team2] != 3 {
		t.Errorf("vorstand should see both teams ×3, got %+v", seen)
	}
}

func TestListTeamGroups_SportlicheLeitungSeesAllTeams(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	outsider := testutil.CreateUser(t, f.db, "standard")

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, outsider, "standard", []string{"sportliche_leitung"}))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	seen := map[int]int{}
	for _, g := range groups {
		seen[g.TeamID]++
	}
	if seen[f.team1] != 3 || seen[f.team2] != 3 {
		t.Errorf("sportliche_leitung should see both teams ×3, got %+v", seen)
	}
}

func TestListTeamGroups_TrainerSeesOnlyOwnTeam(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, f.trainerU1, "standard", nil))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	for _, g := range groups {
		if g.TeamID == f.team2 {
			t.Errorf("trainer of T1 must not see T2, got %+v", g)
		}
	}
}

func TestListTeamGroups_InactiveSeasonHidden(t *testing.T) {
	db := testutil.NewDB(t)
	if _, err := db.Exec(`INSERT INTO seasons (name, start_date, end_date, is_active) VALUES (?, ?, ?, 0)`,
		"2024/25", "2024-09-01", "2025-06-30"); err != nil {
		t.Fatalf("insert season: %v", err)
	}
	var oldSeason int
	db.QueryRow(`SELECT id FROM seasons WHERE name='2024/25'`).Scan(&oldSeason)
	team := testutil.CreateTeam(t, db, "Alt")
	kader := testutil.CreateKader(t, db, team, oldSeason)
	u := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, u)
	testutil.AddKaderTrainer(t, db, kader, m)

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/team-groups", h.ListTeamGroups)
	})

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, u, "standard", nil))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	if len(groups) != 0 {
		t.Errorf("expected empty list for inactive-season-only trainer, got %+v", groups)
	}
}

// setupTwoTeams legt team1/team2 mit identischer Altersklasse+Geschlecht
// (Erwachsene/mixed, team_number 1 und 2) an → kanonische Kurzformen "gE1"/"gE2".
// Der Trainer von team1 sieht nur team1; displayShort MUSS trotzdem die Nummer
// behalten, weil saisonweit zwei Teams die Gruppe teilen.
func TestListTeamGroups_ShortNameKeepsNumberWhenSeasonHasMultiple(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, f.trainerU1, "standard", nil))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	if len(groups) == 0 {
		t.Fatalf("trainer should see own team groups, got none")
	}
	for _, g := range groups {
		if g.TeamID != f.team1 {
			t.Fatalf("trainer should only see team1, got team %d", g.TeamID)
		}
		if g.DisplayShort != "gE1" {
			t.Errorf("expected displayShort 'gE1' (season-wide disambiguation despite single visible team), got %q", g.DisplayShort)
		}
	}
}

// Bei genau einem Team der Altersklasse+Geschlecht entfällt die Team-Nummer.
func TestListTeamGroups_ShortNameOmitsNumberWhenTeamUnique(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Solo")
	kader := testutil.CreateKader(t, db, team, season)
	u := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, u)
	testutil.AddKaderTrainer(t, db, kader, m)
	// ein Spieler außer dem Caller, damit die Spieler-Gruppe count>0 hat
	pu := testutil.CreateUser(t, db, "standard")
	pm := testutil.CreateMember(t, db, pu)
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kader, pm); err != nil {
		t.Fatalf("insert kader_members: %v", err)
	}

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/team-groups", h.ListTeamGroups)
	})

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, u, "standard", nil))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	if len(groups) == 0 {
		t.Fatalf("trainer should see own team groups, got none")
	}
	for _, g := range groups {
		if g.DisplayShort != "gE" {
			t.Errorf("expected displayShort 'gE' (unique team, no number), got %q", g.DisplayShort)
		}
	}
}

func TestResolveTeamGroup_SpielerIncludesExtended(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv,
		"/api/chat/team-groups/"+itoa(f.team1)+"/spieler/members",
		testutil.Token(t, f.trainerU1, "standard", nil))
	members := decodeJSON[[]chat.TeamGroupMember](t, res)

	ids := map[int]bool{}
	for _, m := range members {
		ids[m.ID] = true
	}
	if !ids[f.playerU1] {
		t.Errorf("expected regular player %d in spieler resolve, got %+v", f.playerU1, members)
	}
	if !ids[f.extPlayerU1] {
		t.Errorf("expected extended player %d in spieler resolve, got %+v", f.extPlayerU1, members)
	}
}

func TestResolveTeamGroup_ElternIncludesBothSources(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv,
		"/api/chat/team-groups/"+itoa(f.team1)+"/eltern/members",
		testutil.Token(t, f.trainerU1, "standard", nil))
	members := decodeJSON[[]chat.TeamGroupMember](t, res)

	ids := map[int]bool{}
	for _, m := range members {
		ids[m.ID] = true
	}
	if !ids[f.parentU1] {
		t.Errorf("expected parent of regular player in resolve, got %+v", members)
	}
	if !ids[f.extParentU1] {
		t.Errorf("expected parent of extended player in resolve, got %+v", members)
	}
}

func TestResolveTeamGroup_ForeignTeamForbidden(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv,
		"/api/chat/team-groups/"+itoa(f.team2)+"/spieler/members",
		testutil.Token(t, f.playerU1, "standard", nil))
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for foreign team, got %d", res.StatusCode)
	}
}

func TestResolveTeamGroup_InvalidKindRejected(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv,
		"/api/chat/team-groups/"+itoa(f.team1)+"/foobar/members",
		testutil.Token(t, f.trainerU1, "standard", nil))
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid kind, got %d", res.StatusCode)
	}
}

func TestResolveTeamGroup_CallerExcluded(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv,
		"/api/chat/team-groups/"+itoa(f.team1)+"/trainer/members",
		testutil.Token(t, f.trainerU1, "standard", nil))
	members := decodeJSON[[]chat.TeamGroupMember](t, res)

	for _, m := range members {
		if m.ID == f.trainerU1 {
			t.Errorf("caller must not appear in own trainer resolve, got %+v", members)
		}
	}
}
