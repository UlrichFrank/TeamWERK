package config_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func activeSeasonServer(t *testing.T, h *config.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/seasons/active", h.GetActiveSeason)
	})
}

// Happy-Path: /termine holt sich hierüber das Saisonfenster, um seine Liste
// vollständig bis zum Saisonende zu laden. Die Route ist bewusst für alle
// Eingeloggten offen (anders als GET /api/seasons).
func TestGetActiveSeason_LiefertFensterFuerStandardnutzer(t *testing.T) {
	database := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, database, "2025/26")
	h := config.NewHandler(database, hub.NewHub())
	srv := activeSeasonServer(t, h)

	tok := testutil.Token(t, 1, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/seasons/active", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	var got map[string]any
	json.NewDecoder(res.Body).Decode(&got)
	if int(got["id"].(float64)) != seasonID {
		t.Errorf("id = %v, want %d", got["id"], seasonID)
	}
	// Reine "2006-01-02"-Form: die Werte gehen direkt als from/to-Query zurück
	// an /api/games/my und /api/training-sessions, die auf DATE vergleichen.
	if got["start_date"] != "2025-09-01" || got["end_date"] != "2026-06-30" {
		t.Errorf("Fenster = %v..%v, want 2025-09-01..2026-06-30", got["start_date"], got["end_date"])
	}
}

// Ohne aktive Saison gibt es kein Fenster — 404 statt eines erfundenen Zeitraums.
// Das Frontend fällt darauf auf ein rollierendes Fenster zurück.
func TestGetActiveSeason_OhneAktiveSaison404(t *testing.T) {
	database := testutil.NewDB(t)
	testutil.CreateSeason(t, database, "2025/26")
	if _, err := database.Exec(`UPDATE seasons SET is_active=0`); err != nil {
		t.Fatalf("deaktivieren: %v", err)
	}
	h := config.NewHandler(database, hub.NewHub())
	srv := activeSeasonServer(t, h)

	res := testutil.Get(t, srv, "/api/seasons/active", testutil.Token(t, 1, "standard", nil))
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", res.StatusCode)
	}
}

// Nicht eingeloggt → 401 (Auth-Middleware der Testserver-Fixture).
func TestGetActiveSeason_OhneToken401(t *testing.T) {
	database := testutil.NewDB(t)
	testutil.CreateSeason(t, database, "2025/26")
	h := config.NewHandler(database, hub.NewHub())
	srv := activeSeasonServer(t, h)

	res := testutil.Get(t, srv, "/api/seasons/active", "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", res.StatusCode)
	}
}
