package app_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/app"
	"github.com/teamstuttgart/teamwerk/internal/games"
	"github.com/teamstuttgart/teamwerk/internal/health"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/settings"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// E2E-Test: bei aktivem Wartungsmodus liefert POST /api/games als Nicht-Admin
// den erwarteten 503 mit dem X-Maintenance-Mode-Header — auch durch die volle
// Router-Kette. Admin-JWTs dagegen kommen durch (weitergereicht an den Games-
// Handler; die konkrete Response-Form hängt davon ab, aber KEIN 503 vom
// Maintenance-Layer).
func TestRouter_MaintenanceModeOn_BlocksNonAdminMutation(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	standardID := testutil.CreateUser(t, db, "standard")

	settingsStore := settings.NewStoreForTest(db, 0)
	if err := settingsStore.SetMaintenanceMode(context.Background(), true, adminID); err != nil {
		t.Fatalf("preset on: %v", err)
	}
	settingsHandler := settings.NewHandler(settingsStore, hub.NewHub())

	// Minimaler Handlers-Container mit den für die Router-Kette Pflicht-
	// Handlern. Nicht-benutzte Handler bleiben nil; der Router legt sie
	// unter Auth-Gruppen an, die diese Tests nicht ansteuern.
	handlers := &app.Handlers{
		Games:         games.NewHandler(db, nil, hub.NewHub()),
		Health:        health.NewHandler(db, "", ""),
		Hub:           hub.NewHandler(hub.NewHub(), "test"),
		Settings:      settingsHandler,
		SettingsStore: settingsStore,
		JWTSecret:     testutil.TestJWTSecret,
		Database:      db,
	}
	router := app.BuildRouter(handlers, nil)

	// Non-Admin POST → 503
	req := httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(`{}`))
	req.Header.Set("Authorization", testutil.Token(t, standardID, "standard", []string{"vorstand"}))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Non-Admin POST /api/games: erwartet 503, bekam %d", rr.Code)
	}
	if rr.Header().Get("X-Maintenance-Mode") != "1" {
		t.Errorf("erwartet X-Maintenance-Mode: 1, bekam %q", rr.Header().Get("X-Maintenance-Mode"))
	}
	if !strings.Contains(rr.Body.String(), `"error":"maintenance_mode"`) {
		t.Errorf("erwartet maintenance_mode-Body, bekam %q", rr.Body.String())
	}

	// Public status endpoint bleibt trotz aktivem Modus erreichbar (kein Auth).
	req2 := httptest.NewRequest(http.MethodGet, "/api/maintenance-status", nil)
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("GET /api/maintenance-status: erwartet 200, bekam %d", rr2.Code)
	}

	// Admin POST läuft durch die Maintenance-Middleware (der Games-Handler
	// selbst wird 400/401/… liefern — Hauptsache NICHT 503).
	req3 := httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(`{}`))
	req3.Header.Set("Authorization", testutil.Token(t, adminID, "admin", nil))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	router.ServeHTTP(rr3, req3)
	if rr3.Code == http.StatusServiceUnavailable {
		t.Errorf("Admin POST sollte NICHT 503 sein, bekam %d", rr3.Code)
	}
	if rr3.Header().Get("X-Maintenance-Mode") != "" {
		t.Errorf("Admin-Response darf X-Maintenance-Mode nicht setzen, bekam %q", rr3.Header().Get("X-Maintenance-Mode"))
	}
}
