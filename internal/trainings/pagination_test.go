package trainings_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

type sessionsListResponse struct {
	Items []struct {
		ID       int  `json:"id"`
		SeriesID *int `json:"series_id"`
	} `json:"items"`
	Total int `json:"total"`
}

// TestListSessions_Paginated: ?limit=&offset= begrenzt die Items, total bleibt
// die Gesamtzahl der sichtbaren Sessions; über offset sind die restlichen
// Sessions erreichbar (disjunkte Ausschnitte).
func TestListSessions_Paginated(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dates := []string{"2026-03-05", "2026-03-06", "2026-03-07"}
	for _, d := range dates {
		testutil.CreateTrainingSession(t, db, teamID, seasonID, d)
	}

	adminUserID := testutil.CreateUser(t, db, "admin")
	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, adminUserID, "admin", nil)

	res := testutil.Get(t, srv, "/api/training-sessions?from=2026-03-01&to=2026-03-31&limit=2&offset=0", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var page1 sessionsListResponse
	json.NewDecoder(res.Body).Decode(&page1)
	res.Body.Close()
	if len(page1.Items) != 2 {
		t.Errorf("expected 2 items on page 1, got %d", len(page1.Items))
	}
	if page1.Total != 3 {
		t.Errorf("expected total=3, got %d", page1.Total)
	}

	res2 := testutil.Get(t, srv, "/api/training-sessions?from=2026-03-01&to=2026-03-31&limit=2&offset=2", token)
	var page2 sessionsListResponse
	json.NewDecoder(res2.Body).Decode(&page2)
	res2.Body.Close()
	if len(page2.Items) != 1 {
		t.Errorf("expected 1 item on page 2, got %d", len(page2.Items))
	}
	seen := map[int]bool{}
	for _, it := range append(page1.Items, page2.Items...) {
		if seen[it.ID] {
			t.Errorf("session id=%d appears on multiple pages", it.ID)
		}
		seen[it.ID] = true
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 distinct sessions across pages, got %d", len(seen))
	}
}

// TestListSessions_ExcludeSeriesFilter: ?exclude_series=1 liefert nur Sessions
// ohne Serienbezug (series_id IS NULL); ohne den Parameter erscheinen beide.
func TestListSessions_ExcludeSeriesFilter(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	creatorID := testutil.CreateUser(t, db, "admin")

	// Ein Einzeltermin (series_id NULL) + ein Serientermin (series_id gesetzt).
	singleID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-05")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, creatorID)
	res, err := db.Exec(
		`INSERT INTO training_sessions (team_id, season_id, series_id, date, start_time, end_time, title)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		teamID, seasonID, seriesID, "2026-03-06", "18:00", "20:00", "Serientermin")
	if err != nil {
		t.Fatalf("insert series session: %v", err)
	}
	seriesSessionID64, _ := res.LastInsertId()
	seriesSessionID := int(seriesSessionID64)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, creatorID, "admin", nil)

	// Ohne Filter: beide Termine sichtbar.
	resAll := testutil.Get(t, srv, "/api/training-sessions?from=2026-03-01&to=2026-03-31", token)
	var all sessionsListResponse
	json.NewDecoder(resAll.Body).Decode(&all)
	resAll.Body.Close()
	if all.Total != 2 {
		t.Errorf("ohne Filter: expected total=2, got %d", all.Total)
	}

	// Mit exclude_series=1: nur der Einzeltermin.
	resFiltered := testutil.Get(t, srv, "/api/training-sessions?from=2026-03-01&to=2026-03-31&exclude_series=1", token)
	if resFiltered.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resFiltered.StatusCode)
	}
	var filtered sessionsListResponse
	json.NewDecoder(resFiltered.Body).Decode(&filtered)
	resFiltered.Body.Close()
	if filtered.Total != 1 {
		t.Errorf("exclude_series: expected total=1, got %d", filtered.Total)
	}
	if len(filtered.Items) != 1 {
		t.Fatalf("exclude_series: expected 1 item, got %d", len(filtered.Items))
	}
	got := filtered.Items[0]
	if got.ID != singleID {
		t.Errorf("expected single session id=%d, got %d", singleID, got.ID)
	}
	if got.SeriesID != nil {
		t.Errorf("expected series_id nil, got %v", *got.SeriesID)
	}
	for _, it := range filtered.Items {
		if it.ID == seriesSessionID {
			t.Errorf("series session %d must not appear with exclude_series=1", seriesSessionID)
		}
	}
}

// Fehlerfall (401 ohne Token) ist bereits durch TestListSessions_Unauthenticated
// in handler_test.go abgedeckt.
