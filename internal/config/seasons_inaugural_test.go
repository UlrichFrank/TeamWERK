package config_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// is_inaugural lässt sich beim Anlegen setzen und über die Liste auslesen.
func TestCreateSeason_Inaugural(t *testing.T) {
	database := testutil.NewDB(t)
	h := config.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/seasons", h.CreateSeason)
		r.Put("/api/seasons/{id}", h.UpdateSeason)
		r.Get("/api/seasons", h.ListSeasons)
	})
	tok := testutil.Token(t, 1, "admin", []string{"vorstand"})

	body := map[string]any{"name": "2026/27", "start_date": "2026-07-01", "end_date": "2027-06-30", "is_inaugural": true}
	if res := testutil.Do(t, srv, http.MethodPost, "/api/seasons", tok, body); res.StatusCode != http.StatusCreated {
		t.Fatalf("POST season: status %d", res.StatusCode)
	}

	var inaugural int
	database.QueryRow(`SELECT is_inaugural FROM seasons WHERE name='2026/27'`).Scan(&inaugural)
	if inaugural != 1 {
		t.Fatalf("is_inaugural in DB = %d, want 1", inaugural)
	}

	res := testutil.Get(t, srv, "/api/seasons", tok)
	var list []map[string]any
	json.NewDecoder(res.Body).Decode(&list)
	res.Body.Close()
	if len(list) != 1 || list[0]["is_inaugural"] != true {
		t.Fatalf("ListSeasons is_inaugural falsch: %v", list)
	}
}

// is_inaugural lässt sich per Update umschalten.
func TestUpdateSeason_ToggleInaugural(t *testing.T) {
	database := testutil.NewDB(t)
	res0, _ := database.Exec(`INSERT INTO seasons (name, start_date, end_date) VALUES ('2027/28','2027-09-01','2028-06-30')`)
	id, _ := res0.LastInsertId()
	h := config.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Put("/api/seasons/{id}", h.UpdateSeason)
	})
	tok := testutil.Token(t, 1, "admin", []string{"vorstand"})

	body := map[string]any{"name": "2027/28", "start_date": "2027-09-01", "end_date": "2028-06-30", "is_inaugural": true}
	if res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/seasons/%d", id), tok, body); res.StatusCode != http.StatusOK {
		t.Fatalf("PUT season: status %d", res.StatusCode)
	}
	var inaugural int
	database.QueryRow(`SELECT is_inaugural FROM seasons WHERE id=?`, id).Scan(&inaugural)
	if inaugural != 1 {
		t.Fatalf("is_inaugural nach Update = %d, want 1", inaugural)
	}

	// wieder ausschalten
	body["is_inaugural"] = false
	testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/seasons/%d", id), tok, body)
	database.QueryRow(`SELECT is_inaugural FROM seasons WHERE id=?`, id).Scan(&inaugural)
	if inaugural != 0 {
		t.Fatalf("is_inaugural nach Reset = %d, want 0", inaugural)
	}
}
