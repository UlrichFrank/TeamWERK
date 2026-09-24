package db_test

import (
	"database/sql"
	"slices"
	"testing"

	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func userTeams(t *testing.T, db *sql.DB, kind appdb.UserTeamKind, userID int) []int {
	t.Helper()
	rows, err := db.Query(appdb.UserTeamsSQL(kind, "?")+` ORDER BY 1`, appdb.UserArgs(kind, userID)...)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		out = append(out, id)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// Eine Konstellation deckt alle Zweige ab: Spieler (Stamm A, erweitert B),
// Trainer (C), Elternteil eines Förderkindes im erweiterten Kader (D), ein
// ausgetretenes Mitglied im erweiterten Kader (E) und eine inaktive Saison (F).
func TestUserTeamsSQL(t *testing.T) {
	db := testutil.NewDB(t)
	old := testutil.CreateSeason(t, db, "2024/25")
	teamF := testutil.CreateTeam(t, db, "F")
	kaderF := testutil.CreateKader(t, db, teamF, old)
	season := testutil.CreateSeason(t, db, "2025/26") // aktiviert die neue, deaktiviert die alte
	teamA := testutil.CreateTeam(t, db, "A")
	teamB := testutil.CreateTeam(t, db, "B")
	teamC := testutil.CreateTeam(t, db, "C")
	teamD := testutil.CreateTeam(t, db, "D")
	teamE := testutil.CreateTeam(t, db, "E")
	kA := testutil.CreateKader(t, db, teamA, season)
	kB := testutil.CreateKader(t, db, teamB, season)
	kC := testutil.CreateKader(t, db, teamC, season)
	kD := testutil.CreateKader(t, db, teamD, season)
	kE := testutil.CreateKader(t, db, teamE, season)

	player := testutil.CreateUser(t, db, "standard")
	pm := testutil.CreateMember(t, db, player)
	testutil.AddKaderMember(t, db, kA, pm)
	testutil.AddExtendedKaderMember(t, db, kB, pm)
	testutil.AddExtendedKaderMember(t, db, kaderF, pm) // inaktive Saison zählt nicht

	trainer := testutil.CreateUser(t, db, "standard")
	tm := testutil.CreateMember(t, db, trainer)
	testutil.AddKaderTrainer(t, db, kC, tm)

	parent := testutil.CreateUser(t, db, "standard")
	child := testutil.CreateMember(t, db, 0)
	db.Exec(`UPDATE members SET status='foerderkind' WHERE id=?`, child)
	testutil.AddFamilyLink(t, db, parent, child)
	testutil.AddExtendedKaderMember(t, db, kD, child)

	gone := testutil.CreateMember(t, db, 0)
	db.Exec(`UPDATE members SET status='ausgetreten' WHERE id=?`, gone)
	testutil.AddFamilyLink(t, db, parent, gone)
	testutil.AddExtendedKaderMember(t, db, kE, gone)

	cases := []struct {
		name string
		kind appdb.UserTeamKind
		user int
		want []int
	}{
		{"Spieler Stamm", appdb.TeamsStamm, player, []int{teamA}},
		{"Spieler erweitert", appdb.TeamsExtended, player, []int{teamB}},
		{"Spieler ohne Kinder", appdb.TeamsChildren, player, nil},
		{"Trainer Stamm", appdb.TeamsStamm, trainer, []int{teamC}},
		{"Trainer nie erweitert", appdb.TeamsExtended, trainer, nil},
		{"Elternteil Förderkind erweitert, Ausgetretene nicht", appdb.TeamsExtended, parent, []int{teamD}},
		{"Elternteil Kinder-Teams", appdb.TeamsChildren, parent, []int{teamD}},
		{"Elternteil ohne Stamm", appdb.TeamsStamm, parent, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := userTeams(t, db, c.kind, c.user); !slices.Equal(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}

	// Stamm-Kind eines Elternteils zählt als Stamm und als Kinder-Team.
	stammChild := testutil.CreateMember(t, db, 0)
	testutil.AddFamilyLink(t, db, parent, stammChild)
	testutil.AddKaderMember(t, db, kA, stammChild)
	if got := userTeams(t, db, appdb.TeamsStamm, parent); !slices.Equal(got, []int{teamA}) {
		t.Errorf("Elternteil Stamm: got %v, want [%d]", got, teamA)
	}
	if got := userTeams(t, db, appdb.TeamsChildren, parent); !slices.Equal(got, []int{teamA, teamD}) {
		t.Errorf("Elternteil Kinder: got %v, want [%d %d]", got, teamA, teamD)
	}
}
