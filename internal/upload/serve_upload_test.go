package upload_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/upload"
)

// B-5: /api/uploads/* erfordert Authentifizierung (Cookie); härtet die Antwort.
func TestServeUpload_RequiresCookieAuth(t *testing.T) {
	db := testutil.NewDB(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("imgdata"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	h := upload.NewHandler(db, dir, testutil.TestJWTSecret, hub.NewHub())
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.CookieMiddleware(db))
		r.Get("/api/uploads/*", h.ServeUpload)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	// Ohne Cookie → 401, keine Datei.
	res, err := http.Get(srv.URL + "/api/uploads/photo.jpg")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("ohne Cookie: erwartet 401, bekam %d", res.StatusCode)
	}

	// Mit gültigem Refresh-Cookie → 200 + Härtungs-Header.
	userID := testutil.CreateUser(t, db, "standard")
	plain := testutil.CreateRefreshToken(t, db, userID)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/uploads/photo.jpg", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: plain})
	res2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET mit Cookie: %v", err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("mit Cookie: erwartet 200, bekam %d", res2.StatusCode)
	}
	if got := res2.Header.Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy: erwartet no-referrer, bekam %q", got)
	}
	if got := res2.Header.Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control: erwartet 'private, no-store', bekam %q", got)
	}
}
