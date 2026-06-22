package members_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// ── local helpers ─────────────────────────────────────────────────────────────

func newMembersServer(t *testing.T, database *sql.DB) *httptest.Server {
	t.Helper()
	return prodserver.New(t, database)
}

func newStatusServer(t *testing.T, database *sql.DB) *httptest.Server {
	t.Helper()
	return prodserver.New(t, database)
}

// addKaderMember inserts a member into a kader directly (player_memberships is a view).
func addKaderMember(t *testing.T, database *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT OR IGNORE INTO kader_members (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("addKaderMember: %v", err)
	}
}

// countFamilyLinks returns how many family_links rows exist for the given member.
func countFamilyLinks(t *testing.T, database *sql.DB, memberID int) int {
	t.Helper()
	var n int
	if err := database.QueryRow(`SELECT COUNT(*) FROM family_links WHERE member_id=?`, memberID).Scan(&n); err != nil {
		t.Fatalf("countFamilyLinks: %v", err)
	}
	return n
}

// memberUserID returns the user_id column for a member, or 0 if NULL.
func memberUserID(t *testing.T, database *sql.DB, memberID int) int {
	t.Helper()
	var uid sql.NullInt64
	if err := database.QueryRow(`SELECT user_id FROM members WHERE id=?`, memberID).Scan(&uid); err != nil {
		t.Fatalf("memberUserID: %v", err)
	}
	if uid.Valid {
		return int(uid.Int64)
	}
	return 0
}

// userCanLogin returns the can_login value for a user.
func userCanLogin(t *testing.T, database *sql.DB, userID int) int {
	t.Helper()
	var v int
	if err := database.QueryRow(`SELECT can_login FROM users WHERE id=?`, userID).Scan(&v); err != nil {
		t.Fatalf("userCanLogin: %v", err)
	}
	return v
}

// listResponse matches the { items: [...], total: N } shape returned by List.
type listResponse struct {
	Items []json.RawMessage `json:"items"`
	Total int               `json:"total"`
}

func decodeList(t *testing.T, res *http.Response) listResponse {
	t.Helper()
	defer res.Body.Close()
	var lr listResponse
	if err := json.NewDecoder(res.Body).Decode(&lr); err != nil {
		t.Fatalf("decodeList: %v", err)
	}
	return lr
}

// ── TC-M01: pagination ────────────────────────────────────────────────────────

func TestList_Pagination(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandUserID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})

	for i := 0; i < 25; i++ {
		testutil.CreateMember(t, database, 0)
	}

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members?limit=10&offset=10", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)
	if lr.Total != 25 {
		t.Errorf("expected total=25, got %d", lr.Total)
	}
	if len(lr.Items) != 10 {
		t.Errorf("expected 10 items, got %d", len(lr.Items))
	}
}

// ── TC-M02: search by name ────────────────────────────────────────────────────

func TestList_SearchByName(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	// Insert Anna Müller
	database.Exec(`INSERT INTO members (first_name, last_name, status) VALUES (?, ?, ?)`,
		"Anna", "Müller", "aktiv")
	// Insert Karl Schmidt
	database.Exec(`INSERT INTO members (first_name, last_name, status) VALUES (?, ?, ?)`,
		"Karl", "Schmidt", "aktiv")

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members?search=Müller", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)
	if lr.Total != 1 {
		t.Errorf("expected total=1 (only Müller), got %d", lr.Total)
	}
	if len(lr.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(lr.Items))
	}
}

// ── TC-M03: ausgetreten members are hidden ────────────────────────────────────

func TestList_AusgetretenHidden(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandUserID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandUserID, "standard", []string{"vorstand"})

	m1 := testutil.CreateMember(t, database, 0)
	m2 := testutil.CreateMember(t, database, 0)
	m3 := testutil.CreateMember(t, database, 0)

	// Set m3 to ausgetreten
	database.Exec(`UPDATE members SET status='ausgetreten' WHERE id=?`, m3)
	_ = m1
	_ = m2

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)
	if lr.Total != 2 {
		t.Errorf("expected total=2 (ausgetreten excluded), got %d", lr.Total)
	}
	if len(lr.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(lr.Items))
	}
}

// ── TC-M04: trainer sees only members of their team ───────────────────────────

// GET /api/members ist für Trainer freigeschaltet (Kadersuche). Ohne
// ?club_function-Filter sieht ein Trainer nur Mitglieder der eigenen Kader
// (policy.ScopeMembersQuery).
func TestList_TrainerScope(t *testing.T) {
	database := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, database, "2025/26")

	// Create teams
	teamA := testutil.CreateTeam(t, database, "Team A")
	teamB := testutil.CreateTeam(t, database, "Team B")

	// Create trainer user and their linked member
	trainerUserID := testutil.CreateUser(t, database, "standard")
	trainerMemberID := testutil.CreateMember(t, database, trainerUserID)

	// Create kader for team A and link trainer
	kaderA := testutil.CreateKader(t, database, teamA, seasonID)
	testutil.AddKaderTrainer(t, database, kaderA, trainerMemberID)

	// Create 3 members in team A via kader_members (player_memberships is a view)
	for range 3 {
		mID := testutil.CreateMember(t, database, 0)
		addKaderMember(t, database, kaderA, mID)
	}

	// Create kader for team B (trainer not linked here)
	kaderB := testutil.CreateKader(t, database, teamB, seasonID)

	// Create 2 members in team B
	for range 2 {
		mID := testutil.CreateMember(t, database, 0)
		addKaderMember(t, database, kaderB, mID)
	}

	// Trainer token: DB role "standard", JWT club_functions "trainer"
	tok := testutil.Token(t, trainerUserID, "standard", []string{"trainer"})

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)
	if lr.Total != 3 {
		t.Errorf("expected total=3 (team A only), got %d", lr.Total)
	}
	if len(lr.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(lr.Items))
	}
}

// Ein Kassierer ist (anders als ein Trainer) keiner Kader-Mannschaft zugeordnet,
// verwaltet aber Bankdaten und Beitragslauf vereinsweit. policy.ScopeMembersQuery
// muss ihm daher die ungescopte (1=1) Mitgliederliste liefern — sonst bliebe die
// Liste für einen reinen Kassierer leer (Regression: /mitglieder zeigte nichts).
func TestList_KassiererSeesAll(t *testing.T) {
	database := testutil.NewDB(t)

	// Kassierer-Nutzer ohne Kader-/Trainer-Zuordnung.
	kassiererUserID := testutil.CreateUser(t, database, "standard")

	// 4 vereinsweite Mitglieder, in keinem Kader.
	for range 4 {
		testutil.CreateMember(t, database, 0)
	}

	tok := testutil.Token(t, kassiererUserID, "standard", []string{"kassierer"})

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)
	if lr.Total != 4 {
		t.Errorf("kassierer should see all 4 members club-wide, got total=%d", lr.Total)
	}
	if len(lr.Items) != 4 {
		t.Errorf("kassierer should see all 4 members club-wide, got %d items", len(lr.Items))
	}
}

// TC-MCAN-04: Feld-Redaction für kader-gescopte Trainer. Ein Trainer sieht in der
// Mitgliederliste nur Name, Jahrgang (birth_year, nicht das exakte Datum), Passnummer
// und Vereinsfunktionen. Mitgliedsnummer, exaktes Geburtsdatum und user_id werden
// entfernt. Vorstand sieht weiterhin alle Felder.
func TestList_TrainerFieldRedaction(t *testing.T) {
	database := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, database, "2025/26")
	teamA := testutil.CreateTeam(t, database, "Team A")

	trainerUserID := testutil.CreateUser(t, database, "standard")
	trainerMemberID := testutil.CreateMember(t, database, trainerUserID)
	kaderA := testutil.CreateKader(t, database, teamA, seasonID)
	testutil.AddKaderTrainer(t, database, kaderA, trainerMemberID)

	// Ein Spieler im Kader mit sensiblen Feldern.
	playerUserID := testutil.CreateUser(t, database, "standard")
	playerID := testutil.CreateMember(t, database, playerUserID)
	addKaderMember(t, database, kaderA, playerID)
	addClubFunction(t, database, playerID, "spieler")
	if _, err := database.Exec(
		`UPDATE members SET date_of_birth='2008-04-15', member_number='M-100', pass_number='P-200' WHERE id=?`,
		playerID); err != nil {
		t.Fatalf("update player: %v", err)
	}

	type item struct {
		FirstName     string   `json:"first_name"`
		LastName      string   `json:"last_name"`
		DateOfBirth   string   `json:"date_of_birth"`
		BirthYear     *int     `json:"birth_year"`
		MemberNumber  string   `json:"member_number"`
		PassNumber    string   `json:"pass_number"`
		UserID        *int     `json:"user_id"`
		ClubFunctions []string `json:"club_functions"`
	}
	srv := newMembersServer(t, database)

	// Trainer: redigierte Felder.
	resT := testutil.Get(t, srv, "/api/members",
		testutil.Token(t, trainerUserID, "standard", []string{"trainer"}))
	if resT.StatusCode != http.StatusOK {
		t.Fatalf("trainer: expected 200, got %d", resT.StatusCode)
	}
	var bodyT struct {
		Items []item `json:"items"`
	}
	if err := json.NewDecoder(resT.Body).Decode(&bodyT); err != nil {
		resT.Body.Close()
		t.Fatalf("decode trainer: %v", err)
	}
	resT.Body.Close()
	if len(bodyT.Items) != 1 {
		t.Fatalf("trainer: expected 1 item, got %d", len(bodyT.Items))
	}
	it := bodyT.Items[0]
	if it.FirstName == "" || it.LastName == "" {
		t.Error("trainer: name must be present")
	}
	if it.BirthYear == nil || *it.BirthYear != 2008 {
		t.Errorf("trainer: expected birth_year=2008, got %v", it.BirthYear)
	}
	if it.DateOfBirth != "" {
		t.Errorf("trainer: date_of_birth must be redacted, got %q", it.DateOfBirth)
	}
	if it.MemberNumber != "" {
		t.Errorf("trainer: member_number must be redacted, got %q", it.MemberNumber)
	}
	if it.UserID != nil {
		t.Errorf("trainer: user_id must be redacted, got %v", *it.UserID)
	}
	if it.PassNumber != "P-200" {
		t.Errorf("trainer: pass_number must be kept, got %q", it.PassNumber)
	}
	if len(it.ClubFunctions) == 0 {
		t.Error("trainer: club_functions must be kept")
	}

	// Vorstand: volle Felder.
	resV := testutil.Get(t, srv, "/api/members",
		testutil.Token(t, trainerUserID, "standard", []string{"vorstand"}))
	if resV.StatusCode != http.StatusOK {
		t.Fatalf("vorstand: expected 200, got %d", resV.StatusCode)
	}
	var bodyV struct {
		Items []item `json:"items"`
	}
	if err := json.NewDecoder(resV.Body).Decode(&bodyV); err != nil {
		resV.Body.Close()
		t.Fatalf("decode vorstand: %v", err)
	}
	resV.Body.Close()
	var player *item
	for i := range bodyV.Items {
		if bodyV.Items[i].MemberNumber == "M-100" {
			player = &bodyV.Items[i]
		}
	}
	if player == nil {
		t.Fatal("vorstand: player with member_number M-100 not found (fields must not be redacted)")
	}
	if player.DateOfBirth == "" {
		t.Error("vorstand: date_of_birth must be present (full)")
	}
	if player.BirthYear != nil {
		t.Error("vorstand: birth_year must not be set (uses full date_of_birth)")
	}
}

// addClubFunction assigns a Vereinsfunktion to a member directly.
func addClubFunction(t *testing.T, database *sql.DB, memberID int, fn string) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT OR IGNORE INTO member_club_functions (member_id, function) VALUES (?, ?)`,
		memberID, fn); err != nil {
		t.Fatalf("addClubFunction: %v", err)
	}
}

// TC-M04b: Ein Trainer, der gezielt nach Trainern sucht (?club_function=trainer),
// bekommt vereinsweit alle Trainer — auch ohne eigene Kader-Verknüpfung. Das ist
// der /kader-Anwendungsfall (Trainer einem Kader zuordnen). Spieler tauchen nicht
// auf. Sportliche Leitung muss denselben Zugriff haben.
func TestList_TrainerSearchTrainers_Wide(t *testing.T) {
	database := testutil.NewDB(t)

	// Two trainers anywhere in the club + one plain player.
	t1 := testutil.CreateMember(t, database, 0)
	addClubFunction(t, database, t1, "trainer")
	t2 := testutil.CreateMember(t, database, 0)
	addClubFunction(t, database, t2, "trainer")
	testutil.CreateMember(t, database, 0) // Spieler ohne Funktion

	// Caller is a trainer with no kader of their own.
	callerUserID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, callerUserID, "standard", []string{"trainer"})

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members?club_function=trainer&limit=20", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)
	if lr.Total != 2 {
		t.Errorf("expected total=2 (club-wide trainers), got %d", lr.Total)
	}
}

// ── TC-M05: create family link ────────────────────────────────────────────────

func TestFamilyLink_Create(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	parentUserID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, 0)

	srv := newMembersServer(t, database)
	res := testutil.Post(t, srv, "/api/family-links", tok,
		map[string]int{"parent_user_id": parentUserID, "member_id": memberID})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if n := countFamilyLinks(t, database, memberID); n != 1 {
		t.Errorf("expected 1 family_link in DB, got %d", n)
	}
}

// ── TC-M06: max 2 parents per member ─────────────────────────────────────────

func TestFamilyLink_MaxTwo(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	memberID := testutil.CreateMember(t, database, 0)

	// Insert 2 existing parents directly
	parent1 := testutil.CreateUser(t, database, "standard")
	parent2 := testutil.CreateUser(t, database, "standard")
	database.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parent1, memberID)
	database.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parent2, memberID)

	// Attempt to add a third parent
	parent3 := testutil.CreateUser(t, database, "standard")
	srv := newMembersServer(t, database)
	res := testutil.Post(t, srv, "/api/family-links", tok,
		map[string]int{"parent_user_id": parent3, "member_id": memberID})
	defer res.Body.Close()

	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

// ── TC-M07: duplicate family link is idempotent ───────────────────────────────

func TestFamilyLink_DuplicateIdempotent(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	parentUserID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, 0)

	srv := newMembersServer(t, database)
	body := map[string]int{"parent_user_id": parentUserID, "member_id": memberID}

	// First POST
	res1 := testutil.Post(t, srv, "/api/family-links", tok, body)
	res1.Body.Close()
	if res1.StatusCode != http.StatusNoContent {
		t.Fatalf("first POST: expected 204, got %d", res1.StatusCode)
	}

	// Second POST — same link
	res2 := testutil.Post(t, srv, "/api/family-links", tok, body)
	res2.Body.Close()
	if res2.StatusCode != http.StatusNoContent {
		t.Fatalf("second POST: expected 204, got %d", res2.StatusCode)
	}

	// Only 1 row should exist in DB
	if n := countFamilyLinks(t, database, memberID); n != 1 {
		t.Errorf("expected 1 family_link after duplicate insert, got %d", n)
	}
}

// ── TC-M08: delete non-existent family link returns 404 ──────────────────────

func TestFamilyLink_DeleteNotFound(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	srv := newMembersServer(t, database)
	// Use IDs that don't exist in family_links
	res := testutil.Do(t, srv, http.MethodDelete, "/api/family-links", tok,
		map[string]int{"parent_user_id": 9999, "member_id": 9999})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}

// ── TC-M09: create proxy account ─────────────────────────────────────────────

func TestProxyAccount_Create(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	// Member without a user link
	memberID := testutil.CreateMember(t, database, 0)

	srv := newMembersServer(t, database)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/members/%d/proxy-account", memberID), tok, map[string]any{})
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	// Decode response to get new user_id
	var body map[string]int
	json.NewDecoder(res.Body).Decode(&body)
	newUserID, ok := body["user_id"]
	if !ok || newUserID == 0 {
		t.Fatal("response missing user_id")
	}

	// can_login must be 0
	if cl := userCanLogin(t, database, newUserID); cl != 0 {
		t.Errorf("expected can_login=0, got %d", cl)
	}

	// members.user_id must be updated
	if uid := memberUserID(t, database, memberID); uid != newUserID {
		t.Errorf("expected members.user_id=%d, got %d", newUserID, uid)
	}
}

// ── TC-M10: proxy account creation fails if member already has a user ─────────

func TestProxyAccount_AlreadyHasAccount(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	// Member WITH an existing user link
	existingUserID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, existingUserID)

	srv := newMembersServer(t, database)
	res := testutil.Post(t, srv, fmt.Sprintf("/api/members/%d/proxy-account", memberID), tok, map[string]any{})
	defer res.Body.Close()

	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}
}

// ── GetProfile / UpdateProfile ────────────────────────────────────────────────

// TC: GET /api/profile/me liefert eigene Daten zurück (HTTP 200).
func TestGetProfile_ReturnsOwnData(t *testing.T) {
	database := testutil.NewDB(t)
	userID := testutil.CreateUser(t, database, "standard")
	database.Exec(`UPDATE users SET first_name='Klara', last_name='Mustermann' WHERE id=?`, userID)
	srv := newMembersServer(t, database)

	res := testutil.Get(t, srv, "/api/profile/me", testutil.Token(t, userID, "standard", nil))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	if body == nil {
		t.Error("expected non-nil profile response")
	}
}

// TC: PUT /api/profile/me ändert first_name in DB.
func TestUpdateProfile_PersistsChange(t *testing.T) {
	database := testutil.NewDB(t)
	userID := testutil.CreateUser(t, database, "standard")
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, "/api/profile/me",
		testutil.Token(t, userID, "standard", nil),
		map[string]string{"first_name": "Neuer", "last_name": "Name"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var firstName string
	database.QueryRow(`SELECT first_name FROM users WHERE id=?`, userID).Scan(&firstName)
	if firstName != "Neuer" {
		t.Errorf("expected first_name='Neuer', got %q", firstName)
	}
}

// TC-SEC-M01: GetChildProfile returns correct parent name (u.name bug fix).
func TestGetChildProfile_ReturnsParentName(t *testing.T) {
	database := testutil.NewDB(t)
	// Create a parent user (DB role is "standard"; JWT role is "elternteil" via token)
	parentID := testutil.CreateUser(t, database, "standard")
	// Create a member linked to the parent (userID=0 means no linked user)
	memberID := testutil.CreateMember(t, database, 0)
	// Insert a family link
	if _, err := database.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentID, memberID); err != nil {
		t.Fatalf("insert family_link: %v", err)
	}
	// Update parent's name for the assertion
	database.Exec(`UPDATE users SET first_name='Anna', last_name='Müller' WHERE id=?`, parentID)

	srv := newMembersServer(t, database)
	token := testutil.Token(t, parentID, "elternteil", nil)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/profile/kind/%d", memberID), token)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Parents []struct {
			Name string `json:"name"`
		} `json:"parents"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Parents) == 0 {
		t.Fatal("parents array is empty — u.name bug may be present")
	}
	if body.Parents[0].Name != "Anna Müller" {
		t.Errorf("expected 'Anna Müller', got %q", body.Parents[0].Name)
	}
}

// ── TC-M-F01: ?unlinked_user=1 — nur Mitglieder ohne user_id und ohne family_links ──

func TestList_UnlinkedUserFilter(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	// Mitglied mit direkt verknüpftem User
	linkedUserID := testutil.CreateUser(t, database, "standard")
	testutil.CreateMember(t, database, linkedUserID)

	// Mitglied ohne direkten User, aber mit family_link-Elternteil
	parentUserID := testutil.CreateUser(t, database, "standard")
	parentMemberID := testutil.CreateMember(t, database, 0)
	database.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, parentUserID, parentMemberID)

	// Mitglied ohne jede Verknüpfung
	unlinkedMemberID := testutil.CreateMember(t, database, 0)

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members?unlinked_user=1", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)
	if lr.Total != 1 {
		t.Errorf("expected total=1 (only unlinked member), got %d", lr.Total)
	}
	if len(lr.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(lr.Items))
	}
	var item struct {
		ID int `json:"id"`
	}
	json.Unmarshal(lr.Items[0], &item)
	if item.ID != unlinkedMemberID {
		t.Errorf("expected unlinked member %d, got %d", unlinkedMemberID, item.ID)
	}
}

// ── TC-M-F02: ?has_draft=1 — nur Mitglieder mit offenem Änderungsantrag ─────────

func TestList_HasDraftFilter(t *testing.T) {
	database := testutil.NewDB(t)
	adminUserID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminUserID, "admin", nil)

	memberWithDraft := testutil.CreateMember(t, database, 0)
	memberWithoutDraft := testutil.CreateMember(t, database, 0)
	database.Exec(`INSERT INTO member_change_drafts (member_id, field_name, old_value, new_value) VALUES (?,?,?,?)`,
		memberWithDraft, "profil", "{}", "{\"first_name\":\"Neu\"}")
	_ = memberWithoutDraft

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members?has_draft=1", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)
	if lr.Total != 1 {
		t.Errorf("expected total=1 (only member with draft), got %d", lr.Total)
	}
	if len(lr.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(lr.Items))
	}
	var item struct {
		ID int `json:"id"`
	}
	json.Unmarshal(lr.Items[0], &item)
	if item.ID != memberWithDraft {
		t.Errorf("expected member %d with draft, got %d", memberWithDraft, item.ID)
	}
}

// ── TC-M: Anwärter-Status ─────────────────────────────────────────────────────

func TestMemberStatus_Anwaerter_Update(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})
	memberID := testutil.CreateMember(t, database, 0)

	srv := newStatusServer(t, database)
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/members/%d/status", memberID), tok,
		map[string]string{"status": "anwaerter"})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var got string
	database.QueryRow(`SELECT status FROM members WHERE id=?`, memberID).Scan(&got)
	if got != "anwaerter" {
		t.Errorf("expected status=anwaerter, got %q", got)
	}
}

func TestMemberStatus_Invalid(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})
	memberID := testutil.CreateMember(t, database, 0)

	srv := newStatusServer(t, database)
	res := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/members/%d/status", memberID), tok,
		map[string]string{"status": "unbekannt"})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestMemberStatus_Anwaerter_Create(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})

	srv := newStatusServer(t, database)
	res := testutil.Post(t, srv, "/api/members", tok,
		map[string]string{"first_name": "Tom", "last_name": "Probe"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var body struct {
		ID int `json:"id"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	// promote to anwaerter
	res2 := testutil.Do(t, srv, http.MethodPut,
		fmt.Sprintf("/api/members/%d/status", body.ID), tok,
		map[string]string{"status": "anwaerter"})
	if res2.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 on status update, got %d", res2.StatusCode)
	}
}

// GET /api/members/{id} liefert beitragsfrei_grund in der Response (Vorstand sieht Bankfelder).
func TestGetMember_BeitragsfreiGrundField(t *testing.T) {
	database := testutil.NewDB(t)
	memberID := testutil.CreateMember(t, database, 0)
	if _, err := database.Exec(
		`UPDATE members SET beitragsfrei=1, beitragsfrei_grund='kein aktiver Sportler mehr' WHERE id=?`,
		memberID); err != nil {
		t.Fatalf("seed: %v", err)
	}
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})
	srv := newMembersServer(t, database)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d", memberID), tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Beitragsfrei      bool    `json:"beitragsfrei"`
		BeitragsfreiGrund *string `json:"beitragsfrei_grund"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	if !body.Beitragsfrei {
		t.Fatalf("beitragsfrei: got false, want true")
	}
	if body.BeitragsfreiGrund == nil || *body.BeitragsfreiGrund != "kein aktiver Sportler mehr" {
		got := "<nil>"
		if body.BeitragsfreiGrund != nil {
			got = *body.BeitragsfreiGrund
		}
		t.Errorf("beitragsfrei_grund: got %q, want %q", got, "kein aktiver Sportler mehr")
	}
}

// PUT /api/members/{id} mit beitragsfrei=false setzt beitragsfrei_grund auf NULL
// (Kopplungs-Invariante, siehe Spec).
func TestUpdateMember_BeitragsfreiFalseClearsGrund(t *testing.T) {
	database := testutil.NewDB(t)
	memberID := testutil.CreateMember(t, database, 0)
	if _, err := database.Exec(
		`UPDATE members SET first_name='Petra', last_name='Test', beitragsfrei=1, beitragsfrei_grund='Zweitspielrecht' WHERE id=?`,
		memberID); err != nil {
		t.Fatalf("seed: %v", err)
	}
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})
	srv := newMembersServer(t, database)

	// Trotz mitgesendetem Grund-Wert MUSS beitragsfrei=false den Grund leeren.
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", memberID), tok,
		map[string]any{
			"first_name":         "Petra",
			"last_name":          "Test",
			"status":             "aktiv",
			"beitragsfrei":       false,
			"beitragsfrei_grund": "wird ignoriert",
		})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT /members/{id}: expected 204, got %d", res.StatusCode)
	}

	var beitragsfrei int
	var grund sql.NullString
	if err := database.QueryRow(
		`SELECT beitragsfrei, beitragsfrei_grund FROM members WHERE id=?`, memberID).
		Scan(&beitragsfrei, &grund); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if beitragsfrei != 0 {
		t.Errorf("beitragsfrei: got %d, want 0", beitragsfrei)
	}
	if grund.Valid {
		t.Errorf("beitragsfrei_grund: got %q, want NULL", grund.String)
	}
}
