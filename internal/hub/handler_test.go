package hub_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newSSEServer(t *testing.T) *httptest.Server {
	t.Helper()
	db := testutil.NewDB(t)
	h := hub.NewHandler(hub.NewHub(), "test-build-hash")

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.CookieMiddleware(db))
		r.Get("/api/events", h.Events)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// TestSSE_CookieAuth_Valid: gültiges Refresh-Token-Cookie → 200 (SSE connection accepted)
func TestSSE_CookieAuth_Valid(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	plainToken := testutil.CreateRefreshToken(t, db, userID)

	h := hub.NewHandler(hub.NewHub(), "test-hash")
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.CookieMiddleware(db))
		r.Get("/api/events", h.Events)
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/events", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: plainToken})

	// Use a client that doesn't follow redirects and closes quickly
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with valid cookie, got %d", resp.StatusCode)
	}
}

// TestSSE_CookieAuth_NoCookie: kein Cookie → 401
func TestSSE_CookieAuth_NoCookie(t *testing.T) {
	srv := newSSEServer(t)

	req, _ := http.NewRequest("GET", srv.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without cookie, got %d", resp.StatusCode)
	}
}

// TestSSE_CookieAuth_QueryTokenRejected: alter ?token=-Parameter wird nicht akzeptiert
func TestSSE_CookieAuth_QueryTokenRejected(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	_ = testutil.CreateRefreshToken(t, db, userID)
	jwtToken := testutil.Token(t, userID, "standard", nil)

	h := hub.NewHandler(hub.NewHub(), "test-hash")
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.CookieMiddleware(db))
		r.Get("/api/events", h.Events)
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/api/events?token="+url.QueryEscape(jwtToken), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for ?token= param (only cookie accepted), got %d", resp.StatusCode)
	}
}
