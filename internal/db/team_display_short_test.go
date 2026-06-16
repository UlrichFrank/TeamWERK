package db_test

import (
	"database/sql"
	"testing"

	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// helper: insert a team with custom age_class+gender (testutil.CreateTeam is fixed to "Erwachsene"/"mixed")
func mkTeam(t *testing.T, db *sql.DB, name, ageClass, gender string) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO teams (name, age_class, gender) VALUES (?, ?, ?)`, name, ageClass, gender)
	if err != nil {
		t.Fatalf("mkTeam: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func mkKader(t *testing.T, db *sql.DB, seasonID, teamID int, ageClass, gender string, teamNumber int) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, ageClass, gender, teamID, teamNumber)
	if err != nil {
		t.Fatalf("mkKader: %v", err)
	}
}

func short(t *testing.T, db *sql.DB, teamID int) (string, bool) {
	t.Helper()
	var s sql.NullString
	q := `SELECT ` + appdb.TeamDisplayShort("t") + ` FROM teams t WHERE t.id = ?`
	if err := db.QueryRow(q, teamID).Scan(&s); err != nil {
		t.Fatalf("query: %v", err)
	}
	return s.String, s.Valid
}

func TestTeamDisplayShort_SingleTeam_NoSuffix(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	team := mkTeam(t, db, "Team A", "A-Jugend", "m")
	mkKader(t, db, seasonID, team, "A-Jugend", "m", 1)

	got, ok := short(t, db, team)
	if !ok {
		t.Fatalf("expected non-NULL display_short")
	}
	if got != "mA" {
		t.Errorf("expected 'mA', got %q", got)
	}
}

func TestTeamDisplayShort_MultipleTeams_AddSuffix(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	t1 := mkTeam(t, db, "Team B1", "B-Jugend", "m")
	t2 := mkTeam(t, db, "Team B2", "B-Jugend", "m")
	mkKader(t, db, seasonID, t1, "B-Jugend", "m", 1)
	mkKader(t, db, seasonID, t2, "B-Jugend", "m", 2)

	if got, _ := short(t, db, t1); got != "mB1" {
		t.Errorf("team 1: expected 'mB1', got %q", got)
	}
	if got, _ := short(t, db, t2); got != "mB2" {
		t.Errorf("team 2: expected 'mB2', got %q", got)
	}
}

func TestTeamDisplayShort_AllGenders(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	tm := mkTeam(t, db, "Tm", "C-Jugend", "m")
	tf := mkTeam(t, db, "Tf", "C-Jugend", "f")
	tg := mkTeam(t, db, "Tg", "C-Jugend", "mixed")
	mkKader(t, db, seasonID, tm, "C-Jugend", "m", 1)
	mkKader(t, db, seasonID, tf, "C-Jugend", "f", 1)
	mkKader(t, db, seasonID, tg, "C-Jugend", "mixed", 1)

	cases := []struct {
		teamID int
		want   string
	}{
		{tm, "mC"},
		{tf, "wC"},
		{tg, "gC"},
	}
	for _, c := range cases {
		if got, _ := short(t, db, c.teamID); got != c.want {
			t.Errorf("team %d: expected %q, got %q", c.teamID, c.want, got)
		}
	}
}

func TestTeamDisplayShort_UnknownAgeClass_FirstChar(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	team := mkTeam(t, db, "Erwachsene 1", "Erwachsene", "m")
	mkKader(t, db, seasonID, team, "Erwachsene", "m", 1)

	got, _ := short(t, db, team)
	if got != "mE" {
		t.Errorf("expected 'mE', got %q", got)
	}
}

func TestTeamDisplayShort_NoKaderInActiveSeason_ReturnsNull(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateSeason(t, db, "2025/26")
	team := mkTeam(t, db, "Lonely", "A-Jugend", "m")
	// kein Kader-Eintrag

	got, ok := short(t, db, team)
	if ok {
		t.Errorf("expected NULL, got %q", got)
	}
}

func TestTeamDisplayShort_OtherSeasonKaderIgnored(t *testing.T) {
	db := testutil.NewDB(t)
	// inactive Saison anlegen, dann aktive
	_, _ = db.Exec(`INSERT INTO seasons (name, start_date, end_date, is_active) VALUES (?, ?, ?, 0)`,
		"2024/25", "2024-09-01", "2025-06-30")
	var oldSeason int
	db.QueryRow(`SELECT id FROM seasons WHERE name='2024/25'`).Scan(&oldSeason)
	activeSeason := testutil.CreateSeason(t, db, "2025/26")

	team := mkTeam(t, db, "Team D", "D-Jugend", "f")
	// nur Kader in alter Saison → kein Kader in aktiver → NULL
	mkKader(t, db, oldSeason, team, "D-Jugend", "f", 1)

	if _, ok := short(t, db, team); ok {
		t.Errorf("expected NULL for team without active-season kader")
	}

	// jetzt auch in aktiver Saison anlegen → liefert Kurzform
	mkKader(t, db, activeSeason, team, "D-Jugend", "f", 1)
	if got, _ := short(t, db, team); got != "wD" {
		t.Errorf("expected 'wD', got %q", got)
	}
}
