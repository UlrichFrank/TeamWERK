package policy

import "slices"

// Principal is a lightweight, cycle-free representation of the authenticated caller.
// Construct it from *auth.Claims using FromClaims (in the handler layer).
type Principal struct {
	UserID        int
	Role          string
	ClubFunctions []string
	IsParent      bool
	HasMember     bool
}

// NavItem represents a navigation entry returned by /api/me.
type NavItem struct {
	Label string `json:"label"`
	Route string `json:"route"`
}

func (p *Principal) hasFunction(f string) bool {
	return slices.Contains(p.ClubFunctions, f)
}

func (p *Principal) hasAnyFunction(fns ...string) bool {
	for _, f := range fns {
		if slices.Contains(p.ClubFunctions, f) {
			return true
		}
	}
	return false
}

// IsTrainerLike returns true for trainer and sportliche_leitung (plus admin pass-through).
func IsTrainerLike(p *Principal) bool {
	return p.Role == "admin" || p.hasAnyFunction("trainer", "sportliche_leitung")
}

// IsVorstandLike returns true for vorstand (plus admin pass-through).
func IsVorstandLike(p *Principal) bool {
	return p.Role == "admin" || p.hasFunction("vorstand")
}

// IsMatchReportReviewer returns true for match-report Freigeber:
// Vereinsfunktion 'medien' oder 'vorstand' oder Admin. Gedanklich der
// Frontend-Spiegel des Backend-Guards in matchreports.isReviewer.
func IsMatchReportReviewer(p *Principal) bool {
	return p.Role == "admin" || p.hasAnyFunction("medien", "vorstand")
}

// IsKassiererLike returns true for kassierer (plus vorstand/admin pass-through).
// Kassierer dürfen Mitglieder lesen/Bankdaten pflegen und den Beitragslauf ausführen.
func IsKassiererLike(p *Principal) bool {
	return IsVorstandLike(p) || p.hasFunction("kassierer")
}

// CanReadMemberAdminFields returns true if the caller may see administrative member
// fields in the members list (exact date of birth, member number, account linkage).
// Pure trainers are kader-scoped and only get a reduced set: name, year of birth (not
// the exact date), pass number, club functions, plus sport fields.
func CanReadMemberAdminFields(p *Principal) bool {
	return p.Role == "admin" || p.hasAnyFunction("vorstand", "kassierer", "sportliche_leitung")
}

// CanEditMember returns true if the caller may edit the given member (identified by their user_id).
func CanEditMember(p *Principal, memberUserID int) bool {
	if p.Role == "admin" || p.hasFunction("vorstand") {
		return true
	}
	return memberUserID != 0 && p.UserID == memberUserID
}

// CanDeleteMember returns true if the caller may delete a member.
func CanDeleteMember(p *Principal) bool {
	return p.Role == "admin" || p.hasFunction("vorstand")
}

// CanEditGame returns true if the caller may create/edit/delete games.
func CanEditGame(p *Principal) bool {
	return p.Role == "admin" || p.hasAnyFunction("vorstand", "trainer", "sportliche_leitung")
}

// CanDeleteGame returns true if the caller may delete a game.
func CanDeleteGame(p *Principal) bool {
	return CanEditGame(p)
}

// CanViewAllGames returns true if the caller sees all games without team scoping.
func CanViewAllGames(p *Principal) bool {
	return p.Role == "admin" || p.hasAnyFunction("vorstand", "sportliche_leitung")
}

// ScopeGamesQuery returns a SQL WHERE fragment and bound args for games queries.
// Games are assumed to be aliased "g" with a game_teams junction table.
// Trainers see only their teams' games; spieler/parents see games their teams play.
func ScopeGamesQuery(p *Principal) (where string, args []any) {
	if CanViewAllGames(p) {
		return "1=1", nil
	}
	if p.hasFunction("trainer") {
		return `EXISTS (
			SELECT 1 FROM game_teams gt
			JOIN kader k ON k.team_id = gt.team_id AND k.season_id = g.season_id
			JOIN kader_trainers kt ON kt.kader_id = k.id
			JOIN members m ON m.id = kt.member_id
			WHERE gt.game_id = g.id AND m.user_id = ?
		)`, []any{p.UserID}
	}
	return `EXISTS (
		SELECT 1 FROM game_teams gt2
		JOIN team_memberships tm ON tm.team_id = gt2.team_id AND tm.season_id = g.season_id
		WHERE gt2.game_id = g.id AND (
			EXISTS(SELECT 1 FROM members m WHERE m.id = tm.member_id AND m.user_id = ?)
			OR EXISTS(SELECT 1 FROM family_links fl WHERE fl.member_id = tm.member_id AND fl.parent_user_id = ?)
		)
	)`, []any{p.UserID, p.UserID}
}

// CanEditKader returns true if the caller may create/edit/delete kader.
func CanEditKader(p *Principal) bool {
	return p.Role == "admin" || p.hasAnyFunction("vorstand", "trainer", "sportliche_leitung")
}

// CanEditDutySlot returns true if the caller may create/edit/delete duty slots.
func CanEditDutySlot(p *Principal) bool {
	return p.Role == "admin" || p.hasAnyFunction("vorstand", "trainer", "sportliche_leitung")
}

// CanFulfillAssignment returns true if the caller may mark an assignment as fulfilled.
func CanFulfillAssignment(p *Principal) bool {
	return p.Role == "admin" || p.hasAnyFunction("trainer", "sportliche_leitung")
}

// CanManageTrainings returns true if the caller may create/edit trainings.
// Mirrors the POST /api/training-series gate (trainer + sportliche_leitung, plus admin).
func CanManageTrainings(p *Principal) bool {
	return IsTrainerLike(p)
}

// CanManageDocuments returns true if the caller may manage the document tree.
func CanManageDocuments(p *Principal) bool {
	return p.Role == "admin"
}

// CanBroadcast returns true if the caller may send broadcast messages.
func CanBroadcast(p *Principal) bool {
	return p.Role == "admin" || p.hasAnyFunction("vorstand", "trainer", "sportliche_leitung")
}

// CanBroadcastAll returns true if the caller may broadcast org-wide (admin-level broadcast features).
func CanBroadcastAll(p *Principal) bool {
	return IsVorstandLike(p)
}

// CanModerateChat returns true if the caller may delete other users' chat messages.
func CanModerateChat(p *Principal) bool {
	return p.Role == "admin"
}

// CanSuppressEventNotification returns true if the caller may delete a game,
// training or duty slot *without* notifying the affected people.
//
// Deliberately narrower than the delete right itself: CanDeleteGame (and
// trainings.hasTeamAccess) also cover trainer and sportliche_leitung. A trainer
// may cancel a game, but the team always hears about it. Suppression exists for
// corrections — import duplicates, typo events — where an "abgesagt" push to
// players and parents would be pure noise.
//
// Callers must treat a silent request from someone without this right as
// "notify anyway", not as an error: the deletion itself is allowed, and failing
// it over an extra flag would block a legitimate action.
func CanSuppressEventNotification(p *Principal) bool {
	return IsVorstandLike(p)
}

// CanBulkRegenDuties gates the mass duty-slot regeneration endpoints
// (/api/duty-slots/bulk-regen/*). Bewusst enger als CanManageDuties (das auch
// Trainer/sportliche Leitung erlaubt) — siehe CapBulkRegenDuties.
func CanBulkRegenDuties(p *Principal) bool {
	return IsVorstandLike(p)
}

// ScopeMembersQuery returns a SQL WHERE fragment that restricts a members query to the
// set visible to the caller. needsUserIDArg=true means the caller must supply Principal.UserID
// as the next query argument.
func ScopeMembersQuery(p *Principal) (where string, needsUserIDArg bool) {
	if p.Role == "admin" || p.hasFunction("vorstand") || p.hasFunction("sportliche_leitung") || p.hasFunction("kassierer") {
		return "1=1", false
	}
	return `EXISTS (
		SELECT 1 FROM kader k
		JOIN kader_trainers kt ON kt.kader_id = k.id
		JOIN members tm ON tm.id = kt.member_id
		JOIN player_memberships pm ON pm.member_id = m.id AND pm.team_id = k.team_id
		WHERE tm.user_id = ?
	)`, true
}

// CanCreateRootFolder returns true if the caller may create a folder on the top level
// of /dokumente. Top-level folders have no parent whose ACL could authorize the write,
// so this is a standalone right: admin plus vorstand.
//
// Deliberately narrower than CapManageDocuments (blanket read+write on *every* folder,
// admin only): vorstand creates top-level folders and holds them afterwards through the
// created_by owner precedence in FolderAccess — not through a global override.
func CanCreateRootFolder(p *Principal) bool {
	return IsVorstandLike(p)
}

// Capability constants used in /api/me responses.
const (
	CapManageMembers    = "manage_members"
	CapManageGames      = "manage_games"
	CapImportGames      = "import_games"
	CapManageDuties     = "manage_duties"
	CapManageKader      = "manage_kader"
	CapManageUsers      = "manage_users"
	CapManageSeasons    = "manage_seasons"
	CapManageClub       = "manage_club"
	CapManageFees       = "manage_fees"
	CapManageDutyTypes  = "manage_duty_types"
	CapImpersonate      = "impersonate"
	CapManageTrainings  = "manage_trainings"
	CapFulfillDuties    = "fulfill_duties"
	CapManageDocuments  = "manage_documents"
	CapCreateRootFolder = "create_root_folder"
	CapBroadcast        = "broadcast_messages"
	CapBroadcastAll     = "broadcast_all"
	CapModerateChat     = "moderate_chat"

	CapSuppressEventNotification = "suppress_event_notification"

	// CapBulkRegenDuties ist bewusst enger als CapManageDuties (das auch Trainer
	// haben): ein Lauf über die Restsaison kann hunderte Slots löschen/neu
	// anlegen. Gleiche Begründung und gleiches Tier wie CapImportGames.
	CapBulkRegenDuties = "bulk_regen_duties"
)

// Capabilities returns the list of capability strings for a given principal.
func Capabilities(p *Principal) []string {
	caps := []string{}
	if IsVorstandLike(p) {
		// CapImportGames ist bewusst enger als CapManageGames: der H4A-Import nimmt
		// fremde Zugangsdaten entgegen und schreibt den halben Spielplan — das
		// bleibt beim Vorstand, obwohl Trainer/sportliche Leitung Spiele pflegen
		// dürfen. Spiegelt das Vorstand-Tier der Import-Routen in router.go.
		caps = append(caps, CapManageMembers, CapManageGames, CapImportGames, CapManageDuties, CapManageKader, CapManageUsers, CapManageSeasons, CapManageDutyTypes)
	} else if IsTrainerLike(p) {
		caps = append(caps, CapManageGames, CapManageDuties, CapManageKader)
	}
	// Kassierer-like (kassierer + vorstand + admin): Vereins-Stammdaten (SEPA) und
	// Beitragswesen — steuert die Einstellungen-Tabs „Verein" und „Beiträge".
	if IsKassiererLike(p) {
		caps = append(caps, CapManageClub, CapManageFees)
	}
	// Trainer-like personas (admin, trainer, sportliche_leitung) — excludes pure vorstand.
	if CanManageTrainings(p) {
		caps = append(caps, CapManageTrainings)
	}
	if CanFulfillAssignment(p) {
		caps = append(caps, CapFulfillDuties)
	}
	// Messaging.
	if CanBroadcast(p) {
		caps = append(caps, CapBroadcast)
	}
	if CanBroadcastAll(p) {
		caps = append(caps, CapBroadcastAll)
	}
	if CanCreateRootFolder(p) {
		caps = append(caps, CapCreateRootFolder)
	}
	// Stummes Löschen — enger als das Löschrecht selbst, siehe
	// CanSuppressEventNotification.
	if CanSuppressEventNotification(p) {
		caps = append(caps, CapSuppressEventNotification)
	}
	if CanBulkRegenDuties(p) {
		caps = append(caps, CapBulkRegenDuties)
	}
	if p.Role == "admin" {
		caps = append(caps, CapImpersonate, CapManageDocuments, CapModerateChat)
	}
	return caps
}

// NavFor returns the ordered list of navigation items for a given principal,
// using the actual frontend route paths.
func NavFor(p *Principal) []NavItem {
	nav := []NavItem{}

	// Nutzer
	nav = append(nav, NavItem{"Dashboard", "/"})
	// „Mein Profil" für alle mit eigenem Mitglieds-Datensatz oder Elternteilen.
	// Reine Admin-Accounts ohne Mitglied/Kind (Beispiel: initialer Setup-Admin)
	// sehen den Eintrag nicht — für sie ist /profil leer.
	if p.Role != "admin" || p.HasMember || p.IsParent {
		nav = append(nav, NavItem{"Mein Profil", "/profil"})
	}

	// Spielbetrieb — visible to all
	nav = append(nav, NavItem{"Kalender", "/kalender"})
	nav = append(nav, NavItem{"Termine", "/termine"})
	nav = append(nav, NavItem{"Videos", "/videos"})
	// Anwesenheits-Statistik nur für Trainer / sportliche Leitung / Admin.
	if IsTrainerLike(p) {
		nav = append(nav, NavItem{"Anwesenheit", "/anwesenheit"})
		// Mannschaftsübersicht des Trainingstagebuchs — dieselbe Zielgruppe
		// wie die Anwesenheitsstatistik.
		nav = append(nav, NavItem{"Trainingstagebuch", "/trainingstagebuch"})
	}
	// Eigenes Trainingstagebuch: ausschließlich für Spieler. Ein Mitglieds-
	// Datensatz allein genügt nicht — Trainer, Vorstand, Kassierer und
	// Elternteile mit eigenem Mitglied trainieren nicht als Spieler und führen
	// deshalb kein eigenes Tagebuch. Das Tagebuch der Kinder erreichen Eltern
	// über den Tab auf der Kind-Profilseite, nicht über diesen Eintrag.
	// Spiegelt die Tab-Sichtbarkeit im Frontend (hasPlayerFunction) und das
	// Schreibrecht in trainingdiary.CreateEntry.
	if p.hasFunction("spieler") {
		nav = append(nav, NavItem{"Mein Trainingstagebuch", "/profil/trainingstagebuch"})
	}

	// Verein — visible to all
	nav = append(nav, NavItem{"Mein Team", "/mein-team"})
	nav = append(nav, NavItem{"Dokumente", "/dokumente"})
	nav = append(nav, NavItem{"Dienste", "/dienste"})
	nav = append(nav, NavItem{"Mitfahrten", "/mitfahrgelegenheiten"})
	nav = append(nav, NavItem{"Nachrichten", "/chat"})
	// Spielberichte nur für Presseteam (+Admin, hierarchisch).
	if p.Role == "presseteam" || p.Role == "admin" {
		nav = append(nav, NavItem{"Spielberichte", "/spielberichte"})
	}
	// Berichte zur Prüfung — Freigeber (Vereinsfunktion medien/vorstand + Admin).
	if IsMatchReportReviewer(p) {
		nav = append(nav, NavItem{"Berichte prüfen", "/spielberichte/pruefen"})
	}

	// Verwaltung — role-restricted
	if IsTrainerLike(p) || IsVorstandLike(p) {
		nav = append(nav, NavItem{"Kader", "/kader"})
	}
	if IsVorstandLike(p) {
		nav = append(nav, NavItem{"Nutzerverwaltung", "/nutzer"})
	}
	// Mitglieder + Beitragslauf + Einstellungen: auch für Kassierer sichtbar.
	if IsKassiererLike(p) {
		nav = append(nav, NavItem{"Mitglieder", "/mitglieder"})
	}
	if IsVorstandLike(p) {
		nav = append(nav, NavItem{"Diensttypen", "/diensttypen"})
		nav = append(nav, NavItem{"Dienstplan-Vorlagen", "/dienstplan-vorlagen"})
		nav = append(nav, NavItem{"Veranstaltungsorte", "/veranstaltungsorte"})
	}
	if IsKassiererLike(p) {
		nav = append(nav, NavItem{"Beitragslauf", "/beitragslauf"})
		nav = append(nav, NavItem{"Tresor", "/tresor"})
		nav = append(nav, NavItem{"Einstellungen", "/einstellungen"})
	}
	// Wartungsmodus: nur System-Rolle admin (Bypass-Nutzer, kann sich selbst
	// nicht aussperren, ist verantwortlich für Toggle in Migrations-Fenstern).
	if p.Role == "admin" {
		nav = append(nav, NavItem{"Wartungsmodus", "/wartung"})
	}

	return nav
}
