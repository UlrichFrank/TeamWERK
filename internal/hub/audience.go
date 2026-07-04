package hub

import (
	"context"
	"database/sql"
	"strings"
)

// Audience resolves the set of user IDs that should receive a scoped domain
// event. It lives in the hub (FOUNDATION) package and works purely through
// generic SQL joins over shared tables (users, members, member_club_functions,
// team_memberships) so it introduces no domain↔domain import — the domain
// handlers pass in the IDs they already hold (userID, teamIDs) and receive back
// a []int for BroadcastToUsers.
//
// It never widens data visibility: the returned IDs only decide who is asked to
// reload; the read routes stay authoritative. When a team cannot be resolved,
// callers fall back to the global Broadcast (see design.md "Konservativ scopen").
type Audience struct {
	db *sql.DB
}

// NewAudience wires an Audience resolver to the shared database handle.
func NewAudience(db *sql.DB) *Audience {
	return &Audience{db: db}
}

// financeFunctions are the Vereinsfunktionen that may read member/user data.
var financeFunctions = []string{"vorstand", "vorstand_beisitzer", "kassierer"}

// FinanceGroup returns the user IDs of the finance group: every admin plus every
// user linked to a member with a vorstand/vorstand_beisitzer/kassierer function.
// extraUserIDs (e.g. the affected user's own ID on a self-profile change) are
// appended; 0 entries are ignored. Used for the members/users topics.
func (a *Audience) FinanceGroup(ctx context.Context, extraUserIDs ...int) []int {
	set := newIDSet()
	a.collectAdmins(ctx, set)
	a.collectByFunctions(ctx, set, financeFunctions)
	for _, uid := range extraUserIDs {
		if uid > 0 {
			set.add(uid)
		}
	}
	return set.slice()
}

// Team returns the user IDs that should be notified about an event bound to the
// given teams: members (players) and trainers of those teams (via the
// team_memberships view), parents linked to those members, plus the club-wide
// staff who oversee every team (sportliche_leitung + finance group + admins).
// An empty teamIDs slice yields only the staff/finance audience — callers with
// no resolvable team should prefer the global Broadcast instead.
func (a *Audience) Team(ctx context.Context, teamIDs []int, extraUserIDs ...int) []int {
	set := newIDSet()
	// Club-wide staff always see team events (they manage across teams).
	a.collectAdmins(ctx, set)
	a.collectByFunctions(ctx, set, []string{"vorstand", "vorstand_beisitzer", "sportliche_leitung"})

	if len(teamIDs) > 0 {
		a.collectTeamMembers(ctx, set, teamIDs)
		a.collectTeamParents(ctx, set, teamIDs)
	}

	for _, uid := range extraUserIDs {
		if uid > 0 {
			set.add(uid)
		}
	}
	return set.slice()
}

// TeamIDsForGame returns the team IDs linked to a game via game_teams.
func (a *Audience) TeamIDsForGame(ctx context.Context, gameID int) []int {
	return a.teamIDs(ctx, `SELECT team_id FROM game_teams WHERE game_id = ?`, gameID)
}

// TeamIDsForTraining returns the team ID of a training session (single team per
// session). Returned as a slice for symmetry with the games path.
func (a *Audience) TeamIDsForTraining(ctx context.Context, trainingID int) []int {
	return a.teamIDs(ctx, `SELECT team_id FROM training_sessions WHERE id = ?`, trainingID)
}

// MembersAudience resolves the audience for events bound to one or more members
// (e.g. absences and their cascaded training/game RSVP auto-declines): the
// teams those members belong to (players + trainers + parents + club-wide
// staff), plus each member's own linked user and their parents explicitly (so a
// member not currently in any kader is still covered). Empty memberIDs yields
// only the club-wide staff.
func (a *Audience) MembersAudience(ctx context.Context, memberIDs []int, extraUserIDs ...int) []int {
	if len(memberIDs) == 0 {
		return a.Team(ctx, nil, extraUserIDs...)
	}
	ids := a.Team(ctx, a.teamIDsForMembers(ctx, memberIDs), extraUserIDs...)
	set := newIDSet()
	for _, id := range ids {
		set.add(id)
	}
	a.collectMemberOwners(ctx, set, memberIDs)
	a.collectMemberParents(ctx, set, memberIDs)
	return set.slice()
}

// teamIDsForMembers returns the distinct team IDs the given members belong to.
func (a *Audience) teamIDsForMembers(ctx context.Context, memberIDs []int) []int {
	if len(memberIDs) == 0 {
		return nil
	}
	args := make([]any, len(memberIDs))
	for i, id := range memberIDs {
		args[i] = id
	}
	return a.teamIDs(ctx,
		`SELECT DISTINCT team_id FROM team_memberships WHERE member_id IN (`+placeholders(len(memberIDs))+`)`,
		args...)
}

// collectMemberOwners adds the user_id of each given member.
func (a *Audience) collectMemberOwners(ctx context.Context, set *idSet, memberIDs []int) {
	args := make([]any, len(memberIDs))
	for i, id := range memberIDs {
		args[i] = id
	}
	rows, err := a.db.QueryContext(ctx,
		`SELECT user_id FROM members WHERE user_id IS NOT NULL AND id IN (`+placeholders(len(memberIDs))+`)`,
		args...)
	if err != nil {
		return
	}
	scanIDs(rows, set)
}

// collectMemberParents adds the parent user IDs linked to the given members.
func (a *Audience) collectMemberParents(ctx context.Context, set *idSet, memberIDs []int) {
	args := make([]any, len(memberIDs))
	for i, id := range memberIDs {
		args[i] = id
	}
	rows, err := a.db.QueryContext(ctx,
		`SELECT parent_user_id FROM family_links WHERE member_id IN (`+placeholders(len(memberIDs))+`)`,
		args...)
	if err != nil {
		return
	}
	scanIDs(rows, set)
}

// TeamIDsForDutySlot returns the team IDs a duty slot belongs to: its own
// team_id if set, otherwise the teams of its linked game (game_teams). This
// mirrors the /api/duty-board team filter (slot's team, or its game's teams).
// Empty result → slot has neither a team nor a game (fall back to global).
func (a *Audience) TeamIDsForDutySlot(ctx context.Context, slotID any) []int {
	return a.teamIDs(ctx, `
		SELECT team_id FROM duty_slots WHERE id = ? AND team_id IS NOT NULL
		UNION
		SELECT gt.team_id
		FROM duty_slots ds
		JOIN game_teams gt ON gt.game_id = ds.game_id
		WHERE ds.id = ? AND ds.team_id IS NULL`, slotID, slotID)
}

// TeamIDsForKader returns the team ID of a kader (may be empty if the kader has
// no team assigned yet — team_id is nullable). kaderID accepts int or the raw
// string path value (SQLite coerces the bind parameter).
func (a *Audience) TeamIDsForKader(ctx context.Context, kaderID any) []int {
	return a.teamIDs(ctx, `SELECT team_id FROM kader WHERE id = ? AND team_id IS NOT NULL`, kaderID)
}

// TeamIDsForTrainingSeries returns the team ID of a training series.
func (a *Audience) TeamIDsForTrainingSeries(ctx context.Context, seriesID int) []int {
	return a.teamIDs(ctx, `SELECT team_id FROM training_series WHERE id = ?`, seriesID)
}

// GameAudience resolves the team audience for a game (its teams' players +
// trainers + parents, plus club-wide staff). If the game has no resolvable
// teams, the audience is only the club-wide staff — callers should then prefer
// the global Broadcast (see design.md "Konservativ scopen").
func (a *Audience) GameAudience(ctx context.Context, gameID int, extraUserIDs ...int) []int {
	return a.Team(ctx, a.TeamIDsForGame(ctx, gameID), extraUserIDs...)
}

// TrainingAudience resolves the team audience for a training session.
func (a *Audience) TrainingAudience(ctx context.Context, trainingID int, extraUserIDs ...int) []int {
	return a.Team(ctx, a.TeamIDsForTraining(ctx, trainingID), extraUserIDs...)
}

// teamIDs runs a single-int-column query and returns the distinct positive IDs.
func (a *Audience) teamIDs(ctx context.Context, query string, args ...any) []int {
	rows, err := a.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	set := newIDSet()
	scanIDs(rows, set)
	return set.slice()
}

// collectAdmins adds every admin user ID to set.
func (a *Audience) collectAdmins(ctx context.Context, set *idSet) {
	rows, err := a.db.QueryContext(ctx, `SELECT id FROM users WHERE role = 'admin'`)
	if err != nil {
		return
	}
	scanIDs(rows, set)
}

// collectByFunctions adds the user IDs of members holding any of the given
// Vereinsfunktionen (joined members→users).
func (a *Audience) collectByFunctions(ctx context.Context, set *idSet, functions []string) {
	if len(functions) == 0 {
		return
	}
	args := make([]any, len(functions))
	for i, f := range functions {
		args[i] = f
	}
	q := `SELECT DISTINCT m.user_id
	      FROM member_club_functions mcf
	      JOIN members m ON m.id = mcf.member_id
	      WHERE m.user_id IS NOT NULL
	        AND mcf.function IN (` + placeholders(len(functions)) + `)`
	rows, err := a.db.QueryContext(ctx, q, args...)
	if err != nil {
		return
	}
	scanIDs(rows, set)
}

// collectTeamMembers adds the user IDs of members (players + trainers) belonging
// to any of the given teams, via the team_memberships view.
func (a *Audience) collectTeamMembers(ctx context.Context, set *idSet, teamIDs []int) {
	args := make([]any, len(teamIDs))
	for i, id := range teamIDs {
		args[i] = id
	}
	q := `SELECT DISTINCT m.user_id
	      FROM team_memberships tm
	      JOIN members m ON m.id = tm.member_id
	      WHERE m.user_id IS NOT NULL
	        AND tm.team_id IN (` + placeholders(len(teamIDs)) + `)`
	rows, err := a.db.QueryContext(ctx, q, args...)
	if err != nil {
		return
	}
	scanIDs(rows, set)
}

// collectTeamParents adds the user IDs of parents linked (family_links) to
// members who belong to any of the given teams.
func (a *Audience) collectTeamParents(ctx context.Context, set *idSet, teamIDs []int) {
	args := make([]any, len(teamIDs))
	for i, id := range teamIDs {
		args[i] = id
	}
	q := `SELECT DISTINCT fl.parent_user_id
	      FROM family_links fl
	      JOIN team_memberships tm ON tm.member_id = fl.member_id
	      WHERE tm.team_id IN (` + placeholders(len(teamIDs)) + `)`
	rows, err := a.db.QueryContext(ctx, q, args...)
	if err != nil {
		return
	}
	scanIDs(rows, set)
}

// scanIDs drains rows of a single INTEGER column into set, skipping NULL/0, and
// closes rows.
func scanIDs(rows *sql.Rows, set *idSet) {
	defer rows.Close()
	for rows.Next() {
		var id sql.NullInt64
		if err := rows.Scan(&id); err != nil {
			return
		}
		if id.Valid && id.Int64 > 0 {
			set.add(int(id.Int64))
		}
	}
}

// placeholders returns "?,?,...,?" with n placeholders for an IN clause.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// idSet is an insertion-order-preserving set of int IDs.
type idSet struct {
	seen  map[int]struct{}
	order []int
}

func newIDSet() *idSet { return &idSet{seen: map[int]struct{}{}} }

func (s *idSet) add(id int) {
	if _, ok := s.seen[id]; ok {
		return
	}
	s.seen[id] = struct{}{}
	s.order = append(s.order, id)
}

func (s *idSet) slice() []int { return s.order }
