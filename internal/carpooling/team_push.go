package carpooling

import (
	"context"
	"fmt"
	"strings"
)

// qualifyingTeamsForNextGame returns the team IDs (from game_teams for gameID)
// for which gameID is the team's earliest upcoming game (g.date >= today).
// A team contributes recipients to the team-push only if it qualifies here.
func (h *Handler) qualifyingTeamsForNextGame(ctx context.Context, gameID int) []int {
	rows, err := h.db.QueryContext(ctx, `
		SELECT gt.team_id
		FROM game_teams gt
		WHERE gt.game_id = ?
		  AND gt.game_id = (
		      SELECT g2.id
		      FROM games g2
		      JOIN game_teams gt2 ON gt2.game_id = g2.id
		      WHERE gt2.team_id = gt.team_id
		        AND g2.date >= date('now')
		      ORDER BY g2.date, g2.time, g2.id
		      LIMIT 1
		  )`, gameID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// kaderRecipients returns the DISTINCT user IDs to notify for a team-push,
// computed as the union of:
//   - parents (family_links.parent_user_id) of members in kader_members
//     ∪ kader_extended_members for any kader matching (team_id IN teamIDs,
//     season_id = seasonID),
//   - users (members.user_id) behind kader_trainers for the same kaders.
//
// excludeUserID (the suche-Steller) is filtered out. NULL user_ids (members
// without a user account) are dropped.
func (h *Handler) kaderRecipients(ctx context.Context, teamIDs []int, seasonID, excludeUserID int) []int {
	if len(teamIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(teamIDs))
	for i := range teamIDs {
		placeholders[i] = "?"
	}
	in := strings.Join(placeholders, ",")

	query := fmt.Sprintf(`
		WITH target_kader AS (
		    SELECT id FROM kader WHERE season_id = ? AND team_id IN (%s)
		)
		SELECT DISTINCT user_id FROM (
		    SELECT fl.parent_user_id AS user_id
		    FROM family_links fl
		    JOIN kader_members km ON km.member_id = fl.member_id
		    WHERE km.kader_id IN (SELECT id FROM target_kader)
		    UNION
		    SELECT fl.parent_user_id
		    FROM family_links fl
		    JOIN kader_extended_members kem ON kem.member_id = fl.member_id
		    WHERE kem.kader_id IN (SELECT id FROM target_kader)
		    UNION
		    SELECT m.user_id
		    FROM kader_trainers kt
		    JOIN members m ON m.id = kt.member_id
		    WHERE kt.kader_id IN (SELECT id FROM target_kader)
		      AND m.user_id IS NOT NULL
		)
		WHERE user_id IS NOT NULL AND user_id != ?`, in)

	args := make([]any, 0, len(teamIDs)+2)
	args = append(args, seasonID)
	for _, id := range teamIDs {
		args = append(args, id)
	}
	args = append(args, excludeUserID)

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// teamPushRecipients resolves the recipients for a team-push triggered by a new
// "suche" entry for gameID. Returns nil (silent skip) when:
//   - gameID has no season (defensive),
//   - no team for which gameID is the next upcoming game,
//   - no kader nominated for any qualifying (team, season) pair,
//   - the only recipient candidate is the sender themselves.
func (h *Handler) teamPushRecipients(ctx context.Context, gameID, senderID int) []int {
	var seasonID int
	if err := h.db.QueryRowContext(ctx,
		`SELECT season_id FROM games WHERE id = ?`, gameID).Scan(&seasonID); err != nil {
		return nil
	}
	teamIDs := h.qualifyingTeamsForNextGame(ctx, gameID)
	if len(teamIDs) == 0 {
		return nil
	}
	return h.kaderRecipients(ctx, teamIDs, seasonID, senderID)
}
