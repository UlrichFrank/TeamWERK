package videos

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// newCRUDServer mounts the CRUD/list routes behind JWT auth, mirroring the
// Authenticated-tier mounting in internal/app/router.go.
func newCRUDServer(t *testing.T, h *Handler) *httptest.Server {
	t.Helper()
	r := chi.NewRouter()
	r.Use(auth.Middleware(testutil.TestJWTSecret))
	r.Get("/api/videos", h.List)
	r.Get("/api/videos/{id}", h.Get)
	r.Patch("/api/videos/{id}", h.Update)
	r.Delete("/api/videos/{id}", h.Delete)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// crudHandler returns a handler with storage rooted at a temp dir and a real hub.
func crudHandler(t *testing.T, db *sql.DB) (*Handler, string) {
	t.Helper()
	root := t.TempDir()
	cfg := &appconfig.Config{JWTSecret: testutil.TestJWTSecret, VideoStorageDir: root}
	return NewHandler(db, hub.NewHub(), cfg), root
}

func patch(t *testing.T, srv *httptest.Server, path, token string, body any) *http.Response {
	t.Helper()
	return testutil.Do(t, srv, http.MethodPatch, path, token, body)
}

// listResp is the decoded shape of GET /api/videos.
type listResp struct {
	Items []videoListItem `json:"items"`
	Total int             `json:"total"`
}

func decodeList(t *testing.T, res *http.Response) listResp {
	t.Helper()
	defer res.Body.Close()
	var lr listResp
	if err := json.NewDecoder(res.Body).Decode(&lr); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	return lr
}

// --- LIST: visibility per persona -------------------------------------------

func TestList_VisibilityPerPersona(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := crudHandler(t, db)
	srv := newCRUDServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	kaderB := testutil.CreateKader(t, db, teamB, season)

	uploader := testutil.CreateUser(t, db, "standard")
	vidA := testutil.CreateVideo(t, db, teamA, season, uploader, "ready")
	_ = testutil.CreateVideo(t, db, teamB, season, uploader, "ready")

	// player of team A
	playerUser := testutil.CreateUser(t, db, "standard")
	playerMember := testutil.CreateMember(t, db, playerUser)
	addKaderMember(t, db, kaderA, playerMember)

	// trainer of team B
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderB, trainerMember)

	// parent of the team-A player
	parentUser := testutil.CreateUser(t, db, "standard")
	addFamilyLink(t, db, parentUser, playerMember)

	vorstandUser := testutil.CreateUser(t, db, "standard")

	cases := []struct {
		name      string
		token     string
		wantTotal int
		wantTeam  int // 0 = don't assert a single team
	}{
		{"player sees only team A", testutil.Token(t, playerUser, "standard", []string{"spieler"}), 1, teamA},
		{"parent sees only team A", testutil.Token(t, parentUser, "standard", nil), 1, teamA},
		{"trainer sees only team B", testutil.Token(t, trainerUser, "standard", []string{"trainer"}), 1, teamB},
		{"vorstand sees all", testutil.Token(t, vorstandUser, "standard", []string{"vorstand"}), 2, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.Get(t, srv, "/api/videos", tc.token)
			if res.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", res.StatusCode)
			}
			lr := decodeList(t, res)
			if lr.Total != tc.wantTotal || len(lr.Items) != tc.wantTotal {
				t.Fatalf("total=%d items=%d, want %d", lr.Total, len(lr.Items), tc.wantTotal)
			}
			if tc.wantTeam != 0 {
				for _, it := range lr.Items {
					if it.TeamID != tc.wantTeam {
						t.Errorf("item team_id=%d, want %d", it.TeamID, tc.wantTeam)
					}
				}
			}
		})
	}

	// outsider sees nothing
	outsider := testutil.CreateUser(t, db, "standard")
	res := testutil.Get(t, srv, "/api/videos", testutil.Token(t, outsider, "standard", nil))
	lr := decodeList(t, res)
	if lr.Total != 0 {
		t.Errorf("outsider total=%d, want 0", lr.Total)
	}
	_ = vidA
}

// --- LIST: filters -----------------------------------------------------------

func TestList_Filters(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := crudHandler(t, db)
	srv := newCRUDServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	uploader := testutil.CreateUser(t, db, "standard")
	testutil.CreateVideo(t, db, teamA, season, uploader, "ready")
	testutil.CreateVideo(t, db, teamA, season, uploader, "queued")
	testutil.CreateVideo(t, db, teamB, season, uploader, "ready")

	admin := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)

	res := testutil.Get(t, srv, "/api/videos?team_id="+itoa(teamA), admin)
	if lr := decodeList(t, res); lr.Total != 2 {
		t.Errorf("team_id filter total=%d, want 2", lr.Total)
	}
	res = testutil.Get(t, srv, "/api/videos?status=ready", admin)
	if lr := decodeList(t, res); lr.Total != 2 {
		t.Errorf("status filter total=%d, want 2", lr.Total)
	}
	res = testutil.Get(t, srv, "/api/videos?team_id="+itoa(teamA)+"&status=ready", admin)
	if lr := decodeList(t, res); lr.Total != 1 {
		t.Errorf("combined filter total=%d, want 1", lr.Total)
	}
}

// --- LIST: pagination --------------------------------------------------------

func TestList_Pagination(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := crudHandler(t, db)
	srv := newCRUDServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	uploader := testutil.CreateUser(t, db, "standard")
	for i := 0; i < 3; i++ {
		testutil.CreateVideo(t, db, team, season, uploader, "ready")
	}
	admin := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)

	res := testutil.Get(t, srv, "/api/videos?limit=2&offset=0", admin)
	lr := decodeList(t, res)
	if lr.Total != 3 || len(lr.Items) != 2 {
		t.Fatalf("page1 total=%d items=%d, want total=3 items=2", lr.Total, len(lr.Items))
	}
	res = testutil.Get(t, srv, "/api/videos?limit=2&offset=2", admin)
	lr = decodeList(t, res)
	if lr.Total != 3 || len(lr.Items) != 1 {
		t.Fatalf("page2 total=%d items=%d, want total=3 items=1", lr.Total, len(lr.Items))
	}
}

// --- GET detail --------------------------------------------------------------

func TestGet_VisibleAndHidden(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := crudHandler(t, db)
	srv := newCRUDServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, teamA, season, uploader, "ready")

	playerUser := testutil.CreateUser(t, db, "standard")
	pm := testutil.CreateMember(t, db, playerUser)
	addKaderMember(t, db, kaderA, pm)
	outsider := testutil.CreateUser(t, db, "standard")

	// visible → 200
	res := testutil.Get(t, srv, "/api/videos/"+itoa(vid), testutil.Token(t, playerUser, "standard", []string{"spieler"}))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("player get status = %d, want 200", res.StatusCode)
	}
	res.Body.Close()

	// not visible → 404 (no existence leak)
	res = testutil.Get(t, srv, "/api/videos/"+itoa(vid), testutil.Token(t, outsider, "standard", nil))
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("outsider get status = %d, want 404", res.StatusCode)
	}
	res.Body.Close()

	// nonexistent → 404
	res = testutil.Get(t, srv, "/api/videos/999999", testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil))
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing get status = %d, want 404", res.StatusCode)
	}
	res.Body.Close()
}

// --- PATCH -------------------------------------------------------------------

func TestUpdate_HappyAndForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := crudHandler(t, db)
	srv := newCRUDServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, teamA, season, uploader, "ready")

	// trainer of team A may manage
	trainerUser := testutil.CreateUser(t, db, "standard")
	tm := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderA, tm)
	trainer := testutil.Token(t, trainerUser, "standard", []string{"trainer"})

	res := patch(t, srv, "/api/videos/"+itoa(vid), trainer, map[string]any{"title": "Neuer Titel", "description": "Hallo"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d, want 200", res.StatusCode)
	}
	res.Body.Close()
	var gotTitle, gotDesc string
	if err := db.QueryRow(`SELECT title, description FROM videos WHERE id=?`, vid).Scan(&gotTitle, &gotDesc); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if gotTitle != "Neuer Titel" || gotDesc != "Hallo" {
		t.Errorf("after patch title=%q desc=%q", gotTitle, gotDesc)
	}

	// plain player (team A) may NOT manage → 403
	playerUser := testutil.CreateUser(t, db, "standard")
	pmem := testutil.CreateMember(t, db, playerUser)
	addKaderMember(t, db, kaderA, pmem)
	res = patch(t, srv, "/api/videos/"+itoa(vid), testutil.Token(t, playerUser, "standard", []string{"spieler"}), map[string]any{"title": "X"})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("player patch status = %d, want 403", res.StatusCode)
	}
	res.Body.Close()

	// empty title → 400
	res = patch(t, srv, "/api/videos/"+itoa(vid), trainer, map[string]any{"title": "   "})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty title status = %d, want 400", res.StatusCode)
	}
	res.Body.Close()

	// missing video → 404
	res = patch(t, srv, "/api/videos/999999", trainer, map[string]any{"title": "X"})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing patch status = %d, want 404", res.StatusCode)
	}
	res.Body.Close()
}

// --- DELETE: removes row and files ------------------------------------------

func TestDelete_RemovesRowAndFiles(t *testing.T) {
	db := testutil.NewDB(t)
	h, root := crudHandler(t, db)
	srv := newCRUDServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, teamA, season, uploader, "ready")

	// lay down raw + processed files
	if err := os.MkdirAll(ProcessedDir(root, vid)+"/720p", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ProcessedDir(root, vid)+"/master.m3u8", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(rawDir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(RawPath(root, vid), []byte("rawdata"), 0o644); err != nil {
		t.Fatal(err)
	}

	trainerUser := testutil.CreateUser(t, db, "standard")
	tm := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderA, tm)
	trainer := testutil.Token(t, trainerUser, "standard", []string{"trainer"})

	res := testutil.Do(t, srv, http.MethodDelete, "/api/videos/"+itoa(vid), trainer, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", res.StatusCode)
	}
	res.Body.Close()

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM videos WHERE id=?`, vid).Scan(&n)
	if n != 0 {
		t.Errorf("row still present after delete")
	}
	if _, err := os.Stat(RawPath(root, vid)); !os.IsNotExist(err) {
		t.Errorf("raw file still present: %v", err)
	}
	if _, err := os.Stat(ProcessedDir(root, vid)); !os.IsNotExist(err) {
		t.Errorf("processed dir still present: %v", err)
	}
}

// --- DELETE: forbidden / not found ------------------------------------------

func TestDelete_ForbiddenAndNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	h, _ := crudHandler(t, db)
	srv := newCRUDServer(t, h)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderB := testutil.CreateKader(t, db, teamB, season)
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, teamA, season, uploader, "ready")

	// trainer of FOREIGN team B → 403 on team-A video
	foreignTrainerUser := testutil.CreateUser(t, db, "standard")
	ftm := testutil.CreateMember(t, db, foreignTrainerUser)
	testutil.AddKaderTrainer(t, db, kaderB, ftm)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/videos/"+itoa(vid),
		testutil.Token(t, foreignTrainerUser, "standard", []string{"trainer"}), nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign-team delete status = %d, want 403", res.StatusCode)
	}
	res.Body.Close()

	// row must survive the forbidden attempt
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM videos WHERE id=?`, vid).Scan(&n)
	if n != 1 {
		t.Errorf("video deleted despite 403")
	}

	// missing → 404
	res = testutil.Do(t, srv, http.MethodDelete, "/api/videos/999999",
		testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil), nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing delete status = %d, want 404", res.StatusCode)
	}
	res.Body.Close()
}

// --- helpers -----------------------------------------------------------------

func itoa(i int) string { return strconv.Itoa(i) }

// rawDir returns {root}/raw, the directory that holds raw/{id}.mp4.
func rawDir(root string) string { return filepath.Dir(RawPath(root, 0)) }
