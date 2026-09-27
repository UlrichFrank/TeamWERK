package attendance_test

import (
	"database/sql"
	"net/http"
	"testing"

	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Spielspalten tragen den Aufstellungsstatus, Trainingsspalten nicht. Eine
// Spalte ohne Aufstellung ist für alle „offen"; die Matrix leitet jede Zelle
// genauso ab wie das SQL-Fragment der übrigen Routen.
func TestRSVPMatrix_Aufstellungsstatus(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	_, kaderID := makeTrainer(t, db, teamID, seasonID)
	playerUser, a := makePlayer(t, db, kaderID)
	_, b := makePlayer(t, db, kaderID)

	set := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-01")
	open := testutil.CreateGame(t, db, seasonID, teamID, "2026-05-08")
	testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-02")
	testutil.AddLineup(t, db, set, a)

	code, body := getMatrix(t, db, teamID, matrixRange,
		testutil.Token(t, playerUser, "standard", []string{"spieler"}))
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	want := map[int]map[int]string{ // game -> member -> lineup
		set:  {a: appdb.LineupIn, b: appdb.LineupOut},
		open: {a: appdb.LineupOpen, b: appdb.LineupOpen},
	}
	for i, ev := range body.Events {
		for _, m := range body.Members {
			got := m.Cells[i].Lineup
			if ev.Kind == "training" {
				if got != "" {
					t.Errorf("Training: Mitglied %d trägt lineup=%q", m.MemberID, got)
				}
				continue
			}
			if w := want[ev.ID][m.MemberID]; got != w {
				t.Errorf("Spiel %d, Mitglied %d: lineup=%q, want %q", ev.ID, m.MemberID, got, w)
			}
			if sqlState := lineupViaSQL(t, db, ev.ID, m.MemberID); got != sqlState {
				t.Errorf("Spiel %d, Mitglied %d: Matrix %q ≠ SQL-Fragment %q", ev.ID, m.MemberID, got, sqlState)
			}
		}
	}
}

func lineupViaSQL(t *testing.T, db *sql.DB, gameID, memberID int) string {
	t.Helper()
	var s sql.NullString
	if err := db.QueryRow(`SELECT `+appdb.LineupStateSQL("g.event_type", "g.id", "m.id")+`
		FROM games g, members m WHERE g.id = ? AND m.id = ?`, gameID, memberID).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s.String
}
