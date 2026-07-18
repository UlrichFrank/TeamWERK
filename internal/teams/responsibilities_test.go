package teams_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/teams"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func itoa(n int) string { return strconv.Itoa(n) }

func readBody(t *testing.T, res *http.Response) []byte {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return b
}

func respRoutes(h *teams.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/api/teams/{id}/roster", h.GetRoster)
		r.Get("/api/teams/{id}/responsibility-types", h.ListResponsibilityTypes)
		r.Post("/api/teams/{id}/responsibility-types", h.CreateResponsibilityType)
		r.Delete("/api/teams/{id}/responsibility-types/{typeId}", h.DeleteResponsibilityType)
		r.Post("/api/teams/{id}/responsibilities", h.CreateResponsibility)
		r.Delete("/api/teams/{id}/responsibilities/{respId}", h.DeleteResponsibility)
	}
}

// rosterPlayerLabels dekodiert die Roster-Response und liefert die Responsibilities
// des Spielers mit der gegebenen memberId (aus players + extended_players).
func rosterPlayerLabels(t *testing.T, body []byte, memberID int) []string {
	t.Helper()
	type respEntry struct {
		ID    int    `json:"id"`
		Label string `json:"label"`
	}
	var resp struct {
		Players []struct {
			MemberID         int         `json:"memberId"`
			Responsibilities []respEntry `json:"responsibilities"`
		} `json:"players"`
		ExtendedPlayers []struct {
			MemberID         int         `json:"memberId"`
			Responsibilities []respEntry `json:"responsibilities"`
		} `json:"extended_players"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode roster: %v", err)
	}
	collect := func(items []respEntry) []string {
		labels := []string{}
		for _, it := range items {
			labels = append(labels, it.Label)
		}
		return labels
	}
	for _, p := range resp.Players {
		if p.MemberID == memberID {
			return collect(p.Responsibilities)
		}
	}
	for _, p := range resp.ExtendedPlayers {
		if p.MemberID == memberID {
			return collect(p.Responsibilities)
		}
	}
	return nil
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func TestResponsibilityCatalog_TrainerCreates_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, respRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	trUID := testutil.CreateUser(t, db, "standard")
	trM := testutil.CreateMember(t, db, trUID)
	testutil.AddKaderTrainer(t, db, kaderID, trM)
	tok := testutil.Token(t, trUID, "standard", []string{"trainer"})

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/responsibility-types", tok, map[string]string{"label": "Harz"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d", res.StatusCode)
	}
}

func TestResponsibilityCatalog_NonTrainer_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, respRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)
	tok := testutil.Token(t, plUID, "standard", []string{"spieler"})

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/responsibility-types", tok, map[string]string{"label": "Harz"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403, got %d", res.StatusCode)
	}
}

func TestResponsibilities_Unauthenticated_401(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, respRoutes(h))
	teamID := testutil.CreateTeam(t, db, "H1")

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/responsibility-types", "", map[string]string{"label": "Harz"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", res.StatusCode)
	}
}

func TestResponsibilityAssign_Trainer_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, respRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	trUID := testutil.CreateUser(t, db, "standard")
	trM := testutil.CreateMember(t, db, trUID)
	testutil.AddKaderTrainer(t, db, kaderID, trM)
	trTok := testutil.Token(t, trUID, "standard", []string{"trainer"})

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)
	plTok := testutil.Token(t, plUID, "standard", []string{"spieler"})

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/responsibilities", trTok,
		map[string]any{"memberId": plM, "label": "Mannschaftskasse"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("assign: want 201, got %d", res.StatusCode)
	}

	rres := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/roster", plTok)
	defer rres.Body.Close()
	if rres.StatusCode != http.StatusOK {
		t.Fatalf("roster: want 200, got %d", rres.StatusCode)
	}
	labels := rosterPlayerLabels(t, readBody(t, rres), plM)
	if !contains(labels, "Mannschaftskasse") {
		t.Fatalf("player responsibilities %v missing 'Mannschaftskasse'", labels)
	}
}

// TestRoster_IncludesResponsibilities: auch ein Elternteil (Roster-Sichtbarkeit)
// sieht die Aufgaben des Kindes.
func TestRoster_IncludesResponsibilities(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, respRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	trUID := testutil.CreateUser(t, db, "standard")
	trM := testutil.CreateMember(t, db, trUID)
	testutil.AddKaderTrainer(t, db, kaderID, trM)
	trTok := testutil.Token(t, trUID, "standard", []string{"trainer"})

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)

	parentUID := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentUID, plM)
	parentTok := testutil.TokenWithIsParent(t, parentUID, "standard", nil, true)

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/responsibilities", trTok,
		map[string]any{"memberId": plM, "label": "Leibchen"})
	res.Body.Close()

	rres := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/roster", parentTok)
	defer rres.Body.Close()
	if rres.StatusCode != http.StatusOK {
		t.Fatalf("parent roster: want 200, got %d", rres.StatusCode)
	}
	labels := rosterPlayerLabels(t, readBody(t, rres), plM)
	if !contains(labels, "Leibchen") {
		t.Fatalf("parent-visible responsibilities %v missing 'Leibchen'", labels)
	}
}

// TestResponsibility_CatalogEditKeepsSnapshot: Löschen des Catalog-Eintrags ändert
// eine bereits vergebene Zuweisung nicht (Snapshot-Invariante).
func TestResponsibility_CatalogEditKeepsSnapshot(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, respRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	trUID := testutil.CreateUser(t, db, "standard")
	trM := testutil.CreateMember(t, db, trUID)
	testutil.AddKaderTrainer(t, db, kaderID, trM)
	trTok := testutil.Token(t, trUID, "standard", []string{"trainer"})

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)
	plTok := testutil.Token(t, plUID, "standard", []string{"spieler"})

	// Catalog-Eintrag anlegen und ID lesen.
	cres := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/responsibility-types", trTok, map[string]string{"label": "Harz"})
	var created struct {
		ID int `json:"id"`
	}
	json.Unmarshal(readBody(t, cres), &created)
	cres.Body.Close()

	// Zuweisung "Harz".
	ares := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/responsibilities", trTok, map[string]any{"memberId": plM, "label": "Harz"})
	ares.Body.Close()

	// Catalog-Eintrag löschen.
	dres := testutil.Delete(t, srv, "/api/teams/"+itoa(teamID)+"/responsibility-types/"+itoa(created.ID), trTok)
	dres.Body.Close()

	// Zuweisung muss bestehen bleiben.
	rres := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/roster", plTok)
	defer rres.Body.Close()
	labels := rosterPlayerLabels(t, readBody(t, rres), plM)
	if !contains(labels, "Harz") {
		t.Fatalf("snapshot lost: player responsibilities %v missing 'Harz' after catalog delete", labels)
	}
}
