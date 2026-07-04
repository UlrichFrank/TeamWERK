package venues_test

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/venues"
)

// TestVenues_ETag304 — GET /api/venues liefert ETag + private, no-cache;
// unveränderter Bestand revalidiert per 304, nach einer Mutation ändert sich
// der ETag. Fehlerfall (403 ohne Vereinsfunktion) bleibt unverändert.
func TestVenues_ETag304(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(
		`INSERT INTO venues (name, street, city, postal_code) VALUES ('Sporthalle Ost', 'Teststr. 1', 'Stuttgart', '70000')`,
	); err != nil {
		t.Fatalf("seed venue: %v", err)
	}
	h := venues.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Get("/api/venues", h.List)
		})
	})
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Get(t, srv, "/api/venues", tok)
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

	// Unverändert → 304.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/venues", nil)
	req.Header.Set("Authorization", tok)
	req.Header.Set("If-None-Match", etag)
	res304, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("revalidierter GET: %v", err)
	}
	defer res304.Body.Close()
	if res304.StatusCode != http.StatusNotModified {
		t.Fatalf("revalidierter Abruf: status %d, want 304", res304.StatusCode)
	}

	// Mutation → neuer ETag, voller Body.
	if _, err := database.Exec(`UPDATE venues SET name='Sporthalle West'`); err != nil {
		t.Fatalf("update venue: %v", err)
	}
	req2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/venues", nil)
	req2.Header.Set("Authorization", tok)
	req2.Header.Set("If-None-Match", etag)
	resAfter, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("GET nach Mutation: %v", err)
	}
	defer resAfter.Body.Close()
	if resAfter.StatusCode != http.StatusOK {
		t.Fatalf("nach Mutation: status %d, want 200", resAfter.StatusCode)
	}
	if newTag := resAfter.Header.Get("ETag"); newTag == etag {
		t.Errorf("ETag nach Mutation unverändert: %q", newTag)
	}

	// Fehlerfall unverändert: ohne berechtigte Vereinsfunktion → 403.
	spieler := testutil.Token(t, userID, "standard", []string{"spieler"})
	resForbidden := testutil.Get(t, srv, "/api/venues", spieler)
	defer resForbidden.Body.Close()
	if resForbidden.StatusCode != http.StatusForbidden {
		t.Errorf("Spieler: status %d, want 403", resForbidden.StatusCode)
	}
}
