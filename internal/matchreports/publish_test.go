package matchreports_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/matchreports"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TC-MR-P01 · Publisher-Payload trägt TeamCategoryName aus TeamDisplayShort.
//
// Setup: Team mit age_class="Erwachsene", gender="mixed" + Kader in aktiver
// Saison. TeamDisplayShort → "g" (mixed) + "E" (first char age_class) → "gE".
// Da nur ein Kader existiert, entfällt das Team-Number-Suffix.
func TestPublish_PayloadHasTeamCategoryName(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	// Kader für die aktive Saison — ohne den liefert TeamDisplayShort NULL.
	testutil.CreateKader(t, db, teamID, seasonID)

	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}

	reviewerID := testutil.CreateMedienUser(t, db)
	fp := &fakePublisher{
		Result: &matchreports.PublishResult{PageUID: 1, URL: "https://example.org/x"},
	}
	h := newHandlerWithPublisher(db, fp)
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}
	if fp.Last == nil {
		t.Fatal("publisher never called")
	}
	if fp.Last.Meta.TeamCategoryName != "gE" {
		t.Errorf("TeamCategoryName = %q, want %q", fp.Last.Meta.TeamCategoryName, "gE")
	}
}

// TC-MR-P02 · Publisher-Payload trägt Season aus aktiver Saison.
//
// Aktive Saison heißt "2025/26" → Slug-Segment "2025-2026".
func TestPublish_PayloadHasSeasonFromActiveSeason(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db) // Season name "2025/26"
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}

	reviewerID := testutil.CreateMedienUser(t, db)
	fp := &fakePublisher{
		Result: &matchreports.PublishResult{PageUID: 1, URL: "https://example.org/x"},
	}
	h := newHandlerWithPublisher(db, fp)
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}
	if got, want := fp.Last.Meta.Season, "2025-2026"; got != want {
		t.Errorf("Season = %q, want %q", got, want)
	}
}

// TC-MR-P03 · Publisher-Payload trägt user-editierten Titel aus DB.
//
// Setup: title="Sieg vs. Muster" wird direkt in der DB gesetzt (simuliert das,
// was der Update-Handler nach dem Frontend-Save schreibt). Der Payload muss
// diesen Titel 1:1 tragen, nicht mehr `BuildTitle(date, opponent)`.
func TestPublish_PayloadUsesUserTitleFromDB(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', title=?, submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		"Sieg vs. Muster", reportID); err != nil {
		t.Fatal(err)
	}

	reviewerID := testutil.CreateMedienUser(t, db)
	fp := &fakePublisher{
		Result: &matchreports.PublishResult{PageUID: 1, URL: "https://example.org/x"},
	}
	h := newHandlerWithPublisher(db, fp)
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}
	if got, want := fp.Last.Meta.Title, "Sieg vs. Muster"; got != want {
		t.Errorf("Title = %q, want %q", got, want)
	}
	// Slug wird aus Titel abgeleitet.
	if got, want := fp.Last.Meta.Slug, "sieg-vs-muster"; got != want {
		t.Errorf("Slug = %q, want %q", got, want)
	}
}

// TC-MR-P04 · Publish ohne aktive Saison → HTTP 500 no_active_season,
// State bleibt pending_review.
//
// Guard-Check läuft VOR dem State-Wechsel — deshalb kein publish_failed.
func TestPublish_NoActiveSeason(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}
	// Aktive Saison rausnehmen — simuliert „noch keine Saison gesetzt".
	if _, err := db.Exec(`UPDATE seasons SET is_active=0`); err != nil {
		t.Fatal(err)
	}

	reviewerID := testutil.CreateMedienUser(t, db)
	fp := &fakePublisher{}
	h := newHandlerWithPublisher(db, fp)
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d — %s", res.StatusCode, readBody(t, res))
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "no_active_season" {
		t.Errorf("error = %q, want %q", body["error"], "no_active_season")
	}
	// State bleibt pending_review, KEIN publish_failed.
	var state string
	var errMsg sql.NullString
	if err := db.QueryRow(
		`SELECT state, error_message FROM match_reports WHERE id=?`, reportID,
	).Scan(&state, &errMsg); err != nil {
		t.Fatal(err)
	}
	if state != "pending_review" {
		t.Errorf("state = %q, want pending_review", state)
	}
	// Publisher darf gar nicht aufgerufen worden sein.
	if fp.Last != nil {
		t.Error("publisher was called but should not have been")
	}
}
