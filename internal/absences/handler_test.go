package absences_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/absences"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// parentToken issues a Bearer token with is_parent=true, since the default
// testutil.Token always passes isParent=false.
func parentToken(t *testing.T, userID int) string {
	t.Helper()
	tok, err := auth.IssueAccessToken(testutil.TestJWTSecret, userID, "parent@test.local", "standard", nil, true)
	if err != nil {
		t.Fatalf("parentToken: %v", err)
	}
	return "Bearer " + tok
}

func linkFamily(t *testing.T, db *sql.DB, parentUserID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentUserID, memberID); err != nil {
		t.Fatalf("linkFamily: %v", err)
	}
}

func countAbsences(t *testing.T, db *sql.DB, memberID int) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM member_absences WHERE member_id = ?`, memberID).Scan(&n); err != nil {
		t.Fatalf("countAbsences: %v", err)
	}
	return n
}

func newAbsenceServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	h := absences.NewHandler(db, hub.NewHub())
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/absences/preview", h.Preview)
		r.Post("/api/absences", h.Create)
	})
}

// TestCreateAbsence_MultiChild_Success: parent posts a vacation for two
// linked children → both rows inserted, response carries absence_ids.
func TestCreateAbsence_MultiChild_Success(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childA := testutil.CreateMember(t, db, 0)
	childB := testutil.CreateMember(t, db, 0)
	linkFamily(t, db, parentUserID, childA)
	linkFamily(t, db, parentUserID, childB)

	srv := newAbsenceServer(t, db)
	body, _ := json.Marshal(map[string]any{
		"member_ids": []int{childA, childB},
		"type":       "vacation",
		"start_date": "2026-07-01",
		"end_date":   "2026-07-14",
		"note":       "Familienurlaub",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/absences", bytes.NewReader(body))
	req.Header.Set("Authorization", parentToken(t, parentUserID))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var resp struct {
		AbsenceIDs []int `json:"absence_ids"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	if len(resp.AbsenceIDs) != 2 {
		t.Errorf("expected 2 absence_ids, got %v", resp.AbsenceIDs)
	}
	if countAbsences(t, db, childA) != 1 {
		t.Errorf("childA absence not inserted")
	}
	if countAbsences(t, db, childB) != 1 {
		t.Errorf("childB absence not inserted")
	}
}

// TestCreateAbsence_MultiChild_AllOrNothing: one of two children has an
// overlapping vacation → 409 with conflicts list, no row inserted for either.
func TestCreateAbsence_MultiChild_AllOrNothing(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childA := testutil.CreateMember(t, db, 0)
	childB := testutil.CreateMember(t, db, 0)
	linkFamily(t, db, parentUserID, childA)
	linkFamily(t, db, parentUserID, childB)

	// Pre-seed childA with an overlapping vacation.
	if _, err := db.Exec(
		`INSERT INTO member_absences (member_id, type, start_date, end_date, created_by)
		 VALUES (?, 'vacation', '2026-07-05', '2026-07-10', ?)`,
		childA, parentUserID); err != nil {
		t.Fatalf("seed pre-existing: %v", err)
	}

	srv := newAbsenceServer(t, db)
	body, _ := json.Marshal(map[string]any{
		"member_ids": []int{childA, childB},
		"type":       "vacation",
		"start_date": "2026-07-01",
		"end_date":   "2026-07-14",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/absences", bytes.NewReader(body))
	req.Header.Set("Authorization", parentToken(t, parentUserID))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
	var resp struct {
		Error     string `json:"error"`
		Conflicts []struct {
			MemberID int `json:"member_id"`
		} `json:"conflicts"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	if resp.Error != "overlap" {
		t.Errorf("expected error=overlap, got %q", resp.Error)
	}
	if len(resp.Conflicts) != 1 || resp.Conflicts[0].MemberID != childA {
		t.Errorf("expected conflicts=[childA=%d], got %v", childA, resp.Conflicts)
	}
	// All-or-nothing: childB still has only 0 absences, childA still has 1.
	if got := countAbsences(t, db, childB); got != 0 {
		t.Errorf("childB should have no new absence, got %d", got)
	}
	if got := countAbsences(t, db, childA); got != 1 {
		t.Errorf("childA should still have only the pre-existing absence, got %d", got)
	}
}

// TestCreateAbsence_Legacy_SingleMemberID: old { member_id: N } body still works
// and returns { id: N } (no breaking change).
func TestCreateAbsence_Legacy_SingleMemberID(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childA := testutil.CreateMember(t, db, 0)
	linkFamily(t, db, parentUserID, childA)

	srv := newAbsenceServer(t, db)
	body, _ := json.Marshal(map[string]any{
		"member_id":  childA,
		"type":       "vacation",
		"start_date": "2026-07-01",
		"end_date":   "2026-07-14",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/absences", bytes.NewReader(body))
	req.Header.Set("Authorization", parentToken(t, parentUserID))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var resp map[string]any
	json.NewDecoder(res.Body).Decode(&resp)
	if _, hasID := resp["id"]; !hasID {
		t.Errorf("legacy response should contain 'id' field, got %v", resp)
	}
	if _, hasIDs := resp["absence_ids"]; hasIDs {
		t.Errorf("legacy response must NOT contain absence_ids, got %v", resp)
	}
	if countAbsences(t, db, childA) != 1 {
		t.Errorf("childA absence not inserted")
	}
}

// TestPreview_MultiChild_Union verifies that when both kids have a confirmed
// response to the same training, that training appears in the preview only once.
func TestPreview_MultiChild_Union(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-07-05")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childA := testutil.CreateMember(t, db, 0)
	childB := testutil.CreateMember(t, db, 0)
	linkFamily(t, db, parentUserID, childA)
	linkFamily(t, db, parentUserID, childB)

	// Both children confirmed for the same training.
	for _, mid := range []int{childA, childB} {
		if _, err := db.Exec(
			`INSERT INTO training_responses (training_id, member_id, responded_by, status)
			 VALUES (?, ?, ?, 'confirmed')`,
			sessionID, mid, parentUserID); err != nil {
			t.Fatalf("seed response: %v", err)
		}
	}

	srv := newAbsenceServer(t, db)
	url := fmt.Sprintf("%s/api/absences/preview?member_ids=%d,%d&from=2026-07-01&to=2026-07-14",
		srv.URL, childA, childB)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", parentToken(t, parentUserID))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var events []map[string]any
	json.NewDecoder(res.Body).Decode(&events)
	if len(events) != 1 {
		t.Fatalf("expected 1 deduped event, got %d: %v", len(events), events)
	}
}

// TC-AB-EXT01: Fremdes Mitglied (kein Familienlink) → 403.
func TestCreateAbsence_UnauthorizedMember(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	parentUserID := testutil.CreateUser(t, db, "standard")
	unlinkedMember := testutil.CreateMember(t, db, 0) // no family_links to parentUser

	srv := newAbsenceServer(t, db)
	body, _ := json.Marshal(map[string]any{
		"member_ids": []int{unlinkedMember},
		"type":       "vacation",
		"start_date": "2026-07-01",
		"end_date":   "2026-07-14",
	})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/absences", bytes.NewReader(body))
	req.Header.Set("Authorization", parentToken(t, parentUserID))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for unlinked member, got %d", res.StatusCode)
	}
	if countAbsences(t, db, unlinkedMember) != 0 {
		t.Error("no absence should be created for unauthorized member")
	}
}

// TC-AB-EXT02: Preview mit leerem Zeitraum (keine Events) gibt [] zurück.
func TestPreview_Empty(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")

	parentUserID := testutil.CreateUser(t, db, "standard")
	childMember := testutil.CreateMember(t, db, 0)
	linkFamily(t, db, parentUserID, childMember)

	srv := newAbsenceServer(t, db)
	url := fmt.Sprintf("%s/api/absences/preview?member_ids=%d&from=2026-08-01&to=2026-08-31",
		srv.URL, childMember)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", parentToken(t, parentUserID))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var events []map[string]any
	json.NewDecoder(res.Body).Decode(&events)
	if len(events) != 0 {
		t.Errorf("expected 0 events for empty range, got %d", len(events))
	}
}

// ── Sichtbarkeit & Mutation: Owner/Staff-Scoping ─────────────────────────────

func absenceRWServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	h := absences.NewHandler(db, hub.NewHub())
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/absences", h.List)
		r.Get("/api/absences/calendar", h.Calendar)
		r.Put("/api/absences/{id}", h.Update)
		r.Delete("/api/absences/{id}", h.Delete)
	})
}

func decodeAbsenceMemberIDs(t *testing.T, res *http.Response) []int {
	t.Helper()
	var items []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()
	ids := make([]int, 0, len(items))
	for _, it := range items {
		if v, ok := it["member_id"].(float64); ok {
			ids = append(ids, int(v))
		}
	}
	return ids
}

// TestCalendar_ShowTeam_MemberSeesNoTeamAbsences nagelt die Leak-Prävention fest: ein
// einfaches Mitglied (weder vorstand/trainer noch admin) sieht mit ?show_team=true KEINE
// fremden Team-Abwesenheiten — der Team-Block wird durch das canSeeTeam-Gate übersprungen.
func TestCalendar_ShowTeam_MemberSeesNoTeamAbsences(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	requesterUserID := testutil.CreateUser(t, db, "standard")

	foreignMemberID := testutil.CreateMember(t, db, 0)
	testutil.AddKaderMember(t, db, kaderID, foreignMemberID)
	db.Exec(`UPDATE members SET absences_public=1 WHERE id=?`, foreignMemberID)
	testutil.CreateAbsence(t, db, foreignMemberID, "vacation", "2026-03-01", "2026-03-05", testutil.CreateUser(t, db, "standard"))

	srv := absenceRWServer(t, db)
	tok := testutil.Token(t, requesterUserID, "standard", nil)
	res := testutil.Get(t, srv, fmt.Sprintf("/api/absences/calendar?from=2026-03-01&to=2026-03-31&show_team=true&team_id=%d", teamID), tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	ids := decodeAbsenceMemberIDs(t, res)
	for _, id := range ids {
		if id == foreignMemberID {
			t.Fatalf("plain member must not see foreign team absence (member %d leaked)", foreignMemberID)
		}
	}
}

// TestCalendar_ShowTeam_VorstandSeesTeam ist der Gegenpol: ein vorstand mit Kader-Zugang
// (user_accessible_teams) sieht die öffentliche Team-Abwesenheit.
func TestCalendar_ShowTeam_VorstandSeesTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	// vorstand user must be in user_accessible_teams for this team → via kader_trainers
	vorstandUserID := testutil.CreateUser(t, db, "standard")
	vorstandMemberID := testutil.CreateMember(t, db, vorstandUserID)
	testutil.AddKaderTrainer(t, db, kaderID, vorstandMemberID)

	foreignMemberID := testutil.CreateMember(t, db, 0)
	testutil.AddKaderMember(t, db, kaderID, foreignMemberID)
	db.Exec(`UPDATE members SET absences_public=1 WHERE id=?`, foreignMemberID)
	testutil.CreateAbsence(t, db, foreignMemberID, "vacation", "2026-03-01", "2026-03-05", testutil.CreateUser(t, db, "standard"))

	srv := absenceRWServer(t, db)
	tok := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/absences/calendar?from=2026-03-01&to=2026-03-31&show_team=true&team_id=%d", teamID), tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	ids := decodeAbsenceMemberIDs(t, res)
	found := false
	for _, id := range ids {
		if id == foreignMemberID {
			found = true
		}
	}
	if !found {
		t.Errorf("vorstand with kader access must see public team absence of member %d", foreignMemberID)
	}
}

func TestUpdate_ForeignForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	ownerUserID := testutil.CreateUser(t, db, "standard")
	ownerMemberID := testutil.CreateMember(t, db, ownerUserID)
	absID := testutil.CreateAbsence(t, db, ownerMemberID, "vacation", "2026-03-01", "2026-03-05", ownerUserID)

	strangerUserID := testutil.CreateUser(t, db, "standard")

	srv := absenceRWServer(t, db)
	tok := testutil.Token(t, strangerUserID, "standard", nil)
	res := testutil.Put(t, srv, fmt.Sprintf("/api/absences/%d", absID), tok,
		map[string]any{"type": "vacation", "start_date": "2026-03-02", "end_date": "2026-03-06"})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 updating foreign absence, got %d", res.StatusCode)
	}
}

func TestDelete_ForeignForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	ownerUserID := testutil.CreateUser(t, db, "standard")
	ownerMemberID := testutil.CreateMember(t, db, ownerUserID)
	absID := testutil.CreateAbsence(t, db, ownerMemberID, "vacation", "2026-03-01", "2026-03-05", ownerUserID)

	strangerUserID := testutil.CreateUser(t, db, "standard")

	srv := absenceRWServer(t, db)
	tok := testutil.Token(t, strangerUserID, "standard", nil)
	res := testutil.Delete(t, srv, fmt.Sprintf("/api/absences/%d", absID), tok)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 deleting foreign absence, got %d", res.StatusCode)
	}
	if got := countAbsences(t, db, ownerMemberID); got != 1 {
		t.Errorf("foreign absence must remain, count=%d", got)
	}
}

func TestList_NoForeignAbsences(t *testing.T) {
	db := testutil.NewDB(t)
	ownerUserID := testutil.CreateUser(t, db, "standard")
	ownerMemberID := testutil.CreateMember(t, db, ownerUserID)
	testutil.CreateAbsence(t, db, ownerMemberID, "vacation", "2026-03-01", "2026-03-05", ownerUserID)

	strangerUserID := testutil.CreateUser(t, db, "standard")

	srv := absenceRWServer(t, db)
	tok := testutil.Token(t, strangerUserID, "standard", nil)
	res := testutil.Get(t, srv, "/api/absences", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	ids := decodeAbsenceMemberIDs(t, res)
	if len(ids) != 0 {
		t.Errorf("stranger must not see any foreign absences, got member_ids %v", ids)
	}
}
