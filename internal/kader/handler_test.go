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
		r.Post("/api/kader/auto-assign", h.AutoAssign)
	})

	resp := testutil.Post(t, srv, "/api/kader/auto-assign", token,
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
		r.Post("/api/kader/auto-assign", h.AutoAssign)
	})

	resp := testutil.Post(t, srv, "/api/kader/auto-assign", token,
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
		r.Post("/api/kader/auto-assign", h.AutoAssign)
	})

	resp := testutil.Post(t, srv, "/api/kader/auto-assign", token,
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
		r.Get("/api/kader/{id}/member-suggestions", h.MemberSuggestions)
	})

	path := fmt.Sprintf("/api/kader/%d/member-suggestions", kaderID)
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
		r.Get("/api/kader/{id}/member-suggestions", h.MemberSuggestions)
	})

	path := fmt.Sprintf("/api/kader/%d/member-suggestions?filter_age_bracket=false", kaderID)
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

// ── Trainingsgruppen (Förderkader/Perspektivkader) ───────────────────────────

// TestMemberSuggestions_TrainingGroupNoDedicatedYear ist der Regressionstest für
// den ursprünglichen Bug: eine Trainingsgruppe (Förderkader/Perspektivkader) hat
// keinen Spiel-Bracket. Ohne dedicated_birth_year lieferte der Default-Filter
// (filter_age_bracket=true) früher `BETWEEN 0 AND 0` → NULL Treffer, sodass man
// den Filter manuell abschalten musste. Jetzt wird der Jahresfilter mangels
// Bracket übersprungen → Kandidaten erscheinen.
func TestMemberSuggestions_TrainingGroupNoDedicatedYear(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Förderkader no-year")

	// Förderkader kader, NO dedicated_birth_year (the reported situation).
	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "Förderkader", "mixed", teamID, 1)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	kaderIDRaw, _ := res.LastInsertId()
	kaderID := int(kaderIDRaw)

	// A D+2 Förderkind (born 2016) and an unrelated older member (born 2005).
	youngRes, _ := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Fritz", "Förderkind", "foerderkind", "2016-04-04", "m")
	youngID, _ := youngRes.LastInsertId()
	oldRes, _ := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Otto", "Alt", "aktiv", "2005-02-20", "m")
	oldID, _ := oldRes.LastInsertId()

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kader/{id}/member-suggestions", h.MemberSuggestions)
	})

	// Default request (filter_age_bracket defaults to true).
	resp := testutil.Get(t, srv, fmt.Sprintf("/api/kader/%d/member-suggestions", kaderID), token)
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
	if len(payload.Suggestions) == 0 {
		t.Fatal("training-group suggestions without dedicated year must not be empty (regression: BETWEEN 0 AND 0)")
	}
	youngFound, oldFound := false, false
	for _, s := range payload.Suggestions {
		if int64(s.ID) == youngID {
			youngFound = true
		}
		if int64(s.ID) == oldID {
			oldFound = true
		}
	}
	if !youngFound {
		t.Errorf("expected Förderkind (born 2016, id=%d) in suggestions", youngID)
	}
	// No bracket → no year filter applied; all active members are candidates.
	if !oldFound {
		t.Errorf("expected unfiltered member (id=%d) in suggestions when no bracket applies", oldID)
	}
}

// TestMemberSuggestions_TrainingGroupDedicatedYear verifies that once a training
// group carries a dedicated_birth_year, the default filter narrows to exactly
// that Jahrgang.
func TestMemberSuggestions_TrainingGroupDedicatedYear(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Perspektivkader dedicated")

	// Perspektivkader = D+1 = 2015.
	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number, dedicated_birth_year) VALUES (?, ?, ?, ?, ?, ?)`,
		seasonID, "Perspektivkader", "mixed", teamID, 1, 2015)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	kaderIDRaw, _ := res.LastInsertId()
	kaderID := int(kaderIDRaw)

	inRes, _ := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Paul", "Passt", "foerderkind", "2015-06-06", "m")
	inID, _ := inRes.LastInsertId()
	outRes, _ := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth, gender) VALUES (?, ?, ?, ?, ?)`,
		"Nina", "Nachbarjahr", "foerderkind", "2016-06-06", "f")
	outID, _ := outRes.LastInsertId()

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kader/{id}/member-suggestions", h.MemberSuggestions)
	})

	resp := testutil.Get(t, srv, fmt.Sprintf("/api/kader/%d/member-suggestions", kaderID), token)
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
		t.Errorf("expected 2015 member (id=%d) in dedicated-year suggestions", inID)
	}
	if outFound {
		t.Errorf("expected 2016 member (id=%d) NOT in 2015-dedicated suggestions", outID)
	}
}

// TestGetKader_TrainingGroupBracketYears verifies that a training-group kader
// exposes selectable bracket_years (computed relative to D-Jugend) so the
// "Jahrgang wählen" dropdown is populated — previously it was empty.
func TestGetKader_TrainingGroupBracketYears(t *testing.T) {
	db := testutil.NewDB(t)
	h := kader.NewHandler(db, hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Förderkader brackets")
	res, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "Förderkader", "mixed", teamID, 1)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	kaderIDRaw, _ := res.LastInsertId()
	kaderID := int(kaderIDRaw)

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/kader/{id}", h.GetKader)
	})

	resp := testutil.Get(t, srv, fmt.Sprintf("/api/kader/%d", kaderID), token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	var payload struct {
		BracketYears []int `json:"bracket_years"`
		BirthYears   []int `json:"birth_years"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	// 2025/26: D+1..D+6 = 2015..2020.
	if len(payload.BracketYears) == 0 {
		t.Fatal("training-group bracket_years must be populated (dropdown source)")
	}
	if payload.BracketYears[0] != 2015 {
		t.Errorf("first bracket year: got %d, want 2015 (D+1)", payload.BracketYears[0])
	}
	// Without a dedicated year a training group has no implied roster range.
	if len(payload.BirthYears) != 0 {
		t.Errorf("training-group birth_years without dedicated year: got %v, want []", payload.BirthYears)
	}
}

// ── TC-K06/K07: CopyFromSeason ────────────────────────────────────────────────

// TC-K06: CopyFromSeason mit member_source=same-age-previous übernimmt Mitglieder.
func TestCopyFromSeason_SameAgePrevious(t *testing.T) {
	db := testutil.NewDB(t)

	// Quell-Saison 2024/25 mit A-Jugend-Kader und Mitgliedern im Bracket.
	fromSeasonID := testutil.CreateSeason(t, db, "2024/25")
	// Override season dates so seasonStartYear=2024.
	db.Exec(`UPDATE seasons SET start_date='2024-09-01', end_date='2025-06-30' WHERE id=?`, fromSeasonID)
	teamID := testutil.CreateTeam(t, db, "A-Jugend")
	db.Exec(`UPDATE teams SET age_class='A-Jugend', gender='mixed' WHERE id=?`, teamID)

	srcKaderRes, _ := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		fromSeasonID, "A-Jugend", "mixed", teamID, 1)
	srcKaderID64, _ := srcKaderRes.LastInsertId()
	srcKaderID := int(srcKaderID64)

	// Mitglieder Jg. 2006–2007 (in A-Jugend 2024/25 bracket) in Quell-Kader.
	m2006, _ := db.Exec(`INSERT INTO members (first_name, last_name, status, date_of_birth) VALUES (?,?,?,?)`,
		"Alt", "M2006", "aktiv", "2006-05-01")
	m2007, _ := db.Exec(`INSERT INTO members (first_name, last_name, status, date_of_birth) VALUES (?,?,?,?)`,
		"Alt", "M2007", "aktiv", "2007-03-15")
	id2006, _ := m2006.LastInsertId()
	id2007, _ := m2007.LastInsertId()
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, srcKaderID, id2006)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, srcKaderID, id2007)

	// Ziel-Saison 2025/26.
	toSeasonRes, _ := db.Exec(
		`INSERT INTO seasons (name, start_date, end_date, is_active) VALUES (?, ?, ?, 0)`,
		"2025/26", "2025-09-01", "2026-06-30")
	toSeasonID64, _ := toSeasonRes.LastInsertId()
	toSeasonID := int(toSeasonID64)

	adminID := testutil.CreateUser(t, db, "admin")
	srv := testutil.NewServer(t, func(r chi.Router) {
		h := kader.NewHandler(db, hub.NewHub())
		r.Post("/api/kader/copy-from-season", h.CopyFromSeason)
	})

	res := testutil.Post(t, srv, "/api/kader/copy-from-season",
		testutil.Token(t, adminID, "admin", nil),
		map[string]any{
			"from_season_id": fromSeasonID,
			"to_season_id":   toSeasonID,
			"assignments": []map[string]any{
				{"age_class": "A-Jugend", "gender": "mixed", "member_source": "same-age-previous"},
			},
		})
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	var newKaderID int
	db.QueryRow(`SELECT id FROM kader WHERE season_id=? AND age_class='A-Jugend'`, toSeasonID).Scan(&newKaderID)
	if newKaderID == 0 {
		t.Fatal("new kader not created in target season")
	}
	var memberCount int
	db.QueryRow(`SELECT COUNT(*) FROM kader_members WHERE kader_id=?`, newKaderID).Scan(&memberCount)
	if memberCount == 0 {
		t.Errorf("expected members copied to new kader, got 0")
	}
}

// TC-K07: CopyFromSeason mit member_source="" legt Kader ohne Mitglieder an.
func TestCopyFromSeason_EmptyMemberSource(t *testing.T) {
	db := testutil.NewDB(t)

	fromSeasonID := testutil.CreateSeason(t, db, "2024/25")
	db.Exec(`UPDATE seasons SET start_date='2024-09-01', end_date='2025-06-30' WHERE id=?`, fromSeasonID)
	teamID := testutil.CreateTeam(t, db, "B-Jugend")
	db.Exec(`UPDATE teams SET age_class='B-Jugend', gender='mixed' WHERE id=?`, teamID)

	db.Exec(`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		fromSeasonID, "B-Jugend", "mixed", teamID, 1)

	toSeasonRes, _ := db.Exec(
		`INSERT INTO seasons (name, start_date, end_date, is_active) VALUES (?, ?, ?, 0)`,
		"2025/26", "2025-09-01", "2026-06-30")
	toSeasonID64, _ := toSeasonRes.LastInsertId()
	toSeasonID := int(toSeasonID64)

	adminID := testutil.CreateUser(t, db, "admin")
	srv := testutil.NewServer(t, func(r chi.Router) {
		h := kader.NewHandler(db, hub.NewHub())
		r.Post("/api/kader/copy-from-season", h.CopyFromSeason)
	})

	res := testutil.Post(t, srv, "/api/kader/copy-from-season",
		testutil.Token(t, adminID, "admin", nil),
		map[string]any{
			"from_season_id": fromSeasonID,
			"to_season_id":   toSeasonID,
			"assignments": []map[string]any{
				{"age_class": "B-Jugend", "gender": "mixed", "member_source": ""},
			},
		})
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var newKaderID int
	db.QueryRow(`SELECT id FROM kader WHERE season_id=? AND age_class='B-Jugend'`, toSeasonID).Scan(&newKaderID)
	if newKaderID == 0 {
		t.Fatal("new kader not created")
	}
	var memberCount int
	db.QueryRow(`SELECT COUNT(*) FROM kader_members WHERE kader_id=?`, newKaderID).Scan(&memberCount)
	if memberCount != 0 {
		t.Errorf("expected 0 members for empty member_source, got %d", memberCount)
	}
}
