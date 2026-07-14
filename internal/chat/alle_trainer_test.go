package chat_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func alleTrainerEntry(groups []chat.TeamGroup) *chat.TeamGroup {
	for i := range groups {
		if groups[i].Kind == "alle_trainer" {
			return &groups[i]
		}
	}
	return nil
}

func memberIDs(members []chat.TeamGroupMember) map[int]bool {
	ids := map[int]bool{}
	for _, m := range members {
		ids[m.ID] = true
	}
	return ids
}

// --- 5.1: Listing-Sichtbarkeit der "Alle Trainer"-Kachel ---

func TestListTeamGroups_AlleTrainer_TrainerSiehtKachel(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, f.trainerU1, "standard", nil))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	e := alleTrainerEntry(groups)
	if e == nil {
		t.Fatalf("trainer should see 'Alle Trainer' tile, got %+v", groups)
	}
	if e.TeamID != 0 {
		t.Errorf("expected teamId 0, got %d", e.TeamID)
	}
	if e.DisplayShort != "Alle Trainer" {
		t.Errorf("expected displayShort 'Alle Trainer', got %q", e.DisplayShort)
	}
	// Trainerkreis-Inhalt = beide Trainer; Caller (trainerU1) ausgeschlossen → 1.
	if e.Count != 1 {
		t.Errorf("expected count 1 (trainerU2, caller excluded), got %d", e.Count)
	}
}

func TestListTeamGroups_AlleTrainer_VorstandSiehtKachel(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	outsider := testutil.CreateUser(t, f.db, "standard")

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, outsider, "standard", []string{"vorstand"}))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	e := alleTrainerEntry(groups)
	if e == nil {
		t.Fatalf("vorstand should see 'Alle Trainer' tile, got %+v", groups)
	}
	// Reiner Vorstand ist kein Trainer → count = beide Trainer.
	if e.Count != 2 {
		t.Errorf("expected count 2 (both trainers), got %d", e.Count)
	}
}

func TestListTeamGroups_AlleTrainer_SportlicheLeitungSiehtKachel(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	outsider := testutil.CreateUser(t, f.db, "standard")

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, outsider, "standard", []string{"sportliche_leitung"}))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	if alleTrainerEntry(groups) == nil {
		t.Fatalf("sportliche_leitung should see 'Alle Trainer' tile, got %+v", groups)
	}
}

func TestListTeamGroups_AlleTrainer_BeisitzerSiehtKachel(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	outsider := testutil.CreateUser(t, f.db, "standard")

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, outsider, "standard", []string{"vorstand_beisitzer"}))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	if alleTrainerEntry(groups) == nil {
		t.Fatalf("vorstand_beisitzer should see 'Alle Trainer' tile, got %+v", groups)
	}
}

func TestListTeamGroups_AlleTrainer_SpielerSiehtNicht(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv, "/api/chat/team-groups",
		testutil.Token(t, f.playerU1, "standard", nil))
	groups := decodeJSON[[]chat.TeamGroup](t, res)

	if alleTrainerEntry(groups) != nil {
		t.Errorf("player must not see 'Alle Trainer' tile, got %+v", groups)
	}
}

// --- 5.1: Auflösung der "Alle Trainer"-Gruppe ---

func TestResolveTeamGroup_AlleTrainer_Teamuebergreifend(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	outsider := testutil.CreateUser(t, f.db, "standard")

	res := testutil.Get(t, srv, "/api/chat/team-groups/0/alle_trainer/members",
		testutil.Token(t, outsider, "standard", []string{"vorstand"}))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	ids := memberIDs(decodeJSON[[]chat.TeamGroupMember](t, res))
	if !ids[f.trainerU1] || !ids[f.trainerU2] {
		t.Errorf("expected trainers of both teams, got %+v", ids)
	}
}

func TestResolveTeamGroup_AlleTrainer_ReinerVorstandNichtImErgebnis(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	// Reiner Vorstand (member_club_functions), kein Kader-Trainer.
	vu := testutil.CreateUser(t, f.db, "standard")
	vm := testutil.CreateMember(t, f.db, vu)
	testutil.AddClubFunction(t, f.db, vm, "vorstand")

	res := testutil.Get(t, srv, "/api/chat/team-groups/0/alle_trainer/members",
		testutil.Token(t, f.trainerU1, "standard", nil))
	ids := memberIDs(decodeJSON[[]chat.TeamGroupMember](t, res))

	if ids[vu] {
		t.Errorf("pure vorstand must NOT be in members (content = trainers only), got %+v", ids)
	}
	if !ids[f.trainerU2] {
		t.Errorf("expected trainerU2 in members, got %+v", ids)
	}
}

func TestResolveTeamGroup_AlleTrainer_VorstandDarfAuflösen(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	outsider := testutil.CreateUser(t, f.db, "standard")

	res := testutil.Get(t, srv, "/api/chat/team-groups/0/alle_trainer/members",
		testutil.Token(t, outsider, "standard", []string{"vorstand"}))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("vorstand should be allowed to resolve, got %d", res.StatusCode)
	}
}

func TestResolveTeamGroup_AlleTrainer_NichtKreisForbidden(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv, "/api/chat/team-groups/0/alle_trainer/members",
		testutil.Token(t, f.playerU1, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("player must get 403, got %d", res.StatusCode)
	}
}

func TestResolveTeamGroup_AlleTrainer_CallerExcluded(t *testing.T) {
	f, mux := setupTwoTeams(t)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	res := testutil.Get(t, srv, "/api/chat/team-groups/0/alle_trainer/members",
		testutil.Token(t, f.trainerU1, "standard", nil))
	ids := memberIDs(decodeJSON[[]chat.TeamGroupMember](t, res))

	if ids[f.trainerU1] {
		t.Errorf("caller must not appear in own resolve, got %+v", ids)
	}
}

// --- 5.2: Kontaktierbarkeit + Nutzersuche ---

func newContactServer(t *testing.T, db *sql.DB) func(r chi.Router) {
	t.Helper()
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	return func(r chi.Router) {
		r.Post("/api/chat/conversations", h.CreateConversation)
		r.Get("/api/chat/users", h.Users)
	}
}

func TestCreateDirect_TrainerZuTeamfremdemTrainer_Erlaubt(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))

	res := testutil.Post(t, srv, "/api/chat/conversations",
		testutil.Token(t, f.trainerU1, "standard", nil),
		map[string]any{"type": "direct", "userId": f.trainerU2})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Errorf("trainer↔teamfremder trainer should be allowed, got %d", res.StatusCode)
	}
}

func TestCreateDirect_SportlicheLeitungZuTrainer_Erlaubt(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))
	sl := testutil.CreateUser(t, f.db, "standard")

	res := testutil.Post(t, srv, "/api/chat/conversations",
		testutil.Token(t, sl, "standard", []string{"sportliche_leitung"}),
		map[string]any{"type": "direct", "userId": f.trainerU1})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Errorf("sportliche_leitung → trainer should be allowed, got %d", res.StatusCode)
	}
}

func TestCreateDirect_SpielerZuTeamfremdemTrainer_Forbidden(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))

	res := testutil.Post(t, srv, "/api/chat/conversations",
		testutil.Token(t, f.playerU1, "standard", nil),
		map[string]any{"type": "direct", "userId": f.trainerU2})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("player → teamfremder trainer must be 403, got %d", res.StatusCode)
	}
}

func TestChatUsers_TrainerFindetTeamfremdenTrainer(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))

	res := testutil.Get(t, srv, "/api/chat/users",
		testutil.Token(t, f.trainerU1, "standard", nil))
	type row struct {
		ID int `json:"id"`
	}
	rows := decodeJSON[[]row](t, res)
	found := false
	for _, r := range rows {
		if r.ID == f.trainerU2 {
			found = true
		}
	}
	if !found {
		t.Errorf("trainer should find teamfremden trainer %d in search, got %+v", f.trainerU2, rows)
	}
}

func TestChatUsers_SpielerFindetTeamfremdenTrainerNicht(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))

	res := testutil.Get(t, srv, "/api/chat/users",
		testutil.Token(t, f.playerU1, "standard", nil))
	type row struct {
		ID int `json:"id"`
	}
	rows := decodeJSON[[]row](t, res)
	for _, r := range rows {
		if r.ID == f.trainerU2 {
			t.Errorf("player must NOT find teamfremden trainer %d, got %+v", f.trainerU2, rows)
		}
	}
}
