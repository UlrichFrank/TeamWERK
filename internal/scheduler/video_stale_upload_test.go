package scheduler

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// setVideoCreatedAt überschreibt created_at relativ zu jetzt (negativ = Vergangenheit, in Stunden).
func setVideoCreatedAt(t *testing.T, db *sql.DB, videoID, hoursFromNow int) {
	t.Helper()
	spec := "+" + itoa(hoursFromNow) + " hours"
	if hoursFromNow < 0 {
		spec = itoa(hoursFromNow) + " hours"
	}
	if _, err := db.Exec(`UPDATE videos SET created_at = datetime('now', ?) WHERE id = ?`, spec, videoID); err != nil {
		t.Fatalf("setVideoCreatedAt: %v", err)
	}
}

func videoStatus(t *testing.T, db *sql.DB, id int) (string, string) {
	t.Helper()
	var status string
	var reason sql.NullString
	if err := db.QueryRow(`SELECT status, failure_reason FROM videos WHERE id = ?`, id).Scan(&status, &reason); err != nil {
		t.Fatalf("videoStatus: %v", err)
	}
	return status, reason.String
}

// TestFailStaleVideoUploads prüft, dass nur alte uploading-Zeilen (>24 h) auf
// failed gesetzt werden — frische uploading-Zeilen und bereits abgeschlossene
// Videos bleiben unangetastet.
func TestFailStaleVideoUploads(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	user := testutil.CreateUser(t, db, "standard")

	stale := testutil.CreateVideo(t, db, team, season, user, "uploading")
	setVideoCreatedAt(t, db, stale, -25)

	fresh := testutil.CreateVideo(t, db, team, season, user, "uploading")
	setVideoCreatedAt(t, db, fresh, -1)

	queued := testutil.CreateVideo(t, db, team, season, user, "queued")
	setVideoCreatedAt(t, db, queued, -48)

	s := New(db, testutil.TestConfig(), nil)
	s.failStaleVideoUploads()

	if status, reason := videoStatus(t, db, stale); status != "failed" || reason != "Upload abgebrochen" {
		t.Errorf("stale upload: status=%q reason=%q, want failed / \"Upload abgebrochen\"", status, reason)
	}
	if status, _ := videoStatus(t, db, fresh); status != "uploading" {
		t.Errorf("fresh upload: status=%q, want uploading (unverändert)", status)
	}
	if status, _ := videoStatus(t, db, queued); status != "queued" {
		t.Errorf("queued video: status=%q, want queued (unverändert)", status)
	}
}
