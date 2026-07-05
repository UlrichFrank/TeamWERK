package auth

// System-Rollen (users.role). Hierarchisch: admin ⊇ presseteam ⊇ standard.
//
// Konstanten statt String-Literale, damit Grep und Rename verlässlich sind.
// Semantik: RolePressTeam kann alles, was RoleStandard kann, plus die
// presseteam-Endpunkte (Spielberichte schreiben/publizieren). RoleAdmin
// kann alles.
const (
	RoleAdmin     = "admin"
	RoleStandard  = "standard"
	RolePressTeam = "presseteam"
)
