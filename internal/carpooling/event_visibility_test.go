package carpooling_test

import (
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/carpooling"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestCarpooling_FremdGame_404 verifiziert die event-team-visibility-Regel:
// Ein User ohne Team-Bezug zum referenzierten Game kann kein Mitfahr-Gesuch
// anlegen — Antwort ist 404, nicht 403 oder 204.
func TestCarpooling_FremdGame_404(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	// Game gehört Team A; Caller ist nicht im Kader von Team A.
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	callerUID := testutil.CreateUser(t, db, "standard")
	// Kein Member, kein Kader-Eintrag → keine Sichtbarkeit.

	h := carpooling.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testFullServer(t, h)

	token := testutil.Token(t, callerUID, "standard", nil)
	res := testutil.Post(t, srv, "/api/mitfahrgelegenheiten", token, map[string]any{
		"gameId":  gameID,
		"typ":     "suche",
		"plaetze": 1,
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("Fremd-Game-Carpool: erwartet 404, got %d", res.StatusCode)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM mitfahrgelegenheiten WHERE game_id=?`, gameID).Scan(&count)
	if count != 0 {
		t.Errorf("kein Eintrag in DB erwartet, got %d", count)
	}
}
