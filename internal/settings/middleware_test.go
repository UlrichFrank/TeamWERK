package settings_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/settings"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newTestStore(t *testing.T, enabled bool) *settings.Store {
	t.Helper()
	db := testutil.NewDB(t)
	s := settings.NewStoreForTest(db, 0)
	if enabled {
		if err := s.SetMaintenanceMode(context.Background(), true, 0); err != nil {
			t.Fatalf("set maintenance on: %v", err)
		}
	}
	return s
}

func passthroughNext() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestMiddleware_ModeOff_AllowsMutation(t *testing.T) {
	store := newTestStore(t, false)
	mw := settings.MaintenanceMiddleware(store, testutil.TestJWTSecret)(passthroughNext())

	req := httptest.NewRequest(http.MethodPost, "/api/games", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("erwartet 200 bei off, bekam %d", rr.Code)
	}
	if rr.Header().Get("X-Maintenance-Mode") != "" {
		t.Error("erwartet keinen X-Maintenance-Mode-Header bei off")
	}
}

func TestMiddleware_ModeOn_BlocksNonAdminMutation(t *testing.T) {
	store := newTestStore(t, true)
	mw := settings.MaintenanceMiddleware(store, testutil.TestJWTSecret)(passthroughNext())

	req := httptest.NewRequest(http.MethodPost, "/api/games", nil)
	// Standard-User (kein Admin)
	req.Header.Set("Authorization", testutil.Token(t, 1, "standard", nil))
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("erwartet 503, bekam %d", rr.Code)
	}
	if got := rr.Header().Get("X-Maintenance-Mode"); got != "1" {
		t.Errorf("erwartet X-Maintenance-Mode: 1, bekam %q", got)
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("erwartet Content-Type application/json, bekam %q", got)
	}
	if !strings.Contains(rr.Body.String(), `"error":"maintenance_mode"`) {
		t.Errorf("erwartet error-body, bekam %q", rr.Body.String())
	}
}

func TestMiddleware_ModeOn_AllowsAdminMutation(t *testing.T) {
	store := newTestStore(t, true)
	mw := settings.MaintenanceMiddleware(store, testutil.TestJWTSecret)(passthroughNext())

	req := httptest.NewRequest(http.MethodPost, "/api/games", nil)
	req.Header.Set("Authorization", testutil.Token(t, 1, "admin", nil))
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("erwartet 200 für Admin, bekam %d", rr.Code)
	}
	if rr.Header().Get("X-Maintenance-Mode") != "" {
		t.Error("erwartet keinen X-Maintenance-Mode-Header für Admin")
	}
}

func TestMiddleware_ModeOn_AllowsAuthRoutes(t *testing.T) {
	store := newTestStore(t, true)
	mw := settings.MaintenanceMiddleware(store, testutil.TestJWTSecret)(passthroughNext())

	authPaths := []string{
		"/api/auth/login",
		"/api/auth/refresh",
		"/api/auth/logout",
		"/api/auth/forgot-password",
		"/api/auth/reset-password",
	}
	for _, p := range authPaths {
		req := httptest.NewRequest(http.MethodPost, p, nil)
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("%s: erwartet 200 (Auth-Whitelist), bekam %d", p, rr.Code)
		}
	}
}

func TestMiddleware_ModeOn_AllowsGetHeadOptions(t *testing.T) {
	store := newTestStore(t, true)
	mw := settings.MaintenanceMiddleware(store, testutil.TestJWTSecret)(passthroughNext())

	methods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	for _, m := range methods {
		req := httptest.NewRequest(m, "/api/games", nil)
		rr := httptest.NewRecorder()
		mw.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("%s: erwartet 200, bekam %d", m, rr.Code)
		}
		if rr.Header().Get("X-Maintenance-Mode") != "" {
			t.Errorf("%s: erwartet keinen X-Maintenance-Mode-Header", m)
		}
	}
}

func TestMiddleware_ModeOn_UnauthenticatedNonAuthMutation_Blocked(t *testing.T) {
	store := newTestStore(t, true)
	mw := settings.MaintenanceMiddleware(store, testutil.TestJWTSecret)(passthroughNext())

	req := httptest.NewRequest(http.MethodPost, "/api/games", nil)
	// Kein Authorization-Header
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("erwartet 503 auch ohne Auth, bekam %d", rr.Code)
	}
}

func TestMiddleware_InvalidToken_TreatedAsNonAdmin(t *testing.T) {
	store := newTestStore(t, true)
	mw := settings.MaintenanceMiddleware(store, testutil.TestJWTSecret)(passthroughNext())

	req := httptest.NewRequest(http.MethodPost, "/api/games", nil)
	req.Header.Set("Authorization", "Bearer garbage-token")
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("erwartet 503 bei kaputtem Token (nicht Admin), bekam %d", rr.Code)
	}
}
