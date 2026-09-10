package dutyfairness_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/dutyfairness"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

const ranglistePath = "/api/duty-fairness/rangliste"

func ranglisteServer(t *testing.T, db *sql.DB) *httptest.Server {
	h := dutyfairness.NewHandler(db)
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get(ranglistePath, h.Rangliste)
	})
}

type ranglisteRow struct {
	Rank       int     `json:"rank"`
	MemberID   *int    `json:"memberId"`
	Name       *string `json:"name"`
	IsOwn      bool    `json:"isOwn"`
	Geleistet  float64 `json:"geleistet"`
	Vorhersage float64 `json:"vorhersage"`
}

type ranglisteBody struct {
	Teams []struct {
		ID    int    `json:"id"`
		Label string `json:"label"`
	} `json:"teams"`
	Blocks []struct {
		TeamID int            `json:"teamId"`
		Soll   float64        `json:"soll"`
		Rows   []ranglisteRow `json:"rows"`
	} `json:"blocks"`
}

func getRangliste(t *testing.T, srv *httptest.Server, query, token string) (int, ranglisteBody) {
	t.Helper()
	res := testutil.Get(t, srv, ranglistePath+query, token)
	defer res.Body.Close()
	var body ranglisteBody
	if res.StatusCode == http.StatusOK {
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
	}
	return res.StatusCode, body
}

// scenario: Team A mit drei Spielern (x: 3 Dienste, das Kind des Elternteils:
// 2, y: 1), Team B mit zwei Spielern ohne Dienste.
type scenario struct {
	f               *fixture
	teamA, teamB    int
	kaderB          int
	parentUserID    int
	kidID, xID, yID int
	zID, wID        int
}

func newScenario(t *testing.T) *scenario {
	t.Helper()
	f := newFixture(t)
	s := &scenario{f: f}
	var kaderA int
	s.teamA, kaderA = f.team("A-Jugend")
	s.teamB, s.kaderB = f.team("B-Jugend")

	s.parentUserID = testutil.CreateUser(t, f.db, "standard")
	kidUser := testutil.CreateUser(t, f.db, "standard")
	xUser := testutil.CreateUser(t, f.db, "standard")
	yUser := testutil.CreateUser(t, f.db, "standard")
	s.kidID = f.player(kaderA, kidUser)
	testutil.AddFamilyLink(t, f.db, s.parentUserID, s.kidID)
	s.xID = f.player(kaderA, xUser)
	s.yID = f.player(kaderA, yUser)
	s.zID = f.player(s.kaderB, 0)
	s.wID = f.player(s.kaderB, 0)

	for i := 1; i <= 3; i++ {
		f.assign(f.teamSlot(s.teamA, day(-i), 1), xUser, "assigned")
	}
	f.assign(f.teamSlot(s.teamA, day(-1), 1), s.parentUserID, "assigned")
	f.assign(f.teamSlot(s.teamA, day(5), 1), kidUser, "assigned")
	f.assign(f.teamSlot(s.teamA, day(-4), 1), yUser, "assigned")
	return s
}

func parentToken(t *testing.T, s *scenario) string {
	return testutil.TokenWithIsParent(t, s.parentUserID, "standard", nil, true)
}

// Happy-Path Standard-Nutzer: nur das Team des eigenen Kindes im Scope, die
// Zeile des Kindes benannt, alle anderen nur mit Platzierung.
func TestRangliste_StandardNutzer_EigeneZeileBenanntRestNurPlatz(t *testing.T) {
	s := newScenario(t)
	srv := ranglisteServer(t, s.f.db)

	status, body := getRangliste(t, srv, "", parentToken(t, s))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(body.Teams) != 1 || body.Teams[0].ID != s.teamA {
		t.Fatalf("teams = %+v, want nur Team A", body.Teams)
	}
	if len(body.Blocks) != 1 || len(body.Blocks[0].Rows) != 3 {
		t.Fatalf("blocks = %+v, want ein Block mit 3 Zeilen", body.Blocks)
	}
	rows := body.Blocks[0].Rows
	for i, r := range rows {
		if r.Rank != i+1 {
			t.Errorf("row %d rank = %d, want %d", i, r.Rank, i+1)
		}
	}
	if rows[1].Name == nil || rows[1].MemberID == nil || *rows[1].MemberID != s.kidID || !rows[1].IsOwn {
		t.Errorf("Platz 2 = %+v, want eigenes Kind benannt", rows[1])
	}
	if rows[1].Geleistet != 1 || rows[1].Vorhersage != 1 {
		t.Errorf("Kind = %v/%v, want 1/1", rows[1].Geleistet, rows[1].Vorhersage)
	}
	for _, i := range []int{0, 2} {
		if rows[i].Name != nil || rows[i].MemberID != nil || rows[i].IsOwn {
			t.Errorf("Platz %d = %+v, want anonymisiert (kein Name, keine ID)", i+1, rows[i])
		}
	}
}

// Ein Spieler ohne family_links sieht sein eigenes Team mit der eigenen Zeile.
func TestRangliste_SpielerOhneEltern_SiehtEigeneZeile(t *testing.T) {
	s := newScenario(t)
	playerUser := testutil.CreateUser(t, s.f.db, "standard")
	own := s.f.player(s.kaderB, playerUser)
	srv := ranglisteServer(t, s.f.db)

	status, body := getRangliste(t, srv, "", testutil.Token(t, playerUser, "standard", []string{"spieler"}))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(body.Teams) != 1 || body.Teams[0].ID != s.teamB {
		t.Fatalf("teams = %+v, want nur Team B", body.Teams)
	}
	named := 0
	for _, r := range body.Blocks[0].Rows {
		if r.Name != nil {
			named++
			if *r.MemberID != own || !r.IsOwn {
				t.Errorf("benannte Zeile = %+v, want eigene", r)
			}
		}
	}
	if named != 1 {
		t.Errorf("benannte Zeilen = %d, want 1", named)
	}
}

// Fehlerfall: ein Team ohne eigene Verbindung ist für Standard-Nutzer 403 —
// allein oder in einer Mehrfachauswahl.
func TestRangliste_FremdesTeam_403(t *testing.T) {
	s := newScenario(t)
	srv := ranglisteServer(t, s.f.db)

	for _, q := range []string{
		fmt.Sprintf("?team=%d", s.teamB),
		fmt.Sprintf("?team=%d,%d", s.teamA, s.teamB),
	} {
		if status, _ := getRangliste(t, srv, q, parentToken(t, s)); status != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403", q, status)
		}
	}
}

// Vorstand: alle Teams wählbar, alle Zeilen benannt — auch ohne eigene Verbindung.
func TestRangliste_Vorstand_AlleTeamsUndAlleNamen(t *testing.T) {
	s := newScenario(t)
	vorstand := testutil.CreateUser(t, s.f.db, "standard")
	srv := ranglisteServer(t, s.f.db)

	status, body := getRangliste(t, srv, "", testutil.Token(t, vorstand, "standard", []string{"vorstand"}))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(body.Teams) != 2 || len(body.Blocks) != 2 {
		t.Fatalf("teams/blocks = %d/%d, want 2/2", len(body.Teams), len(body.Blocks))
	}
	for _, b := range body.Blocks {
		for _, r := range b.Rows {
			if r.Name == nil || r.MemberID == nil {
				t.Errorf("team %d rank %d unbenannt, want alle Namen für Vorstand", b.TeamID, r.Rank)
			}
		}
	}
}

// Mehrfachauswahl: ein eigener, für sich sortierter Block je Team, keine
// Vermischung; Blöcke in Scope-Reihenfolge, nicht in Query-Reihenfolge.
func TestRangliste_Mehrfachauswahl_GetrennteSortierteBloecke(t *testing.T) {
	s := newScenario(t)
	admin := testutil.CreateUser(t, s.f.db, "admin")
	srv := ranglisteServer(t, s.f.db)
	token := testutil.Token(t, admin, "admin", nil)

	status, body := getRangliste(t, srv, fmt.Sprintf("?team=%d,%d", s.teamB, s.teamA), token)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(body.Blocks) != 2 || body.Blocks[0].TeamID != s.teamA || body.Blocks[1].TeamID != s.teamB {
		t.Fatalf("blocks = %+v, want [A, B]", body.Blocks)
	}
	wantA := []int{s.xID, s.kidID, s.yID}
	for i, r := range body.Blocks[0].Rows {
		if *r.MemberID != wantA[i] {
			t.Errorf("A Platz %d = member %d, want %d (absteigend 3, 2, 1)", i+1, *r.MemberID, wantA[i])
		}
	}
	if len(body.Blocks[1].Rows) != 2 {
		t.Errorf("B rows = %d, want 2 (keine Vermischung mit A)", len(body.Blocks[1].Rows))
	}

	_, only := getRangliste(t, srv, fmt.Sprintf("?team=%d", s.teamB), token)
	if len(only.Blocks) != 1 || only.Blocks[0].TeamID != s.teamB || len(only.Teams) != 2 {
		t.Errorf("?team=B: blocks=%+v teams=%d, want nur Block B bei vollem Scope", only.Blocks, len(only.Teams))
	}
}

// Gleichstand: eindeutige Plätze, sekundär nach member_id aufsteigend.
func TestRangliste_Gleichstand_StabileEindeutigeReihenfolge(t *testing.T) {
	s := newScenario(t)
	vorstand := testutil.CreateUser(t, s.f.db, "standard")
	srv := ranglisteServer(t, s.f.db)
	token := testutil.Token(t, vorstand, "standard", []string{"vorstand"})

	for range 2 {
		_, body := getRangliste(t, srv, fmt.Sprintf("?team=%d", s.teamB), token)
		rows := body.Blocks[0].Rows
		if len(rows) != 2 || rows[0].Rank != 1 || rows[1].Rank != 2 {
			t.Fatalf("rows = %+v, want Plätze 1 und 2", rows)
		}
		if *rows[0].MemberID != s.zID || *rows[1].MemberID != s.wID {
			t.Errorf("Reihenfolge = %d, %d, want %d, %d (member_id aufsteigend)", *rows[0].MemberID, *rows[1].MemberID, s.zID, s.wID)
		}
	}
}

func TestRangliste_OhneAktiveSaison_Leer(t *testing.T) {
	db := testutil.NewDB(t)
	db.Exec(`UPDATE seasons SET is_active=0`)
	srv := ranglisteServer(t, db)
	user := testutil.CreateUser(t, db, "standard")

	status, body := getRangliste(t, srv, "", testutil.Token(t, user, "standard", nil))
	if status != http.StatusOK || len(body.Teams) != 0 || len(body.Blocks) != 0 {
		t.Errorf("status=%d body=%+v, want 200 mit leeren Listen", status, body)
	}
}

func TestRangliste_OhneAuth_401(t *testing.T) {
	srv := ranglisteServer(t, testutil.NewDB(t))
	if status, _ := getRangliste(t, srv, "", ""); status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
}
