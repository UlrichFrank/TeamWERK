package db_test

import (
	"database/sql"
	"testing"

	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func lineupState(t *testing.T, db *sql.DB, gameID, memberID int) sql.NullString {
	t.Helper()
	var s sql.NullString
	err := db.QueryRow(`SELECT `+appdb.LineupStateSQL("g.event_type", "g.id", "m.id")+`
		FROM games g, members m WHERE g.id = ? AND m.id = ?`, gameID, memberID).Scan(&s)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	return s
}

// Drei Spiele decken die drei Zustände ab, ein generisches Event den
// Termin ohne Aufstellung. Die Go-Form (ResolveLineupState) muss für jede
// Kombination dasselbe liefern wie das SQL-Fragment.
func TestLineupStateSQL(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "mB1")
	a := testutil.CreateMember(t, db, 0)
	b := testutil.CreateMember(t, db, 0)

	open := testutil.CreateGame(t, db, season, team, "2026-10-01")
	set := testutil.CreateGame(t, db, season, team, "2026-10-02")
	testutil.AddLineup(t, db, set, a)
	generic := testutil.CreateGame(t, db, season, team, "2026-10-03")
	if _, err := db.Exec(`UPDATE games SET event_type = 'generisch' WHERE id = ?`, generic); err != nil {
		t.Fatal(err)
	}
	testutil.AddLineup(t, db, generic, a)

	cases := []struct {
		name         string
		game, member int
		want         string // "" = NULL
	}{
		{"leere Aufstellung ist offen, nicht nicht-aufgestellt", open, a, appdb.LineupOpen},
		{"Mitglied in der Aufstellung", set, a, appdb.LineupIn},
		{"Aufstellung ohne das Mitglied", set, b, appdb.LineupOut},
		{"generisches Event hat keine Aufstellung", generic, a, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := lineupState(t, db, c.game, c.member)
			if got.String != c.want || got.Valid != (c.want != "") {
				t.Errorf("SQL: got %+v, want %q", got, c.want)
			}
			// Die Go-Form auf denselben Fakten muss dasselbe liefern.
			var eventType string
			var exists, in bool
			db.QueryRow(`SELECT event_type,
				EXISTS(SELECT 1 FROM game_lineup WHERE game_id = ?),
				EXISTS(SELECT 1 FROM game_lineup WHERE game_id = ? AND member_id = ?)
				FROM games WHERE id = ?`, c.game, c.game, c.member, c.game).Scan(&eventType, &exists, &in)
			if g := appdb.ResolveLineupState(eventType, exists, in); g != c.want {
				t.Errorf("ResolveLineupState: got %q, want %q", g, c.want)
			}
		})
	}
}

func TestResolveLineupState(t *testing.T) {
	cases := []struct {
		eventType        string
		exists, inLineup bool
		want             string
	}{
		{"heim", false, false, appdb.LineupOpen},
		{"auswärts", true, true, appdb.LineupIn},
		{"heim", true, false, appdb.LineupOut},
		{"generisch", true, true, ""},
		{"training", false, false, ""},
	}
	for _, c := range cases {
		if got := appdb.ResolveLineupState(c.eventType, c.exists, c.inLineup); got != c.want {
			t.Errorf("ResolveLineupState(%q, %v, %v) = %q, want %q", c.eventType, c.exists, c.inLineup, got, c.want)
		}
	}
	// Ohne Typ-Gate (Detailseite, auch „Sonstiges"): dieselbe Regel.
	if got := appdb.LineupFromFacts(false, false); got != appdb.LineupOpen {
		t.Errorf("LineupFromFacts(false, false) = %q, want open", got)
	}
	if got := appdb.LineupFromFacts(true, false); got != appdb.LineupOut {
		t.Errorf("LineupFromFacts(true, false) = %q, want out", got)
	}
}
