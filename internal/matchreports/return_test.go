package matchreports_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Rückgabe eines eingereichten Berichts an den Autor (POST /return).

func setupPendingReport(t *testing.T, db *sql.DB) (reportID, authorID int) {
	t.Helper()
	_, _, gameID := setupBasicGame(t, db)
	authorID = testutil.CreateUser(t, db, auth.RoleStandard)
	reportID = testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}
	return reportID, authorID
}

func returnPath(id int) string { return fmt.Sprintf("/api/match-reports/%d/return", id) }

func TestReturn_HappyPath_AutorDarfWiederBearbeiten(t *testing.T) {
	db := testutil.NewDB(t)
	reportID, authorID := setupPendingReport(t, db)
	reviewerID := testutil.CreateMedienUser(t, db)
	srv := testServer(t, newHandlerWithPublisher(db, &fakePublisher{}))
	reviewerTok := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, returnPath(reportID), reviewerTok, map[string]string{"comment": "  Bitte Halbzeitstand ergänzen.  "})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}

	var state, comment string
	var returnedAt sql.NullString
	if err := db.QueryRow(`SELECT state, review_comment, returned_at FROM match_reports WHERE id=?`, reportID).
		Scan(&state, &comment, &returnedAt); err != nil {
		t.Fatal(err)
	}
	if state != "draft" || comment != "Bitte Halbzeitstand ergänzen." || !returnedAt.Valid {
		t.Errorf("state=%q comment=%q returned_at=%v", state, comment, returnedAt)
	}

	// Der eigentliche Zweck: der Autor darf wieder speichern und erneut einreichen.
	authorTok := testutil.Token(t, authorID, auth.RoleStandard, nil)
	if res := testutil.Put(t, srv, fmt.Sprintf("/api/match-reports/%d", reportID), authorTok,
		map[string]any{"abstract": "neu", "body_md": "neu"}); res.StatusCode != http.StatusOK {
		t.Fatalf("Autor-Update nach Rückgabe: %d — %s", res.StatusCode, readBody(t, res))
	}
	if res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/submit-for-review", reportID), authorTok, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("erneutes Einreichen: %d — %s", res.StatusCode, readBody(t, res))
	}

	// Der Kommentar bleibt für den Freigeber sichtbar.
	res = testutil.Get(t, srv, fmt.Sprintf("/api/match-reports/%d", reportID), reviewerTok)
	if body := readBody(t, res); !strings.Contains(body, "Bitte Halbzeitstand ergänzen.") {
		t.Errorf("review_comment fehlt im Get nach erneutem Einreichen: %s", body)
	}
}

func TestReturn_BenachrichtigtAutor(t *testing.T) {
	db := testutil.NewDB(t)
	reportID, authorID := setupPendingReport(t, db)
	reviewerID := testutil.CreateMedienUser(t, db)
	srv := testServer(t, newHandlerWithPublisher(db, &fakePublisher{}))
	tok := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	if res := testutil.Post(t, srv, returnPath(reportID), tok, map[string]string{"comment": "Tippfehler im Titel"}); res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	// Versand läuft asynchron — auf die Event-Log-Zeile warten.
	deadline := time.Now().Add(3 * time.Second)
	for {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM user_events WHERE user_id=? AND body LIKE '%Tippfehler im Titel%'`, authorID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("keine Meldung an den Autor im Event-Log")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestReturn_OhneKommentar_400(t *testing.T) {
	db := testutil.NewDB(t)
	reportID, _ := setupPendingReport(t, db)
	reviewerID := testutil.CreateMedienUser(t, db)
	srv := testServer(t, newHandlerWithPublisher(db, &fakePublisher{}))
	tok := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, returnPath(reportID), tok, map[string]string{"comment": "   "})
	if res.StatusCode != http.StatusBadRequest || !contains(readBody(t, res), "comment_required") {
		t.Fatalf("expected 400 comment_required, got %d", res.StatusCode)
	}
	res = testutil.Post(t, srv, returnPath(reportID), tok, map[string]string{"comment": strings.Repeat("x", 2001)})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 comment_too_long, got %d", res.StatusCode)
	}
	var state string
	_ = db.QueryRow(`SELECT state FROM match_reports WHERE id=?`, reportID).Scan(&state)
	if state != "pending_review" {
		t.Errorf("state=%q, abgelehnte Rückgabe darf nichts ändern", state)
	}
}

func TestReturn_AutorOhneFreigeberRecht_403(t *testing.T) {
	db := testutil.NewDB(t)
	reportID, authorID := setupPendingReport(t, db)
	srv := testServer(t, newHandlerWithPublisher(db, &fakePublisher{}))
	tok := testutil.Token(t, authorID, auth.RoleStandard, nil)

	res := testutil.Post(t, srv, returnPath(reportID), tok, map[string]string{"comment": "selbst zurückholen"})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestReturn_FalscherState_409(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreateUser(t, db, auth.RoleStandard)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0) // draft
	reviewerID := testutil.CreateMedienUser(t, db)
	srv := testServer(t, newHandlerWithPublisher(db, &fakePublisher{}))
	tok := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, returnPath(reportID), tok, map[string]string{"comment": "x"})
	if res.StatusCode != http.StatusConflict || !contains(readBody(t, res), "not_pending_review") {
		t.Fatalf("expected 409 not_pending_review, got %d", res.StatusCode)
	}
}

func TestReturn_Unbekannt_404(t *testing.T) {
	db := testutil.NewDB(t)
	reviewerID := testutil.CreateMedienUser(t, db)
	srv := testServer(t, newHandlerWithPublisher(db, &fakePublisher{}))
	tok := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	if res := testutil.Post(t, srv, returnPath(9999), tok, map[string]string{"comment": "x"}); res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}
