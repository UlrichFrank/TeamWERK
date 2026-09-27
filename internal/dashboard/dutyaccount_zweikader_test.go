package dashboard_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/dashboard"

	"github.com/teamstuttgart/teamwerk/internal/dutyfairness"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Ein Kind in zwei Stammkadern hat zwei Positionen mit den Zahlen und dem Soll
// des jeweiligen Kaders — dieselben Werte wie in der Rangliste
// (Change dienst-bilanz-je-kader).
func TestDashboard_DutyAccount_ZweiKaderEigeneZahlen(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2026/27")
	dt := testutil.CreateDutyType(t, db, "Kasse", 1.0)
	k1Team, k2Team := testutil.CreateTeam(t, db, "K1"), testutil.CreateTeam(t, db, "K2")
	k1, k2 := testutil.CreateKader(t, db, k1Team, season), testutil.CreateKader(t, db, k2Team, season)

	parent := testutil.CreateUser(t, db, "standard")
	kid := testutil.CreateMember(t, db, 0)
	testutil.AddKaderMember(t, db, k1, kid)
	testutil.AddKaderMember(t, db, k2, kid)
	testutil.AddFamilyLink(t, db, parent, kid)
	testutil.AddKaderMember(t, db, k1, testutil.CreateMember(t, db, 0))
	for range 3 {
		testutil.AddKaderMember(t, db, k2, testutil.CreateMember(t, db, 0))
	}

	past := testutil.CreateGame(t, db, season, k1Team, dayOffset(-3))
	slot := testutil.CreateDutySlot(t, db, dt, season, k1Team, past, dayOffset(-3))
	testutil.CreateDutySlot(t, db, dt, season, k1Team, testutil.CreateGame(t, db, season, k1Team, dayOffset(5)), dayOffset(5))
	testutil.CreateDutySlot(t, db, dt, season, k2Team, testutil.CreateGame(t, db, season, k2Team, dayOffset(5)), dayOffset(5))
	if _, err := db.Exec(`INSERT INTO duty_assignments (duty_slot_id, user_id) VALUES (?, ?)`, slot, parent); err != nil {
		t.Fatal(err)
	}

	body := loadDashboardFull(t, db, parent)
	snap, err := dutyfairness.Compute(context.Background(), db, season)
	if err != nil {
		t.Fatal(err)
	}
	ranked := map[int]*dutyfairness.Position{}
	for _, tid := range []int{k1Team, k2Team} {
		for _, p := range snap.Teams[tid].Ranked() {
			if p.Member.MemberID == kid {
				ranked[tid] = p
			}
		}
	}

	want := map[int]struct{ geleistet, soll float64 }{k1Team: {1, 2}, k2Team: {0, 0.5}} // slots_total je Slot = 2
	if len(body) != 2 {
		t.Fatalf("dutyAccount = %+v, want zwei Positionen", body)
	}
	for _, e := range body {
		w, ok := want[e.TeamID]
		if !ok || e.MemberID != kid {
			t.Fatalf("unerwartete Position %+v", e)
		}
		if e.Geleistet != w.geleistet || e.Soll != w.soll {
			t.Errorf("Team %d: geleistet/soll = %v/%v, want %v/%v", e.TeamID, e.Geleistet, e.Soll, w.geleistet, w.soll)
		}
		if r := ranked[e.TeamID]; r == nil || dutyfairness.Round2(r.Geleistet) != e.Geleistet || dutyfairness.Round2(r.Vorhersage) != e.Vorhersage {
			t.Errorf("Team %d: Kachel %+v weicht von der Rangliste %+v ab", e.TeamID, e, r)
		}
	}
}

func loadDashboardFull(t *testing.T, db *sql.DB, userID int) []dashboard.DutyAccountEntry {
	t.Helper()
	srv := testServer(t, dashboard.NewHandler(db))
	res := testutil.Get(t, srv, "/api/dashboard", testutil.TokenWithIsParent(t, userID, "standard", nil, true))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		MeineDienste struct {
			DutyAccount []dashboard.DutyAccountEntry `json:"dutyAccount"`
		} `json:"meineDienste"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.MeineDienste.DutyAccount
}
