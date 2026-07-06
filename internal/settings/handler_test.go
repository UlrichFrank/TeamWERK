package settings_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/settings"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// serverSetup baut einen Test-Server mit den drei maintenance-mode-Routen:
// Public-Status (kein Auth), Admin-Status (auth+admin), Toggle (auth+admin).
// createUser wird zurückgegeben, damit Tests reale User anlegen können —
// die FK auf users(id) in system_settings.updated_by würde sonst 500 werfen.
func serverSetup(t *testing.T, enabled bool) (srv *testHTTPServer, store *settings.Store, evHub *hub.EventHub, createUser func(role string) int) {
	t.Helper()
	db := testutil.NewDB(t)
	store = settings.NewStoreForTest(db, 0)
	if enabled {
		// Preset ohne updated_by (0 → NULL), damit kein FK-Konflikt entsteht.
		if err := store.SetMaintenanceMode(context.Background(), true, 0); err != nil {
			t.Fatalf("preset on: %v", err)
		}
	}
	evHub = hub.NewHub()
	handler := settings.NewHandler(store, evHub)

	srv = newTestHTTPServer(t, func(r chi.Router) {
		// Public — kein Auth
		r.Get("/api/maintenance-status", handler.GetPublicStatus)
		// Authenticated → Admin-only
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(testutil.TestJWTSecret))
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole("admin"))
				r.Get("/api/admin/maintenance-mode", handler.GetAdminStatus)
				r.Post("/api/admin/maintenance-mode", handler.SetMaintenanceMode)
			})
		})
	})
	createUser = func(role string) int { return testutil.CreateUser(t, db, role) }
	return
}

func TestHandler_PublicStatus_NoAuth_Returns200(t *testing.T) {
	srv, _, _, _ := serverSetup(t, false)
	defer srv.Close()

	res := testutil.Get(t, srv.raw, "/api/maintenance-status", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	if body["enabled"] != false {
		t.Errorf("erwartet enabled=false, bekam %v", body["enabled"])
	}
	// Info-Leak-Check: keine Metadaten im public status
	if _, ok := body["updated_at"]; ok {
		t.Error("public status darf updated_at nicht enthalten")
	}
	if _, ok := body["updated_by_name"]; ok {
		t.Error("public status darf updated_by_name nicht enthalten")
	}
}

func TestHandler_PublicStatus_ReflectsEnabled(t *testing.T) {
	srv, _, _, _ := serverSetup(t, true)
	defer srv.Close()

	res := testutil.Get(t, srv.raw, "/api/maintenance-status", "")
	defer res.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	if body["enabled"] != true {
		t.Errorf("erwartet enabled=true, bekam %v", body["enabled"])
	}
}

func TestHandler_Toggle_AsAdmin_Returns200_AndBroadcasts(t *testing.T) {
	srv, store, evHub, createUser := serverSetup(t, false)
	defer srv.Close()

	sub := evHub.Subscribe()
	defer evHub.Unsubscribe(sub)

	adminID := createUser("admin")
	adminToken := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Post(t, srv.raw, "/api/admin/maintenance-mode", adminToken, map[string]any{"enabled": true})
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	if !store.MaintenanceMode() {
		t.Error("Store sollte nach Toggle on sein")
	}

	select {
	case ev := <-sub:
		if ev != "settings-changed" {
			t.Errorf("erwartet 'settings-changed', bekam %q", ev)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("erwartet Broadcast 'settings-changed', kam nicht an")
	}
}

func TestHandler_Toggle_AsNonAdmin_Returns403(t *testing.T) {
	srv, store, _, createUser := serverSetup(t, false)
	defer srv.Close()

	userID := createUser("standard")
	standardToken := testutil.Token(t, userID, "standard", []string{"vorstand"})
	res := testutil.Post(t, srv.raw, "/api/admin/maintenance-mode", standardToken, map[string]any{"enabled": true})
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Errorf("erwartet 403, bekam %d", res.StatusCode)
	}
	if store.MaintenanceMode() {
		t.Error("Store sollte unverändert off sein")
	}
}

func TestHandler_Toggle_Unauthenticated_Returns401(t *testing.T) {
	srv, store, _, _ := serverSetup(t, false)
	defer srv.Close()

	res := testutil.Post(t, srv.raw, "/api/admin/maintenance-mode", "", map[string]any{"enabled": true})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("erwartet 401, bekam %d", res.StatusCode)
	}
	if store.MaintenanceMode() {
		t.Error("Store sollte unverändert off sein")
	}
}

func TestHandler_AdminStatus_IncludesMetadata(t *testing.T) {
	srv, store, _, createUser := serverSetup(t, false)
	defer srv.Close()

	adminID := createUser("admin")
	adminToken := testutil.Token(t, adminID, "admin", nil)
	// Wir müssen einen echten User in der DB haben, damit updated_by_name via JOIN
	// aufgelöst wird. Der Store hat den Admin selbst aber noch nicht in users.
	// Der bestehende Testutil erlaubt CreateUser mit Rolle — machen wir das
	// separat und rufen den Store direkt auf.
	if err := store.SetMaintenanceMode(context.Background(), true, 0); err != nil {
		t.Fatalf("set: %v", err)
	}

	res := testutil.Get(t, srv.raw, "/api/admin/maintenance-mode", adminToken)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(res.Body).Decode(&body)
	if body["enabled"] != true {
		t.Errorf("erwartet enabled=true, bekam %v", body["enabled"])
	}
	if _, ok := body["updated_at"]; !ok {
		t.Error("erwartet updated_at im admin-status")
	}
}
