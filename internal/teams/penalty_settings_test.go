package teams_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/teams"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// penaltySettingsRoutes registriert Einheiten- + Vergabe-Routen für die Tests.
func penaltySettingsRoutes(h *teams.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/api/teams/{id}/penalty-settings", h.GetPenaltySettings)
		r.Get("/api/teams/{id}/penalty-settings/preview", h.PreviewPenaltySettings)
		r.Put("/api/teams/{id}/penalty-settings", h.SetPenaltySettings)
		r.Post("/api/teams/{id}/penalties", h.CreatePenalty)
		r.Post("/api/teams/{id}/penalty-types", h.CreatePenaltyType)
	}
}

func TestPenaltySettings_Read_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltySettingsRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)
	testutil.AddKaderMember(t, db, kaderID, m)
	tok := testutil.Token(t, uid, "standard", []string{"spieler"})

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/penalty-settings", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("read settings: want 200, got %d", res.StatusCode)
	}
	var s struct {
		Unit string `json:"unit"`
	}
	json.Unmarshal(readBody(t, res), &s)
	if s.Unit != "euro" {
		t.Fatalf("default unit: want euro, got %q", s.Unit)
	}
}

func TestPenaltySettings_Trainer_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltySettingsRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	trUID := testutil.CreateUser(t, db, "standard")
	trM := testutil.CreateMember(t, db, trUID)
	testutil.AddKaderTrainer(t, db, kaderID, trM)
	trTok := testutil.Token(t, trUID, "standard", []string{"trainer"})

	res := testutil.Put(t, srv, "/api/teams/"+itoa(teamID)+"/penalty-settings", trTok,
		map[string]any{"unit": "striche"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("trainer set unit: want 200, got %d", res.StatusCode)
	}
	var unit string
	db.QueryRow(`SELECT unit FROM penalty_settings WHERE kader_id=?`, kaderID).Scan(&unit)
	if unit != "striche" {
		t.Fatalf("unit not persisted: got %q", unit)
	}
}

func TestPenaltySettings_NonTrainer_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltySettingsRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)
	testutil.AddKaderMember(t, db, kaderID, m)
	tok := testutil.Token(t, uid, "standard", []string{"spieler"})

	res := testutil.Put(t, srv, "/api/teams/"+itoa(teamID)+"/penalty-settings", tok,
		map[string]any{"unit": "striche"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("non-trainer set unit: want 403, got %d", res.StatusCode)
	}
}

func TestPenaltySettings_EurToStriche_RoundsUpAndConverts(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltySettingsRoutes(h))

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

	// 5,50 € (aufrunden → 6 Striche = 600) und 5,00 € (exakt → 5 Striche = 500).
	p1 := testutil.CreatePenalty(t, db, kaderID, plM, 550, "A", trM)
	p2 := testutil.CreatePenalty(t, db, kaderID, plM, 500, "B", trM)
	typeID := testutil.AddPenaltyType(t, db, kaderID, "T", 550)

	res := testutil.Put(t, srv, "/api/teams/"+itoa(teamID)+"/penalty-settings", trTok,
		map[string]any{"unit": "striche"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("convert: want 200, got %d", res.StatusCode)
	}

	var a1, a2, at int
	db.QueryRow(`SELECT amount_cent FROM team_penalties WHERE id=?`, p1).Scan(&a1)
	db.QueryRow(`SELECT amount_cent FROM team_penalties WHERE id=?`, p2).Scan(&a2)
	db.QueryRow(`SELECT default_amount_cent FROM penalty_types WHERE id=?`, typeID).Scan(&at)
	if a1 != 600 {
		t.Fatalf("550 → want 600 (6 Striche), got %d", a1)
	}
	if a2 != 500 {
		t.Fatalf("500 → want 500 (5 Striche), got %d", a2)
	}
	if at != 600 {
		t.Fatalf("catalog 550 → want 600, got %d", at)
	}
}

func TestPenaltySettings_StricheToEur_ExactConversion(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltySettingsRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.SetPenaltyUnit(t, db, kaderID, "striche")

	trUID := testutil.CreateUser(t, db, "standard")
	trM := testutil.CreateMember(t, db, trUID)
	testutil.AddKaderTrainer(t, db, kaderID, trM)
	trTok := testutil.Token(t, trUID, "standard", []string{"trainer"})

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)

	// 6 Striche = 600 Cent → soll 6,00 € = 600 Cent bleiben (verlustfrei).
	p := testutil.CreatePenalty(t, db, kaderID, plM, 600, "A", trM)

	res := testutil.Put(t, srv, "/api/teams/"+itoa(teamID)+"/penalty-settings", trTok,
		map[string]any{"unit": "euro"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("convert: want 200, got %d", res.StatusCode)
	}
	var a int
	db.QueryRow(`SELECT amount_cent FROM team_penalties WHERE id=?`, p).Scan(&a)
	if a != 600 {
		t.Fatalf("6 Striche → want 600 (6,00 €), got %d", a)
	}
}

func TestPenaltySettings_Preview_NoMutation(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltySettingsRoutes(h))

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
	p := testutil.CreatePenalty(t, db, kaderID, plM, 550, "A", trM)

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/penalty-settings/preview?to=striche", trTok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("preview: want 200, got %d", res.StatusCode)
	}
	var pv struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Affected  int    `json:"affected"`
		RoundedUp int    `json:"roundedUp"`
		Penalties []struct {
			OldAmount int `json:"oldAmount"`
			NewAmount int `json:"newAmount"`
		} `json:"penalties"`
	}
	json.Unmarshal(readBody(t, res), &pv)
	if pv.From != "euro" || pv.To != "striche" {
		t.Fatalf("preview from/to: got %q/%q", pv.From, pv.To)
	}
	if pv.Affected != 1 || pv.RoundedUp != 1 {
		t.Fatalf("preview affected/roundedUp: want 1/1, got %d/%d", pv.Affected, pv.RoundedUp)
	}
	if len(pv.Penalties) != 1 || pv.Penalties[0].OldAmount != 550 || pv.Penalties[0].NewAmount != 600 {
		t.Fatalf("preview delta wrong: %+v", pv.Penalties)
	}

	// DB unverändert.
	var a int
	db.QueryRow(`SELECT amount_cent FROM team_penalties WHERE id=?`, p).Scan(&a)
	if a != 550 {
		t.Fatalf("preview mutated DB: want 550, got %d", a)
	}
}

func TestPenaltyCreate_StricheUnit_NonInteger_400(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltySettingsRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.SetPenaltyUnit(t, db, kaderID, "striche")

	swUID := testutil.CreateUser(t, db, "standard")
	swM := testutil.CreateMember(t, db, swUID)
	testutil.AddKaderMember(t, db, kaderID, swM)
	testutil.AppointStrafenwart(t, db, kaderID, swM)
	swTok := testutil.Token(t, swUID, "standard", nil)

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)

	// 2,5 Striche (250 Cent) — nicht durch 100 teilbar → 400.
	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", swTok,
		map[string]any{"memberId": plM, "amountCent": 250, "reason": "krumm"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("non-integer striche: want 400, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM team_penalties WHERE kader_id=?`, kaderID).Scan(&n)
	if n != 0 {
		t.Fatalf("no penalty row expected, got %d", n)
	}

	// 3 Striche (300 Cent) — zulässig → 201.
	ok := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", swTok,
		map[string]any{"memberId": plM, "amountCent": 300, "reason": "ganz"})
	defer ok.Body.Close()
	if ok.StatusCode != http.StatusCreated {
		t.Fatalf("whole striche: want 201, got %d", ok.StatusCode)
	}
}
