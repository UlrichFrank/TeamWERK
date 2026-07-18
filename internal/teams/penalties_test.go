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

// penaltyRoutes registriert alle Strafen-/Strafenwart-Routen für die Tests.
func penaltyRoutes(h *teams.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/api/teams/{id}/penalties", h.ListPenalties)
		r.Post("/api/teams/{id}/penalties", h.CreatePenalty)
		r.Delete("/api/teams/{id}/penalties/{penaltyId}", h.DeletePenalty)
		// ResetMemberPenalties teilt sich den DELETE-Pfad mit DeletePenalty im
		// Router (?member=), wird in den Tests aber direkt aufgerufen.
		r.Get("/api/teams/{id}/penalty-types", h.ListPenaltyTypes)
		r.Post("/api/teams/{id}/penalty-types", h.CreatePenaltyType)
		r.Delete("/api/teams/{id}/penalty-types/{typeId}", h.DeletePenaltyType)
		r.Get("/api/teams/{id}/strafenwarte", h.ListStrafenwarte)
		r.Post("/api/teams/{id}/strafenwarte", h.AppointStrafenwart)
		r.Delete("/api/teams/{id}/strafenwarte/{memberId}", h.RemoveStrafenwart)
	}
}

// resetRoutes registriert den ResetMemberPenalties-Handler auf dem Collection-Pfad
// (DELETE /api/teams/{id}/penalties?member=…) — im echten Router liegt er dort.
func resetRoutes(h *teams.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/api/teams/{id}/penalties", h.ListPenalties)
		r.Post("/api/teams/{id}/penalties", h.CreatePenalty)
		r.Delete("/api/teams/{id}/penalties", h.ResetMemberPenalties)
	}
}

type penaltyEntry struct {
	ID         int    `json:"id"`
	MemberID   int    `json:"memberId"`
	MemberName string `json:"memberName"`
	AmountCent int    `json:"amountCent"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"createdAt"`
}

type penaltyTotal struct {
	MemberID   int    `json:"memberId"`
	MemberName string `json:"memberName"`
	TotalCent  int    `json:"totalCent"`
}

type penaltyList struct {
	Penalties []penaltyEntry `json:"penalties"`
	Totals    []penaltyTotal `json:"totals"`
	CanLevy   bool           `json:"canLevy"`
}

func decodePenaltyList(t *testing.T, body []byte) penaltyList {
	t.Helper()
	var pl penaltyList
	if err := json.Unmarshal(body, &pl); err != nil {
		t.Fatalf("decode penalty list: %v (body=%s)", err, body)
	}
	return pl
}

func totalFor(pl penaltyList, memberID int) int {
	for _, tot := range pl.Totals {
		if tot.MemberID == memberID {
			return tot.TotalCent
		}
	}
	return 0
}

// --- Read-Sichtbarkeit --------------------------------------------------------

func TestPenalties_Player_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)
	testutil.AddKaderMember(t, db, kaderID, m)
	tok := testutil.Token(t, uid, "standard", []string{"spieler"})

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("player read: want 200, got %d", res.StatusCode)
	}
	pl := decodePenaltyList(t, readBody(t, res))
	if pl.CanLevy {
		t.Fatalf("plain player must not have canLevy=true")
	}
}

func TestPenalties_ExtendedMember_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)
	testutil.AddExtendedKaderMember(t, db, kaderID, m)
	tok := testutil.Token(t, uid, "standard", nil)

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("extended member read: want 200, got %d", res.StatusCode)
	}
}

func TestPenalties_Parent_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	childUID := testutil.CreateUser(t, db, "standard")
	childM := testutil.CreateMember(t, db, childUID)
	testutil.AddKaderMember(t, db, kaderID, childM)

	parentUID := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentUID, childM)
	tok := testutil.TokenWithIsParent(t, parentUID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("parent read: want 403, got %d", res.StatusCode)
	}
}

func TestPenalties_Outsider_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, uid)
	tok := testutil.Token(t, uid, "standard", nil)

	res := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("outsider read: want 403, got %d", res.StatusCode)
	}
}

// --- Vergeben (Levy) ----------------------------------------------------------

func TestPenaltyCreate_Strafenwart_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	// Strafenwart.
	swUID := testutil.CreateUser(t, db, "standard")
	swM := testutil.CreateMember(t, db, swUID)
	testutil.AddKaderMember(t, db, kaderID, swM)
	testutil.AppointStrafenwart(t, db, kaderID, swM)
	swTok := testutil.Token(t, swUID, "standard", nil)

	// Ziel-Spieler.
	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", swTok,
		map[string]any{"memberId": plM, "amountCent": 500, "reason": "Zu spät"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("levy: want 201, got %d", res.StatusCode)
	}

	// In der Liste + Totals sichtbar.
	lres := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", swTok)
	defer lres.Body.Close()
	pl := decodePenaltyList(t, readBody(t, lres))
	if !pl.CanLevy {
		t.Fatalf("strafenwart must have canLevy=true")
	}
	if len(pl.Penalties) != 1 || pl.Penalties[0].AmountCent != 500 || pl.Penalties[0].Reason != "Zu spät" {
		t.Fatalf("penalty not in list as expected: %+v", pl.Penalties)
	}
	if got := totalFor(pl, plM); got != 500 {
		t.Fatalf("total for member: want 500, got %d", got)
	}
}

func TestPenaltyCreate_NonStrafenwart_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)
	plTok := testutil.Token(t, plUID, "standard", []string{"spieler"})

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", plTok,
		map[string]any{"memberId": plM, "amountCent": 500, "reason": "Zu spät"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("plain player levy: want 403, got %d", res.StatusCode)
	}
}

func TestPenaltyCreate_ForeignTeamStrafenwart_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	seasonID := testutil.CreateSeason(t, db, "2025/26")

	// Team A + dessen Strafenwart.
	teamA := testutil.CreateTeam(t, db, "A")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	swUID := testutil.CreateUser(t, db, "standard")
	swM := testutil.CreateMember(t, db, swUID)
	testutil.AddKaderMember(t, db, kaderA, swM)
	testutil.AppointStrafenwart(t, db, kaderA, swM)
	swTok := testutil.Token(t, swUID, "standard", nil)

	// Team B + ein Spieler dort.
	teamB := testutil.CreateTeam(t, db, "B")
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)
	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderB, plM)

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamB)+"/penalties", swTok,
		map[string]any{"memberId": plM, "amountCent": 500, "reason": "Fremd"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign-team strafenwart levy: want 403, got %d", res.StatusCode)
	}

	// Keine Row angelegt.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM team_penalties WHERE kader_id=?`, kaderB).Scan(&n)
	if n != 0 {
		t.Fatalf("no penalty row expected, got %d", n)
	}
}

// --- Storno + Reset -----------------------------------------------------------

func TestPenaltyStorno_Strafenwart_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	swUID := testutil.CreateUser(t, db, "standard")
	swM := testutil.CreateMember(t, db, swUID)
	testutil.AddKaderMember(t, db, kaderID, swM)
	testutil.AppointStrafenwart(t, db, kaderID, swM)
	swTok := testutil.Token(t, swUID, "standard", nil)

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)
	penaltyID := testutil.CreatePenalty(t, db, kaderID, plM, 500, "Zu spät", swM)

	res := testutil.Delete(t, srv, "/api/teams/"+itoa(teamID)+"/penalties/"+itoa(penaltyID), swTok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("storno: want 204, got %d", res.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM team_penalties WHERE id=?`, penaltyID).Scan(&n)
	if n != 0 {
		t.Fatalf("penalty must be gone after storno, still %d rows", n)
	}
}

func TestPenaltyReset_PerMember_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, resetRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	swUID := testutil.CreateUser(t, db, "standard")
	swM := testutil.CreateMember(t, db, swUID)
	testutil.AddKaderMember(t, db, kaderID, swM)
	testutil.AppointStrafenwart(t, db, kaderID, swM)
	swTok := testutil.Token(t, swUID, "standard", nil)

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)

	otherUID := testutil.CreateUser(t, db, "standard")
	otherM := testutil.CreateMember(t, db, otherUID)
	testutil.AddKaderMember(t, db, kaderID, otherM)

	// Zwei Strafen für plM, eine für otherM.
	testutil.CreatePenalty(t, db, kaderID, plM, 500, "A", swM)
	testutil.CreatePenalty(t, db, kaderID, plM, 300, "B", swM)
	testutil.CreatePenalty(t, db, kaderID, otherM, 200, "C", swM)

	res := testutil.Delete(t, srv, "/api/teams/"+itoa(teamID)+"/penalties?member="+itoa(plM), swTok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("reset: want 204, got %d", res.StatusCode)
	}

	lres := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", swTok)
	defer lres.Body.Close()
	pl := decodePenaltyList(t, readBody(t, lres))
	if got := totalFor(pl, plM); got != 0 {
		t.Fatalf("reset member total: want 0, got %d", got)
	}
	if got := totalFor(pl, otherM); got != 200 {
		t.Fatalf("other member untouched: want 200, got %d", got)
	}
}

func TestPenaltyReset_MissingMember_400(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, resetRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	swUID := testutil.CreateUser(t, db, "standard")
	swM := testutil.CreateMember(t, db, swUID)
	testutil.AddKaderMember(t, db, kaderID, swM)
	testutil.AppointStrafenwart(t, db, kaderID, swM)
	swTok := testutil.Token(t, swUID, "standard", nil)

	res := testutil.Delete(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", swTok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("reset without member: want 400, got %d", res.StatusCode)
	}
}

// --- Strafenwart-Ernennung ----------------------------------------------------

func TestStrafenwartAppoint_Trainer_200(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

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

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/strafenwarte", trTok,
		map[string]any{"memberId": plM})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("appoint: want 201, got %d", res.StatusCode)
	}

	// In der Liste sichtbar.
	lres := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/strafenwarte", trTok)
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
		t.Fatalf("appointed strafenwart %d not in list %+v", plM, warte)
	}
}

func TestStrafenwartAppoint_NonTrainer_403(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)
	plTok := testutil.Token(t, plUID, "standard", []string{"spieler"})

	res := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/strafenwarte", plTok,
		map[string]any{"memberId": plM})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("non-trainer appoint: want 403, got %d", res.StatusCode)
	}
}

// --- Snapshot + Rollen-Invariante --------------------------------------------

// TestPenalty_CatalogEditKeepsSnapshot: Löschen des penalty_types-Eintrags ändert
// eine bereits vergebene team_penalties-Row nicht (Snapshot-Invariante).
func TestPenalty_CatalogEditKeepsSnapshot(t *testing.T) {
	db := testutil.NewDB(t)
	h := teams.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, penaltyRoutes(h))

	teamID := testutil.CreateTeam(t, db, "H1")
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	trUID := testutil.CreateUser(t, db, "standard")
	trM := testutil.CreateMember(t, db, trUID)
	testutil.AddKaderTrainer(t, db, kaderID, trM)
	trTok := testutil.Token(t, trUID, "standard", []string{"trainer"})

	swUID := testutil.CreateUser(t, db, "standard")
	swM := testutil.CreateMember(t, db, swUID)
	testutil.AddKaderMember(t, db, kaderID, swM)
	testutil.AppointStrafenwart(t, db, kaderID, swM)
	swTok := testutil.Token(t, swUID, "standard", nil)

	plUID := testutil.CreateUser(t, db, "standard")
	plM := testutil.CreateMember(t, db, plUID)
	testutil.AddKaderMember(t, db, kaderID, plM)

	// Catalog-Eintrag "X" (Betrag 500) anlegen.
	typeID := testutil.AddPenaltyType(t, db, kaderID, "X", 500)

	// Strafe mit reason "X"/500 vergeben.
	lres := testutil.Post(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", swTok,
		map[string]any{"memberId": plM, "amountCent": 500, "reason": "X"})
	lres.Body.Close()

	// Catalog-Eintrag "X" löschen (durch Trainer).
	dres := testutil.Delete(t, srv, "/api/teams/"+itoa(teamID)+"/penalty-types/"+itoa(typeID), trTok)
	dres.Body.Close()

	// Vergebene Strafe bleibt unverändert in der Liste.
	gres := testutil.Get(t, srv, "/api/teams/"+itoa(teamID)+"/penalties", swTok)
	defer gres.Body.Close()
	pl := decodePenaltyList(t, readBody(t, gres))
	if len(pl.Penalties) != 1 || pl.Penalties[0].Reason != "X" || pl.Penalties[0].AmountCent != 500 {
		t.Fatalf("snapshot lost after catalog delete: %+v", pl.Penalties)
	}
}

// TestClubFunctions_NoStrafenwartValue beweist, dass 'strafenwart' KEINE globale
// Vereinsfunktion ist: der CHECK-Constraint auf member_club_functions.function
// muss den Wert ablehnen.
func TestClubFunctions_NoStrafenwartValue(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	m := testutil.CreateMember(t, db, uid)

	_, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'strafenwart')`, m)
	if err == nil {
		t.Fatalf("expected CHECK constraint to reject function='strafenwart', but insert succeeded")
	}
}
