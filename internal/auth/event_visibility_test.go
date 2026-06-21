package auth_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// eventVisFixture richtet zwei Teams + zwei Spiele in der aktiven Saison ein.
// Spiel A gehört Team A, Spiel B gehört Team B; Member-Ids gehören jeweils ins
// reguläre Kader des passenden Teams.
type eventVisFixture struct {
	seasonID                                                      int
	teamA, teamB, kaderA, kaderB                                  int
	gameA, gameB                                                  int
	memberA, memberB                                              int
	userOfA, userOfB                                              int
	userWithoutTeam, parentOfA, trainerUID, vorstandUID, adminUID int
}

func setupEventVisFixture(t *testing.T, db *sql.DB) eventVisFixture {
	t.Helper()
	fx := eventVisFixture{}
	fx.seasonID = testutil.CreateSeason(t, db, "2025/26")
	if _, err := db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, fx.seasonID); err != nil {
		t.Fatalf("activate season: %v", err)
	}

	mkTeam := func(name string) int {
		res, err := db.Exec(`INSERT INTO teams (name, age_class, gender) VALUES (?,?,?)`, name, "Erwachsene", "m")
		if err != nil {
			t.Fatalf("insert team: %v", err)
		}
		id, _ := res.LastInsertId()
		return int(id)
	}
	mkKader := func(seasonID, teamID, number int) int {
		res, err := db.Exec(`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?,?,?,?,?)`,
			seasonID, "Erwachsene", "m", teamID, number)
		if err != nil {
			t.Fatalf("insert kader: %v", err)
		}
		id, _ := res.LastInsertId()
		return int(id)
	}

	fx.teamA = mkTeam("Team A")
	fx.teamB = mkTeam("Team B")
	fx.kaderA = mkKader(fx.seasonID, fx.teamA, 1)
	fx.kaderB = mkKader(fx.seasonID, fx.teamB, 2)

	fx.userOfA = testutil.CreateUser(t, db, "standard")
	fx.memberA = testutil.CreateMember(t, db, fx.userOfA)
	fx.userOfB = testutil.CreateUser(t, db, "standard")
	fx.memberB = testutil.CreateMember(t, db, fx.userOfB)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?,?)`, fx.kaderA, fx.memberA)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?,?)`, fx.kaderB, fx.memberB)

	fx.gameA = testutil.CreateGame(t, db, fx.seasonID, fx.teamA, "2026-03-10")
	fx.gameB = testutil.CreateGame(t, db, fx.seasonID, fx.teamB, "2026-03-11")

	// Außenstehende: User ohne Member, Eltern eines A-Members, Trainer, Vorstand, Admin.
	fx.userWithoutTeam = testutil.CreateUser(t, db, "standard")
	fx.parentOfA = testutil.CreateUser(t, db, "standard")
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, fx.parentOfA, fx.memberA); err != nil {
		t.Fatalf("family_links: %v", err)
	}

	fx.trainerUID = testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, fx.trainerUID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?,?)`, trainerMember, "trainer")

	fx.vorstandUID = testutil.CreateUser(t, db, "standard")
	vorstandMember := testutil.CreateMember(t, db, fx.vorstandUID)
	db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?,?)`, vorstandMember, "vorstand")

	fx.adminUID = testutil.CreateUser(t, db, "admin")
	return fx
}

func TestUserCanSeeGame_EigenesTeam(t *testing.T) {
	db := testutil.NewDB(t)
	fx := setupEventVisFixture(t, db)

	ok, err := auth.UserCanSeeGame(context.Background(), db, fx.userOfA, fx.gameA)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !ok {
		t.Errorf("UserOfA muss gameA sehen")
	}
	ok, _ = auth.UserCanSeeGame(context.Background(), db, fx.userOfA, fx.gameB)
	if ok {
		t.Errorf("UserOfA darf gameB (anderes Team) NICHT sehen")
	}
}

func TestUserCanSeeGame_ElternsKindIstImTeam(t *testing.T) {
	db := testutil.NewDB(t)
	fx := setupEventVisFixture(t, db)

	ok, err := auth.UserCanSeeGame(context.Background(), db, fx.parentOfA, fx.gameA)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !ok {
		t.Errorf("Elternteil eines A-Members muss gameA sehen")
	}
	ok, _ = auth.UserCanSeeGame(context.Background(), db, fx.parentOfA, fx.gameB)
	if ok {
		t.Errorf("Elternteil eines A-Members darf gameB NICHT sehen")
	}
}

func TestUserCanSeeGame_KeineZugehoerigkeit(t *testing.T) {
	db := testutil.NewDB(t)
	fx := setupEventVisFixture(t, db)

	for _, gid := range []int{fx.gameA, fx.gameB} {
		ok, err := auth.UserCanSeeGame(context.Background(), db, fx.userWithoutTeam, gid)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if ok {
			t.Errorf("User ohne Team-Bezug darf game %d NICHT sehen", gid)
		}
	}
}

func TestUserCanSeeGame_Funktionstraeger(t *testing.T) {
	db := testutil.NewDB(t)
	fx := setupEventVisFixture(t, db)

	for _, uid := range []int{fx.trainerUID, fx.vorstandUID, fx.adminUID} {
		for _, gid := range []int{fx.gameA, fx.gameB} {
			ok, err := auth.UserCanSeeGame(context.Background(), db, uid, gid)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if !ok {
				t.Errorf("Funktionsträger UID %d muss game %d sehen (Bypass)", uid, gid)
			}
		}
	}
}

func TestUserCanSeeGame_MultiTeamEvent(t *testing.T) {
	db := testutil.NewDB(t)
	fx := setupEventVisFixture(t, db)
	// gameA bekommt Team B als zusätzliches Team
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`, fx.gameA, fx.teamB); err != nil {
		t.Fatalf("insert game_teams: %v", err)
	}

	// UserOfB ist jetzt durch sein Team B beteiligt
	ok, err := auth.UserCanSeeGame(context.Background(), db, fx.userOfB, fx.gameA)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !ok {
		t.Errorf("UserOfB muss gameA sehen, weil Team B mit dabei ist")
	}
}

func TestGameIDsVisibleToUser_EigenesTeam(t *testing.T) {
	db := testutil.NewDB(t)
	fx := setupEventVisFixture(t, db)

	ids, unrestricted, err := auth.GameIDsVisibleToUser(context.Background(), db, fx.userOfA, fx.seasonID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if unrestricted {
		t.Errorf("Standard-Spieler darf nicht unrestricted sein")
	}
	if len(ids) != 1 || ids[0] != fx.gameA {
		t.Errorf("UserOfA: erwartet [%d], got %v", fx.gameA, ids)
	}
}

func TestGameIDsVisibleToUser_Funktionstraeger(t *testing.T) {
	db := testutil.NewDB(t)
	fx := setupEventVisFixture(t, db)

	for _, uid := range []int{fx.trainerUID, fx.vorstandUID, fx.adminUID} {
		ids, unrestricted, err := auth.GameIDsVisibleToUser(context.Background(), db, uid, fx.seasonID)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !unrestricted {
			t.Errorf("UID %d (Funktionsträger): erwartet unrestricted=true", uid)
		}
		if ids != nil {
			t.Errorf("UID %d: unrestricted heißt ids=nil, got %v", uid, ids)
		}
	}
}

func TestGameIDsVisibleToUser_KeineGames(t *testing.T) {
	db := testutil.NewDB(t)
	fx := setupEventVisFixture(t, db)

	ids, unrestricted, err := auth.GameIDsVisibleToUser(context.Background(), db, fx.userWithoutTeam, fx.seasonID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if unrestricted {
		t.Errorf("ohne Team-Bezug darf nicht unrestricted sein")
	}
	if len(ids) != 0 {
		t.Errorf("erwartet leere Liste, got %v", ids)
	}
}
