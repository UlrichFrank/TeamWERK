package hub_test

import (
	"context"
	"sort"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func sorted(ids []int) []int {
	out := append([]int(nil), ids...)
	sort.Ints(out)
	return out
}

func contains(ids []int, want int) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

// TestAudience_FinanceGroup: only admins and users whose member holds a
// vorstand/vorstand_beisitzer/kassierer function are in the finance audience;
// a plain player is not.
func TestAudience_FinanceGroup(t *testing.T) {
	db := testutil.NewDB(t)

	adminU := testutil.CreateUser(t, db, "admin")

	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	kassiererU := testutil.CreateUser(t, db, "standard")
	kassiererM := testutil.CreateMember(t, db, kassiererU)
	testutil.AddClubFunction(t, db, kassiererM, "kassierer")

	beisitzerU := testutil.CreateUser(t, db, "standard")
	beisitzerM := testutil.CreateMember(t, db, beisitzerU)
	testutil.AddClubFunction(t, db, beisitzerM, "vorstand_beisitzer")

	playerU := testutil.CreateUser(t, db, "standard")
	playerM := testutil.CreateMember(t, db, playerU)
	testutil.AddClubFunction(t, db, playerM, "spieler")

	a := hub.NewAudience(db)
	got := a.FinanceGroup(context.Background())

	for _, want := range []int{adminU, vorstandU, kassiererU, beisitzerU} {
		if !contains(got, want) {
			t.Errorf("finance group must contain user %d, got %v", want, sorted(got))
		}
	}
	if contains(got, playerU) {
		t.Errorf("finance group must NOT contain plain player %d, got %v", playerU, sorted(got))
	}
}

// TestAudience_FinanceGroup_ExtraUser: an explicitly passed affected user is
// included even without a finance function.
func TestAudience_FinanceGroup_ExtraUser(t *testing.T) {
	db := testutil.NewDB(t)
	ownerU := testutil.CreateUser(t, db, "standard")

	a := hub.NewAudience(db)
	got := a.FinanceGroup(context.Background(), ownerU, 0) // 0 must be ignored

	if !contains(got, ownerU) {
		t.Errorf("extra affected user %d must be included, got %v", ownerU, sorted(got))
	}
	if contains(got, 0) {
		t.Errorf("zero id must be ignored, got %v", sorted(got))
	}
}

// TestAudience_Team: players and trainers of the target team, their parents, and
// club-wide staff are in the team audience; a member of another team is not.
func TestAudience_Team(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")

	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, season)
	kaderB := testutil.CreateKader(t, db, teamB, season)

	// Player on team A.
	playerAU := testutil.CreateUser(t, db, "standard")
	playerAM := testutil.CreateMember(t, db, playerAU)
	testutil.AddKaderMember(t, db, kaderA, playerAM)

	// Trainer of team A.
	trainerAU := testutil.CreateUser(t, db, "standard")
	trainerAM := testutil.CreateMember(t, db, trainerAU)
	testutil.AddKaderTrainer(t, db, kaderA, trainerAM)

	// Parent of player A.
	parentAU := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentAU, playerAM)

	// Player on team B (should NOT be in team A's audience).
	playerBU := testutil.CreateUser(t, db, "standard")
	playerBM := testutil.CreateMember(t, db, playerBU)
	testutil.AddKaderMember(t, db, kaderB, playerBM)

	// Club-wide sportliche_leitung (no team) — should always see team events.
	slU := testutil.CreateUser(t, db, "standard")
	slM := testutil.CreateMember(t, db, slU)
	testutil.AddClubFunction(t, db, slM, "sportliche_leitung")

	a := hub.NewAudience(db)
	got := a.Team(context.Background(), []int{teamA})

	for _, want := range []int{playerAU, trainerAU, parentAU, slU} {
		if !contains(got, want) {
			t.Errorf("team A audience must contain user %d, got %v", want, sorted(got))
		}
	}
	if contains(got, playerBU) {
		t.Errorf("team A audience must NOT contain team-B player %d, got %v", playerBU, sorted(got))
	}
}

// TestAudience_Team_EmptyTeams: with no team IDs the audience contains only the
// club-wide staff (no player leaks) — callers with no resolvable team should
// prefer the global Broadcast instead.
func TestAudience_Team_EmptyTeams(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "Team A")
	kader := testutil.CreateKader(t, db, team, season)

	playerU := testutil.CreateUser(t, db, "standard")
	playerM := testutil.CreateMember(t, db, playerU)
	testutil.AddKaderMember(t, db, kader, playerM)

	vorstandU := testutil.CreateUser(t, db, "standard")
	vorstandM := testutil.CreateMember(t, db, vorstandU)
	testutil.AddClubFunction(t, db, vorstandM, "vorstand")

	a := hub.NewAudience(db)
	got := a.Team(context.Background(), nil)

	if contains(got, playerU) {
		t.Errorf("empty-team audience must not contain a player, got %v", sorted(got))
	}
	if !contains(got, vorstandU) {
		t.Errorf("empty-team audience must still contain vorstand %d, got %v", vorstandU, sorted(got))
	}
}
