package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/db"
)

type TeamGroup struct {
	TeamID       int    `json:"teamId"`
	DisplayShort string `json:"displayShort"`
	Kind         string `json:"kind"`
	Count        int    `json:"count"`
}

type TeamGroupMember struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var teamGroupKinds = map[string]bool{
	"trainer":      true,
	"spieler":      true,
	"eltern":       true,
	"alle_trainer": true,
}

// allTrainersMemberQuery liefert DISTINCT (user_id, name) für die Mitgliedermenge
// der "Alle Trainer"-Gruppe (= T): alle Trainer aller Kader der aktiven Saison.
// Identisch zu teamGroupMemberQuery("trainer"), nur ohne den team_id-Filter.
// Nimmt keine Platzhalter.
func allTrainersMemberQuery() string {
	return `
		SELECT DISTINCT m.user_id AS user_id,
		       u.first_name || ' ' || u.last_name AS name
		FROM kader_trainers kt
		JOIN kader k ON k.id = kt.kader_id
		JOIN seasons s ON s.id = k.season_id
		JOIN members m ON m.id = kt.member_id
		JOIN users u ON u.id = m.user_id
		WHERE s.is_active = 1 AND m.user_id IS NOT NULL`
}

// trainerCircleMemberQuery liefert DISTINCT (user_id, name) für den Zugriffskreis
// (= Z): Kader-Trainer der aktiven Saison ∪ vorstand ∪ sportliche_leitung ∪
// vorstand_beisitzer. Nur für die Nutzersuche-Erweiterung, nicht für die
// Gruppenauflösung. Nimmt keine Platzhalter.
func trainerCircleMemberQuery() string {
	return `
		SELECT DISTINCT user_id, name FROM (
			` + allTrainersMemberQuery() + `
			UNION
			SELECT m.user_id AS user_id,
			       u.first_name || ' ' || u.last_name AS name
			FROM member_club_functions mcf
			JOIN members m ON m.id = mcf.member_id
			JOIN users u ON u.id = m.user_id
			WHERE mcf.function IN ('vorstand', 'sportliche_leitung', 'vorstand_beisitzer')
			  AND m.user_id IS NOT NULL
		)`
}

// isInTrainerCircle prüft die Zugehörigkeit zum Zugriffskreis Z:
// (a) Kader-Trainer der aktiven Saison ODER (b) vorstand ODER
// (c) sportliche_leitung ODER (d) vorstand_beisitzer.
func (h *Handler) isInTrainerCircle(ctx context.Context, userID int) (bool, error) {
	var exists bool
	err := h.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM kader_trainers kt
			JOIN kader k ON k.id = kt.kader_id
			JOIN seasons s ON s.id = k.season_id
			JOIN members m ON m.id = kt.member_id
			WHERE s.is_active = 1 AND m.user_id = ?
			UNION ALL
			SELECT 1 FROM member_club_functions mcf
			JOIN members m ON m.id = mcf.member_id
			WHERE m.user_id = ?
			  AND mcf.function IN ('vorstand', 'sportliche_leitung', 'vorstand_beisitzer')
		)`, userID, userID).Scan(&exists)
	return exists, err
}

// callerInTrainerCircle prüft die Zugriffskreis-Zugehörigkeit des Callers.
// Vereinsfunktionen werden — wie im übrigen Chat-Code — aus den JWT-Claims
// gelesen; die Kader-Trainer-Zugehörigkeit (nicht in der Claim) aus der DB.
// admin ist stets berechtigt.
func (h *Handler) callerInTrainerCircle(ctx context.Context, claims *auth.Claims) (bool, error) {
	if claims.Role == "admin" ||
		claims.HasAnyFunction("vorstand", "sportliche_leitung", "vorstand_beisitzer") {
		return true, nil
	}
	return h.isInTrainerCircle(ctx, claims.UserID)
}

// hasGlobalTeamGroupAccess returns true if the caller may see every team's
// standard groups regardless of personal team membership.
func hasGlobalTeamGroupAccess(claims *auth.Claims) bool {
	if claims == nil {
		return false
	}
	return claims.Role == "admin" || claims.HasFunction("vorstand") || claims.HasFunction("sportliche_leitung")
}

// canSeeTeamGroup checks whether the caller may see/resolve standard groups
// for the given team. Verified against the active season.
func (h *Handler) canSeeTeamGroup(r *http.Request, claims *auth.Claims, teamID int) (bool, error) {
	if hasGlobalTeamGroupAccess(claims) {
		var exists int
		err := h.db.QueryRowContext(r.Context(), `
			SELECT 1 FROM teams t
			JOIN kader k ON k.team_id = t.id
			JOIN seasons s ON s.id = k.season_id
			WHERE t.id = ? AND s.is_active = 1
			LIMIT 1`, teamID).Scan(&exists)
		if err == sql.ErrNoRows {
			return false, nil
		}
		return err == nil, err
	}
	var count int
	err := h.db.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM user_accessible_teams uat
		JOIN seasons s ON s.id = uat.season_id
		WHERE uat.user_id = ? AND uat.team_id = ? AND s.is_active = 1`,
		claims.UserID, teamID).Scan(&count)
	return count > 0, err
}

// GET /api/chat/team-groups
func (h *Handler) ListTeamGroups(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())

	// display_short = kanonische Team-Kurzform; die Team-Nummer wird saisonweit
	// disambiguiert (unabhängig von der Sichtbarkeit des Callers).
	displayShort := db.TeamDisplayShort("t")

	var teamRows *sql.Rows
	var err error
	if hasGlobalTeamGroupAccess(claims) {
		teamRows, err = h.db.QueryContext(r.Context(), `
			SELECT DISTINCT t.id, COALESCE(`+displayShort+`, t.name)
			FROM teams t
			JOIN kader k ON k.team_id = t.id
			JOIN seasons s ON s.id = k.season_id
			WHERE s.is_active = 1
			ORDER BY t.age_class, t.gender, k.team_number`)
	} else {
		teamRows, err = h.db.QueryContext(r.Context(), `
			SELECT DISTINCT t.id, COALESCE(`+displayShort+`, t.name)
			FROM user_accessible_teams uat
			JOIN teams t ON t.id = uat.team_id
			JOIN kader k ON k.team_id = t.id AND k.season_id = uat.season_id
			JOIN seasons s ON s.id = uat.season_id
			WHERE uat.user_id = ? AND s.is_active = 1
			ORDER BY t.age_class, t.gender, k.team_number`, claims.UserID)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer teamRows.Close()

	type teamInfo struct {
		id           int
		displayShort string
	}
	var teams []teamInfo
	for teamRows.Next() {
		var t teamInfo
		if err := teamRows.Scan(&t.id, &t.displayShort); err != nil {
			continue
		}
		teams = append(teams, t)
	}
	teamRows.Close()

	results := []TeamGroup{}

	// "Alle Trainer": synthetische, teamübergreifende Kachel für den
	// Zugriffskreis (Trainer/Vorstand/sL/Beisitzer) + admin. Vorangestellt.
	eligible, _ := h.callerInTrainerCircle(r.Context(), claims)
	if eligible {
		if count, err := h.countTeamGroupMembers(r, 0, "alle_trainer", claims.UserID); err == nil && count > 0 {
			results = append(results, TeamGroup{
				TeamID:       0,
				DisplayShort: "Alle Trainer",
				Kind:         "alle_trainer",
				Count:        count,
			})
		}
	}

	for _, t := range teams {
		for _, kind := range []string{"trainer", "spieler", "eltern"} {
			count, err := h.countTeamGroupMembers(r, t.id, kind, claims.UserID)
			if err != nil || count == 0 {
				continue
			}
			results = append(results, TeamGroup{
				TeamID:       t.id,
				DisplayShort: t.displayShort,
				Kind:         kind,
				Count:        count,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *Handler) countTeamGroupMembers(r *http.Request, teamID int, kind string, excludeUserID int) (int, error) {
	if kind == "alle_trainer" {
		var count int
		err := h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM (`+allTrainersMemberQuery()+`) WHERE user_id != ?`,
			excludeUserID).Scan(&count)
		return count, err
	}
	q := teamGroupMemberQuery(kind)
	if q == "" {
		return 0, nil
	}
	var count int
	var err error
	if kind == "trainer" {
		err = h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM (`+q+`) WHERE user_id != ?`,
			teamID, excludeUserID).Scan(&count)
	} else {
		err = h.db.QueryRowContext(r.Context(),
			`SELECT COUNT(*) FROM (`+q+`) WHERE user_id != ?`,
			teamID, teamID, excludeUserID).Scan(&count)
	}
	return count, err
}

// teamGroupMemberQuery returns a SQL fragment that yields DISTINCT (user_id, name)
// rows for the given kind. The first placeholder must be team_id.
func teamGroupMemberQuery(kind string) string {
	switch kind {
	case "trainer":
		return `
			SELECT DISTINCT m.user_id AS user_id,
			       u.first_name || ' ' || u.last_name AS name
			FROM kader_trainers kt
			JOIN kader k ON k.id = kt.kader_id
			JOIN seasons s ON s.id = k.season_id
			JOIN members m ON m.id = kt.member_id
			JOIN users u ON u.id = m.user_id
			WHERE k.team_id = ? AND s.is_active = 1 AND m.user_id IS NOT NULL`
	case "spieler":
		return `
			SELECT DISTINCT user_id, name FROM (
				SELECT m.user_id AS user_id,
				       u.first_name || ' ' || u.last_name AS name
				FROM kader_members km
				JOIN kader k ON k.id = km.kader_id
				JOIN seasons s ON s.id = k.season_id
				JOIN members m ON m.id = km.member_id
				JOIN users u ON u.id = m.user_id
				WHERE k.team_id = ? AND s.is_active = 1 AND m.user_id IS NOT NULL
				UNION ALL
				SELECT m.user_id AS user_id,
				       u.first_name || ' ' || u.last_name AS name
				FROM kader_extended_members kem
				JOIN kader k ON k.id = kem.kader_id
				JOIN seasons s ON s.id = k.season_id
				JOIN members m ON m.id = kem.member_id
				JOIN users u ON u.id = m.user_id
				WHERE k.team_id = ? AND s.is_active = 1 AND m.user_id IS NOT NULL
			)`
	case "eltern":
		return `
			SELECT DISTINCT user_id, name FROM (
				SELECT fl.parent_user_id AS user_id,
				       u.first_name || ' ' || u.last_name AS name
				FROM family_links fl
				JOIN kader_members km ON km.member_id = fl.member_id
				JOIN kader k ON k.id = km.kader_id
				JOIN seasons s ON s.id = k.season_id
				JOIN users u ON u.id = fl.parent_user_id
				WHERE k.team_id = ? AND s.is_active = 1
				UNION ALL
				SELECT fl.parent_user_id AS user_id,
				       u.first_name || ' ' || u.last_name AS name
				FROM family_links fl
				JOIN kader_extended_members kem ON kem.member_id = fl.member_id
				JOIN kader k ON k.id = kem.kader_id
				JOIN seasons s ON s.id = k.season_id
				JOIN users u ON u.id = fl.parent_user_id
				WHERE k.team_id = ? AND s.is_active = 1
			)`
	}
	return ""
}

// GET /api/chat/team-groups/{teamId}/{kind}/members
func (h *Handler) ResolveTeamGroup(w http.ResponseWriter, r *http.Request) {
	claims := auth.ClaimsFromCtx(r.Context())
	teamID, err := strconv.Atoi(chi.URLParam(r, "teamId"))
	if err != nil {
		http.Error(w, "invalid teamId", http.StatusBadRequest)
		return
	}
	kind := chi.URLParam(r, "kind")
	if !teamGroupKinds[kind] {
		http.Error(w, "invalid kind", http.StatusBadRequest)
		return
	}

	if kind == "alle_trainer" {
		h.resolveAllTrainers(w, r, claims)
		return
	}

	ok, err := h.canSeeTeamGroup(r, claims, teamID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	q := teamGroupMemberQuery(kind)
	if q == "" {
		http.Error(w, "invalid kind", http.StatusBadRequest)
		return
	}
	var rows *sql.Rows
	if kind == "trainer" {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT user_id, name FROM (`+q+`) WHERE user_id != ? ORDER BY name`,
			teamID, claims.UserID)
	} else {
		rows, err = h.db.QueryContext(r.Context(),
			`SELECT user_id, name FROM (`+q+`) WHERE user_id != ? ORDER BY name`,
			teamID, teamID, claims.UserID)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	members := []TeamGroupMember{}
	for rows.Next() {
		var m TeamGroupMember
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			continue
		}
		members = append(members, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// resolveAllTrainers liefert die Mitgliedermenge der "Alle Trainer"-Gruppe
// (= alle Kader-Trainer der aktiven Saison, ohne den Caller). Zugriff nur für
// den Zugriffskreis (Trainer/Vorstand/sL/Beisitzer) oder admin.
func (h *Handler) resolveAllTrainers(w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	eligible, err := h.callerInTrainerCircle(r.Context(), claims)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !eligible {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT user_id, name FROM (`+allTrainersMemberQuery()+`) WHERE user_id != ? ORDER BY name`,
		claims.UserID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	members := []TeamGroupMember{}
	for rows.Next() {
		var m TeamGroupMember
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			continue
		}
		members = append(members, m)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}
