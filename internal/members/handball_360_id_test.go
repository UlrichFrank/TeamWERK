package members_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Happy-Path: Vorstand setzt handball_360_id → 204, Feld persistiert und wird
// über GET /members/{id} wieder ausgeliefert.
func TestUpdateMember_Handball360ID_PersistsAndReturned(t *testing.T) {
	database := testutil.NewDB(t)
	memberID := testutil.CreateMember(t, database, 0)
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", memberID), tok,
		map[string]any{
			"first_name":      "Hans",
			"last_name":       "Dampf",
			"status":          "aktiv",
			"join_date":       "2026-01-01",
			"handball_360_id": "H360-42",
		})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT /members/{id}: expected 204, got %d", res.StatusCode)
	}

	res2 := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d", memberID), tok)
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("GET /members/{id}: expected 200, got %d", res2.StatusCode)
	}
	var got struct {
		Handball360ID string `json:"handball_360_id"`
	}
	if err := json.NewDecoder(res2.Body).Decode(&got); err != nil {
		res2.Body.Close()
		t.Fatalf("decode: %v", err)
	}
	res2.Body.Close()
	if got.Handball360ID != "H360-42" {
		t.Errorf("handball_360_id: got %q, want %q", got.Handball360ID, "H360-42")
	}
}

// Invariante: anders als pass_number (spielerspezifisch) ist handball_360_id
// nicht auf die Vereinsfunktion "spieler" beschränkt und wird auch bei Status
// 'extern' NICHT geleert — Vereinshelfer ohne eigene Mitgliedschaft (z. B.
// externe Trainer) können ebenfalls einen Handball4All-Account haben.
func TestUpdateMember_Handball360ID_KeptForExternStatus(t *testing.T) {
	database := testutil.NewDB(t)
	memberID := testutil.CreateMember(t, database, 0)
	if _, err := database.Exec(`UPDATE members SET handball_360_id='H360-99' WHERE id=?`, memberID); err != nil {
		t.Fatalf("seed handball_360_id: %v", err)
	}
	vorstandID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/members/%d", memberID), tok,
		map[string]any{
			"first_name":      "Hans",
			"last_name":       "Dampf",
			"status":          "extern",
			"join_date":       "2026-01-01",
			"handball_360_id": "H360-99",
		})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT /members/{id}: expected 204, got %d", res.StatusCode)
	}

	var handball360 *string
	if err := database.QueryRow(`SELECT handball_360_id FROM members WHERE id=?`, memberID).Scan(&handball360); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if handball360 == nil || *handball360 != "H360-99" {
		t.Errorf("handball_360_id: erwartet weiterhin 'H360-99' bei Status 'extern', war %v", handball360)
	}
}
