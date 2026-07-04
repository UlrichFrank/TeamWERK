//go:build measure

// Package measure is the payload-measurement harness (OpenSpec change
// payload-measurement-harness). It is build-tagged `measure` so it never runs
// in the default `go test ./...` suite — only via `make measure`. This keeps
// the timing-sensitive SSE fan-out measurement out of the blocking gate, as the
// design mandates.
//
// All measurements go through the production router (testutil/prodserver) over
// real HTTP — no internal shortcuts — so the numbers reflect the real client
// experience.
package measure

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// measureRefTime is the fixed reference instant the whole dataset is anchored
// to. NOTHING in the seed uses time.Now(): every date is derived from this
// constant, so two seed runs are byte-identical.
var measureRefTime = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

// Pinned dataset sizes (design.md, "Deterministisches Seeding (verbindlich)").
const (
	seedTeams            = 4
	seedSpielerPerTeam   = 45  // 180 total
	seedChildren         = 8   // 2 per team, each with a parent user
	seedGames            = 100 // 60 past / 40 future
	seedDutyTypes        = 20  // 10 with instruction_md
	seedDutyWithInstr    = 10
	seedInstrBytes       = 3072
	seedDutySlots        = 500
	seedTrainingSessions = 100 // 60 series-bound / 40 standalone
	seedTrainingSeries   = 60
	seedChatMessages     = 100 // 80 short / 15 long / 5 deleted
	seedVideos           = 5

	// Derived member total: 180 spieler + 4 trainer + 2 sL + 4 vorstand-ish
	// + 2 kassierer + 8 children = 200. The 8 SSE fan-out clients (C1..C8) are
	// drawn FROM this population (plus one admin user without a member row), so
	// the roster adds no extra members.
	seedMembersTotal = seedSpielerPerTeam*seedTeams + seedTeams + 2 + 4 + 2 + seedChildren
)

// dayOffset returns a "2006-01-02" date measureRefTime + days.
func dayOffset(days int) string {
	return measureRefTime.AddDate(0, 0, days).Format("2006-01-02")
}

// fanoutClient is one of the fixed C1..C8 SSE clients used to measure the
// per-mutation fan-out. `Cookie` is the plaintext refresh token (the SSE
// endpoint authenticates via the refresh_token cookie, not a Bearer token).
type fanoutClient struct {
	Label     string
	UserID    int
	Role      string
	Functions []string
	Team      string
	Cookie    string
	// Designed audiences (the target state of scoped-live-updates). Used only by
	// the roster-composition assertion; on main every client receives every
	// event (global broadcast → 8/8/8).
	InMembersAudience bool
	InGamesT1Audience bool
}

// measureData holds the IDs/tokens the measurement functions need.
type measureData struct {
	db         *sql.DB
	adminToken string // C1 Bearer, bypasses all club-function tiers
	seasonID   int
	teamT1     int
	gameT1     int
	gameT1Date string
	c5MemberID int
	convID     int
	roster     []fanoutClient
}

// representatives are population members/users captured during seeding so the
// fan-out roster can be drawn from the population (keeping member counts exact).
type representatives struct {
	adminUser     int // admin user, no member row
	vorstandUser  int
	kassiererUser int
	trainerT1User int
	spielerT1User int
	spielerT1Mem  int
	spielerT2User int
	spielerT3User int
	parentT1User  int
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) sql.Result {
	t.Helper()
	res, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("seed exec %q: %v", query, err)
	}
	return res
}

// measureSeed builds the full deterministic dataset and the C1..C8 fan-out
// roster. It uses testutil fixtures + verified direct SQL; no randomness.
func measureSeed(t *testing.T, db *sql.DB) *measureData {
	t.Helper()
	var rep representatives

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	// Two extra inactive seasons so /api/seasons returns 3 rows.
	mustExec(t, db, `INSERT INTO seasons (name, start_date, end_date, is_active) VALUES ('2024/25','2024-09-01','2025-06-30',0)`)
	mustExec(t, db, `INSERT INTO seasons (name, start_date, end_date, is_active) VALUES ('2026/27','2026-09-01','2027-06-30',0)`)

	teams := make([]int, seedTeams)
	kaders := make([]int, seedTeams)
	for i := 0; i < seedTeams; i++ {
		teams[i] = testutil.CreateTeam(t, db, fmt.Sprintf("Team %d", i+1))
		kaders[i] = testutil.CreateKader(t, db, teams[i], seasonID)
	}

	// 180 spieler (45/team): member + function + kader membership. Capture the
	// first spieler of teams 1..3 as fan-out representatives.
	for ti := 0; ti < seedTeams; ti++ {
		for j := 0; j < seedSpielerPerTeam; j++ {
			uid := testutil.CreateUser(t, db, "standard")
			mid := testutil.CreateMember(t, db, uid)
			mustExec(t, db, `INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, mid)
			mustExec(t, db, `INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaders[ti], mid)
			if j == 0 {
				switch ti {
				case 0:
					rep.spielerT1User, rep.spielerT1Mem = uid, mid
				case 1:
					rep.spielerT2User = uid
				case 2:
					rep.spielerT3User = uid
				}
			}
		}
	}

	// 4 trainer (1/team) → member + function + kader_trainers. Capture team 1's.
	for ti := 0; ti < seedTeams; ti++ {
		uid := testutil.CreateUser(t, db, "standard")
		mid := testutil.CreateMember(t, db, uid)
		mustExec(t, db, `INSERT INTO member_club_functions (member_id, function) VALUES (?, 'trainer')`, mid)
		mustExec(t, db, `INSERT INTO kader_trainers (kader_id, member_id) VALUES (?, ?)`, kaders[ti], mid)
		if ti == 0 {
			rep.trainerT1User = uid
		}
	}
	// 2 sportliche_leitung, 3 vorstand, 1 vorstand_beisitzer, 2 kassierer.
	for i := 0; i < 2; i++ {
		insertMemberWithFunction(t, db, "sportliche_leitung")
	}
	for i := 0; i < 3; i++ {
		uid := insertMemberWithFunction(t, db, "vorstand")
		if i == 0 {
			rep.vorstandUser = uid
		}
	}
	insertMemberWithFunction(t, db, "vorstand_beisitzer")
	for i := 0; i < 2; i++ {
		uid := insertMemberWithFunction(t, db, "kassierer")
		if i == 0 {
			rep.kassiererUser = uid
		}
	}

	// 8 children (2/team) each with a parent user via family_links. Capture the
	// parent of the first T1 child.
	for i := 0; i < seedChildren; i++ {
		childUID := testutil.CreateUser(t, db, "standard")
		childMID := testutil.CreateMember(t, db, childUID)
		mustExec(t, db, `INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, childMID)
		mustExec(t, db, `INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaders[i%seedTeams], childMID)
		parentUID := testutil.CreateUser(t, db, "standard")
		mustExec(t, db, `INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentUID, childMID)
		if i == 0 {
			rep.parentT1User = parentUID
		}
	}

	// 100 games: 60 past, 40 future; teams round-robin. Capture a future T1 game.
	var gameT1 int
	var gameT1Date string
	for i := 0; i < seedGames; i++ {
		var day int
		if i < 60 {
			day = -(i + 1)
		} else {
			day = i - 60 + 1
		}
		team := teams[i%seedTeams]
		date := dayOffset(day)
		gid := testutil.CreateGame(t, db, seasonID, team, date)
		if gameT1 == 0 && team == teams[0] && day > 0 {
			gameT1, gameT1Date = gid, date
		}
	}

	// 20 duty types, first 10 with a fixed 3072-byte instruction.
	instr := strings.Repeat("A", seedInstrBytes)
	dutyTypes := make([]int, seedDutyTypes)
	for i := 0; i < seedDutyTypes; i++ {
		dutyTypes[i] = testutil.CreateDutyType(t, db, fmt.Sprintf("Dienst %d", i+1), 1.0)
		if i < seedDutyWithInstr {
			testutil.SetDutyInstruction(t, db, dutyTypes[i], instr)
		}
	}

	// 500 duty slots across duty types / teams.
	for i := 0; i < seedDutySlots; i++ {
		testutil.CreateDutySlot(t, db, dutyTypes[i%seedDutyTypes], seasonID, teams[i%seedTeams], 0, dayOffset(i%40-20))
	}

	// 100 training sessions: 60 series-bound, 40 standalone.
	seriesID := testutil.CreateTrainingSeries(t, db, teams[0], seasonID, rep.adminUserOrCreate(t, db))
	for i := 0; i < seedTrainingSessions; i++ {
		var seriesArg any
		if i < seedTrainingSeries {
			seriesArg = seriesID
		}
		mustExec(t, db,
			`INSERT INTO training_sessions (series_id, team_id, season_id, date, start_time, end_time, title)
			 VALUES (?, ?, ?, ?, '18:00', '20:00', 'Training')`,
			seriesArg, teams[i%seedTeams], seasonID, dayOffset(i%50-25))
	}

	// A few venues so /api/venues has content.
	for i := 0; i < 4; i++ {
		mustExec(t, db, `INSERT INTO venues (name, street, city, postal_code) VALUES (?, 'Str 1', 'Stuttgart', '70000')`,
			fmt.Sprintf("Halle %d", i+1))
	}

	// A few videos.
	for i := 0; i < seedVideos; i++ {
		testutil.CreateVideo(t, db, teams[0], seasonID, rep.adminUser, "ready")
	}

	roster, adminToken := buildFanoutRoster(t, db, rep)

	// Chat conversation the admin (C1) is a member of, with 100 messages.
	convID := seedConversation(t, db, rep.adminUser)

	return &measureData{
		db:         db,
		adminToken: adminToken,
		seasonID:   seasonID,
		teamT1:     teams[0],
		gameT1:     gameT1,
		gameT1Date: gameT1Date,
		c5MemberID: rep.spielerT1Mem,
		convID:     convID,
		roster:     roster,
	}
}

// insertMemberWithFunction creates a user+member and grants a club function,
// returning the user ID.
func insertMemberWithFunction(t *testing.T, db *sql.DB, function string) int {
	t.Helper()
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	mustExec(t, db, `INSERT INTO member_club_functions (member_id, function) VALUES (?, ?)`, memberID, function)
	return userID
}

// adminUserOrCreate lazily creates the single admin user (no member row) used as
// C1 and as the created_by for series/videos.
func (r *representatives) adminUserOrCreate(t *testing.T, db *sql.DB) int {
	t.Helper()
	if r.adminUser == 0 {
		r.adminUser = testutil.CreateUser(t, db, "admin")
	}
	return r.adminUser
}

// buildFanoutRoster wraps the captured population representatives as the fixed
// C1..C8 roster (design.md table) and returns it plus the admin (C1) Bearer
// token. No new members are created here.
func buildFanoutRoster(t *testing.T, db *sql.DB, rep representatives) (roster []fanoutClient, adminToken string) {
	t.Helper()
	rep.adminUserOrCreate(t, db)

	mk := func(label, role string, functions []string, team string, uid int, inMembers, inGames bool) fanoutClient {
		cookie := testutil.CreateRefreshToken(t, db, uid)
		return fanoutClient{
			Label: label, UserID: uid, Role: role, Functions: functions, Team: team,
			Cookie: cookie, InMembersAudience: inMembers, InGamesT1Audience: inGames,
		}
	}

	roster = []fanoutClient{
		mk("C1", "admin", nil, "", rep.adminUser, true, true),
		mk("C2", "standard", []string{"vorstand"}, "", rep.vorstandUser, true, true),
		mk("C3", "standard", []string{"kassierer"}, "", rep.kassiererUser, true, false),
		mk("C4", "standard", []string{"trainer"}, "T1", rep.trainerT1User, false, true),
		mk("C5", "standard", []string{"spieler"}, "T1", rep.spielerT1User, false, true),
		mk("C6", "standard", []string{"spieler"}, "T2", rep.spielerT2User, false, false),
		mk("C7", "standard", []string{"spieler"}, "T3", rep.spielerT3User, false, false),
		mk("C8", "standard", nil, "T1", rep.parentT1User, false, true),
	}
	adminToken = testutil.Token(t, rep.adminUser, "admin", nil)
	return roster, adminToken
}

func seedConversation(t *testing.T, db *sql.DB, ownerUserID int) int {
	t.Helper()
	res := mustExec(t, db, `INSERT INTO conversations (type, name, created_by) VALUES ('group', 'Team-Chat', ?)`, ownerUserID)
	convID64, _ := res.LastInsertId()
	convID := int(convID64)
	mustExec(t, db, `INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, ownerUserID)
	longBody := strings.Repeat("m", 1800)
	for i := 0; i < seedChatMessages; i++ {
		body := "hi"
		if i >= 80 && i < 95 {
			body = longBody
		}
		res := mustExec(t, db, `INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`, convID, ownerUserID, body)
		if i >= 95 {
			mid, _ := res.LastInsertId()
			mustExec(t, db, `UPDATE messages SET deleted_at=? WHERE id=?`, measureRefTime, mid)
		}
	}
	return convID
}

// startServer builds the full production router over the seeded DB.
func startServer(t *testing.T, data *measureData) string {
	t.Helper()
	srv := prodserver.New(t, data.db)
	return srv.URL
}

func TestMeasure_SeedIsDeterministic(t *testing.T) {
	count := func(db *sql.DB, table string) int {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}

	db1 := testutil.NewDB(t)
	measureSeed(t, db1)
	db2 := testutil.NewDB(t)
	measureSeed(t, db2)

	for _, tc := range []struct {
		table string
		want  int
	}{
		{"members", seedMembersTotal},
		{"games", seedGames},
		{"duty_slots", seedDutySlots},
		{"duty_types", seedDutyTypes},
		{"training_sessions", seedTrainingSessions},
		{"seasons", 3},
		{"messages", seedChatMessages},
	} {
		g1, g2 := count(db1, tc.table), count(db2, tc.table)
		if g1 != g2 {
			t.Errorf("%s: non-deterministic seed: run1=%d run2=%d", tc.table, g1, g2)
		}
		if g1 != tc.want {
			t.Errorf("%s: got %d, want pinned %d", tc.table, g1, tc.want)
		}
	}
}
