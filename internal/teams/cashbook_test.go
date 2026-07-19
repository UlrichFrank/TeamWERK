package teams_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/teams"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// cashbookRoutes registriert alle Kassen-/Kassenwart-Routen für die Tests.
func cashbookRoutes(h *teams.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/api/teams/{id}/cashbook", h.ListCashbook)
		r.Post("/api/teams/{id}/cashbook", h.CreateCashbookEntry)
		r.Delete("/api/teams/{id}/cashbook/{entryId}", h.DeleteCashbookEntry)
		r.Get("/api/teams/{id}/treasurers", h.ListKassenwarte)
		r.Post("/api/teams/{id}/treasurers", h.AppointKassenwart)
		r.Delete("/api/teams/{id}/treasurers/{memberId}", h.RemoveKassenwart)
	}
}

type cashbookEntry struct {
	ID         int    `json:"id"`
	AmountCent int    `json:"amountCent"`
	Note       string `json:"note"`
}

type cashbookResp struct {
	Entries     []cashbookEntry `json:"entries"`
	BalanceCent int             `json:"balanceCent"`
	CanManage   bool            `json:"canManage"`
}

func decodeCashbook(t *testing.T, body []byte) cashbookResp {
	t.Helper()
	var c cashbookResp
	if err := json.Unmarshal(body, &c); err != nil {
		t.Fatalf("decode cashbook: %v (body=%s)", err, body)
	}
	return c
}

// --- Read-Gate ----------------------------------------------------------------

func TestCashbookRead_Player_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)
	testutil.AddKaderMember(t, db, kaderID, m)
	tok := testutil.Token(t, uid, "standard", []string{"spieler"})

	// Saldo aus +5000, +3000, -2000 = 6000.
	testutil.CreateCashbookEntry(t, db, kaderID, m, 5000, "Einzahlung A")
	testutil.CreateCashbookEntry(t, db, kaderID, m, 3000, "Einzahlung B")
	testutil.CreateCashbookEntry(t, db, kaderID, m, -2000, "Ausgabe C")

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("player read cashbook: want 200, got %d", res.StatusCode)
	}
	c := decodeCashbook(t, readBody(t, res))
	if c.BalanceCent != 6000 {
		t.Fatalf("balance: want 6000, got %d", c.BalanceCent)
	}
	if len(c.Entries) != 3 {
		t.Fatalf("entries: want 3, got %d", len(c.Entries))
	}
	if c.CanManage {
		t.Fatalf("plain player must not have canManage=true")
	}
}

func TestCashbookRead_Trainer_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	trUID := testutil.CreateUser(t, db, "standard")
	trM := testutil.CreateMember(t, db, trUID)
	testutil.AddKaderTrainer(t, db, kaderID, trM)
	tok := testutil.Token(t, trUID, "standard", []string{"trainer"})

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("trainer read cashbook: want 200, got %d", res.StatusCode)
	}
	c := decodeCashbook(t, readBody(t, res))
	if !c.CanManage {
		t.Fatalf("trainer must have canManage=true")
	}
}

func TestCashbookRead_Extended_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)
	testutil.AddExtendedKaderMember(t, db, kaderID, m)
	tok := testutil.Token(t, uid, "standard", nil)

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("extended member read cashbook: want 200, got %d", res.StatusCode)
	}
}

func TestCashbookRead_Parent_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	childUID := testutil.CreateUser(t, db, "standard")
	childM := testutil.CreateMember(t, db, childUID)
	testutil.AddKaderMember(t, db, kaderID, childM)

	parentUID := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentUID, childM)
	tok := testutil.TokenWithIsParent(t, parentUID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("parent read cashbook: want 403, got %d", res.StatusCode)
	}
}

func TestCashbookRead_Outsider_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, uid)
	tok := testutil.Token(t, uid, "standard", nil)

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("outsider read cashbook: want 403, got %d", res.StatusCode)
	}
}

// --- Write-Gate ---------------------------------------------------------------

func TestCashbookCreate_Trainer_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	trUID := testutil.CreateUser(t, db, "standard")
	trM := testutil.CreateMember(t, db, trUID)
	testutil.AddKaderTrainer(t, db, kaderID, trM)
	tok := testutil.Token(t, trUID, "standard", []string{"trainer"})

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook", tok,
		map[string]any{"amountCent": 5000, "note": "Startgeld Turnier"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("trainer create: want 201, got %d", res.StatusCode)
	}
}

func TestCashbookCreate_Kassenwart_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	kwUID := testutil.CreateUser(t, db, "standard")
	kwM := testutil.CreateMember(t, db, kwUID)
	testutil.AddKaderMember(t, db, kaderID, kwM)
	testutil.AppointKassenwart(t, db, kaderID, kwM)
	tok := testutil.Token(t, kwUID, "standard", nil)

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook", tok,
		map[string]any{"amountCent": -2500, "note": "Trainergeschenk"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("kassenwart create: want 201, got %d", res.StatusCode)
	}

	lres := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook", tok)
	defer lres.Body.Close()
	c := decodeCashbook(t, readBody(t, lres))
	if c.BalanceCent != -2500 {
		t.Fatalf("balance after expense: want -2500, got %d", c.BalanceCent)
	}
	if !c.CanManage {
		t.Fatalf("kassenwart must have canManage=true")
	}
}

func TestCashbookCreate_Spieler_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)
	testutil.AddKaderMember(t, db, kaderID, m)
	tok := testutil.Token(t, uid, "standard", []string{"spieler"})

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook", tok,
		map[string]any{"amountCent": 5000, "note": "unerlaubt"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("player create: want 403, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM team_cashbook_entries WHERE kader_id=?`, kaderID).Scan(&n)
	if n != 0 {
		t.Fatalf("no cashbook row expected, got %d", n)
	}
}

func TestCashbookCreate_ForeignTeamKassenwart_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	seasonID := testutil.CreateSeason(t, db, "2025/26")

	// Team A + dessen Kassenwart.
	teamA := testutil.CreateTeam(t, db, "A")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	kwUID := testutil.CreateUser(t, db, "standard")
	kwM := testutil.CreateMember(t, db, kwUID)
	testutil.AddKaderMember(t, db, kaderA, kwM)
	testutil.AppointKassenwart(t, db, kaderA, kwM)
	kwTok := testutil.Token(t, kwUID, "standard", nil)

	// Team B.
	teamB := testutil.CreateTeam(t, db, "B")
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamB)+"/cashbook", kwTok,
		map[string]any{"amountCent": 5000, "note": "Fremd"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign-team kassenwart create: want 403, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM team_cashbook_entries WHERE kader_id=?`, kaderB).Scan(&n)
	if n != 0 {
		t.Fatalf("no cashbook row expected, got %d", n)
	}
}

func TestCashbookDelete_Kassenwart_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	kwUID := testutil.CreateUser(t, db, "standard")
	kwM := testutil.CreateMember(t, db, kwUID)
	testutil.AddKaderMember(t, db, kaderID, kwM)
	testutil.AppointKassenwart(t, db, kaderID, kwM)
	tok := testutil.Token(t, kwUID, "standard", nil)

	entryID := testutil.CreateCashbookEntry(t, db, kaderID, kwM, 5000, "Einzahlung")

	res := testutil.Delete(t, srv, "/api/teams/"+itoa(teamID)+"/cashbook/"+itoa(entryID), tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete entry: want 204, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM team_cashbook_entries WHERE id=?`, entryID).Scan(&n)
	if n != 0 {
		t.Fatalf("entry must be gone after delete, still %d rows", n)
	}
}

// --- Kassenwart-Ernennung -----------------------------------------------------

func TestKassenwartAppoint_Trainer_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

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

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/treasurers", trTok,
		map[string]any{"memberId": plM})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("appoint kassenwart: want 201, got %d", res.StatusCode)
	}

	lres := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/treasurers", trTok)
	defer lres.Body.Close()
	var warte []struct {
		MemberID int `json:"memberId"`
	}
	json.Unmarshal(readBody(t, lres), &warte)
	found := false
	for _, w := range warte {
		if w.MemberID == plM {
			found = true
		}
	}
	if !found {
		t.Fatalf("appointed kassenwart %d not in list %+v", plM, warte)
	}
}

func TestKassenwartAppoint_NonTrainer_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, cashbookRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)
	plTok := testutil.Token(t, plUID, "standard", []string{"spieler"})

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/treasurers", plTok,
		map[string]any{"memberId": plM})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("non-trainer appoint: want 403, got %d", res.StatusCode)
	}
}

// TestCashbook_ExcludedFromRoster: die Roster-Response enthält keinerlei
// Kassendaten (Saldo, Einträge, Kassenwarte) — die Kasse lebt ausschließlich
// hinter ihrem eigenen Read-Gate. Analog zur Strafen-Ausschluss-Invariante.
func TestCashbook_ExcludedFromRoster(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) { r.Get("/api/teams/{id}/roster", h.GetRoster) })

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)
	testutil.AddKaderMember(t, db, kaderID, m)
	testutil.AppointKassenwart(t, db, kaderID, m)
	testutil.CreateCashbookEntry(t, db, kaderID, m, 5000, "Einzahlung")
	tok := testutil.Token(t, uid, "standard", []string{"spieler"})

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/roster", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("roster: want 200, got %d", res.StatusCode)
	}
	body := string(readBody(t, res))
	for _, needle := range []string{"balanceCent", "cashbook", "amountCent", "kassenwart", "Einzahlung"} {
		if contains2(body, needle) {
			t.Fatalf("roster response leaks cashbook data (%q): %s", needle, body)
		}
	}
}

// contains2 ist ein lokaler substring-Check (vermeidet Kollision mit contains aus
// responsibilities_test.go, das über []string arbeitet).
func contains2(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// TestClubFunctions_NoKassenwartValue beweist, dass 'kassenwart' KEINE globale
// Vereinsfunktion ist: der CHECK-Constraint auf member_club_functions.function
// muss den Wert ablehnen.
func TestClubFunctions_NoKassenwartValue(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)

	_, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'kassenwart')`, m)
	if err == nil {
		t.Fatalf("expected CHECK constraint to reject function='kassenwart', but insert succeeded")
	}
}
