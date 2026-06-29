package videos

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// addKaderMember inserts a member into a kader (→ player via player_memberships view).
func addKaderMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("addKaderMember: %v", err)
	}
}

// addFamilyLink links a parent user to a member.
func addFamilyLink(t *testing.T, db *sql.DB, parentUserID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, memberID); err != nil {
		t.Fatalf("addFamilyLink: %v", err)
	}
}

func claims(userID int, role string, fns ...string) *auth.Claims {
	return &auth.Claims{UserID: userID, Role: role, ClubFunctions: fns}
}

// --- CanUploadToTeam ---------------------------------------------------------

func TestCanUploadToTeam(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)

	// trainer of team A only
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMember)

	tests := []struct {
		name   string
		claims *auth.Claims
		teamID int
		want   bool
	}{
		{"admin any team", claims(testutil.CreateUser(t, db, "admin"), "admin"), teamA, true},
		{"vorstand any team", claims(testutil.CreateUser(t, db, "standard"), "standard", "vorstand"), teamB, true},
		{"sportliche_leitung any team", claims(testutil.CreateUser(t, db, "standard"), "standard", "sportliche_leitung"), teamB, true},
		{"trainer of own team", claims(trainerUser, "standard", "trainer"), teamA, true},
		{"trainer of foreign team", claims(trainerUser, "standard", "trainer"), teamB, false},
		{"plain player", claims(testutil.CreateUser(t, db, "standard"), "standard", "spieler"), teamA, false},
		{"nil claims", nil, teamA, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := h.CanUploadToTeam(tc.claims, tc.teamID)
			if err != nil {
				t.Fatalf("CanUploadToTeam: %v", err)
			}
			if got != tc.want {
				t.Errorf("CanUploadToTeam = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- CanManageTeamVideos -----------------------------------------------------

func TestCanManageTeamVideos(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)

	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMember)

	tests := []struct {
		name   string
		claims *auth.Claims
		teamID int
		want   bool
	}{
		{"admin", claims(testutil.CreateUser(t, db, "admin"), "admin"), teamA, true},
		{"vorstand", claims(testutil.CreateUser(t, db, "standard"), "standard", "vorstand"), teamA, true},
		{"trainer of own team", claims(trainerUser, "standard", "trainer"), teamA, true},
		{"trainer of foreign team", claims(trainerUser, "standard", "trainer"), teamB, false},
		{"sportliche_leitung is not management", claims(testutil.CreateUser(t, db, "standard"), "standard", "sportliche_leitung"), teamA, false},
		{"plain player", claims(testutil.CreateUser(t, db, "standard"), "standard", "spieler"), teamA, false},
		{"nil claims", nil, teamA, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := h.CanManageTeamVideos(tc.claims, tc.teamID)
			if err != nil {
				t.Fatalf("CanManageTeamVideos: %v", err)
			}
			if got != tc.want {
				t.Errorf("CanManageTeamVideos = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- CanViewVideo ------------------------------------------------------------

func TestCanViewVideo(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)

	// active player of team A
	playerUser := testutil.CreateUser(t, db, "standard")
	playerMember := testutil.CreateMember(t, db, playerUser)
	addKaderMember(t, db, kaderA, playerMember)

	// trainer of team A
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kaderA, trainerMember)

	// parent of the active player
	parentUser := testutil.CreateUser(t, db, "standard")
	addFamilyLink(t, db, parentUser, playerMember)

	// outsider: user with a member but no team relation
	outsiderUser := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, outsiderUser)

	videoA := &Video{ID: 1, TeamID: teamA}
	videoB := &Video{ID: 2, TeamID: teamB}

	tests := []struct {
		name   string
		claims *auth.Claims
		video  *Video
		want   bool
	}{
		{"admin always", claims(testutil.CreateUser(t, db, "admin"), "admin"), videoA, true},
		{"vorstand always", claims(testutil.CreateUser(t, db, "standard"), "standard", "vorstand"), videoA, true},
		{"active player of team", claims(playerUser, "standard", "spieler"), videoA, true},
		{"trainer of team", claims(trainerUser, "standard", "trainer"), videoA, true},
		{"parent of player", claims(parentUser, "standard"), videoA, true},
		{"player of foreign team", claims(playerUser, "standard", "spieler"), videoB, false},
		{"outsider", claims(outsiderUser, "standard"), videoA, false},
		{"nil claims", nil, videoA, false},
		{"nil video", claims(playerUser, "standard"), nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := h.CanViewVideo(tc.claims, tc.video)
			if err != nil {
				t.Fatalf("CanViewVideo: %v", err)
			}
			if got != tc.want {
				t.Errorf("CanViewVideo = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- inactive player is excluded from view -----------------------------------

func TestCanViewVideo_InactivePlayerExcluded(t *testing.T) {
	db := testutil.NewDB(t)
	h := NewHandler(db, nil, nil)

	season := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	kaderA := testutil.CreateKader(t, db, teamA, season)

	user := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateMember(t, db, user)
	addKaderMember(t, db, kaderA, member)
	if _, err := db.Exec(`UPDATE members SET status='ausgetreten' WHERE id=?`, member); err != nil {
		t.Fatalf("deactivate member: %v", err)
	}

	got, err := h.CanViewVideo(claims(user, "standard", "spieler"), &Video{ID: 1, TeamID: teamA})
	if err != nil {
		t.Fatalf("CanViewVideo: %v", err)
	}
	if got {
		t.Error("an inactive (ausgetreten) player must not be able to view team videos")
	}
}
