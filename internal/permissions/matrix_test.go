// Package permissions_test enthält den tabellen-getriebenen Backend-Permission-Matrix-Test.
// Für jeden registrierten Endpoint × jede der 11 Personas wird der erwartete HTTP-Status
// geprüft. Testet primär Middleware-Verhalten (RequireRole, RequireClubFunction, auth.Middleware)
// sowie bekannte Handler-Level-Gates (Inline-RequireClubFunction-Checks).
//
// Neue Routen MÜSSEN in `matrix` eingetragen werden — sonst failt der Drift-Check.
package permissions_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

const (
	// httpAllowed: Persona darf den Endpoint erreichen (Middleware lässt durch).
	// Test prüft: Status-Code ≠ 401 und ≠ 403.
	httpAllowed = -1
	// httpAnyOK: Jeder Response-Code ist akzeptabel. Verwendung bei:
	// - öffentlichen Routen, bei denen der Handler selbst 401 zurückgibt (z. B. cookie-basiert)
	// - Routen, bei denen der Handler 403 aus Ownership/Membership-Gründen zurückgibt
	//   (kein Fixture in der Test-DB → jeder Nutzer wird geblockt, unabhängig von seiner Rolle)
	httpAnyOK = 0
)

// ── Vordefinierte Expected-Maps pro Middleware-Gate ───────────────────────────

// Alle 11 Personas kommen durch den Authenticated-Middleware-Layer.
var exAuth = map[string]int{
	"admin": httpAllowed, "vorstand": httpAllowed, "vorstand_elternteil": httpAllowed,
	"vorstand_beisitzer": httpAllowed, "kassierer": httpAllowed,
	"trainer": httpAllowed, "trainer_elternteil": httpAllowed,
	"sportliche_leitung": httpAllowed, "sportliche_leitung_elternteil": httpAllowed,
	"spieler": httpAllowed, "elternteil": httpAllowed,
}

// Öffentliche Routen — kein Auth-Gate, beliebiger Response-Code akzeptabel.
var exPublic = map[string]int{
	"admin": httpAnyOK, "vorstand": httpAnyOK, "vorstand_elternteil": httpAnyOK,
	"vorstand_beisitzer": httpAnyOK, "kassierer": httpAnyOK,
	"trainer": httpAnyOK, "trainer_elternteil": httpAnyOK,
	"sportliche_leitung": httpAnyOK, "sportliche_leitung_elternteil": httpAnyOK,
	"spieler": httpAnyOK, "elternteil": httpAnyOK,
}

// RequireClubFunction("trainer","sportliche_leitung")
var exTrainer = map[string]int{
	"admin": httpAllowed, "trainer": httpAllowed, "trainer_elternteil": httpAllowed,
	"sportliche_leitung": httpAllowed, "sportliche_leitung_elternteil": httpAllowed,
	"vorstand": 403, "vorstand_elternteil": 403, "vorstand_beisitzer": 403,
	"kassierer": 403, "spieler": 403, "elternteil": 403,
}

// RequireClubFunction("vorstand","trainer","sportliche_leitung")
var exVorstandTrainer = map[string]int{
	"admin": httpAllowed, "vorstand": httpAllowed, "vorstand_elternteil": httpAllowed,
	"trainer": httpAllowed, "trainer_elternteil": httpAllowed,
	"sportliche_leitung": httpAllowed, "sportliche_leitung_elternteil": httpAllowed,
	"vorstand_beisitzer": 403, "kassierer": 403, "spieler": 403, "elternteil": 403,
}

// RequireClubFunction("vorstand","trainer","sportliche_leitung","kassierer")
// Saisons lesen: Kader (Vorstand/Trainer/sL) + Beitragslauf (Kassierer).
var exSeasonsRead = map[string]int{
	"admin": httpAllowed, "vorstand": httpAllowed, "vorstand_elternteil": httpAllowed,
	"kassierer": httpAllowed,
	"trainer":   httpAllowed, "trainer_elternteil": httpAllowed,
	"sportliche_leitung": httpAllowed, "sportliche_leitung_elternteil": httpAllowed,
	"vorstand_beisitzer": 403, "spieler": 403, "elternteil": 403,
}

// RequireClubFunction("vorstand")
var exVorstand = map[string]int{
	"admin": httpAllowed, "vorstand": httpAllowed, "vorstand_elternteil": httpAllowed,
	"vorstand_beisitzer": 403, "kassierer": 403,
	"trainer": 403, "trainer_elternteil": 403,
	"sportliche_leitung": 403, "sportliche_leitung_elternteil": 403,
	"spieler": 403, "elternteil": 403,
}

// RequireClubFunction("vorstand","kassierer","trainer","sportliche_leitung")
// Member-Liste (Suche): Mitgliederverwaltung + Kader-/Trainersuche.
var exMembersList = map[string]int{
	"admin": httpAllowed, "vorstand": httpAllowed, "vorstand_elternteil": httpAllowed,
	"kassierer": httpAllowed,
	"trainer":   httpAllowed, "trainer_elternteil": httpAllowed,
	"sportliche_leitung": httpAllowed, "sportliche_leitung_elternteil": httpAllowed,
	"vorstand_beisitzer": 403, "spieler": 403, "elternteil": 403,
}

// RequireClubFunction("vorstand","kassierer")
var exVorstandKassierer = map[string]int{
	"admin": httpAllowed, "vorstand": httpAllowed, "vorstand_elternteil": httpAllowed,
	"kassierer":          httpAllowed,
	"vorstand_beisitzer": 403,
	"trainer":            403, "trainer_elternteil": 403,
	"sportliche_leitung": 403, "sportliche_leitung_elternteil": 403,
	"spieler": 403, "elternteil": 403,
}

// RequireRole("admin")
var exAdmin = map[string]int{
	"admin":    httpAllowed,
	"vorstand": 403, "vorstand_elternteil": 403, "vorstand_beisitzer": 403,
	"kassierer": 403, "trainer": 403, "trainer_elternteil": 403,
	"sportliche_leitung": 403, "sportliche_leitung_elternteil": 403,
	"spieler": 403, "elternteil": 403,
}

// Handler-Level-Gate: chat.SendBroadcast — admin || vorstand || trainer || sportliche_leitung
var exBroadcastSend = map[string]int{
	"admin": httpAllowed, "vorstand": httpAllowed, "vorstand_elternteil": httpAllowed,
	"trainer": httpAllowed, "trainer_elternteil": httpAllowed,
	"sportliche_leitung": httpAllowed, "sportliche_leitung_elternteil": httpAllowed,
	"vorstand_beisitzer": 403, "kassierer": 403, "spieler": 403, "elternteil": 403,
}

// Handler-Level-Gate: games.SaveLineup — admin || trainer (nur diese beiden Checks)
var exLineup = map[string]int{
	"admin": httpAllowed, "trainer": httpAllowed, "trainer_elternteil": httpAllowed,
	"vorstand": 403, "vorstand_elternteil": 403, "vorstand_beisitzer": 403,
	"kassierer": 403, "sportliche_leitung": 403, "sportliche_leitung_elternteil": 403,
	"spieler": 403, "elternteil": 403,
}

// Handler-Level-Gate: upload.SepaDownloadToken / DeleteSepaMandat
// Gate: isAdmin || isVorstand || isOwn || isParent
// Mit Member-ID 1 (existiert nicht in Test-DB): isOwn=false, isParent=false für alle.
// → admin und vorstand kommen durch; alle anderen → 403.
var exSepaOwner = map[string]int{
	"admin": httpAllowed, "vorstand": httpAllowed, "vorstand_elternteil": httpAllowed,
	"vorstand_beisitzer": 403, "kassierer": 403,
	"trainer": 403, "trainer_elternteil": 403,
	"sportliche_leitung": 403, "sportliche_leitung_elternteil": 403,
	"spieler": 403, "elternteil": 403,
}

// Handler-Level-Gate: members.canAccessMember — admin || vorstand || kassierer || isOwn || isParent.
// Mit einer Member-ID, die keiner Persona gehört (isOwn=false, isParent=false):
// → admin, vorstand(_elternteil), kassierer kommen durch; alle anderen → 403.
// Quelle: openspec/changes/secure-member-draft-access (B-1/B-3).
var exMemberDraftAccess = map[string]int{
	"admin": httpAllowed, "vorstand": httpAllowed, "vorstand_elternteil": httpAllowed,
	"kassierer":          httpAllowed,
	"vorstand_beisitzer": 403,
	"trainer":            403, "trainer_elternteil": 403,
	"sportliche_leitung": 403, "sportliche_leitung_elternteil": 403,
	"spieler": 403, "elternteil": 403,
}

// endpointCase beschreibt einen einzelnen Endpoint in der Permission-Matrix.
type endpointCase struct {
	method   string
	path     string
	expected map[string]int // persona ID → erwarteter HTTP-Status
}

// matrix enthält einen Eintrag pro registrierter HTTP-Route.
// Quelle der Wahrheit: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md
// Ausschlüsse:
//   - SSE-Routen (/api/events, /api/chat/events): CookieMiddleware, kein Bearer-Token-Flow.
//   - SPA-Fallback (GET /*): kein API-Endpoint.
var matrix = []endpointCase{
	// ── Public ──────────────────────────────────────────────────────────────────
	// /api/uploads/* läuft seit B-5 unter CookieMiddleware (kein Bearer-Flow, wie SSE):
	// ohne Refresh-Cookie → 401. Die Bearer-basierte Matrix akzeptiert das via httpAnyOK;
	// die eigentliche Cookie-Auth wird in upload.TestServeUpload_RequiresCookieAuth geprüft.
	{method: "GET", path: "/api/uploads/*", expected: exPublic},
	{method: "GET", path: "/api/files/{id}/download", expected: exPublic},
	// sepa-mandat/download ist öffentlich, nutzt aber Query-Token-Check → httpAnyOK
	{method: "GET", path: "/api/members/{id}/sepa-mandat/download", expected: exPublic},
	{method: "GET", path: "/api/encryption-pubkey", expected: exPublic},
	{method: "POST", path: "/api/auth/login", expected: exPublic},
	// refresh nutzt Cookie-Auth, kein Bearer → 401 ohne Cookie ist korrekt → exPublic
	{method: "POST", path: "/api/auth/refresh", expected: exPublic},
	{method: "POST", path: "/api/auth/logout", expected: exPublic},
	{method: "POST", path: "/api/auth/request-membership", expected: exPublic},
	{method: "POST", path: "/api/auth/register", expected: exPublic},
	{method: "GET", path: "/api/auth/token-info", expected: exPublic},
	{method: "POST", path: "/api/auth/forgot-password", expected: exPublic},
	{method: "POST", path: "/api/auth/reset-password", expected: exPublic},
	{method: "GET", path: "/api/profile/email/confirm", expected: exPublic},
	{method: "GET", path: "/api/profile/recovery-email/confirm", expected: exPublic},
	// Calendar feed — token im Pfad ist die Authentifizierung
	{method: "GET", path: "/api/calendar/feed/{token}", expected: exPublic},
	// Monitoring-Signale: healthz public; metrics ohne METRICS_TOKEN deaktiviert (404),
	// mit Token Bearer-geschützt — beides ohne Auth-Middleware → exPublic.
	{method: "GET", path: "/api/healthz", expected: exPublic},
	{method: "GET", path: "/api/metrics", expected: exPublic},

	// ── Authenticated ───────────────────────────────────────────────────────────
	// Chat (einfache Operationen ohne Konversations-Membership)
	{method: "GET", path: "/api/chat/users", expected: exAuth},
	{method: "GET", path: "/api/chat/conversations", expected: exAuth},
	{method: "POST", path: "/api/chat/conversations", expected: exAuth},
	{method: "GET", path: "/api/chat/broadcasts", expected: exAuth},
	// Broadcasts senden: Handler-Level-Gate (admin || vorstand || trainer || sportliche_leitung)
	{method: "POST", path: "/api/chat/broadcasts", expected: exBroadcastSend},

	// Konversations-spezifische Routen: isMember-Check → 403 für alle wenn Konversation 1 nicht existiert.
	// httpAnyOK: wir testen hier keine Middleware, sondern dokumentieren das bekannte Verhalten.
	{method: "GET", path: "/api/chat/conversations/{id}/messages", expected: exPublic},
	{method: "POST", path: "/api/chat/conversations/{id}/messages", expected: exPublic},
	{method: "POST", path: "/api/chat/conversations/{id}/read", expected: exPublic},
	{method: "DELETE", path: "/api/chat/conversations/{id}/members/me", expected: exPublic},
	{method: "DELETE", path: "/api/chat/conversations/{id}/members/{uid}", expected: exPublic},
	{method: "DELETE", path: "/api/chat/conversations/{id}/everyone", expected: exPublic},
	{method: "PUT", path: "/api/chat/conversations/{id}", expected: exPublic},
	{method: "POST", path: "/api/chat/conversations/{id}/transfer-ownership", expected: exPublic},
	{method: "DELETE", path: "/api/chat/conversations/{id}", expected: exPublic},
	{method: "POST", path: "/api/chat/conversations/{id}/members", expected: exPublic},
	// Nachrichten: Sender-Check → 403 für alle (kein Fixture)
	{method: "PUT", path: "/api/chat/messages/{id}", expected: exPublic},
	{method: "DELETE", path: "/api/chat/messages/{id}", expected: exPublic},
	{method: "POST", path: "/api/chat/messages/{id}/reactions", expected: exPublic},
	// Broadcast-Mutations: Sender-Check → 403 für alle (kein Fixture)
	{method: "POST", path: "/api/chat/broadcasts/{id}/read", expected: exPublic},
	{method: "PUT", path: "/api/chat/broadcasts/{id}", expected: exPublic},
	{method: "DELETE", path: "/api/chat/broadcasts/{id}", expected: exPublic},
	// Team-Standard-Gruppen (Picker im "Neues Gespräch"-Modal):
	// List ist für alle Eingeloggten erlaubt (liefert leere Liste bei fehlender Saison/Mitgliedschaft).
	// Resolve gating geschieht im Handler (canSeeTeamGroup); für die Matrix gilt: kein Auth-Tier-Gate.
	{method: "GET", path: "/api/chat/team-groups", expected: exAuth},
	{method: "GET", path: "/api/chat/team-groups/{teamId}/{kind}/members", expected: exPublic},

	// Members (self-service)
	{method: "GET", path: "/api/users/{id}/contact", expected: exAuth},
	// Änderungsanträge: Handler-Level-Ownership-Gate (B-1) — Eigentümer/Eltern/admin/vorstand/kassierer.
	{method: "GET", path: "/api/members/{id}/change-drafts", expected: exMemberDraftAccess},
	{method: "POST", path: "/api/members/{id}/change-request", expected: exMemberDraftAccess},
	// sepa-mandat-spezifische Routen: Handler-Level-Gate (isAdmin || isVorstand || isOwn || isParent)
	{method: "GET", path: "/api/members/{id}/sepa-mandat/download-token", expected: exSepaOwner},
	{method: "DELETE", path: "/api/members/{id}/sepa-mandat", expected: exSepaOwner},

	// Me
	{method: "GET", path: "/api/me", expected: exAuth},

	// Profile
	{method: "GET", path: "/api/profile/me", expected: exAuth},
	{method: "PUT", path: "/api/profile/me", expected: exAuth},
	{method: "GET", path: "/api/profile/vehicle", expected: exAuth},
	{method: "PUT", path: "/api/profile/vehicle", expected: exAuth},
	{method: "GET", path: "/api/profile/account", expected: exAuth},
	{method: "PUT", path: "/api/profile/account", expected: exAuth},
	{method: "POST", path: "/api/profile/password", expected: exAuth},
	{method: "POST", path: "/api/profile/email", expected: exAuth},
	{method: "POST", path: "/api/profile/phones", expected: exAuth},
	{method: "PUT", path: "/api/profile/phones/{id}", expected: exAuth},
	{method: "DELETE", path: "/api/profile/phones/{id}", expected: exAuth},
	{method: "PUT", path: "/api/profile/visibility", expected: exAuth},
	{method: "PUT", path: "/api/profile/reminder-preference", expected: exAuth},
	{method: "PUT", path: "/api/profile/absence-visibility", expected: exAuth},
	// cross-team-visible: Handler-interner Ownership-Check (eigenes Member, Eltern,
	// vorstand, admin); andere Authenticated-Caller → 403 im Handler.
	{method: "PUT", path: "/api/members/{id}/cross-team-visible", expected: exAuth},
	{method: "GET", path: "/api/profile/notification-preferences", expected: exAuth},
	{method: "PUT", path: "/api/profile/notification-preferences", expected: exAuth},

	// Calendar token (eigenes Token verwalten)
	{method: "GET", path: "/api/calendar/token", expected: exAuth},
	{method: "POST", path: "/api/calendar/token", expected: exAuth},
	{method: "DELETE", path: "/api/calendar/token", expected: exAuth},

	// Kind-Profil
	{method: "GET", path: "/api/family/proxy-accounts", expected: exAuth},
	// kind/{memberId}-Routen: isParentOf-Handler-Check → 403 für alle ohne family_link-Fixture.
	{method: "GET", path: "/api/profile/kind/{memberId}", expected: exPublic},
	{method: "PUT", path: "/api/profile/kind/{memberId}/account", expected: exPublic},
	{method: "PUT", path: "/api/profile/kind/{memberId}/member", expected: exPublic},
	{method: "PUT", path: "/api/profile/kind/{memberId}/bank", expected: exPublic},
	{method: "POST", path: "/api/profile/kind/{memberId}/photo", expected: exPublic},
	{method: "DELETE", path: "/api/profile/kind/{memberId}/photo", expected: exPublic},
	{method: "POST", path: "/api/profile/kind/{memberId}/phones", expected: exPublic},
	{method: "DELETE", path: "/api/profile/kind/{memberId}/phones/{phoneId}", expected: exPublic},
	{method: "POST", path: "/api/profile/kind/{memberId}/recovery-email", expected: exPublic},
	{method: "PUT", path: "/api/profile/kind/{memberId}/visibility", expected: exPublic},

	// Upload (User-Photo)
	{method: "POST", path: "/api/upload/user-photo", expected: exAuth},
	{method: "DELETE", path: "/api/upload/user-photo", expected: exAuth},

	// Dashboard
	{method: "GET", path: "/api/dashboard", expected: exAuth},

	// Duties (self-service)
	{method: "GET", path: "/api/duty-board", expected: exAuth},
	{method: "POST", path: "/api/duty-board/{slotId}/claim", expected: exAuth},
	{method: "DELETE", path: "/api/duty-board/{slotId}/claim", expected: exAuth},
	{method: "GET", path: "/api/duty-accounts", expected: exAuth},
	{method: "GET", path: "/api/duty-slots", expected: exAuth},
	{method: "GET", path: "/api/duty-slots/{id}/assignments", expected: exAuth},

	// Mitfahrgelegenheiten
	{method: "GET", path: "/api/mitfahrgelegenheiten", expected: exAuth},
	{method: "POST", path: "/api/mitfahrgelegenheiten", expected: exAuth},
	// DELETE/Pairing: Ownership-Check → 403 wenn Fixture nicht existiert
	{method: "DELETE", path: "/api/mitfahrgelegenheiten/{id}", expected: exPublic},
	{method: "POST", path: "/api/mitfahrt-paarungen", expected: exAuth},
	{method: "POST", path: "/api/mitfahrt-paarungen/{id}/confirm", expected: exPublic},
	{method: "POST", path: "/api/mitfahrt-paarungen/{id}/reject", expected: exPublic},

	// Push
	{method: "GET", path: "/api/push/vapid-public-key", expected: exAuth},
	{method: "POST", path: "/api/push/subscribe", expected: exAuth},
	{method: "DELETE", path: "/api/push/subscribe", expected: exAuth},

	// Users picker
	{method: "GET", path: "/api/users/picker", expected: exAuth},

	// Absences
	{method: "GET", path: "/api/absences/preview", expected: exAuth},
	{method: "GET", path: "/api/absences/calendar", expected: exAuth},
	{method: "GET", path: "/api/absences", expected: exAuth},
	{method: "POST", path: "/api/absences", expected: exAuth},
	{method: "PUT", path: "/api/absences/{id}", expected: exAuth},
	{method: "DELETE", path: "/api/absences/{id}", expected: exAuth},

	// Dokumente
	{method: "GET", path: "/api/folders", expected: exAuth},
	{method: "POST", path: "/api/folders", expected: exAuth},
	{method: "GET", path: "/api/folders/{id}/contents", expected: exAuth},
	{method: "PUT", path: "/api/folders/{id}", expected: exAuth},
	{method: "DELETE", path: "/api/folders/{id}", expected: exAuth},
	{method: "GET", path: "/api/folders/{id}/permissions", expected: exAuth},
	{method: "POST", path: "/api/folders/{id}/permissions", expected: exAuth},
	{method: "DELETE", path: "/api/folders/{id}/permissions/{permId}", expected: exAuth},
	{method: "POST", path: "/api/folders/{folderId}/files", expected: exAuth},
	{method: "GET", path: "/api/files/{id}/download-token", expected: exAuth},
	{method: "PUT", path: "/api/files/{id}", expected: exAuth},
	{method: "DELETE", path: "/api/files/{id}", expected: exAuth},

	// Games (read + RSVP)
	{method: "GET", path: "/api/games", expected: exAuth},
	{method: "GET", path: "/api/games/{id}", expected: exAuth},
	{method: "GET", path: "/api/games/my", expected: exAuth},
	{method: "POST", path: "/api/games/{id}/respond", expected: exAuth},
	{method: "GET", path: "/api/games/{id}/responses", expected: exAuth},
	{method: "GET", path: "/api/games/{id}/participants", expected: exAuth},
	// lineup: Handler-Level-Gate (admin || trainer — kein vorstand, kein sportliche_leitung)
	{method: "POST", path: "/api/games/{id}/lineup", expected: exLineup},
	// attendance: GET unter Authenticated (Handler-Authz: admin/sL/trainer);
	// POST unter Trainer+sL (RequireClubFunction-Group).
	{method: "GET", path: "/api/games/{id}/attendances", expected: exAuth},

	// Anwesenheits-Statistik (Authenticated; Handler-Authz)
	{method: "GET", path: "/api/teams/{id}/attendance-stats", expected: exAuth},
	{method: "GET", path: "/api/teams/{id}/attendance-open", expected: exAuth},
	{method: "GET", path: "/api/members/{id}/attendance-stats", expected: exAuth},

	// Trainings (read + RSVP)
	{method: "GET", path: "/api/training-sessions", expected: exAuth},
	{method: "GET", path: "/api/training-sessions/{id}", expected: exAuth},
	{method: "POST", path: "/api/training-sessions/{id}/respond", expected: exAuth},
	{method: "GET", path: "/api/training-sessions/{id}/attendances", expected: exAuth},

	// Teams
	{method: "GET", path: "/api/teams", expected: exAuth},
	{method: "GET", path: "/api/teams/names", expected: exAuth},
	{method: "GET", path: "/api/teams/my", expected: exAuth},
	// roster: user_accessible_teams-Check → 403 wenn kein Season/Membership-Fixture
	{method: "GET", path: "/api/teams/{id}/roster", expected: exPublic},
	{method: "GET", path: "/api/stammvereine", expected: exAuth},

	// Videos (Spielvideo-Ablage) — Lese-/CRUD-Routen im Authenticated-Tier;
	// {id}-Routen prüfen zuerst die Existenz → ohne Fixture 404 (≠401/403).
	{method: "GET", path: "/api/videos", expected: exAuth},
	{method: "GET", path: "/api/videos/{id}", expected: exAuth},
	{method: "GET", path: "/api/videos/{id}/play", expected: exAuth},
	{method: "PATCH", path: "/api/videos/{id}", expected: exAuth},
	{method: "DELETE", path: "/api/videos/{id}", expected: exAuth},
	// Upload-Init: RequireClubFunction("vorstand","trainer","sportliche_leitung")
	{method: "POST", path: "/api/videos", expected: exVorstandTrainer},
	// HLS-Auslieferung: public, nur per Stream-Token (?st=) — ohne Token 403 für alle.
	{method: "GET", path: "/api/videos/{id}/hls/master.m3u8", expected: exPublic},
	{method: "GET", path: "/api/videos/{id}/hls/{rendition}/{segment}", expected: exPublic},

	// ── Trainer + sportliche_leitung ────────────────────────────────────────────
	{method: "GET", path: "/api/training-series", expected: exTrainer},
	{method: "POST", path: "/api/training-series", expected: exTrainer},
	{method: "PUT", path: "/api/training-series/{id}", expected: exTrainer},
	{method: "DELETE", path: "/api/training-series/{id}", expected: exTrainer},
	{method: "POST", path: "/api/training-sessions", expected: exTrainer},
	{method: "PUT", path: "/api/training-sessions/{id}", expected: exTrainer},
	{method: "DELETE", path: "/api/training-sessions/{id}", expected: exTrainer},
	{method: "POST", path: "/api/training-sessions/{id}/attendances", expected: exTrainer},
	{method: "POST", path: "/api/games/{id}/attendances", expected: exTrainer},
	{method: "POST", path: "/api/duty-assignments/{id}/fulfill", expected: exTrainer},
	{method: "POST", path: "/api/duty-assignments/{id}/cash-substitute", expected: exTrainer},

	// ── Vorstand + Trainer + sportliche_leitung ──────────────────────────────────
	{method: "GET", path: "/api/venues", expected: exVorstandTrainer},
	{method: "POST", path: "/api/venues", expected: exVorstandTrainer},
	{method: "POST", path: "/api/venues/import", expected: exVorstandTrainer},
	{method: "DELETE", path: "/api/venues", expected: exVorstandTrainer},
	{method: "PUT", path: "/api/venues/{id}", expected: exVorstandTrainer},
	{method: "DELETE", path: "/api/venues/{id}", expected: exVorstandTrainer},
	{method: "POST", path: "/api/games", expected: exVorstandTrainer},
	{method: "PUT", path: "/api/games/{id}", expected: exVorstandTrainer},
	{method: "PUT", path: "/api/games/{id}/note", expected: exVorstandTrainer},
	{method: "DELETE", path: "/api/games/{id}", expected: exVorstandTrainer},
	{method: "PUT", path: "/api/trainings/{id}/note", expected: exVorstandTrainer},
	{method: "POST", path: "/api/duty-slots", expected: exVorstandTrainer},
	{method: "PUT", path: "/api/duty-slots/{id}", expected: exVorstandTrainer},
	{method: "DELETE", path: "/api/duty-slots/{id}", expected: exVorstandTrainer},
	{method: "POST", path: "/api/games/{id}/regenerate", expected: exVorstandTrainer},
	{method: "POST", path: "/api/games/regenerate-day", expected: exVorstandTrainer},
	{method: "POST", path: "/api/members/{id}/change-drafts/{draftId}/accept", expected: exVorstandTrainer},
	{method: "DELETE", path: "/api/members/{id}/change-drafts/{draftId}", expected: exVorstandTrainer},
	{method: "GET", path: "/api/age-class-rules", expected: exVorstandTrainer},
	// Read-only Vorstand+Trainer-Gruppe
	{method: "GET", path: "/api/duty-types", expected: exVorstandTrainer},
	{method: "GET", path: "/api/duty-templates", expected: exVorstandTrainer},
	{method: "GET", path: "/api/duty-templates/{id}", expected: exVorstandTrainer},
	{method: "GET", path: "/api/duty-templates/{id}/preview", expected: exVorstandTrainer},
	{method: "GET", path: "/api/kader", expected: exVorstandTrainer},
	{method: "POST", path: "/api/kader", expected: exVorstandTrainer},
	{method: "GET", path: "/api/kader/{id}", expected: exVorstandTrainer},
	{method: "PUT", path: "/api/kader/{id}", expected: exVorstandTrainer},
	{method: "DELETE", path: "/api/kader/{id}", expected: exVorstandTrainer},
	{method: "GET", path: "/api/kader/{id}/member-suggestions", expected: exVorstandTrainer},
	{method: "GET", path: "/api/kader/{id}/extended-member-suggestions", expected: exVorstandTrainer},
	{method: "PATCH", path: "/api/kader/{id}/games-per-season", expected: exVorstandTrainer},
	{method: "POST", path: "/api/kader/copy-from-season", expected: exVorstandTrainer},
	{method: "POST", path: "/api/kader/auto-assign", expected: exVorstandTrainer},

	// ── Saisons lesen: Vorstand/Trainer/sL + Kassierer ───────────────────────────
	{method: "GET", path: "/api/seasons", expected: exSeasonsRead},

	// ── Admin only ───────────────────────────────────────────────────────────────
	{method: "POST", path: "/api/impersonate/{id}", expected: exAdmin},

	// ── Vorstand + Kassierer ─────────────────────────────────────────────────────
	{method: "GET", path: "/api/members", expected: exMembersList},
	{method: "GET", path: "/api/members/export", expected: exVorstandKassierer},
	{method: "GET", path: "/api/members/{id}", expected: exVorstandKassierer},
	{method: "GET", path: "/api/members/{id}/parents", expected: exVorstandKassierer},
	{method: "PUT", path: "/api/members/{id}/bank-details", expected: exVorstandKassierer},
	{method: "GET", path: "/api/club", expected: exVorstandKassierer},
	{method: "PUT", path: "/api/club", expected: exVorstandKassierer},
	{method: "GET", path: "/api/admin/encryption-config", expected: exVorstandKassierer},
	{method: "PUT", path: "/api/admin/encryption-config", expected: exVorstandKassierer},
	{method: "PUT", path: "/api/admin/rotate-encryption", expected: exVorstandKassierer},
	{method: "GET", path: "/api/fee-rates", expected: exVorstandKassierer},
	{method: "POST", path: "/api/fee-rates", expected: exVorstandKassierer},
	{method: "DELETE", path: "/api/fee-rates/{id}", expected: exVorstandKassierer},
	{method: "GET", path: "/api/fee-run/preview", expected: exVorstandKassierer},
	{method: "POST", path: "/api/fee-run/export-data", expected: exVorstandKassierer},
	{method: "POST", path: "/api/fee-run/confirm", expected: exVorstandKassierer},
	{method: "GET", path: "/api/fee-run/protocol", expected: exVorstandKassierer},

	// ── Vorstand ─────────────────────────────────────────────────────────────────
	{method: "POST", path: "/api/members", expected: exVorstand},
	{method: "PUT", path: "/api/members/{id}", expected: exVorstand},
	{method: "PUT", path: "/api/members/{id}/status", expected: exVorstand},
	{method: "POST", path: "/api/seasons", expected: exVorstand},
	{method: "PUT", path: "/api/seasons/{id}", expected: exVorstand},
	{method: "PUT", path: "/api/seasons/{id}/activate", expected: exVorstand},
	{method: "DELETE", path: "/api/seasons/{id}", expected: exVorstand},
	{method: "PUT", path: "/api/seasons/{id}/duty-targets", expected: exVorstand},
	{method: "POST", path: "/api/teams", expected: exVorstand},
	{method: "PUT", path: "/api/teams/{id}", expected: exVorstand},
	{method: "GET", path: "/api/users", expected: exVorstand},
	{method: "POST", path: "/api/users", expected: exVorstand},
	{method: "PUT", path: "/api/users/{id}", expected: exVorstand},
	{method: "PUT", path: "/api/users/{id}/role", expected: exVorstand},
	{method: "PUT", path: "/api/users/{id}/recovery-email", expected: exVorstand},
	{method: "DELETE", path: "/api/users/{id}", expected: exVorstand},
	{method: "POST", path: "/api/auth/invite", expected: exVorstand},
	{method: "GET", path: "/api/invitations", expected: exVorstand},
	{method: "DELETE", path: "/api/invitations/{id}", expected: exVorstand},
	{method: "POST", path: "/api/invitations/import-csv", expected: exVorstand},
	{method: "POST", path: "/api/invitations/{id}/send", expected: exVorstand},
	{method: "PUT", path: "/api/invitations/{id}/member", expected: exVorstand},
	{method: "GET", path: "/api/membership-requests", expected: exVorstand},
	{method: "POST", path: "/api/membership-requests/{id}/approve", expected: exVorstand},
	{method: "POST", path: "/api/membership-requests/{id}/reject", expected: exVorstand},
	{method: "DELETE", path: "/api/membership-requests/{id}", expected: exVorstand},
	{method: "POST", path: "/api/members/import", expected: exVorstand},
	{method: "DELETE", path: "/api/members/{id}", expected: exVorstand},
	{method: "PUT", path: "/api/members/{id}/user", expected: exVorstand},
	{method: "POST", path: "/api/members/{id}/proxy-account", expected: exVorstand},
	{method: "POST", path: "/api/members/{id}/welcome-email", expected: exVorstand},
	{method: "POST", path: "/api/users/{id}/create-member", expected: exVorstand},
	{method: "POST", path: "/api/family-links", expected: exVorstand},
	{method: "DELETE", path: "/api/family-links", expected: exVorstand},
	{method: "POST", path: "/api/duty-types", expected: exVorstand},
	{method: "PUT", path: "/api/duty-types/{id}", expected: exVorstand},
	{method: "PUT", path: "/api/duty-types/{id}/instruction", expected: exVorstand},
	{method: "DELETE", path: "/api/duty-types/{id}", expected: exVorstand},
	{method: "GET", path: "/api/duty-accounts/export", expected: exVorstand},
	{method: "POST", path: "/api/duty-templates", expected: exVorstand},
	{method: "PUT", path: "/api/duty-templates/{id}", expected: exVorstand},
	{method: "DELETE", path: "/api/duty-templates/{id}", expected: exVorstand},
	{method: "POST", path: "/api/upload/member-photo/{id}", expected: exVorstand},
	{method: "DELETE", path: "/api/upload/member-photo/{id}", expected: exVorstand},
	{method: "POST", path: "/api/upload/sepa-mandat/{id}", expected: exVorstandKassierer},
	{method: "PUT", path: "/api/age-class-rules/{ageClass}", expected: exVorstand},
	{method: "POST", path: "/api/stammvereine", expected: exVorstand},
	{method: "PUT", path: "/api/stammvereine/{id}", expected: exVorstand},
	{method: "DELETE", path: "/api/stammvereine/{id}", expected: exVorstand},
}

// routeKey gibt den kanonischen Schlüssel für einen Endpoint zurück.
func routeKey(method, path string) string {
	return method + " " + path
}

// resolvePathParams ersetzt chi-Pfadparameter wie {id} durch "1".
func resolvePathParams(path string) string {
	result := path
	for strings.Contains(result, "{") {
		start := strings.Index(result, "{")
		end := strings.Index(result, "}")
		if start == -1 || end == -1 {
			break
		}
		result = result[:start] + "1" + result[end+1:]
	}
	return result
}

func TestPermissionMatrix_Backend(t *testing.T) {
	db := testutil.NewDB(t)
	router := prodserver.BuildRouter(t, db)

	// Drift-Check: alle registrierten Routen müssen in der Matrix stehen.
	matrixKeys := make(map[string]bool, len(matrix))
	for _, c := range matrix {
		matrixKeys[routeKey(c.method, c.path)] = true
	}

	var driftErrors []string
	chiRouter, ok := router.(chi.Routes)
	if ok {
		err := chi.Walk(chiRouter, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			// SPA-Fallback und SSE-Routen ausschließen.
			if route == "/*" || route == "/api/events" || route == "/api/chat/events" {
				return nil
			}
			key := routeKey(method, route)
			if !matrixKeys[key] {
				driftErrors = append(driftErrors, fmt.Sprintf(
					"Route %s ist nicht in der Permission-Matrix gepflegt — "+
						"bitte openspec/changes/permissions-baseline-tests/specs/permissions/spec.md "+
						"und internal/permissions/matrix_test.go ergänzen", key))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("chi.Walk Fehler: %v", err)
		}
	}
	if len(driftErrors) > 0 {
		for _, e := range driftErrors {
			t.Error(e)
		}
		t.FailNow()
	}

	// Persona-Tokens vorbereiten.
	tokens := make(map[string]string, len(Personas))
	for i, p := range Personas {
		tokens[p.ID] = testutil.TokenWithIsParent(t, i+100, p.Role, p.ClubFunctions, p.IsParent)
	}

	// Matrix-Tests: pro Endpoint × Persona.
	for _, tc := range matrix {
		tc := tc
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()
			url := resolvePathParams(tc.path)
			for _, persona := range Personas {
				persona := persona
				expected, ok := tc.expected[persona.ID]
				if !ok {
					t.Errorf("Persona %q hat keinen Eintrag in expected-Map für %s %s", persona.ID, tc.method, tc.path)
					continue
				}

				rec := httptest.NewRecorder()
				req := httptest.NewRequest(tc.method, url, nil)
				req.Header.Set("Authorization", tokens[persona.ID])
				router.ServeHTTP(rec, req)
				got := rec.Code

				switch expected {
				case httpAnyOK:
					// Jeder Response-Code ist akzeptabel.
				case httpAllowed:
					// Persona darf den Endpoint erreichen: nicht 401 oder 403 vom Middleware-Layer.
					if got == http.StatusUnauthorized || got == http.StatusForbidden {
						t.Errorf("Persona %q: %s %s: erwartet erlaubt (nicht 401/403), bekommen %d",
							persona.ID, tc.method, tc.path, got)
					}
				default:
					// Exakter Status-Code erwartet.
					if got != expected {
						t.Errorf("Persona %q: %s %s: erwartet %d, bekommen %d",
							persona.ID, tc.method, tc.path, expected, got)
					}
				}
			}
		})
	}
}
