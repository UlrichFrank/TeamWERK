package matchreports_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/matchreports"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// serveImageServer wires only GET …/images/{imgId}/blob → ServeImage.
func serveImageServer(t *testing.T, h *matchreports.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/match-reports/{id}/images/{imgId}/blob", h.ServeImage)
	})
}

// createReportWithImage inserts a draft match report for authorID plus one image
// whose storage_path points at storagePath. Returns the report and image IDs.
func createReportWithImage(t *testing.T, db *sql.DB, gameID, authorID int, storagePath string) (int, int) {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO match_reports (game_id, author_user_id, state) VALUES (?, ?, 'draft')`,
		gameID, authorID)
	if err != nil {
		t.Fatalf("insert match_report: %v", err)
	}
	reportID, _ := res.LastInsertId()
	res, err = db.Exec(
		`INSERT INTO match_report_images (report_id, position, storage_path) VALUES (?, 0, ?)`,
		reportID, storagePath)
	if err != nil {
		t.Fatalf("insert match_report_image: %v", err)
	}
	imageID, _ := res.LastInsertId()
	return int(reportID), int(imageID)
}

func blobPath(reportID, imageID int) string {
	return "/api/match-reports/" + itoa(reportID) + "/images/" + itoa(imageID) + "/blob"
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestServeImage_Unauthenticated(t *testing.T) {
	db := testutil.NewDB(t)
	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := serveImageServer(t, h)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreateUser(t, db, "standard")
	reportID, imageID := createReportWithImage(t, db, gameID, authorID, "/nonexistent")

	res := testutil.Get(t, srv, blobPath(reportID, imageID), "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", res.StatusCode)
	}
}

func TestServeImage_ForeignForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := serveImageServer(t, h)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreateUser(t, db, "standard")
	strangerID := testutil.CreateUser(t, db, "standard")
	reportID, imageID := createReportWithImage(t, db, gameID, authorID, "/nonexistent")

	tok := testutil.Token(t, strangerID, "standard", nil)
	res := testutil.Get(t, srv, blobPath(reportID, imageID), tok)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for non-author/non-reviewer, got %d", res.StatusCode)
	}
}

func TestServeImage_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := serveImageServer(t, h)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreateUser(t, db, "standard")
	reportID, _ := createReportWithImage(t, db, gameID, authorID, "/nonexistent")

	tok := testutil.Token(t, authorID, "standard", nil)
	// existing report, unknown image id
	res := testutil.Get(t, srv, blobPath(reportID, 99999), tok)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for unknown image, got %d", res.StatusCode)
	}
}

func TestServeImage_AuthorOK(t *testing.T) {
	db := testutil.NewDB(t)
	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := serveImageServer(t, h)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreateUser(t, db, "standard")

	path := filepath.Join(t.TempDir(), "img.jpg")
	if err := os.WriteFile(path, []byte("\xff\xd8\xff jpeg-ish"), 0o600); err != nil {
		t.Fatal(err)
	}
	reportID, imageID := createReportWithImage(t, db, gameID, authorID, path)

	tok := testutil.Token(t, authorID, "standard", nil)
	res := testutil.Get(t, srv, blobPath(reportID, imageID), tok)
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for author, got %d", res.StatusCode)
	}
}

func TestServeImage_ReviewerOK(t *testing.T) {
	db := testutil.NewDB(t)
	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := serveImageServer(t, h)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreateUser(t, db, "standard")
	reviewerID := testutil.CreateUser(t, db, "standard")

	path := filepath.Join(t.TempDir(), "img.jpg")
	if err := os.WriteFile(path, []byte("\xff\xd8\xff jpeg-ish"), 0o600); err != nil {
		t.Fatal(err)
	}
	reportID, imageID := createReportWithImage(t, db, gameID, authorID, path)

	// reviewer via club_function "vorstand" (isReviewer → true)
	tok := testutil.Token(t, reviewerID, "standard", []string{"vorstand"})
	res := testutil.Get(t, srv, blobPath(reportID, imageID), tok)
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for reviewer, got %d", res.StatusCode)
	}
}
