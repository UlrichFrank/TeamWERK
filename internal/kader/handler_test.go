package kader_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/kader"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── TC-K01: AutoAssign filters by DHB age bracket ─────────────────────────────

// TestAutoAssign_BracketFilter verifies that AutoAssign for an A-Jugend
// (birth years 2007-2008 in season 2025/26) adds the in-bracket member
// (born 2007) and skips the out-of-bracket member (born 2005).
func TestAutoAssign_BracketFilter(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "A-Jugend männlich")

	// Insert A-Jugend/mixed kader directly so we control age_class.
	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "A-Jugend", "mixed", teamID, 1)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	kaderIDRaw, _ := res.LastInsertId()
	kaderID := int(kaderIDRaw)

	// Member in bracket (born 2007).
	inRes, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Anna", "In", "aktiv", "2007-05-15", "m")
	if err != nil {
		t.Fatalf("insert in-bracket member: %v", err)
	}
	inID, _ := inRes.LastInsertId()

	// Member out of bracket (born 2005).
	outRes, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Bruno", "Out", "aktiv", "2005-03-10", "m")
	if err != nil {
		t.Fatalf("insert out-of-bracket member: %v", err)
	}
	outID, _ := outRes.LastInsertId()

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/admin/kader/auto-assign", h.AutoAssign)
	})

	resp := testutil.Post(t, srv, "/api/admin/kader/auto-assign", token,
		map[string]any{"kader_ids": []int{kaderID}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Only in-bracket member should be in kader_members.
	var inCount, outCount int
	db.QueryRow(`SELECT COUNT(*) FROM kader_members WHERE kader_id=? AND member_id=?`, kaderID, inID).Scan(&inCount)
	db.QueryRow(`SELECT COUNT(*) FROM kader_members WHERE kader_id=? AND member_id=?`, kaderID, outID).Scan(&outCount)

	if inCount != 1 {
		t.Errorf("expected in-bracket member (born 2007) to be assigned; got count=%d", inCount)
	}
	if outCount != 0 {
		t.Errorf("expected out-of-bracket member (born 2005) to NOT be assigned; got count=%d", outCount)
	}
}

// ── TC-K02: AutoAssign excludes ausgetreten members ───────────────────────────

// TestAutoAssign_ExcludesAusgetreten verifies that members with
// status='ausgetreten' are never included even when their birth year matches.
func TestAutoAssign_ExcludesAusgetreten(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "A-Jugend")

	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "A-Jugend", "mixed", teamID, 1)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	kaderIDRaw, _ := res.LastInsertId()
	kaderID := int(kaderIDRaw)

	// Active member born 2007.
	aktRes, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Aktiv", "Spieler", "aktiv", "2007-04-01", "m")
	if err != nil {
		t.Fatalf("insert aktiv member: %v", err)
	}
	aktID, _ := aktRes.LastInsertId()

	// Ausgetreten member also born 2007.
	ausRes, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Aus", "Getreten", "ausgetreten", "2007-09-20", "m")
	if err != nil {
		t.Fatalf("insert ausgetreten member: %v", err)
	}
	ausID, _ := ausRes.LastInsertId()

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/admin/kader/auto-assign", h.AutoAssign)
	})

	resp := testutil.Post(t, srv, "/api/admin/kader/auto-assign", token,
		map[string]any{"kader_ids": []int{kaderID}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var aktCount, ausCount int
	db.QueryRow(`SELECT COUNT(*) FROM kader_members WHERE kader_id=? AND member_id=?`, kaderID, aktID).Scan(&aktCount)
	db.QueryRow(`SELECT COUNT(*) FROM kader_members WHERE kader_id=? AND member_id=?`, kaderID, ausID).Scan(&ausCount)

	if aktCount != 1 {
		t.Errorf("expected aktiv member to be assigned; got count=%d", aktCount)
	}
	if ausCount != 0 {
		t.Errorf("expected ausgetreten member NOT to be assigned; got count=%d", ausCount)
	}
}

// ── TC-K03: AutoAssign respects dedicated_birth_year ─────────────────────────

// TestAutoAssign_DedicatedBirthYear verifies that when kader.dedicated_birth_year
// is set, only members of that exact birth year are assigned.
func TestAutoAssign_DedicatedBirthYear(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "A-Jugend dedicated")

	// Kader with dedicated_birth_year=2008.
	dedicatedYear := 2008
	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number, dedicated_birth_year) VALUES (?, ?, ?, ?, ?, ?)`,
		seasonID, "A-Jugend", "mixed", teamID, 1, dedicatedYear)
	if err != nil {
		t.Fatalf("insert kader with dedicated_birth_year: %v", err)
	}
	kaderIDRaw, _ := res.LastInsertId()
	kaderID := int(kaderIDRaw)

	insert := func(name, dob string) int64 {
		t.Helper()
		r, e := db.Exec(
			`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
			name, "Test", "aktiv", dob, "m")
		if e != nil {
			t.Fatalf("insert member %s: %v", name, e)
		}
		id, _ := r.LastInsertId()
		return id
	}

	id2007 := insert("Born2007", "2007-01-01")
	id2008 := insert("Born2008", "2008-06-15")
	id2009 := insert("Born2009", "2009-11-30")

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/admin/kader/auto-assign", h.AutoAssign)
	})

	resp := testutil.Post(t, srv, "/api/admin/kader/auto-assign", token,
		map[string]any{"kader_ids": []int{kaderID}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	count := func(memberID int64) int {
		t.Helper()
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM kader_members WHERE kader_id=? AND member_id=?`, kaderID, memberID).Scan(&n)
		return n
	}

	if count(id2007) != 0 {
		t.Errorf("born-2007 should NOT be assigned (dedicated=2008)")
	}
	if count(id2008) != 1 {
		t.Errorf("born-2008 SHOULD be assigned (dedicated=2008)")
	}
	if count(id2009) != 0 {
		t.Errorf("born-2009 should NOT be assigned (dedicated=2008)")
	}
}

// ── TC-K04: MemberSuggestions respects age bracket ───────────────────────────

// TestMemberSuggestions_BracketActive verifies that with default
// filter_age_bracket=true only members matching the kader's age bracket appear.
func TestMemberSuggestions_BracketActive(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "A-Jugend suggestions")

	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "A-Jugend", "mixed", teamID, 1)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	kaderIDRaw, _ := res.LastInsertId()
	kaderID := int(kaderIDRaw)

	// In-bracket: born 2007.
	inRes, _ := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Ingrid", "InBracket", "aktiv", "2007-07-07", "m")
	inID, _ := inRes.LastInsertId()

	// Out-of-bracket: born 2005.
	outRes, _ := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Otto", "OutBracket", "aktiv", "2005-02-20", "m")
	outID, _ := outRes.LastInsertId()

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/admin/kader/{id}/member-suggestions", h.MemberSuggestions)
	})

	path := fmt.Sprintf("/api/admin/kader/%d/member-suggestions", kaderID)
	resp := testutil.Get(t, srv, path, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var payload struct {
		Suggestions []struct {
			ID int `json:"id"`
		} `json:"suggestions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	inFound, outFound := false, false
	for _, s := range payload.Suggestions {
		if int64(s.ID) == inID {
			inFound = true
		}
		if int64(s.ID) == outID {
			outFound = true
		}
	}

	if !inFound {
		t.Errorf("expected in-bracket member (born 2007, id=%d) in suggestions", inID)
	}
	if outFound {
		t.Errorf("expected out-of-bracket member (born 2005, id=%d) NOT in suggestions", outID)
	}
}

// ── TC-K05: MemberSuggestions with bracket disabled returns all ───────────────

// TestMemberSuggestions_BracketDisabled verifies that ?filter_age_bracket=false
// returns all active members regardless of birth year.
func TestMemberSuggestions_BracketDisabled(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "A-Jugend no-filter")

	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "A-Jugend", "mixed", teamID, 1)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	kaderIDRaw, _ := res.LastInsertId()
	kaderID := int(kaderIDRaw)

	// In-bracket: born 2007.
	inRes, _ := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Ingrid", "InBracket2", "aktiv", "2007-07-07", "m")
	inID, _ := inRes.LastInsertId()

	// Out-of-bracket: born 2005.
	outRes, _ := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Otto", "OutBracket2", "aktiv", "2005-02-20", "m")
	outID, _ := outRes.LastInsertId()

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/admin/kader/{id}/member-suggestions", h.MemberSuggestions)
	})

	path := fmt.Sprintf("/api/admin/kader/%d/member-suggestions?filter_age_bracket=false", kaderID)
	resp := testutil.Get(t, srv, path, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var payload struct {
		Suggestions []struct {
			ID int `json:"id"`
		} `json:"suggestions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	inFound, outFound := false, false
	for _, s := range payload.Suggestions {
		if int64(s.ID) == inID {
			inFound = true
		}
		if int64(s.ID) == outID {
			outFound = true
		}
	}

	if !inFound {
		t.Errorf("expected in-bracket member (id=%d) in unfiltered suggestions", inID)
	}
	if !outFound {
		t.Errorf("expected out-of-bracket member (id=%d) in unfiltered suggestions (filter disabled)", outID)
	}
}
