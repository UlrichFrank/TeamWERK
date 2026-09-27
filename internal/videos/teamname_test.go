package videos

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestListVideos_TeamNameIstLangformMitNummerAmEnde: die Videoliste nennt die
// Mannschaft wie der Rest der App ("C-Jugend männlich 1"), nicht mit dem
// gespeicherten teams.name, der für Mannschaft 1 keine Nummer trägt.
func TestListVideos_TeamNameIstLangformMitNummerAmEnde(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := crudHandler(t, db)
	srv := newCRUDServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	var teamIDs [2]int
	for i, name := range []string{"C-Jugend männlich", "C-Jugend männlich 2"} {
		res, err := db.Exec(`INSERT INTO teams (name, age_class, gender) VALUES (?, 'C-Jugend', 'm')`, name)
		if err != nil {
			t.Fatalf("insert team: %v", err)
		}
		id, _ := res.LastInsertId()
		teamIDs[i] = int(id)
		if _, err := db.Exec(`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, 'C-Jugend', 'm', ?, ?)`,
			season, id, i+1); err != nil {
			t.Fatalf("insert kader: %v", err)
		}
	}
	admin := testutil.CreateUser(t, db, "admin")
	videoID := testutil.CreateVideo(t, db, teamIDs[0], season, admin, "ready")
	token := testutil.Token(t, admin, "admin", nil)

	res := testutil.Get(t, srv, "/api/videos", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	lr := decodeList(t, res)
	if len(lr.Items) != 1 || lr.Items[0].TeamName != "C-Jugend männlich 1" {
		t.Fatalf("team_name = %+v, want \"C-Jugend männlich 1\"", lr.Items)
	}

	res = testutil.Get(t, srv, "/api/videos/"+strconv.Itoa(videoID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("detail status = %d, want 200", res.StatusCode)
	}
	var detail struct {
		TeamName string `json:"team_name"`
	}
	json.NewDecoder(res.Body).Decode(&detail)
	if detail.TeamName != "C-Jugend männlich 1" {
		t.Errorf("detail team_name = %q, want %q", detail.TeamName, "C-Jugend männlich 1")
	}
}
