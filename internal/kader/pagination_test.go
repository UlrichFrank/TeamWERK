package kader_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/kader"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type kaderListResponse struct {
	Items []struct {
		ID       int    `json:"id"`
		AgeClass string `json:"age_class"`
	} `json:"items"`
	Total int `json:"total"`
}

// TestListKader_PaginationLimitOffset: ?limit=2&offset=0 liefert genau 2 items,
// total bleibt die Gesamtzahl; über offset sind die restlichen Kader erreichbar
// (disjunkte Ausschnitte derselben Gesamtmenge).
func TestListKader_PaginationLimitOffset(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	ageClasses := []string{"A-Jugend", "B-Jugend", "C-Jugend"}
	for i, ac := range ageClasses {
		teamID := testutil.CreateTeam(t, db, ac)
		if _, err := db.Exec(
			`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?,?,?,?,?)`,
			seasonID, ac, "mixed", teamID, i+1); err != nil {
			t.Fatalf("insert kader: %v", err)
		}
	}

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kader", h.ListKader)
	})

	resp := testutil.Get(t, srv, "/api/kader?limit=2&offset=0", token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var page1 kaderListResponse
	if err := json.NewDecoder(resp.Body).Decode(&page1); err != nil {
		t.Fatalf("decode page1: %v", err)
	}
	resp.Body.Close()

	if len(page1.Items) != 2 {
		t.Errorf("expected exactly 2 items on page 1, got %d", len(page1.Items))
	}
	if page1.Total != 3 {
		t.Errorf("expected total=3, got %d", page1.Total)
	}

	resp2 := testutil.Get(t, srv, "/api/kader?limit=2&offset=2", token)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for page 2, got %d", resp2.StatusCode)
	}
	var page2 kaderListResponse
	if err := json.NewDecoder(resp2.Body).Decode(&page2); err != nil {
		t.Fatalf("decode page2: %v", err)
	}
	resp2.Body.Close()

	if len(page2.Items) != 1 {
		t.Errorf("expected 1 item on page 2, got %d", len(page2.Items))
	}
	if page2.Total != 3 {
		t.Errorf("expected total=3 on page 2, got %d", page2.Total)
	}

	// Disjunkte Ausschnitte: kein Kader taucht auf beiden Seiten auf,
	// gemeinsam decken sie die Gesamtmenge ab.
	seen := map[int]bool{}
	for _, it := range append(page1.Items, page2.Items...) {
		if seen[it.ID] {
			t.Errorf("kader id=%d appears on multiple pages", it.ID)
		}
		seen[it.ID] = true
	}
	if len(seen) != 3 {
		t.Errorf("expected all 3 kader across pages, got %d", len(seen))
	}
}

// TestListKader_DefaultLimitApplied: ohne ?limit greift das Default-Limit (50),
// total bleibt die vollständige Gesamtzahl (hier > 50).
func TestListKader_DefaultLimitApplied(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Bulk")
	for i := 0; i < 55; i++ {
		if _, err := db.Exec(
			`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?,?,?,?,?)`,
			seasonID, "Erwachsene", "mixed", teamID, i+1); err != nil {
			t.Fatalf("insert kader %d: %v", i, err)
		}
	}

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kader", h.ListKader)
	})

	resp := testutil.Get(t, srv, "/api/kader", token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	var payload kaderListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(payload.Items) != 50 {
		t.Errorf("expected default limit of 50 items, got %d", len(payload.Items))
	}
	if payload.Total != 55 {
		t.Errorf("expected total=55, got %d", payload.Total)
	}
}

// Fehlerfall: ohne Token antwortet die Route mit 401.
func TestListKader_Unauthorized(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kader", h.ListKader)
	})

	resp := testutil.Get(t, srv, "/api/kader", "")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}
}
