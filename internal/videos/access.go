package videos

import (
	"github.com/teamstuttgart/teamwerk/internal/auth"
)

// Video ist die für Berechtigungsprüfungen relevante Teilmenge einer videos-Zeile.
type Video struct {
	ID     int
	TeamID int
}

// CanUploadToTeam meldet, ob der Aufrufer ein Video in das gegebene Team hochladen
// darf. Erlaubt sind:
//   - admin (System-Rolle)
//   - vorstand, sportliche_leitung (Vereinsfunktion) — teamübergreifend
//   - trainer NUR für ein Team, in dem er Trainer der laufenden Saison ist
//
// (siehe design.md "Upload-Berechtigung").
func (h *Handler) CanUploadToTeam(claims *auth.Claims, teamID int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if claims.Role == "admin" || claims.HasAnyFunction("vorstand", "sportliche_leitung") {
		return true, nil
	}
	if claims.HasFunction("trainer") {
		return h.isTrainerOfTeam(claims.UserID, teamID)
	}
	return false, nil
}

// CanManageTeamVideos meldet, ob der Aufrufer Videos eines Teams ändern/löschen
// darf: jeder Trainer des Teams, Vorstand, Admin (Trainer vertreten sich
// gegenseitig — nicht nur der ursprüngliche Hochlader; siehe design.md
// "Lösch-Berechtigung").
func (h *Handler) CanManageTeamVideos(claims *auth.Claims, teamID int) (bool, error) {
	if claims == nil {
		return false, nil
	}
	if claims.Role == "admin" || claims.HasFunction("vorstand") {
		return true, nil
	}
	if claims.HasFunction("trainer") {
		return h.isTrainerOfTeam(claims.UserID, teamID)
	}
	return false, nil
}

// CanViewVideo meldet, ob der Aufrufer ein Video ansehen darf. Sichtbar sind
// Videos eines Teams nur für (siehe design.md "Strenge Berechtigung"):
//   - Vorstand und Admin (immer)
//   - aktive Spieler des Teams
//   - Trainer des Teams
//   - Eltern aktiver Spieler des Teams
func (h *Handler) CanViewVideo(claims *auth.Claims, video *Video) (bool, error) {
	if claims == nil || video == nil {
		return false, nil
	}
	if claims.Role == "admin" || claims.HasFunction("vorstand") {
		return true, nil
	}
	return h.userBelongsToTeam(claims, video.TeamID)
}

// isTrainerOfTeam meldet, ob der Nutzer in einem Kader des Teams (in einer
// aktiven Saison) als Trainer eingetragen ist.
func (h *Handler) isTrainerOfTeam(userID, teamID int) (bool, error) {
	var n int
	err := h.db.QueryRow(`
		SELECT COUNT(*) FROM trainer_memberships tm
		JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
		JOIN members m ON m.id = tm.member_id
		WHERE tm.team_id = ? AND m.user_id = ?`,
		teamID, userID).Scan(&n)
	return n > 0, err
}

// userBelongsToTeam meldet, ob der Nutzer als aktiver Spieler, Trainer oder als
// Elternteil eines aktiven Spielers zum Team (in einer aktiven Saison) gehört.
func (h *Handler) userBelongsToTeam(claims *auth.Claims, teamID int) (bool, error) {
	var n int
	err := h.db.QueryRow(`
		SELECT COUNT(*) FROM (
			-- aktiver Spieler des Teams
			SELECT 1 FROM player_memberships pm
			JOIN seasons s ON s.id = pm.season_id AND s.is_active = 1
			JOIN members m ON m.id = pm.member_id AND m.status = 'aktiv'
			WHERE pm.team_id = ? AND m.user_id = ?
			UNION
			-- Trainer des Teams
			SELECT 1 FROM trainer_memberships tm
			JOIN seasons s ON s.id = tm.season_id AND s.is_active = 1
			JOIN members m ON m.id = tm.member_id
			WHERE tm.team_id = ? AND m.user_id = ?
			UNION
			-- Elternteil eines aktiven Spielers des Teams
			SELECT 1 FROM family_links fl
			JOIN members m ON m.id = fl.member_id AND m.status = 'aktiv'
			JOIN player_memberships pm ON pm.member_id = m.id
			JOIN seasons s ON s.id = pm.season_id AND s.is_active = 1
			WHERE pm.team_id = ? AND fl.parent_user_id = ?
		)`,
		teamID, claims.UserID,
		teamID, claims.UserID,
		teamID, claims.UserID).Scan(&n)
	return n > 0, err
}
