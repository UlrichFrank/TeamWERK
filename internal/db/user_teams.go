package db

import "strings"

// UserTeamKind wählt, über welche Kader-Beziehung ein Account mit einem Team
// verbunden ist. Die Unterscheidung trägt die Aushilfe-Regel der Dienste
// (Change dienste-erweiterter-kader): wer mit einem Team nur über den
// erweiterten Kader verbunden ist, hilft dort aus, schuldet aber keine Dienste.
type UserTeamKind int

const (
	// TeamsStamm: eigene Mitglieder oder Kinder (family_links) im Stammkader
	// (kader_members) sowie eigene Mitglieder als Trainer (kader_trainers).
	// Kein Statusfilter — derselbe Umfang wie player_/trainer_memberships.
	TeamsStamm UserTeamKind = iota
	// TeamsExtended: eigene Mitglieder oder Kinder im erweiterten Kader
	// (kader_extended_members). Statusfilter status <> 'ausgetreten', NICHT
	// = 'aktiv' — Förderkinder (status 'foerderkind') sind die typische
	// Besetzung dieser Liste.
	TeamsExtended
	// TeamsChildren: Kinder (family_links) im Stammkader oder im erweiterten
	// Kader — die Team-Menge des Zielgruppen-Eintrags 'eltern'. Der
	// erweiterte Zweig trägt denselben Statusfilter wie TeamsExtended.
	TeamsChildren
)

// UserTeamsSQL liefert ein `SELECT team_id …` über die Teams der aktiven
// Saison, mit denen der Account userExpr in der gewählten Art verbunden ist.
// userExpr ist ein SQL-Ausdruck (Spaltenreferenz wie `da.user_id` oder ein
// Platzhalter `?`); er kommt UserTeamsRefs(kind)-mal im Fragment vor, bei `?`
// müssen die Argumente entsprechend oft übergeben werden.
//
// Ein Baustein für Dienstbörse und Dashboard, damit beide dieselbe Antwort auf
// „Stamm oder Aushilfe?“ geben (design.md Entscheidung 3).
func UserTeamsSQL(kind UserTeamKind, userExpr string) string {
	var q string
	switch kind {
	case TeamsStamm:
		q = `SELECT k_ut.team_id FROM kader_members km_ut
			JOIN kader k_ut ON k_ut.id = km_ut.kader_id
			WHERE k_ut.team_id IS NOT NULL
			  AND k_ut.season_id = (SELECT id FROM seasons WHERE is_active = 1)
			  AND km_ut.member_id IN (
			      SELECT id FROM members WHERE user_id = {U}
			      UNION SELECT member_id FROM family_links WHERE parent_user_id = {U})
			UNION
			SELECT k_ut.team_id FROM kader_trainers kt_ut
			JOIN kader k_ut ON k_ut.id = kt_ut.kader_id
			JOIN members m_ut ON m_ut.id = kt_ut.member_id
			WHERE k_ut.team_id IS NOT NULL
			  AND k_ut.season_id = (SELECT id FROM seasons WHERE is_active = 1)
			  AND m_ut.user_id = {U}`
	case TeamsExtended:
		q = `SELECT k_ut.team_id FROM kader_extended_members kem_ut
			JOIN kader k_ut ON k_ut.id = kem_ut.kader_id
			JOIN members m_ut ON m_ut.id = kem_ut.member_id
			WHERE k_ut.team_id IS NOT NULL
			  AND k_ut.season_id = (SELECT id FROM seasons WHERE is_active = 1)
			  AND m_ut.status <> 'ausgetreten'
			  AND (m_ut.user_id = {U}
			       OR m_ut.id IN (SELECT member_id FROM family_links WHERE parent_user_id = {U}))`
	case TeamsChildren:
		q = `SELECT k_ut.team_id FROM kader_members km_ut
			JOIN kader k_ut ON k_ut.id = km_ut.kader_id
			WHERE k_ut.team_id IS NOT NULL
			  AND k_ut.season_id = (SELECT id FROM seasons WHERE is_active = 1)
			  AND km_ut.member_id IN (SELECT member_id FROM family_links WHERE parent_user_id = {U})
			UNION
			SELECT k_ut.team_id FROM kader_extended_members kem_ut
			JOIN kader k_ut ON k_ut.id = kem_ut.kader_id
			JOIN members m_ut ON m_ut.id = kem_ut.member_id
			WHERE k_ut.team_id IS NOT NULL
			  AND k_ut.season_id = (SELECT id FROM seasons WHERE is_active = 1)
			  AND m_ut.status <> 'ausgetreten'
			  AND m_ut.id IN (SELECT member_id FROM family_links WHERE parent_user_id = {U})`
	default:
		panic("db.UserTeamsSQL: unbekannte Art")
	}
	return strings.ReplaceAll(q, "{U}", userExpr)
}

// UserTeamsRefs ist die Anzahl der Vorkommen von userExpr in UserTeamsSQL(kind).
func UserTeamsRefs(kind UserTeamKind) int {
	switch kind {
	case TeamsStamm:
		return 3
	case TeamsExtended:
		return 2
	case TeamsChildren:
		return 2
	}
	panic("db.UserTeamsRefs: unbekannte Art")
}

// UserArgs wiederholt userID so oft, wie UserTeamsSQL(kind, "?") Platzhalter trägt.
func UserArgs(kind UserTeamKind, userID int) []any {
	n := UserTeamsRefs(kind)
	out := make([]any, n)
	for i := range out {
		out[i] = userID
	}
	return out
}
