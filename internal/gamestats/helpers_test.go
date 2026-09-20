package gamestats

import (
	"context"
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/bwhv"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func intp(n int) *int { return &n }

// newStaffel legt Saison und Staffel an und liefert Store plus IDs.
func newStaffel(t *testing.T) (*sql.DB, *Store, int, int) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "26/27")
	s := NewStore(db)
	st, err := s.upsertStaffel(context.Background(), Staffel{
		SeasonID: seasonID, Code: "mB-RL-BW", Name: "B-Jugend RL",
		OrgID: 216, PeriodID: "142",
	})
	if err != nil {
		t.Fatalf("Staffel anlegen: %v", err)
	}
	return db, s, seasonID, st.ID
}

func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("COUNT(%s): %v", table, err)
	}
	return n
}

// sampleSchedule baut einen Spielplan mit n Begegnungen; die erste trägt ein
// Ergebnis und eine sGID.
func sampleSchedule(n int) *bwhv.Schedule {
	sch := &bwhv.Schedule{ReportURL: "https://example.invalid/r?sGID="}
	for i := 0; i < n; i++ {
		g := bwhv.Game{
			GameNo:    itoa(900000 + i),
			Date:      "2026-09-20",
			Time:      "16:00",
			HomeTeam:  "Verein A",
			GuestTeam: "Verein B",
		}
		if i == 0 {
			g.SGID = "3504061"
			g.HomeGoals, g.GuestGoals = intp(29), intp(25)
			g.HomeGoalsHT, g.GuestGoalsHT = intp(14), intp(13)
		}
		sch.Games = append(sch.Games, g)
	}
	sch.Table = []bwhv.TableRow{{Position: 1, TeamName: "Verein A", Games: 1, PointsPlus: 2}}
	return sch
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
