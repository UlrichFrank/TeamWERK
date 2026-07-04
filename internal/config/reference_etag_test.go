package config_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// getWithNoneMatch führt einen GET mit optionalem If-None-Match aus.
func getWithNoneMatch(t *testing.T, url, token, etag string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return res
}

// TestSeasons_ETagChangesOnMutation — GET /api/seasons liefert einen schwachen
// ETag mit Cache-Control private, no-cache; unveränderter Bestand revalidiert
// per 304; nach einer Saison-Mutation unterscheidet sich der ETag und der
// volle Body wird ausgeliefert. Fehlerfall (403 für Spieler) bleibt unverändert.
func TestSeasons_ETagChangesOnMutation(t *testing.T) {
	database := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, database, "2025/26")
	h := config.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung", "kassierer"))
			r.Get("/api/seasons", h.ListSeasons)
		})
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand"))
			r.Put("/api/seasons/{id}", h.UpdateSeason)
		})
	})
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	// Happy-Path: 200 + ETag + private, no-cache (kein public/max-age).
	res := testutil.Get(t, srv, "/api/seasons", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erster Abruf: status %d, want 200", res.StatusCode)
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("kein ETag gesetzt")
	}
	if cc := res.Header.Get("Cache-Control"); cc != "private, no-cache" {
		t.Errorf("Cache-Control = %q, want private, no-cache", cc)
	}

	// Unveränderter Bestand → 304 mit leerem Body.
	res304 := getWithNoneMatch(t, srv.URL+"/api/seasons", tok, etag)
	defer res304.Body.Close()
	if res304.StatusCode != http.StatusNotModified {
		t.Fatalf("revalidierter Abruf: status %d, want 304", res304.StatusCode)
	}

	// Mutation (Saison umbenennen) → anderer ETag, voller Body.
	body := map[string]any{"name": "2025/26 (neu)", "start_date": "2025-09-01", "end_date": "2026-06-30"}
	if r := testutil.Do(t, srv, http.MethodPut, "/api/seasons/"+strconv.Itoa(seasonID), tok, body); r.StatusCode != http.StatusOK {
		t.Fatalf("PUT season: status %d", r.StatusCode)
	}
	resAfter := getWithNoneMatch(t, srv.URL+"/api/seasons", tok, etag)
	defer resAfter.Body.Close()
	if resAfter.StatusCode != http.StatusOK {
		t.Fatalf("nach Mutation: status %d, want 200 (voller Body)", resAfter.StatusCode)
	}
	if newTag := resAfter.Header.Get("ETag"); newTag == etag {
		t.Errorf("ETag nach Mutation unverändert: %q", newTag)
	}
	var seasons []map[string]any
	json.NewDecoder(resAfter.Body).Decode(&seasons)
	if len(seasons) != 1 || seasons[0]["name"] != "2025/26 (neu)" {
		t.Errorf("Body nach Mutation = %v, want umbenannte Saison", seasons)
	}

	// Fehlerfall unverändert: Spieler ohne Funktion → 403.
	spieler := testutil.Token(t, userID, "standard", []string{"spieler"})
	resForbidden := testutil.Get(t, srv, "/api/seasons", spieler)
	defer resForbidden.Body.Close()
	if resForbidden.StatusCode != http.StatusForbidden {
		t.Errorf("Spieler: status %d, want 403", resForbidden.StatusCode)
	}
}

// TestAgeClassRules_ETag304 — GET /api/age-class-rules revalidiert per 304 und
// liefert nach einer Regel-Änderung einen neuen ETag.
func TestAgeClassRules_ETag304(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(
		`INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes) VALUES ('C-Jugend', 25, 10)`,
	); err != nil {
		t.Fatalf("seed rules: %v", err)
	}
	h := config.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/age-class-rules", h.GetAgeClassRulesHandler)
	})
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Get(t, srv, "/api/age-class-rules", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erster Abruf: status %d, want 200", res.StatusCode)
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("kein ETag gesetzt")
	}
	if cc := res.Header.Get("Cache-Control"); cc != "private, no-cache" {
		t.Errorf("Cache-Control = %q, want private, no-cache", cc)
	}

	res304 := getWithNoneMatch(t, srv.URL+"/api/age-class-rules", tok, etag)
	defer res304.Body.Close()
	if res304.StatusCode != http.StatusNotModified {
		t.Fatalf("revalidierter Abruf: status %d, want 304", res304.StatusCode)
	}

	if err := config.UpdateAgeClassRule(database, "C-Jugend", 30, 10); err != nil {
		t.Fatalf("update rule: %v", err)
	}
	resAfter := getWithNoneMatch(t, srv.URL+"/api/age-class-rules", tok, etag)
	defer resAfter.Body.Close()
	if resAfter.StatusCode != http.StatusOK {
		t.Fatalf("nach Mutation: status %d, want 200", resAfter.StatusCode)
	}
	if newTag := resAfter.Header.Get("ETag"); newTag == etag {
		t.Errorf("ETag nach Regel-Änderung unverändert: %q", newTag)
	}
}
