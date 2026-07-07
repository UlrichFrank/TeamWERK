package matchreports_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/matchreports"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ─── fixtures & helpers ────────────────────────────────────────────────────────

// fakePublisher speichert den letzten Request und liefert konfigurierbare
// Antworten — ersetzt den HTTP-Publisher in Tests.
type fakePublisher struct {
	Last   *matchreports.PublishRequest
	Result *matchreports.PublishResult
	Err    error
}

func (f *fakePublisher) Publish(_ context.Context, req *matchreports.PublishRequest) (*matchreports.PublishResult, error) {
	f.Last = req
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Result, nil
}

func testServer(t *testing.T, h *matchreports.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		// Reihenfolge wichtig: statische Segmente (/pending, /my) VOR den
		// {id}-Routen registrieren, sonst matcht chi „pending" als id.
		r.Get("/api/match-reports/pending", h.Pending)
		r.Post("/api/match-reports", h.Create)
		r.Get("/api/match-reports/{id}", h.Get)
		r.Put("/api/match-reports/{id}", h.Update)
		r.Delete("/api/match-reports/{id}", h.Delete)
		r.Post("/api/match-reports/{id}/publish", h.Publish)
		r.Post("/api/match-reports/{id}/submit-for-review", h.SubmitForReview)
	})
}

func newHandlerWithPublisher(db *sql.DB, p matchreports.Publisher) *matchreports.Handler {
	cfg := &appconfig.Config{
		JWTSecret:           testutil.TestJWTSecret,
		MatchReportImageDir: t_tempDir(),
	}
	return matchreports.NewHandlerWithPublisher(db, hub.NewHub(), cfg, p)
}

// t_tempDir liefert ein statisches Temp-Verzeichnis — die Tests bauen keine
// echten Bilder, daher irrelevant, aber der Handler will einen Pfad.
func t_tempDir() string { return "/tmp/matchreports-test" }

func setupBasicGame(t *testing.T, db *sql.DB) (seasonID, teamID, gameID int) {
	t.Helper()
	seasonID = testutil.CreateSeason(t, db, "2025/26")
	teamID = testutil.CreateTeam(t, db, "Team A")
	gameID = testutil.CreateGame(t, db, seasonID, teamID, "2026-05-15")
	return
}

// createSlotWithAssignee legt einen Duty-Slot inkl. Assignment für den User an.
func createSlotWithAssignee(t *testing.T, db *sql.DB, seasonID, teamID, gameID, userID int) int {
	t.Helper()
	dtID := testutil.CreateDutyType(t, db, "Spielbericht", 0.5)
	slotID := testutil.CreateDutySlot(t, db, dtID, seasonID, teamID, gameID, "2026-05-15")
	testutil.AssignDutySlot(t, db, slotID, userID)
	return slotID
}

// ─── TC-MR01 · Create Happy Path ──────────────────────────────────────────────

func TestCreate_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)

	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)
	res := testutil.Post(t, srv, "/api/match-reports", token,
		map[string]int{"game_id": gameID, "duty_slot_id": slotID})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d — %s", res.StatusCode, readBody(t, res))
	}

	var got struct{ ID int }
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID == 0 {
		t.Fatal("expected id in response")
	}
}

// ─── TC-MR02 · Create Non-Presseteam → 403 ────────────────────────────────────

func TestCreate_NonPressTeamForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	userID := testutil.CreateUser(t, db, auth.RoleStandard)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, userID)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)

	token := testutil.Token(t, userID, auth.RoleStandard, nil)
	res := testutil.Post(t, srv, "/api/match-reports", token,
		map[string]int{"game_id": gameID, "duty_slot_id": slotID})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// ─── TC-MR03 · Create ohne Slot-Ownership → 403 ───────────────────────────────

func TestCreate_ForeignSlot(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	otherID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, otherID)

	// Zweiter Presseteam-User ohne Assignment.
	requesterID := testutil.CreatePressTeamUser(t, db)
	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)

	token := testutil.Token(t, requesterID, auth.RolePressTeam, nil)
	res := testutil.Post(t, srv, "/api/match-reports", token,
		map[string]int{"game_id": gameID, "duty_slot_id": slotID})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// ─── TC-MR03b · Create mit Slot für ein fremdes Spiel → 403 ───────────────────
//
// Regression gegen die IDOR-Lücke: der Requester besitzt den Spielbericht-Slot
// für Spiel A, nennt im Request aber Spiel B. Ohne Slot↔Game-Bindung würde er
// so Autor des Berichts für ein Spiel, für das er nicht zuständig ist.
func TestCreate_SlotForDifferentGame(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameA := setupBasicGame(t, db)
	gameB := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-22")
	authorID := testutil.CreatePressTeamUser(t, db)
	// Slot + Assignment gehören zu Spiel A.
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameA, authorID)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)

	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)
	res := testutil.Post(t, srv, "/api/match-reports", token,
		map[string]int{"game_id": gameB, "duty_slot_id": slotID})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d — %s", res.StatusCode, readBody(t, res))
	}
}

// ─── TC-MR04 · Zweiter Draft für dasselbe Spiel → 409 ─────────────────────────

func TestCreate_Duplicate(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	// Vorhandener Bericht (via Fixture direkt in DB).
	testutil.CreateMatchReport(t, db, gameID, authorID, slotID)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)

	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)
	res := testutil.Post(t, srv, "/api/match-reports", token,
		map[string]int{"game_id": gameID, "duty_slot_id": slotID})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

// ─── TC-MR05 · Update im State published → 409 ────────────────────────────────

func TestUpdate_PublishedIsReadOnly(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)
	if _, err := db.Exec(`UPDATE match_reports SET state='published' WHERE id=?`, reportID); err != nil {
		t.Fatal(err)
	}

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/match-reports/%d", reportID),
		token, map[string]any{"abstract": "neuer text"})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

// ─── TC-MR06 · Publish Happy Path ─────────────────────────────────────────────

func TestPublish_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)
	// Publish erwartet State=pending_review — Bericht muss zuerst eingereicht sein.
	if _, err := db.Exec(
		`UPDATE match_reports SET state='pending_review', submitted_at=CURRENT_TIMESTAMP WHERE id=?`,
		reportID); err != nil {
		t.Fatal(err)
	}
	reviewerID := testutil.CreateMedienUser(t, db)

	fp := &fakePublisher{
		Result: &matchreports.PublishResult{PageUID: 1234, URL: "https://ts.org/spielberichte/x"},
	}
	h := newHandlerWithPublisher(db, fp)
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}
	// State geprüft.
	var state string
	if err := db.QueryRow(`SELECT state FROM match_reports WHERE id=?`, reportID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "published" {
		t.Errorf("expected state=published, got %s", state)
	}
	// reviewer_user_id gesetzt.
	var reviewerUID sql.NullInt64
	if err := db.QueryRow(`SELECT reviewer_user_id FROM match_reports WHERE id=?`, reportID).Scan(&reviewerUID); err != nil {
		t.Fatal(err)
	}
	if !reviewerUID.Valid || int(reviewerUID.Int64) != reviewerID {
		t.Errorf("expected reviewer_user_id=%d, got %v", reviewerID, reviewerUID)
	}
	// Duty-Assignment fulfilled.
	var assignStatus string
	if err := db.QueryRow(
		`SELECT status FROM duty_assignments WHERE duty_slot_id=? AND user_id=?`,
		slotID, authorID).Scan(&assignStatus); err != nil {
		t.Fatal(err)
	}
	if assignStatus != "fulfilled" {
		t.Errorf("expected duty_assignments.status=fulfilled, got %s", assignStatus)
	}
	// Publisher hat einen Request bekommen.
	if fp.Last == nil {
		t.Fatal("publisher never called")
	}
}

// ─── TC-MR07 · Publish Doppel-Klick → 409 ─────────────────────────────────────

func TestPublish_AlreadyPublishing(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)
	if _, err := db.Exec(`UPDATE match_reports SET state='publishing' WHERE id=?`, reportID); err != nil {
		t.Fatal(err)
	}

	reviewerID := testutil.CreateMedienUser(t, db)
	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

// ─── TC-MR08 · Publisher-Fehler → 502 + publish_failed ────────────────────────

func TestPublish_PublisherError(t *testing.T) {
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
	reviewerID := testutil.CreateMedienUser(t, db)

	fp := &fakePublisher{Err: errors.New("typo3 unreachable")}
	h := newHandlerWithPublisher(db, fp)
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d — %s", res.StatusCode, readBody(t, res))
	}
	var state, errMsg string
	if err := db.QueryRow(
		`SELECT state, COALESCE(error_message,'') FROM match_reports WHERE id=?`, reportID,
	).Scan(&state, &errMsg); err != nil {
		t.Fatal(err)
	}
	if state != "publish_failed" {
		t.Errorf("expected state=publish_failed, got %s", state)
	}
	if !strings.Contains(errMsg, "typo3 unreachable") {
		t.Errorf("expected error_message to include publisher error, got %q", errMsg)
	}
}

// ─── TC-MR09 · Publish Already Published → 409 ────────────────────────────────

func TestPublish_AlreadyPublished(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)
	if _, err := db.Exec(`UPDATE match_reports SET state='published' WHERE id=?`, reportID); err != nil {
		t.Fatal(err)
	}

	reviewerID := testutil.CreateMedienUser(t, db)
	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, reviewerID, auth.RoleStandard, []string{"medien"})
	res := testutil.Post(t, srv, fmt.Sprintf("/api/match-reports/%d/publish", reportID), token, nil)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

// ─── TC-MR10 · Create liefert Default-Titel; GET zeigt ihn ────────────────────

func TestCreate_SetsDefaultTitle(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db) // date="2026-05-15", opponent="Test Opponent"
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)

	res := testutil.Post(t, srv, "/api/match-reports", token,
		map[string]int{"game_id": gameID, "duty_slot_id": slotID})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d — %s", res.StatusCode, readBody(t, res))
	}
	var created struct{ ID int }
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	// GET liefert Default-Titel „15.05.2026 — Test Opponent".
	getRes := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/match-reports/%d", created.ID), token, nil)
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getRes.StatusCode)
	}
	var got struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(getRes.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	want := "15.05.2026 — Test Opponent"
	if got.Title != want {
		t.Errorf("title = %q, want %q", got.Title, want)
	}
}

// ─── TC-MR11 · Update mit Custom-Titel → GET zeigt neuen Titel ────────────────

func TestUpdate_CustomTitle(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)

	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/match-reports/%d", reportID), token,
		map[string]any{"title": "Neuer Titel"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("put: expected 200, got %d — %s", res.StatusCode, readBody(t, res))
	}

	getRes := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/match-reports/%d", reportID), token, nil)
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", getRes.StatusCode)
	}
	var got struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(getRes.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Title != "Neuer Titel" {
		t.Errorf("title = %q, want %q", got.Title, "Neuer Titel")
	}
}

// ─── TC-MR12 · Update mit 201-Zeichen-Titel → 400 title_too_long ──────────────

func TestUpdate_TitleTooLong(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)

	longTitle := strings.Repeat("a", 201)
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/match-reports/%d", reportID), token,
		map[string]any{"title": longTitle})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d — %s", res.StatusCode, readBody(t, res))
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "title_too_long" {
		t.Errorf("error = %q, want title_too_long", body["error"])
	}
}

// ─── TC-MR13 · Image-URL ohne /api-Prefix ─────────────────────────────────────
//
// Der Client (axios) hat `baseURL='/api'` und würde `/api/api/…` bauen, wenn
// der Server den Prefix nochmal draufsetzt. GET liefert die URL also relativ
// zum axios-baseURL — ohne `/api`-Prefix.
func TestGet_ImageURLWithoutAPIPrefix(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, gameID := setupBasicGame(t, db)
	authorID := testutil.CreatePressTeamUser(t, db)
	slotID := createSlotWithAssignee(t, db, seasonID, teamID, gameID, authorID)
	reportID := testutil.CreateMatchReport(t, db, gameID, authorID, slotID)

	// Bild-Zeile direkt in DB — Datei-IO braucht der URL-Format-Test nicht.
	res, err := db.Exec(
		`INSERT INTO match_report_images (report_id, position, caption, storage_path)
		 VALUES (?, 1, 'test', '/tmp/nowhere.jpg')`, reportID)
	if err != nil {
		t.Fatal(err)
	}
	imgID, _ := res.LastInsertId()

	h := newHandlerWithPublisher(db, &fakePublisher{})
	srv := testServer(t, h)
	token := testutil.Token(t, authorID, auth.RolePressTeam, nil)

	getRes := testutil.Do(t, srv, http.MethodGet, fmt.Sprintf("/api/match-reports/%d", reportID), token, nil)
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("get: expected 200, got %d — %s", getRes.StatusCode, readBody(t, getRes))
	}
	var got struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.NewDecoder(getRes.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(got.Images))
	}
	want := fmt.Sprintf("/match-reports/%d/images/%d/blob", reportID, imgID)
	if got.Images[0].URL != want {
		t.Errorf("URL = %q, want %q", got.Images[0].URL, want)
	}
	// Explizit: KEIN /api-Prefix.
	if strings.HasPrefix(got.Images[0].URL, "/api/") {
		t.Errorf("URL should not start with /api/, got %q", got.Images[0].URL)
	}
}

func readBody(t *testing.T, res *http.Response) string {
	t.Helper()
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return string(b)
}
