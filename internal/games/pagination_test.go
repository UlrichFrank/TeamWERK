package games_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type gamesListResponse struct {
	Items []struct {
		ID int `json:"id"`
	} `json:"items"`
	Total int `json:"total"`
}

// TestListGames_PaginatedAndSeasonFilter: ?season_id=&limit= begrenzt die Items,
// total bleibt die Gesamtzahl der sichtbaren Spiele; über offset sind die
// restlichen Spiele erreichbar. Ohne season_id greift das Default-Limit (50)
// statt einer unbeschränkten Liste.
func TestListGames_PaginatedAndSeasonFilter(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	if _, err := db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID); err != nil {
		t.Fatalf("activate season: %v", err)
	}
	teamID := testutil.CreateTeam(t, db, "Team A")
	for i := 0; i < 3; i++ {
		testutil.CreateGame(t, db, seasonID, teamID, fmt.Sprintf("2026-01-1%d", i))
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	// Seite 1: limit=2 → genau 2 Items, total=3.
	res := testutil.Get(t, srv, fmt.Sprintf("/api/games?season_id=%d&limit=2&offset=0", seasonID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var page1 gamesListResponse
	json.NewDecoder(res.Body).Decode(&page1)
	res.Body.Close()
	if len(page1.Items) != 2 {
		t.Errorf("expected 2 items on page 1, got %d", len(page1.Items))
	}
	if page1.Total != 3 {
		t.Errorf("expected total=3, got %d", page1.Total)
	}

	// Seite 2: offset=2 → 1 Item, disjunkt zu Seite 1.
	res2 := testutil.Get(t, srv, fmt.Sprintf("/api/games?season_id=%d&limit=2&offset=2", seasonID), token)
	var page2 gamesListResponse
	json.NewDecoder(res2.Body).Decode(&page2)
	res2.Body.Close()
	if len(page2.Items) != 1 {
		t.Errorf("expected 1 item on page 2, got %d", len(page2.Items))
	}
	seen := map[int]bool{}
	for _, it := range append(page1.Items, page2.Items...) {
		if seen[it.ID] {
			t.Errorf("game id=%d appears on multiple pages", it.ID)
		}
		seen[it.ID] = true
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 distinct games across pages, got %d", len(seen))
	}

	// Ohne season_id: Default-Limit statt unbeschränkt, total weiterhin vollständig.
	res3 := testutil.Get(t, srv, "/api/games?limit=2", token)
	var noSeason gamesListResponse
	json.NewDecoder(res3.Body).Decode(&noSeason)
	res3.Body.Close()
	if len(noSeason.Items) != 2 {
		t.Errorf("expected 2 items with limit=2 (no season_id), got %d", len(noSeason.Items))
	}
	if noSeason.Total != 3 {
		t.Errorf("expected total=3 (no season_id), got %d", noSeason.Total)
	}
}

// TestListGames_Unauthorized: ohne Token → 401.
func TestListGames_Unauthorized(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	srv := testServer(t, db)

	res := testutil.Get(t, srv, "/api/games", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", res.StatusCode)
	}
}

// TestParticipants_Paginated: ?limit=&offset= begrenzt die Teilnehmerliste,
// total bleibt die Gesamtzahl der sichtbaren Teilnehmer; Autorisierung
// unverändert (Admin sieht alle).
func TestParticipants_Paginated(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, _, gameID, _ := optOutFixture(t, db, 3, true)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants?limit=2&offset=0", gameID), token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var page1 struct {
		Items []struct {
			MemberID int `json:"member_id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	json.NewDecoder(res.Body).Decode(&page1)
	res.Body.Close()
	if len(page1.Items) != 2 {
		t.Errorf("expected 2 participants on page 1, got %d", len(page1.Items))
	}
	if page1.Total != 3 {
		t.Errorf("expected total=3 participants, got %d", page1.Total)
	}

	res2 := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants?limit=2&offset=2", gameID), token)
	var page2 struct {
		Items []struct {
			MemberID int `json:"member_id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	json.NewDecoder(res2.Body).Decode(&page2)
	res2.Body.Close()
	if len(page2.Items) != 1 {
		t.Errorf("expected 1 participant on page 2, got %d", len(page2.Items))
	}
	if page2.Total != 3 {
		t.Errorf("expected total=3 on page 2, got %d", page2.Total)
	}
	seen := map[int]bool{}
	for _, it := range page1.Items {
		seen[it.MemberID] = true
	}
	for _, it := range page2.Items {
		if seen[it.MemberID] {
			t.Errorf("member id=%d appears on multiple pages", it.MemberID)
		}
		seen[it.MemberID] = true
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 distinct participants across pages, got %d", len(seen))
	}
}

// TestParticipants_Unauthorized: ohne Token → 401.
func TestParticipants_Unauthorized(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, _, gameID, _ := optOutFixture(t, db, 2, true)
	srv := testServer(t, db)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", gameID), "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", res.StatusCode)
	}
}
