package auth

// System-Rollen (users.role). Hierarchisch: admin ⊇ standard.
//
// Konstanten statt String-Literale, damit Grep und Rename verlässlich sind.
// Semantik: RoleAdmin kann alles; RoleStandard ist der Default jedes
// eingeloggten Nutzers. Die fachliche Feingliederung läuft über
// Vereinsfunktionen (member_club_functions), nicht über weitere System-Rollen.
const (
	RoleAdmin    = "admin"
	RoleStandard = "standard"
)
