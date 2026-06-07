package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"

	"github.com/teamstuttgart/teamwerk/internal/auth"
)

const TestJWTSecret = "test-secret-not-for-production"

// TestConfig returns a minimal config suitable for handler tests.
// SMTP and VAPID fields are empty; push notifications will fail silently.
func TestConfig() *appconfig.Config {
	return &appconfig.Config{JWTSecret: TestJWTSecret}
}

// NewServer starts a test HTTP server with auth.Middleware applied.
// routeFn registers the routes under test onto the router.
// The server is closed automatically when the test ends.
func NewServer(t *testing.T, routeFn func(r chi.Router)) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Use(auth.Middleware(TestJWTSecret))
	routeFn(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// Token returns a signed Bearer token for the given user identity.
func Token(t *testing.T, userID int, role string, clubFunctions []string) string {
	t.Helper()
	if clubFunctions == nil {
		clubFunctions = []string{}
	}
	tok, err := auth.IssueAccessToken(TestJWTSecret, userID, "test@test.local", role, clubFunctions, false)
	if err != nil {
		t.Fatalf("testutil.Token: %v", err)
	}
	return "Bearer " + tok
}

// Do sends an HTTP request to the test server and returns the response.
func Do(t *testing.T, srv *httptest.Server, method, path, token string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("testutil.Do encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, srv.URL+path, &buf)
	if err != nil {
		t.Fatalf("testutil.Do new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("testutil.Do: %v", err)
	}
	return res
}

// Get sends a GET request to the test server.
func Get(t *testing.T, srv *httptest.Server, path, token string) *http.Response {
	return Do(t, srv, http.MethodGet, path, token, nil)
}

// Post sends a POST request with a JSON body to the test server.
func Post(t *testing.T, srv *httptest.Server, path, token string, body any) *http.Response {
	return Do(t, srv, http.MethodPost, path, token, body)
}
