package testutil

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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

// Token returns a signed Bearer token for the given user identity (isParent=false).
func Token(t *testing.T, userID int, role string, clubFunctions []string) string {
	t.Helper()
	return TokenWithIsParent(t, userID, role, clubFunctions, false)
}

// TokenWithIsParent returns a signed Bearer token including the isParent flag.
func TokenWithIsParent(t *testing.T, userID int, role string, clubFunctions []string, isParent bool) string {
	t.Helper()
	if clubFunctions == nil {
		clubFunctions = []string{}
	}
	tok, err := auth.IssueAccessToken(TestJWTSecret, userID, "test@test.local", role, clubFunctions, isParent)
	if err != nil {
		t.Fatalf("testutil.TokenWithIsParent: %v", err)
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

// Put sends a PUT request with a JSON body to the test server.
func Put(t *testing.T, srv *httptest.Server, path, token string, body any) *http.Response {
	return Do(t, srv, http.MethodPut, path, token, body)
}

// Delete sends a DELETE request to the test server.
func Delete(t *testing.T, srv *httptest.Server, path, token string) *http.Response {
	return Do(t, srv, http.MethodDelete, path, token, nil)
}

// PostMultipart sends a multipart/form-data POST with a single file field to
// the test server. It sets the Authorization Bearer header (if token != "")
// and the multipart Content-Type with the generated boundary. Consistent with
// Do/Post, the returned response body is NOT closed by this helper.
func PostMultipart(t *testing.T, srv *httptest.Server, path, token, fieldName, filename string, content []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("testutil.PostMultipart create form file: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("testutil.PostMultipart write content: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("testutil.PostMultipart close writer: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, &buf)
	if err != nil {
		t.Fatalf("testutil.PostMultipart new request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("testutil.PostMultipart: %v", err)
	}
	return res
}
