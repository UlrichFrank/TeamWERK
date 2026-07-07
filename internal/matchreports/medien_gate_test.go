package matchreports_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/matchreports"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ─── TC-MG01 · Submit-for-Review Happy Path ───────────────────────────────────

func TestSubmitForReview_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)

	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/submit-for-review", reportID), token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}

	var state, submittedAt string
	if err := db.QueryRow(
		`SELECT state, COALESCE(submitted_at,'') FROM match_reports WHERE id=?`, reportID,
	).Scan(&state, &submittedAt); err != nil {
		t.Fatal(err)
	}
	if state != "pending_review" {
		t.Errorf("expected state=pending_review, got %s", state)
	}
	if submittedAt == "" {
		t.Errorf("expected submitted_at to be set")
	}
}

// ─── TC-MG02 · Submit durch Nicht-Autor → 403 ─────────────────────────────────

func TestSubmitForReview_NotAuthor(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	strangerID := testutil.CreatePressTeamUser(t, db)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, strangerID, auth.RolePressTeam, nil)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/submit-for-review", reportID), token, nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// ─── TC-MG03 · Submit doppelt → 409 ───────────────────────────────────────────

func TestSubmitForReview_AlreadySubmitted(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/submit-for-review", reportID), token, nil)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

// ─── TC-MG04 · Publish auf draft → 409 not_submitted ──────────────────────────

func TestPublish_NotSubmitted(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	reviewerID := testutil.CreateMedienUser(t, db)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
	body := readBody(t, res)
	if !contains(body, "not_submitted") {
		t.Errorf("expected error=not_submitted, got %s", body)
	}
}

// ─── TC-MG05 · Publish durch Autor ohne Medien-Fkt → 403 role_required ────────

func TestPublish_AuthorWithoutReviewer(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d — %s", res.StatusCode, readBody(t, res))
	}
}

// ─── TC-MG06 · Publish durch Vorstand-Freigeber (Fallback) ────────────────────

func TestPublish_VorstandCanPublish(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}
	vorstandID := testutil.CreateVorstandUser(t, db)

	fp := &fakePublisher{Result: &matchreports.PublishResult{PageUID: 42, URL: "https://ts.org/spielberichte/x"}}
	h := newHandlerWithPublisher(db, fp)
	srv := testServer(t, h)
	token := testutil.Token(t, vorstandID, auth.RoleStandard, []string{"vorstand"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}
}

// ─── TC-MG07 · PUT auf pending_review durch Autor → 403 ───────────────────────

func TestUpdate_AuthorCannotEditPending(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)
	res := testutil.Do(t, srv, "PUT", fmt.Sprintf("/api/match-reports/%d", reportID),
		token, map[string]any{"abstract": "neuer text"})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// ─── TC-MG08 · PUT auf pending_review durch Medien-Freigeber → 200 ────────────

func TestUpdate_ReviewerCanEditPending(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}
	reviewerID := testutil.CreateMedienUser(t, db)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})
	res := testutil.Do(t, srv, "PUT", fmt.Sprintf("/api/match-reports/%d", reportID),
		token, map[string]any{"abstract": "vom Freigeber korrigiert"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}
	var abstract string
	if err := db.QueryRow(`SELECT abstract FROM match_reports WHERE id=?`, reportID).Scan(&abstract); err != nil {
		t.Fatal(err)
	}
	if abstract != "vom Freigeber korrigiert" {
		t.Errorf("abstract not updated by reviewer, got %q", abstract)
	}
}

// ─── TC-MG09 · GET /pending liefert Liste für Freigeber, 403 sonst ────────────

func TestGetPending_ReviewerSeesList(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}
	reviewerID := testutil.CreateMedienUser(t, db)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})
	res := testutil.Get(t, srv, "/api/match-reports/pending", token)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}
	if !contains(readBody(t, res), fmt.Sprintf(`"id":%d`, reportID)) {
		t.Errorf("expected report id in pending list")
	}
}

func TestGetPending_NonReviewer(t *testing.T) {
	db := testutil.NewDB(t)
	authorID := testutil.CreatePressTeamUser(t, db)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)
	res := testutil.Get(t, srv, "/api/match-reports/pending", token)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-reviewer, got %d", res.StatusCode)
	}
}

// ─── TC-MG10 · Autor-mit-Medien-Fkt darf sich selbst freigeben (Vier-Augen weich)

func TestPublish_AuthorWithMedienCanSelfPublish(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)

	// User ist gleichzeitig Presseteam (Autor-Rolle) UND hat Vereinsfunktion 'medien'.
	authorID := testutil.CreatePressTeamUser(t, db)
	memberID := testutil.CreateMember(t, db, authorID)
	testutil.AddClubFunction(t, db, memberID, "medien")

	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}

	fp := &fakePublisher{Result: &matchreports.PublishResult{PageUID: 42, URL: "https://ts.org/spielberichte/x"}}
	h := newHandlerWithPublisher(db, fp)
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, []string{"medien"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 (D-2: Vier-Augen weich), got %d — %s", res.StatusCode, readBody(t, res))
	}
}

// ─── Hilfen ───────────────────────────────────────────────────────────────────

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
