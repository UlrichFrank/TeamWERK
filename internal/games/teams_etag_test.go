package games_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// TestTeams_ETag304AndPrivate — GET /api/teams (nutzergefiltert) trägt einen
// ETag und Cache-Control OHNE public/max-age; unveränderter Bestand
// revalidiert per 304. Der ETag wird aus der pro Nutzer gefilterten Antwort
// abgeleitet, ein geteilter Cache kann also nie die Antwort eines anderen
// Nutzers ausliefern.
func TestTeams_ETag304AndPrivate(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	testutil.CreateKader(t, db, teamID, seasonID)
	userID := testutil.CreateUser(t, db, "standard")

	srv := prodserver.New(t, db)
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Get(t, srv, "/api/teams", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erster Abruf: status %d, want 200", res.StatusCode)
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		t.Fatalf("kein ETag gesetzt")
	}
	cc := res.Header.Get("Cache-Control")
	if strings.Contains(cc, "public") || strings.Contains(cc, "max-age") {
		t.Errorf("Cache-Control = %q — nutzergefilterte Route darf kein public/max-age tragen", cc)
	}
	if cc != "private, no-cache" {
		t.Errorf("Cache-Control = %q, want private, no-cache", cc)
	}

	// Unverändert → 304.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/teams", nil)
	req.Header.Set("Authorization", tok)
	req.Header.Set("If-None-Match", etag)
	res304, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("revalidierter GET: %v", err)
	}
	defer res304.Body.Close()
	if res304.StatusCode != http.StatusNotModified {
		t.Fatalf("revalidierter Abruf: status %d, want 304", res304.StatusCode)
	}

	// Anderer Nutzer (andere Filterung: Spieler ohne Teams) → anderer ETag,
	// alter If-None-Match greift nicht.
	otherID := testutil.CreateUser(t, db, "standard")
	otherTok := testutil.Token(t, otherID, "standard", []string{"spieler"})
	req2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/teams", nil)
	req2.Header.Set("Authorization", otherTok)
	req2.Header.Set("If-None-Match", etag)
	resOther, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("GET anderer Nutzer: %v", err)
	}
	defer resOther.Body.Close()
	if resOther.StatusCode != http.StatusOK {
		t.Fatalf("anderer Nutzer: status %d, want 200 (voller, eigener Body)", resOther.StatusCode)
	}
	if otherTag := resOther.Header.Get("ETag"); otherTag == etag {
		t.Errorf("ETag identisch für unterschiedlich gefilterte Antworten: %q", otherTag)
	}

	// Fehlerfall unverändert: ohne Token → 401.
	resUnauth := testutil.Get(t, srv, "/api/teams", "")
	defer resUnauth.Body.Close()
	if resUnauth.StatusCode != http.StatusUnauthorized {
		t.Errorf("ohne Token: status %d, want 401", resUnauth.StatusCode)
	}
}
