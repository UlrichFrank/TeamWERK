package app

import (
	"database/sql"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/teamstuttgart/teamwerk/internal/absences"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/carpooling"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/dashboard"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/files"
	"github.com/teamstuttgart/teamwerk/internal/games"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/kader"
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
	"github.com/teamstuttgart/teamwerk/internal/teams"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
	"github.com/teamstuttgart/teamwerk/internal/upload"
	"github.com/teamstuttgart/teamwerk/internal/venues"
)

// Handlers holds all HTTP handler instances needed to build the router.
type Handlers struct {
	Auth         *auth.Handler
	Config       *appconfig.Handler
	Members      *members.Handler
	WelcomeEmail *members.WelcomeEmailHandler
	Duties       *duties.Handler
	Dashboard    *dashboard.Handler
	Games        *games.Handler
	Kader        *kader.Handler
	Upload       *upload.Handler
	Files        *files.Handler
	Carpool      *carpooling.Handler
	Chat         *chat.Handler
	Notif        *notifications.Handler
	Training     *trainings.Handler
	Absences     *absences.Handler
	Teams        *teams.Handler
	Venues       *venues.Handler
	Hub          *hub.Handler

	JWTSecret string
	Database  *sql.DB
	BaseURL   string
}

// BuildRouter wires all routes, middleware, and handlers.
// spaFS may be nil (e.g. in tests) — the SPA fallback is then skipped.
// CORS middleware is only added when BaseURL is non-empty.
func BuildRouter(h *Handlers, spaFS fs.FS) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	if h.BaseURL != "" {
		r.Use(corsMiddleware(h.BaseURL))
	}

	// Public routes
	r.Get("/api/uploads/*", h.Upload.ServeUpload)
	r.Get("/api/files/{id}/download", h.Files.DownloadFile)
	r.Get("/api/members/{id}/sepa-mandat/download", h.Upload.SepaDownload)
	r.Post("/api/auth/login", h.Auth.Login)
	r.Post("/api/auth/refresh", h.Auth.Refresh)
	r.Post("/api/auth/logout", h.Auth.Logout)
	r.Post("/api/auth/request-membership", h.Auth.RequestMembership)
	r.Post("/api/auth/register", h.Auth.Register)
	r.Get("/api/auth/token-info", h.Auth.GetTokenInfo)
	r.Post("/api/auth/forgot-password", h.Auth.ForgotPassword)
	r.Post("/api/auth/reset-password", h.Auth.ResetPassword)
	r.Get("/api/profile/email/confirm", h.Auth.ConfirmEmailChange)

	// SSE — cookie-authenticated (EventSource cannot send headers)
	r.Group(func(r chi.Router) {
		r.Use(auth.CookieMiddleware(h.Database))
		r.Get("/api/events", h.Hub.Events)
		r.Get("/api/chat/events", h.Chat.ChatEvents)
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(h.JWTSecret))

		// Chat
		r.Get("/api/chat/users", h.Chat.Users)
		r.Get("/api/chat/conversations", h.Chat.ListConversations)
		r.Post("/api/chat/conversations", h.Chat.CreateConversation)
		r.Get("/api/chat/conversations/{id}/messages", h.Chat.ListMessages)
		r.Post("/api/chat/conversations/{id}/messages", h.Chat.SendMessage)
		r.Post("/api/chat/conversations/{id}/read", h.Chat.MarkRead)
		r.Delete("/api/chat/conversations/{id}/members/me", h.Chat.LeaveConversation)
		r.Delete("/api/chat/conversations/{id}/members/{uid}", h.Chat.RemoveMember)
		r.Delete("/api/chat/conversations/{id}/everyone", h.Chat.DeleteConversationForEveryone)
		r.Put("/api/chat/conversations/{id}", h.Chat.UpdateConversation)
		r.Post("/api/chat/conversations/{id}/transfer-ownership", h.Chat.TransferOwnership)
		r.Delete("/api/chat/conversations/{id}", h.Chat.DeleteConversation)
		r.Post("/api/chat/conversations/{id}/members", h.Chat.AddMember)
		r.Put("/api/chat/messages/{id}", h.Chat.EditMessage)
		r.Delete("/api/chat/messages/{id}", h.Chat.DeleteMessage)
		r.Post("/api/chat/messages/{id}/reactions", h.Chat.ToggleReaction)
		r.Get("/api/chat/broadcasts", h.Chat.ListBroadcasts)
		r.Post("/api/chat/broadcasts", h.Chat.SendBroadcast)
		r.Post("/api/chat/broadcasts/{id}/read", h.Chat.MarkBroadcastRead)
		r.Put("/api/chat/broadcasts/{id}", h.Chat.EditBroadcast)
		r.Delete("/api/chat/broadcasts/{id}", h.Chat.DeleteBroadcast)

		// Members
		r.Get("/api/users/{id}/contact", h.Members.GetContact)
		r.Get("/api/members/{id}/change-drafts", h.Members.GetChangeRequestsHandler)
		r.Post("/api/members/{id}/change-request", h.Members.CreateChangeRequestHandler)
		r.Get("/api/members/{id}/sepa-mandat/download-token", h.Upload.SepaDownloadToken)
		r.Delete("/api/members/{id}/sepa-mandat", h.Upload.DeleteSepaMandat)
		r.Get("/api/profile/me", h.Members.GetProfile)
		r.Put("/api/profile/me", h.Members.UpdateProfile)
		r.Get("/api/profile/vehicle", h.Members.GetVehicle)
		r.Put("/api/profile/vehicle", h.Members.UpdateVehicle)
		r.Get("/api/profile/account", h.Auth.GetAccount)
		r.Put("/api/profile/account", h.Auth.UpdateAccount)
		r.Post("/api/profile/password", h.Auth.ChangePassword)
		r.Post("/api/profile/email", h.Auth.RequestEmailChange)
		r.Post("/api/profile/phones", h.Members.AddPhone)
		r.Put("/api/profile/phones/{id}", h.Members.UpdatePhone)
		r.Delete("/api/profile/phones/{id}", h.Members.DeletePhone)
		r.Put("/api/profile/visibility", h.Members.UpdateVisibility)
		r.Put("/api/profile/reminder-preference", h.Members.UpdateReminderPreference)
		r.Put("/api/profile/absence-visibility", h.Members.UpdateAbsenceVisibility)
		r.Get("/api/absences/preview", h.Absences.Preview)
		r.Get("/api/absences/calendar", h.Absences.Calendar)
		r.Get("/api/absences", h.Absences.List)
		r.Post("/api/absences", h.Absences.Create)
		r.Put("/api/absences/{id}", h.Absences.Update)
		r.Delete("/api/absences/{id}", h.Absences.Delete)
		r.Get("/api/family/proxy-accounts", h.Members.GetFamilyProxyAccounts)
		r.Get("/api/profile/kind/{memberId}", h.Members.GetChildProfile)
		r.Put("/api/profile/kind/{memberId}/account", h.Members.UpdateChildAccount)
		r.Put("/api/profile/kind/{memberId}/member", h.Members.UpdateChildMember)
		r.Put("/api/profile/kind/{memberId}/bank", h.Members.UpdateChildBank)
		r.Post("/api/profile/kind/{memberId}/photo", h.Upload.UploadChildPhoto)
		r.Delete("/api/profile/kind/{memberId}/photo", h.Upload.DeleteChildPhoto)
		r.Post("/api/profile/kind/{memberId}/phones", h.Members.AddChildPhone)
		r.Delete("/api/profile/kind/{memberId}/phones/{phoneId}", h.Members.DeleteChildPhone)
		r.Put("/api/profile/kind/{memberId}/visibility", h.Members.UpdateChildVisibility)
		r.Post("/api/upload/user-photo", h.Upload.UploadUserPhoto)
		r.Delete("/api/upload/user-photo", h.Upload.DeleteUserPhoto)

		// Dashboard
		r.Get("/api/dashboard", h.Dashboard.Get)

		// Duties
		r.Get("/api/duty-board", h.Duties.Board)
		r.Post("/api/duty-board/{slotId}/claim", h.Duties.Claim)
		r.Delete("/api/duty-board/{slotId}/claim", h.Duties.Unclaim)
		r.Get("/api/duty-accounts", h.Duties.Accounts)
		r.Get("/api/duty-slots", h.Duties.ListSlots)
		r.Get("/api/duty-slots/{id}/assignments", h.Duties.ListAssignments)

		// Mitfahrgelegenheiten
		r.Get("/api/mitfahrgelegenheiten", h.Carpool.List)
		r.Post("/api/mitfahrgelegenheiten", h.Carpool.Upsert)
		r.Delete("/api/mitfahrgelegenheiten/{id}", h.Carpool.Delete)
		r.Post("/api/mitfahrt-paarungen", h.Carpool.RequestPairing)
		r.Post("/api/mitfahrt-paarungen/{id}/confirm", h.Carpool.ConfirmPairing)
		r.Post("/api/mitfahrt-paarungen/{id}/reject", h.Carpool.RejectPairing)

		// Push Notifications
		r.Get("/api/push/vapid-public-key", h.Notif.GetVAPIDPublicKey)
		r.Post("/api/push/subscribe", h.Notif.Subscribe)
		r.Delete("/api/push/subscribe", h.Notif.Unsubscribe)
		r.Get("/api/profile/notification-preferences", h.Notif.GetNotificationPreferences)
		r.Put("/api/profile/notification-preferences", h.Notif.UpdateNotificationPreferences)

		r.Get("/api/users/picker", h.Auth.UsersPicker)

		// Dokumente
		r.Get("/api/folders", h.Files.ListRootFolders)
		r.Post("/api/folders", h.Files.CreateFolder)
		r.Get("/api/folders/{id}/contents", h.Files.FolderContents)
		r.Put("/api/folders/{id}", h.Files.RenameFolder)
		r.Delete("/api/folders/{id}", h.Files.DeleteFolder)
		r.Get("/api/folders/{id}/permissions", h.Files.ListPermissions)
		r.Post("/api/folders/{id}/permissions", h.Files.AddPermission)
		r.Delete("/api/folders/{id}/permissions/{permId}", h.Files.DeletePermission)
		r.Post("/api/folders/{folderId}/files", h.Files.UploadFile)
		r.Get("/api/files/{id}/download-token", h.Files.HandleDownloadToken)
		r.Put("/api/files/{id}", h.Files.RenameFile)
		r.Delete("/api/files/{id}", h.Files.DeleteFile)

		// Games (read + RSVP — all authenticated)
		r.Get("/api/games", h.Games.ListGames)
		r.Get("/api/games/{id}", h.Games.GetGame)
		r.Get("/api/games/my", h.Games.ListMyGames)
		r.Post("/api/games/{id}/respond", h.Games.RespondToGame)
		r.Get("/api/games/{id}/responses", h.Games.ListGameResponses)
		r.Get("/api/games/{id}/participants", h.Games.GetParticipants)
		r.Post("/api/games/{id}/lineup", h.Games.SaveLineup)

		// Trainings (read + RSVP — all authenticated)
		r.Get("/api/training-sessions", h.Training.ListSessions)
		r.Get("/api/training-sessions/{id}", h.Training.GetSession)
		r.Post("/api/training-sessions/{id}/respond", h.Training.Respond)
		r.Get("/api/training-sessions/{id}/attendances", h.Training.GetAttendances)

		// Teams
		r.Get("/api/teams", h.Games.ListTeamsForUser)
		r.Get("/api/teams/names", h.Games.ListTeamNames)
		r.Get("/api/teams/my", h.Teams.ListMyTeams)
		r.Get("/api/teams/{id}/roster", h.Teams.GetRoster)

		// Trainer + sportliche_leitung
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("trainer", "sportliche_leitung"))
			r.Get("/api/venues", h.Venues.List)
			r.Get("/api/training-series", h.Training.ListSeries)
			r.Post("/api/training-series", h.Training.CreateSeries)
			r.Put("/api/training-series/{id}", h.Training.UpdateSeries)
			r.Delete("/api/training-series/{id}", h.Training.DeleteSeries)
			r.Post("/api/training-sessions", h.Training.CreateSession)
			r.Put("/api/training-sessions/{id}", h.Training.UpdateSession)
			r.Delete("/api/training-sessions/{id}", h.Training.DeleteSession)
			r.Post("/api/training-sessions/{id}/attendances", h.Training.SaveAttendances)
			r.Post("/api/duty-slots", h.Duties.CreateSlot)
			r.Put("/api/duty-slots/{id}", h.Duties.UpdateSlot)
			r.Delete("/api/duty-slots/{id}", h.Duties.DeleteSlot)
			r.Post("/api/duty-assignments/{id}/fulfill", h.Duties.Fulfill)
			r.Post("/api/duty-assignments/{id}/cash-substitute", h.Duties.CashSubstitute)
			r.Get("/api/membership-requests", h.Auth.ListMembershipRequests)
			r.Post("/api/membership-requests/{id}/approve", h.Auth.ApproveMembershipRequest)
			r.Post("/api/membership-requests/{id}/reject", h.Auth.RejectMembershipRequest)
			r.Delete("/api/membership-requests/{id}", h.Auth.DeleteMembershipRequest)
			r.Post("/api/auth/invite", h.Auth.Invite)
		})

		// Vorstand + Trainer + sportliche_leitung
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Post("/api/venues", h.Venues.Create)
			r.Post("/api/venues/import", h.Venues.Import)
			r.Delete("/api/venues", h.Venues.DeleteAll)
			r.Put("/api/venues/{id}", h.Venues.Update)
			r.Delete("/api/venues/{id}", h.Venues.Delete)
			r.Post("/api/games", h.Games.CreateGame)
			r.Put("/api/games/{id}", h.Games.UpdateGame)
			r.Delete("/api/games/{id}", h.Games.DeleteGame)
			r.Post("/api/games/{id}/regenerate", h.Games.RegenerateSlots)
			r.Post("/api/games/regenerate-day", h.Games.RegenerateDaySlots)
			r.Post("/api/members/{id}/change-drafts/{draftId}/accept", h.Members.AcceptChangeRequestHandler)
			r.Delete("/api/members/{id}/change-drafts/{draftId}", h.Members.RejectChangeRequestHandler)
			r.Get("/api/age-class-rules", h.Config.GetAgeClassRulesHandler)
		})

		// Admin only
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin"))
			r.Post("/api/impersonate/{id}", h.Auth.Impersonate)
		})

		// Vorstand
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand"))
			r.Get("/api/members", h.Members.List)
			r.Get("/api/members/export", h.Members.Export)
			r.Get("/api/members/{id}", h.Members.Get)
			r.Post("/api/members", h.Members.Create)
			r.Put("/api/members/{id}", h.Members.Update)
			r.Put("/api/members/{id}/status", h.Members.UpdateStatus)
			r.Get("/api/club", h.Config.GetClub)
			r.Put("/api/club", h.Config.UpdateClub)
			r.Post("/api/seasons", h.Config.CreateSeason)
			r.Put("/api/seasons/{id}", h.Config.UpdateSeason)
			r.Put("/api/seasons/{id}/activate", h.Config.ActivateSeason)
			r.Delete("/api/seasons/{id}", h.Config.DeleteSeason)
			r.Put("/api/seasons/{id}/duty-targets", h.Duties.SetSeasonTargets)
			r.Post("/api/teams", h.Config.CreateTeam)
			r.Put("/api/teams/{id}", h.Config.UpdateTeam)
			r.Get("/api/users", h.Auth.ListUsers)
			r.Post("/api/users", h.Auth.CreateUser)
			r.Put("/api/users/{id}", h.Auth.UpdateUser)
			r.Put("/api/users/{id}/role", h.Auth.UpdateUserRole)
			r.Delete("/api/users/{id}", h.Auth.DeleteUser)
			r.Get("/api/invitations", h.Auth.ListInvitations)
			r.Delete("/api/invitations/{id}", h.Auth.DeleteInvitation)
			r.Post("/api/invitations/import-csv", h.Auth.ImportCSV)
			r.Post("/api/invitations/{id}/send", h.Auth.SendInvitation)
			r.Put("/api/invitations/{id}/member", h.Auth.LinkInvitationMember)
			r.Post("/api/members/import", h.Members.Import)
			r.Delete("/api/members/{id}", h.Members.DeleteMember)
			r.Put("/api/members/{id}/user", h.Members.LinkUser)
			r.Post("/api/members/{id}/proxy-account", h.Members.CreateProxyAccount)
			r.Post("/api/members/{id}/welcome-email", h.WelcomeEmail.Send)
			r.Get("/api/members/{id}/parents", h.Members.GetMemberParents)
			r.Post("/api/users/{id}/create-member", h.Members.CreateMemberFromUser)
			r.Post("/api/family-links", h.Members.CreateFamilyLink)
			r.Delete("/api/family-links", h.Members.DeleteFamilyLink)
			r.Post("/api/duty-types", h.Duties.CreateType)
			r.Put("/api/duty-types/{id}", h.Duties.UpdateType)
			r.Delete("/api/duty-types/{id}", h.Duties.DeleteType)
			r.Get("/api/duty-accounts/export", h.Duties.ExportAccounts)
			r.Post("/api/duty-templates", h.Games.CreateTemplate)
			r.Put("/api/duty-templates/{id}", h.Games.UpdateTemplate)
			r.Delete("/api/duty-templates/{id}", h.Games.DeleteTemplate)
			r.Post("/api/upload/member-photo/{id}", h.Upload.UploadMemberPhoto)
			r.Delete("/api/upload/member-photo/{id}", h.Upload.DeleteMemberPhoto)
			r.Post("/api/upload/sepa-mandat/{id}", h.Upload.UploadSepaMandat)
			r.Put("/api/age-class-rules/{ageClass}", h.Config.UpdateAgeClassRuleHandler)
		})

		// Vorstand + Trainer + sportliche_leitung (read-only)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Get("/api/duty-types", h.Duties.ListTypes)
			r.Get("/api/duty-templates", h.Games.ListTemplates)
			r.Get("/api/duty-templates/{id}", h.Games.GetTemplateByID)
			r.Get("/api/duty-templates/{id}/preview", h.Games.PreviewSlots)
			r.Get("/api/seasons", h.Config.ListSeasons)
			r.Get("/api/kader", h.Kader.ListKader)
			r.Post("/api/kader", h.Kader.InitializeKader)
			r.Get("/api/kader/{id}", h.Kader.GetKader)
			r.Put("/api/kader/{id}", h.Kader.UpdateKader)
			r.Delete("/api/kader/{id}", h.Kader.DeleteKader)
			r.Get("/api/kader/{id}/member-suggestions", h.Kader.MemberSuggestions)
			r.Get("/api/kader/{id}/extended-member-suggestions", h.Kader.ExtendedMemberSuggestions)
			r.Patch("/api/kader/{id}/games-per-season", h.Kader.PatchGamesPerSeason)
			r.Post("/api/kader/copy-from-season", h.Kader.CopyFromSeason)
			r.Post("/api/kader/auto-assign", h.Kader.AutoAssign)
		})
	})

	if spaFS != nil {
		r.Get("/*", spaFallback(spaFS))
	}

	return r
}

func corsMiddleware(baseURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", baseURL)
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func spaFallback(static fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(static))
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = path[1:]
		}
		if _, err := fs.Stat(static, path); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}
