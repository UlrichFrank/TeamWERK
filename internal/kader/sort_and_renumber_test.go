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

type sortListResponse struct {
	Items []struct {
		ID         int    `json:"id"`
		AgeClass   string `json:"age_class"`
		Gender     string `json:"gender"`
		TeamNumber int    `json:"team_number"`
	} `json:"items"`
	Total int `json:"total"`
}

// ── TC-K10: kanonische Sortierreihenfolge (Entscheidung 7) ───────────────────

// TestListKader_CanonicalOrdering verifies A–D-Jugend sort alphabetically first,
// then training-group categories by training_group_categories.sort_order
// (Perspektivkader before Förderkader) — NOT alphabetically, which would put
// "Förderkader" before "Perspektivkader". Secondary team_number order holds
// within a category, and the A–D order is unchanged.
func TestListKader_CanonicalOrdering(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")

	// Deliberately insert in an order that is neither the expected output order
	// nor alphabetical, so a passing assertion proves the ORDER BY did the work.
	type spec struct {
		ageClass   string
		gender     string
		teamNumber int
		birthYear  int
	}
	specs := []spec{
		{"Förderkader", "mixed", 2, 2017},
		{"Förderkader", "mixed", 1, 2016},
		{"Perspektivkader", "mixed", 1, 2015},
		{"D-Jugend", "mixed", 1, 0},
		{"A-Jugend", "m", 1, 0},
		{"C-Jugend", "m", 1, 0},
		{"B-Jugend", "m", 1, 0},
	}
	for _, s := range specs {
		teamID := testutil.CreateTeam(t, db, s.ageClass+" "+s.gender)
		var by any
		if s.birthYear > 0 {
			by = s.birthYear
		}
		if _, err := db.Exec(
			`INSERT INTO kader (season_id, age_class, gender, team_id, team_number, dedicated_birth_year) VALUES (?,?,?,?,?,?)`,
			seasonID, s.ageClass, s.gender, teamID, s.teamNumber, by); err != nil {
			t.Fatalf("insert kader %s: %v", s.ageClass, err)
		}
	}

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kader", h.ListKader)
	})

	resp := testutil.Get(t, srv, "/api/kader?season_id=1&limit=50", token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out sortListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	resp.Body.Close()

	// Expected canonical order: A,B,C,D (alphabetical block 0), then
	// Perspektivkader (sort_order 1), then Förderkader (sort_order 2) with
	// team_number ascending inside the category.
	type want struct {
		ageClass   string
		teamNumber int
	}
	expected := []want{
		{"A-Jugend", 1},
		{"B-Jugend", 1},
		{"C-Jugend", 1},
		{"D-Jugend", 1},
		{"Perspektivkader", 1},
		{"Förderkader", 1},
		{"Förderkader", 2},
	}
	if len(out.Items) != len(expected) {
		t.Fatalf("expected %d items, got %d", len(expected), len(out.Items))
	}
	for i, e := range expected {
		got := out.Items[i]
		if got.AgeClass != e.ageClass || got.TeamNumber != e.teamNumber {
			t.Errorf("position %d: expected %s#%d, got %s#%d",
				i, e.ageClass, e.teamNumber, got.AgeClass, got.TeamNumber)
		}
	}
}

// ── TC-K11: team_number folgt Jahrgang bei Anlage (Task 5.4) ─────────────────

// TestCreateTrainingGroupKader_RenumberByBirthYear verifies that creating
// training-group kader in reverse birth-year order still yields team_number
// following the ascending birth year (older year → lower number), and that the
// team_id is repointed to the correctly numbered team.
func TestCreateTrainingGroupKader_RenumberByBirthYear(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/kader", h.InitializeKader)
	})

	// Create in a deliberately unsorted birth-year order.
	for _, year := range []int{2017, 2015, 2016} {
		resp := testutil.Post(t, srv, "/api/kader", token, map[string]any{
			"season_id":            seasonID,
			"age_class":            "Förderkader",
			"gender":               "mixed",
			"dedicated_birth_year": year,
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create Förderkader %d: expected 201, got %d", year, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// team_number must follow ascending birth year regardless of creation order.
	expected := map[int]int{2015: 1, 2016: 2, 2017: 3}
	for year, wantNum := range expected {
		var gotNum int
		if err := db.QueryRow(
			`SELECT team_number FROM kader WHERE season_id=? AND age_class='Förderkader' AND gender='mixed' AND dedicated_birth_year=?`,
			seasonID, year).Scan(&gotNum); err != nil {
			t.Fatalf("query team_number for %d: %v", year, err)
		}
		if gotNum != wantNum {
			t.Errorf("birth year %d: expected team_number=%d, got %d", year, wantNum, gotNum)
		}
	}

	// team_id must be repointed to the correctly numbered team (name suffix).
	var name1, name3 string
	db.QueryRow(`SELECT tm.name FROM kader k JOIN teams tm ON tm.id=k.team_id
	             WHERE k.age_class='Förderkader' AND k.dedicated_birth_year=2015`).Scan(&name1)
	db.QueryRow(`SELECT tm.name FROM kader k JOIN teams tm ON tm.id=k.team_id
	             WHERE k.age_class='Förderkader' AND k.dedicated_birth_year=2017`).Scan(&name3)
	if name1 != "Förderkader gemischt" {
		t.Errorf("2015 kader (rank 1) team name: expected %q, got %q", "Förderkader gemischt", name1)
	}
	if name3 != "Förderkader gemischt 3" {
		t.Errorf("2017 kader (rank 3) team name: expected %q, got %q", "Förderkader gemischt 3", name3)
	}
}

// ── TC-K12: reguläre A–D-Anlage bleibt unverändert (409 bei Duplikat) ────────

// TestCreateRegularKader_UnchangedConflict verifies that creating a second
// A-Jugend/mixed kader with the same team_number still returns 409 (the classic
// non-training-group path is untouched by the renumber logic).
func TestCreateRegularKader_UnchangedConflict(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/kader", h.InitializeKader)
	})

	body := map[string]any{"season_id": seasonID, "age_class": "A-Jugend", "gender": "mixed", "team_number": 1}

	resp := testutil.Post(t, srv, "/api/kader", token, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first A-Jugend: expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp2 := testutil.Post(t, srv, "/api/kader", token, body)
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate A-Jugend: expected 409, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()
}
