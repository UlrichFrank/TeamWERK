package members_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// memberNumber returns the member_number column for a member, or "" if NULL.
func memberNumber(t *testing.T, database *sql.DB, memberID int) string {
	t.Helper()
	var n sql.NullString
	if err := database.QueryRow(`SELECT member_number FROM members WHERE id=?`, memberID).Scan(&n); err != nil {
		t.Fatalf("memberNumber: %v", err)
	}
	return n.String
}

func memberFirstName(t *testing.T, database *sql.DB, memberID int) string {
	t.Helper()
	var n string
	if err := database.QueryRow(`SELECT first_name FROM members WHERE id=?`, memberID).Scan(&n); err != nil {
		t.Fatalf("memberFirstName: %v", err)
	}
	return n
}

// ── 5.1: Create vergibt höchste numerische + 1 und ignoriert Client-Wert ───────

func TestCreateMember_AutoAssignsNextNumber_IgnoresClientValue(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})

	// Höchste vorhandene numerische Nummer ist 285.
	if _, err := database.Exec(
		`INSERT INTO members (first_name, last_name, status, member_number) VALUES (?,?,?,?)`,
		"Max", "Mustermann", "aktiv", "285"); err != nil {
		t.Fatalf("seed member: %v", err)
	}

	srv := newMembersServer(t, database)
	// Client schickt explizit "999" mit — muss ignoriert werden.
	res := testutil.Post(t, srv, "/api/members", tok,
		map[string]string{"first_name": "Neu", "last_name": "Mitglied", "member_number": "999"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var body struct {
		ID int `json:"id"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	got := memberNumber(t, database, body.ID)
	if got != "286" {
		t.Errorf("expected auto-assigned number 286 (max+1), got %q", got)
	}
}

func TestCreateMember_FirstNumberIsOne(t *testing.T) {
	database := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminID, "admin", nil)

	srv := newMembersServer(t, database)
	res := testutil.Post(t, srv, "/api/members", tok,
		map[string]string{"first_name": "Erstes", "last_name": "Mitglied"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var body struct {
		ID int `json:"id"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	if got := memberNumber(t, database, body.ID); got != "1" {
		t.Errorf("expected first number 1, got %q", got)
	}
}

// ── 5.2: PUT als Admin ändert Nummer (200) / Dublette → 409 ────────────────────

func updateBody(firstName, lastName, memberNumber string) map[string]any {
	return map[string]any{
		"first_name":    firstName,
		"last_name":     lastName,
		"member_number": memberNumber,
		"status":        "aktiv",
		"gender":        "u",
	}
}

func TestUpdateMember_Admin_ChangesNumber(t *testing.T) {
	database := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminID, "admin", nil)

	var mID int64
	res0, _ := database.Exec(
		`INSERT INTO members (first_name, last_name, status, member_number) VALUES (?,?,?,?)`,
		"Anna", "Admin", "aktiv", "1")
	mID, _ = res0.LastInsertId()

	srv := newMembersServer(t, database)
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", mID), tok,
		updateBody("Anna", "Admin", "500"))
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	res.Body.Close()

	if got := memberNumber(t, database, int(mID)); got != "500" {
		t.Errorf("admin should change number to 500, got %q", got)
	}
}

func TestUpdateMember_Admin_DuplicateNumber_409(t *testing.T) {
	database := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, database, "admin")
	tok := testutil.Token(t, adminID, "admin", nil)

	resA, _ := database.Exec(
		`INSERT INTO members (first_name, last_name, status, member_number) VALUES (?,?,?,?)`,
		"Anna", "A", "aktiv", "1")
	idA, _ := resA.LastInsertId()
	if _, err := database.Exec(
		`INSERT INTO members (first_name, last_name, status, member_number) VALUES (?,?,?,?)`,
		"Bert", "B", "aktiv", "2"); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	srv := newMembersServer(t, database)
	// A soll auf "2" gesetzt werden — bereits von B belegt.
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", idA), tok,
		updateBody("Anna", "A", "2"))
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate number, got %d", res.StatusCode)
	}
	res.Body.Close()

	// A behält die alte Nummer.
	if got := memberNumber(t, database, int(idA)); got != "1" {
		t.Errorf("A should keep number 1 after rejected duplicate, got %q", got)
	}
}

// ── 5.3: PUT als Nicht-Admin lässt Nummer unverändert ──────────────────────────

func TestUpdateMember_NonAdmin_CannotChangeNumber(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})

	resA, _ := database.Exec(
		`INSERT INTO members (first_name, last_name, status, member_number) VALUES (?,?,?,?)`,
		"Cora", "C", "aktiv", "1")
	idA, _ := resA.LastInsertId()

	srv := newMembersServer(t, database)
	// Vorstand (kein Admin) versucht Nummer zu ändern und ändert den Vornamen.
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", idA), tok,
		updateBody("Cora-Neu", "C", "777"))
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	res.Body.Close()

	if got := memberNumber(t, database, int(idA)); got != "1" {
		t.Errorf("non-admin must not change number, expected 1, got %q", got)
	}
	if got := memberFirstName(t, database, int(idA)); got != "Cora-Neu" {
		t.Errorf("non-admin should still save other fields, expected first_name Cora-Neu, got %q", got)
	}
}

// ── 5.4: List liefert Konflikt-Flag für alle drei Typen, nicht für honorar ─────

func TestList_MemberNumberConflictFlags(t *testing.T) {
	database := testutil.NewDB(t)
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})

	// Unique-Index entfernen, um Alt-Dubletten zu simulieren.
	if _, err := database.Exec(`DROP INDEX IF EXISTS idx_members_member_number`); err != nil {
		t.Fatalf("drop index: %v", err)
	}

	insert := func(first, status, num string) int {
		t.Helper()
		var numArg any
		if num != "" {
			numArg = num
		}
		res, err := database.Exec(
			`INSERT INTO members (first_name, last_name, status, member_number) VALUES (?,?,?,?)`,
			first, "Test", status, numArg)
		if err != nil {
			t.Fatalf("insert %s: %v", first, err)
		}
		id, _ := res.LastInsertId()
		return int(id)
	}

	dup1 := insert("Dup1", "aktiv", "5")
	dup2 := insert("Dup2", "aktiv", "5")
	nonNum := insert("NonNum", "aktiv", "M-100")
	missing := insert("Missing", "passiv", "")
	honorar := insert("Honorar", "honorar", "")
	ok := insert("Ok", "aktiv", "10")

	srv := newMembersServer(t, database)
	res := testutil.Get(t, srv, "/api/members?limit=100", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	lr := decodeList(t, res)

	flags := map[int]string{}
	for _, raw := range lr.Items {
		var it struct {
			ID                   int    `json:"id"`
			MemberNumberConflict string `json:"member_number_conflict"`
		}
		if err := json.Unmarshal(raw, &it); err != nil {
			t.Fatalf("unmarshal item: %v", err)
		}
		flags[it.ID] = it.MemberNumberConflict
	}

	checks := []struct {
		id   int
		want string
		name string
	}{
		{dup1, "duplicate", "Dup1"},
		{dup2, "duplicate", "Dup2"},
		{nonNum, "non_numeric", "NonNum"},
		{missing, "missing", "Missing"},
		{honorar, "", "Honorar"},
		{ok, "", "Ok"},
	}
	for _, c := range checks {
		if flags[c.id] != c.want {
			t.Errorf("%s: expected conflict %q, got %q", c.name, c.want, flags[c.id])
		}
	}
}
