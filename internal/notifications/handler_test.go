package notifications_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestVapidKey_CacheControlImmutable — GET /api/push/vapid-public-key trägt
// einen Cache-Control-Header mit immutable + ETag; der Body enthält den
// konfigurierten VAPID-Public-Key. Revalidierung per If-None-Match → 304.
func TestVapidKey_CacheControlImmutable(t *testing.T) {
	database := testutil.NewDB(t)
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VAPIDPublicKey: "test-vapid-public-key"}
	h := notifications.NewHandler(database, cfg)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/push/vapid-public-key", h.GetVAPIDPublicKey)
	})
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv, "/api/push/vapid-public-key", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable-Direktive", cc)
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		t.Errorf("kein ETag gesetzt")
	}
	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	if body["publicKey"] != "test-vapid-public-key" {
		t.Errorf("publicKey = %q, want konfigurierter Key", body["publicKey"])
	}

	// Revalidierung: If-None-Match → 304, leerer Body.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/push/vapid-public-key", nil)
	req.Header.Set("Authorization", tok)
	req.Header.Set("If-None-Match", etag)
	res2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("revalidierter GET: %v", err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusNotModified {
		t.Errorf("revalidierter Abruf: status %d, want 304", res2.StatusCode)
	}
}

// Fehlerfall: die Route liegt im Authenticated-Tier — ohne Bearer-Token 401.
func TestVapidKey_Unauthenticated(t *testing.T) {
	database := testutil.NewDB(t)
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VAPIDPublicKey: "test-vapid-public-key"}
	h := notifications.NewHandler(database, cfg)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/push/vapid-public-key", h.GetVAPIDPublicKey)
	})
	res := testutil.Get(t, srv, "/api/push/vapid-public-key", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", res.StatusCode)
	}
}
