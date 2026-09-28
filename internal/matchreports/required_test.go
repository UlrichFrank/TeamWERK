package matchreports_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/matchreports"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Pflichtfelder (Abstract, Berichtstext) werden vor dem Zustandswechsel
// geprüft — der TYPO3-Import lehnt sie sonst mit „missing_fields" ab, und der
// Bericht landete in publish_failed.

func TestPublish_OhneAbstract_400_BleibtBearbeitbar(t *testing.T) {
	db := testutil.NewDB(t)
	reportID, _ := setupPendingReport(t, db)
	if _, err := db.Exec(`UPDATE match_reports SET abstract='  ' WHERE id=?`, reportID); err != nil {
		t.Fatal(err)
	}
	reviewerID := testutil.CreateMedienUser(t, db)
	fp := &fakePublisher{Result: &matchreports.PublishResult{PageUID: 1, URL: "https://example.org/x"}}
	srv := testServer(t, newHandlerWithPublisher(db, fp))
	tok := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), tok, nil)
	body := readBody(t, res)
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(body, `"missing_fields"`) || !strings.Contains(body, "abstract") {
		t.Fatalf("expected 400 missing_fields/abstract, got %d — %s", res.StatusCode, body)
	}
	if fp.Last != nil {
		t.Error("Publisher darf ohne Pflichtfelder nicht aufgerufen werden")
	}
	var state string
	_ = db.QueryRow(`SELECT state FROM match_reports WHERE id=?`, reportID).Scan(&state)
	if state != "pending_review" {
		t.Errorf("state=%q, want pending_review (kein publish_failed)", state)
	}

	// Freigeber ergänzt das Abstract und kann danach veröffentlichen.
	if res := testutil.Put(t, srv, fmt.Sprintf("/api/match-reports/%d", reportID), tok,
		map[string]any{"abstract": "Teaser", "body_md": "Text"}); res.StatusCode != http.StatusOK {
		t.Fatalf("Update: %d — %s", res.StatusCode, readBody(t, res))
	}
	if res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), tok, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("nach Ergänzung: %d — %s", res.StatusCode, readBody(t, res))
	}
}

func TestSubmit_OhneAbstractUndText_400(t *testing.T) {
	db := testutil.NewDB(t)
	_, _, gameID := setupBasicGame(t, db)
	authorID := testutil.CreateUser(t, db, auth.RoleStandard)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, 0)
	if _, err := db.Exec(`UPDATE match_reports SET abstract='', body_md='' WHERE id=?`, reportID); err != nil {
		t.Fatal(err)
	}
	srv := testServer(t, newHandlerWithPublisher(db, &fakePublisher{}))
	tok := testutil.Token(t, authorID, auth.RoleStandard, nil)

	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/submit-for-review", reportID), tok, nil)
	body := readBody(t, res)
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(body, "abstract,body_md") {
		t.Fatalf("expected 400 missing_fields abstract,body_md, got %d — %s", res.StatusCode, body)
	}
	var state string
	_ = db.QueryRow(`SELECT state FROM match_reports WHERE id=?`, reportID).Scan(&state)
	if state != "draft" {
		t.Errorf("state=%q, want draft", state)
	}
}

func TestReturn_AusPublishFailed_AutorKannKorrigieren(t *testing.T) {
	db := testutil.NewDB(t)
	reportID, authorID := setupPendingReport(t, db)
	if _, err := db.Exec(
		`UPDATE match_reports SET state='publish_failed', error_message='typo3 publisher status 400' WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}
	reviewerID := testutil.CreateMedienUser(t, db)
	srv := testServer(t, newHandlerWithPublisher(db, &fakePublisher{}))
	tok := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, returnPath(reportID), tok, map[string]string{"comment": "Bitte Abstract ergänzen"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}
	var state string
	var errMsg *string
	if err := db.QueryRow(`SELECT state, error_message FROM match_reports WHERE id=?`, reportID).Scan(&state, &errMsg); err != nil {
		t.Fatal(err)
	}
	if state != "draft" || errMsg != nil {
		t.Errorf("state=%q error_message=%v, want draft/NULL", state, errMsg)
	}
	authorTok := testutil.Token(t, authorID, auth.RoleStandard, nil)
	if res := testutil.Put(t, srv, fmt.Sprintf("/api/match-reports/%d", reportID), authorTok,
		map[string]any{"abstract": "Teaser", "body_md": "Text"}); res.StatusCode != http.StatusOK {
		t.Fatalf("Autor-Update: %d — %s", res.StatusCode, readBody(t, res))
	}
}
