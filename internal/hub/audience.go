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
