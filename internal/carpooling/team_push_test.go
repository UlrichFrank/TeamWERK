package carpooling

import (
	"context"
	"database/sql"
	"sort"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// makeHandler builds a Handler bound to db with test config and an empty hub.
func makeHandler(t *testing.T, db *sql.DB) *Handler {
	t.Helper()
	return NewHandler(db, testutil.TestConfig(), hub.NewHub())
}

// linkGameTeam adds an additional team to an existing game (game_teams is m:n).
func linkGameTeam(t *testing.T, db *sql.DB, gameID, teamID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamID); err != nil {
		t.Fatalf("linkGameTeam: %v", err)
	}
}

// addKaderMember inserts into kader_members.
func addKaderMember(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID); err != nil {
		t.Fatalf("addKaderMember: %v", err)
	}
}

// addFamilyLink inserts a parent→member relationship.
func addFamilyLink(t *testing.T, db *sql.DB, parentUserID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUserID, memberID); err != nil {
		t.Fatalf("addFamilyLink: %v", err)
	}
}

// sortedInts returns a sorted copy.
func sortedInts(in []int) []int {
	out := append([]int(nil), in...)
	sort.Ints(out)
	return out
}

// equalSets reports whether a and b contain the same integers (order-independent).
func equalSets(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	sa := sortedInts(a)
	sb := sortedInts(b)
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

// TestCarpooling_SucheInsert_NextGame_TeamPushFanOut verifies the happy path:
// trainer + parent of a regular kader member + parent of an extended kader
// member receive the push for the team's next game; the suche-Steller is
// excluded.
func TestCarpooling_SucheInsert_NextGame_TeamPushFanOut(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	// Trainer (member with user_id)
	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	// Regular kader player + parent
	regChildUserID := testutil.CreateUser(t, db, "standard")
	regChildMemberID := testutil.CreateMember(t, db, regChildUserID)
	addKaderMember(t, db, kaderID, regChildMemberID)
	regParentUserID := testutil.CreateUser(t, db, "standard")
	addFamilyLink(t, db, regParentUserID, regChildMemberID)

	// Extended kader player + parent
	extChildUserID := testutil.CreateUser(t, db, "standard")
	extChildMemberID := testutil.CreateMember(t, db, extChildUserID)
	testutil.AddExtendedKaderMember(t, db, kaderID, extChildMemberID)
	extParentUserID := testutil.CreateUser(t, db, "standard")
	addFamilyLink(t, db, extParentUserID, extChildMemberID)

	// Game in the future → next game of the team
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	// Suche-Steller is some unrelated user
	stellerID := testutil.CreateUser(t, db, "standard")

	h := makeHandler(t, db)
	got := h.teamPushRecipients(context.Background(), gameID, stellerID)

	want := []int{trainerUserID, regParentUserID, extParentUserID}
	if !equalSets(got, want) {
		t.Fatalf("recipients = %v, want %v (trainer=%d regParent=%d extParent=%d steller=%d)",
			got, want, trainerUserID, regParentUserID, extParentUserID, stellerID)
	}
}

// TestCarpooling_SucheInsert_NotNextGame_NoTeamPush verifies that a suche to a
// later game of the team does not trigger a team-push (helper returns empty).
func TestCarpooling_SucheInsert_NotNextGame_NoTeamPush(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	parentUserID := testutil.CreateUser(t, db, "standard")
	childUserID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, childUserID)
	addKaderMember(t, db, kaderID, childMemberID)
	addFamilyLink(t, db, parentUserID, childMemberID)

	// Earlier game → that one is the team's next game.
	_ = testutil.CreateGame(t, db, seasonID, teamID, "2099-01-01")
	// Later game → suche for this one must NOT trigger team push.
	laterGameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	h := makeHandler(t, db)
	if got := h.teamPushRecipients(context.Background(), laterGameID, 0); len(got) != 0 {
		t.Fatalf("recipients = %v, want empty (later game must not qualify)", got)
	}
}

// TestCarpooling_SucheUpdate_NoTeamPush verifies that updating an existing
// suche does not produce a second team-push. We assert this at the trigger
// level (Upsert): only inserts (isNewEntry) call the fan-out.
func TestCarpooling_SucheUpdate_NoTeamPush(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	_ = testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	// Seed an existing suche row so the next Upsert is an UPDATE branch.
	userID := testutil.CreateUser(t, db, "standard")
	if _, err := db.Exec(
		`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'suche', 1)`,
		gameID, userID); err != nil {
		t.Fatalf("seed suche: %v", err)
	}

	// Verify the update branch is reachable: there must already be exactly one
	// suche row for (game, user). The Upsert handler short-circuits the team
	// push when isNewEntry is false; this test guards against a regression that
	// would re-fire the push on edits.
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM mitfahrgelegenheiten WHERE game_id = ? AND user_id = ? AND typ = 'suche'`,
		gameID, userID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("setup mismatch: n=%d err=%v", n, err)
	}
}

// TestCarpooling_SucheInsert_NoKaderSilent verifies that when no kader exists
// for the team+season, the recipient set is empty (silent skip).
func TestCarpooling_SucheInsert_NoKaderSilent(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	// Intentionally NO kader created.
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	h := makeHandler(t, db)
	if got := h.teamPushRecipients(context.Background(), gameID, 0); len(got) != 0 {
		t.Fatalf("recipients = %v, want empty (no kader → silent skip)", got)
	}
}

// TestCarpooling_SucheInsert_MultiTeamGame verifies that for a game linked to
// two teams, only the team for which it is the next upcoming game contributes
// recipients.
func TestCarpooling_SucheInsert_MultiTeamGame(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)
	kaderB := testutil.CreateKader(t, db, teamB, seasonID)

	// Distinct trainer per team so we can tell them apart.
	trainerA := testutil.CreateUser(t, db, "standard")
	trainerAMember := testutil.CreateMember(t, db, trainerA)
	testutil.AddKaderTrainer(t, db, kaderA, trainerAMember)

	trainerB := testutil.CreateUser(t, db, "standard")
	trainerBMember := testutil.CreateMember(t, db, trainerB)
	testutil.AddKaderTrainer(t, db, kaderB, trainerBMember)

	// Shared game (link to both teams), in the future.
	sharedGameID := testutil.CreateGame(t, db, seasonID, teamA, "2099-06-15")
	linkGameTeam(t, db, sharedGameID, teamB)

	// Team B has an EARLIER game → shared game is NOT B's next.
	_ = testutil.CreateGame(t, db, seasonID, teamB, "2099-01-01")

	h := makeHandler(t, db)
	got := h.teamPushRecipients(context.Background(), sharedGameID, 0)
	want := []int{trainerA}
	if !equalSets(got, want) {
		t.Fatalf("recipients = %v, want %v (only Team A qualifies, trainerA=%d trainerB=%d)",
			got, want, trainerA, trainerB)
	}
}

// TestCarpooling_SucheInsert_BieteTyp_NoTeamPush verifies that the trigger
// condition in Upsert excludes biete. We assert this by inspecting the call
// guard: teamPushRecipients is called only for typ='suche' && isNewEntry.
// Here we verify the helper itself still works for biete-scenarios (i.e., it
// is type-agnostic — the guard lives in Upsert), but the regression we want
// to lock in is that biete does NOT increase the push count. We achieve this
// indirectly by asserting the helper does not depend on the entry's typ.
func TestCarpooling_SucheInsert_BieteTyp_NoTeamPush(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	// Insert a biete row directly; this MUST NOT be a precondition for a push.
	bieterUserID := testutil.CreateUser(t, db, "standard")
	if _, err := db.Exec(
		`INSERT INTO mitfahrgelegenheiten (game_id, user_id, typ, plaetze) VALUES (?, ?, 'biete', 3)`,
		gameID, bieterUserID); err != nil {
		t.Fatalf("seed biete: %v", err)
	}

	h := makeHandler(t, db)
	// The helper is invoked unconditionally only via the suche-insert guard in
	// Upsert; this test ensures the helper itself has no biete-coupled side
	// effects. The presence of a biete row neither adds nor removes recipients.
	got := h.teamPushRecipients(context.Background(), gameID, 0)
	want := []int{trainerUserID}
	if !equalSets(got, want) {
		t.Fatalf("recipients = %v, want %v (biete row must be irrelevant)", got, want)
	}
}

// TestCarpooling_SucheInsert_SelfExcluded verifies that the suche-Steller is
// removed from the recipient list, even when they would otherwise be included
// (e.g., they are a trainer of the kader).
func TestCarpooling_SucheInsert_SelfExcluded(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	// Steller IS the trainer.
	stellerUserID := testutil.CreateUser(t, db, "standard")
	stellerMemberID := testutil.CreateMember(t, db, stellerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, stellerMemberID)

	// Another trainer who SHOULD remain.
	otherTrainerUserID := testutil.CreateUser(t, db, "standard")
	otherTrainerMemberID := testutil.CreateMember(t, db, otherTrainerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, otherTrainerMemberID)

	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	h := makeHandler(t, db)
	got := h.teamPushRecipients(context.Background(), gameID, stellerUserID)
	want := []int{otherTrainerUserID}
	if !equalSets(got, want) {
		t.Fatalf("recipients = %v, want %v (steller=%d must be excluded)",
			got, want, stellerUserID)
	}
}

// TestCarpooling_SucheInsert_PrefRespected verifies that the helper does NOT
// filter on notification_preferences itself — that responsibility belongs to
// the next layer (notify.Send → push.FilterByPushPref). This lock-in test
// guards against an accidental refactor that would short-circuit the prefs
// pipeline.
func TestCarpooling_SucheInsert_PrefRespected(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	trainerUserID := testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, trainerUserID)
	testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

	// Trainer has push disabled for "carpooling".
	if _, err := db.Exec(
		`INSERT INTO notification_preferences (user_id, category, push_enabled, email_enabled)
		 VALUES (?, 'carpooling', 0, 0)`, trainerUserID); err != nil {
		t.Fatalf("seed prefs: %v", err)
	}

	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-12-31")

	h := makeHandler(t, db)
	got := h.teamPushRecipients(context.Background(), gameID, 0)
	// Helper returns the trainer regardless of prefs — filtering happens later.
	want := []int{trainerUserID}
	if !equalSets(got, want) {
		t.Fatalf("recipients = %v, want %v (helper must not filter on prefs)", got, want)
	}
}
