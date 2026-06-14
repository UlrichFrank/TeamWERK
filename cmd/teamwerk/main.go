package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	webpush "github.com/SherClockHolmes/webpush-go"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/carpooling"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/files"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/dashboard"
	"github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/absences"
	"github.com/teamstuttgart/teamwerk/internal/games"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/kader"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
	"github.com/teamstuttgart/teamwerk/internal/scheduler"
	"github.com/teamstuttgart/teamwerk/internal/teams"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
	"github.com/teamstuttgart/teamwerk/internal/upload"
	"github.com/teamstuttgart/teamwerk/internal/venues"
)

//go:embed all:web/dist
var webFS embed.FS

var buildHash = "dev"

func main() {
	_ = godotenv.Load()

	if len(os.Args) > 1 && os.Args[1] == "scheduler:run" {
		runScheduler()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if len(os.Args) > 2 && os.Args[2] == "force" {
			runMigrateForce()
		} else {
			runMigrate()
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "create-admin" {
		runCreateAdmin()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "gen-vapid" {
		runGenVapid()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "push-test" {
		runPushTest()
		return
	}

	serve()
}

func serve() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatal(err)
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	m := mailer.New(cfg.SMTP, cfg.BaseURL, cfg.MailerDisabled)
	hubInstance := hub.NewHub()
	hubH := hub.NewHandler(hubInstance, buildHash)
	authH := auth.NewHandler(database, cfg, cfg.JWTSecret, m, cfg.BaseURL)
	cfgH := appconfig.NewHandler(database, hubInstance)
	membH := members.NewHandler(database, hubInstance)
	dutyH := duties.NewHandler(database, cfg, hubInstance)
	dashH := dashboard.NewHandler(database)
	gameH := games.NewHandler(database, cfg, hubInstance)
	kaderH := kader.NewHandler(database, hubInstance)
	uploadH := upload.NewHandler(database, cfg.UploadDir, cfg.JWTSecret)
	filesH := files.NewHandler(database, cfg.FilesDir, cfg.JWTSecret)
	carpoolH := carpooling.NewHandler(database, cfg, hubInstance)
	chatH := chat.NewHandler(database, hubInstance, cfg)
	notifH := notifications.NewHandler(database, cfg)
	welcomeH := members.NewWelcomeEmailHandler(database, m)
	trainingH := trainings.NewHandler(database, cfg, hubInstance)
	absenceH := absences.NewHandler(database, hubInstance)
	teamsH := teams.NewHandler(database)
	venueH := venues.NewHandler(database, hubInstance)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(corsMiddleware(cfg.BaseURL))

	// Public routes
	r.Get("/api/uploads/*", uploadH.ServeUpload)
	r.Get("/api/files/{id}/download", filesH.DownloadFile)                     // token-auth handled internally
	r.Get("/api/members/{id}/sepa-mandat/download", uploadH.SepaDownload) // token-auth handled internally
	r.Post("/api/auth/login", authH.Login)
	r.Post("/api/auth/refresh", authH.Refresh)
	r.Post("/api/auth/logout", authH.Logout)
	r.Post("/api/auth/request-membership", authH.RequestMembership)
	r.Post("/api/auth/register", authH.Register)
	r.Get("/api/auth/token-info", authH.GetTokenInfo)
	r.Post("/api/auth/forgot-password", authH.ForgotPassword)
	r.Post("/api/auth/reset-password", authH.ResetPassword)
	r.Get("/api/profile/email/confirm", authH.ConfirmEmailChange)

	// SSE endpoints — authenticated via HttpOnly refresh-token cookie (EventSource cannot send headers)
	r.Group(func(r chi.Router) {
		r.Use(auth.CookieMiddleware(database))
		r.Get("/api/events", hubH.Events)
		r.Get("/api/chat/events", chatH.ChatEvents)
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))

		// Chat
		r.Get("/api/chat/users", chatH.Users)
		r.Get("/api/chat/conversations", chatH.ListConversations)
		r.Post("/api/chat/conversations", chatH.CreateConversation)
		r.Get("/api/chat/conversations/{id}/messages", chatH.ListMessages)
		r.Post("/api/chat/conversations/{id}/messages", chatH.SendMessage)
		r.Post("/api/chat/conversations/{id}/read", chatH.MarkRead)
		r.Delete("/api/chat/conversations/{id}/members/me", chatH.LeaveConversation)
		r.Delete("/api/chat/conversations/{id}/members/{uid}", chatH.RemoveMember)
		r.Delete("/api/chat/conversations/{id}/everyone", chatH.DeleteConversationForEveryone)
		r.Put("/api/chat/conversations/{id}", chatH.UpdateConversation)
		r.Post("/api/chat/conversations/{id}/transfer-ownership", chatH.TransferOwnership)
		r.Delete("/api/chat/conversations/{id}", chatH.DeleteConversation)
		r.Post("/api/chat/conversations/{id}/members", chatH.AddMember)
		r.Put("/api/chat/messages/{id}", chatH.EditMessage)
		r.Delete("/api/chat/messages/{id}", chatH.DeleteMessage)
		r.Post("/api/chat/messages/{id}/reactions", chatH.ToggleReaction)
		r.Get("/api/chat/broadcasts", chatH.ListBroadcasts)
		r.Post("/api/chat/broadcasts", chatH.SendBroadcast)
		r.Post("/api/chat/broadcasts/{id}/read", chatH.MarkBroadcastRead)
		r.Put("/api/chat/broadcasts/{id}", chatH.EditBroadcast)
		r.Delete("/api/chat/broadcasts/{id}", chatH.DeleteBroadcast)

		// Members
		r.Get("/api/users/{id}/contact", membH.GetContact)
		r.Get("/api/members/{id}/change-drafts", membH.GetChangeRequestsHandler)
		r.Post("/api/members/{id}/change-request", membH.CreateChangeRequestHandler)
		r.Get("/api/members/{id}/sepa-mandat/download-token", uploadH.SepaDownloadToken)
		r.Delete("/api/members/{id}/sepa-mandat", uploadH.DeleteSepaMandat)
		r.Get("/api/profile/me", membH.GetProfile)
		r.Put("/api/profile/me", membH.UpdateProfile)
		r.Get("/api/profile/vehicle", membH.GetVehicle)
		r.Put("/api/profile/vehicle", membH.UpdateVehicle)
		r.Get("/api/profile/account", authH.GetAccount)
		r.Put("/api/profile/account", authH.UpdateAccount)
		r.Post("/api/profile/password", authH.ChangePassword)
		r.Post("/api/profile/email", authH.RequestEmailChange)
		r.Post("/api/profile/phones", membH.AddPhone)
		r.Put("/api/profile/phones/{id}", membH.UpdatePhone)
		r.Delete("/api/profile/phones/{id}", membH.DeletePhone)
		r.Put("/api/profile/visibility", membH.UpdateVisibility)
		r.Put("/api/profile/reminder-preference", membH.UpdateReminderPreference)
		r.Put("/api/profile/absence-visibility", membH.UpdateAbsenceVisibility)
		r.Get("/api/absences/preview", absenceH.Preview)
		r.Get("/api/absences/calendar", absenceH.Calendar)
		r.Get("/api/absences", absenceH.List)
		r.Post("/api/absences", absenceH.Create)
		r.Put("/api/absences/{id}", absenceH.Update)
		r.Delete("/api/absences/{id}", absenceH.Delete)
		r.Get("/api/family/proxy-accounts", membH.GetFamilyProxyAccounts)
		r.Get("/api/profile/kind/{memberId}", membH.GetChildProfile)
		r.Put("/api/profile/kind/{memberId}/account", membH.UpdateChildAccount)
		r.Put("/api/profile/kind/{memberId}/member", membH.UpdateChildMember)
		r.Put("/api/profile/kind/{memberId}/bank", membH.UpdateChildBank)
		r.Post("/api/profile/kind/{memberId}/photo", uploadH.UploadChildPhoto)
		r.Delete("/api/profile/kind/{memberId}/photo", uploadH.DeleteChildPhoto)
		r.Post("/api/profile/kind/{memberId}/phones", membH.AddChildPhone)
		r.Delete("/api/profile/kind/{memberId}/phones/{phoneId}", membH.DeleteChildPhone)
		r.Put("/api/profile/kind/{memberId}/visibility", membH.UpdateChildVisibility)
		r.Post("/api/upload/user-photo", uploadH.UploadUserPhoto)
		r.Delete("/api/upload/user-photo", uploadH.DeleteUserPhoto)
		// Dashboard
		r.Get("/api/dashboard", dashH.Get)

		// Duties
		r.Get("/api/duty-board", dutyH.Board)
		r.Post("/api/duty-board/{slotId}/claim", dutyH.Claim)
		r.Delete("/api/duty-board/{slotId}/claim", dutyH.Unclaim)
		r.Get("/api/duty-accounts", dutyH.Accounts)
		r.Get("/api/duty-slots", dutyH.ListSlots)
		r.Get("/api/duty-slots/{id}/assignments", dutyH.ListAssignments)

		// Mitfahrgelegenheiten
		r.Get("/api/mitfahrgelegenheiten", carpoolH.List)
		r.Post("/api/mitfahrgelegenheiten", carpoolH.Upsert)
		r.Delete("/api/mitfahrgelegenheiten/{id}", carpoolH.Delete)
		r.Post("/api/mitfahrt-paarungen", carpoolH.RequestPairing)
		r.Post("/api/mitfahrt-paarungen/{id}/confirm", carpoolH.ConfirmPairing)
		r.Post("/api/mitfahrt-paarungen/{id}/reject", carpoolH.RejectPairing)

		// Push Notifications
		r.Get("/api/push/vapid-public-key", notifH.GetVAPIDPublicKey)
		r.Post("/api/push/subscribe", notifH.Subscribe)
		r.Delete("/api/push/subscribe", notifH.Unsubscribe)
			r.Get("/api/profile/notification-preferences", notifH.GetNotificationPreferences)
			r.Put("/api/profile/notification-preferences", notifH.UpdateNotificationPreferences)

		r.Get("/api/users/picker", authH.UsersPicker)

		// Dokumente
		r.Get("/api/folders", filesH.ListRootFolders)
		r.Post("/api/folders", filesH.CreateFolder)
		r.Get("/api/folders/{id}/contents", filesH.FolderContents)
		r.Put("/api/folders/{id}", filesH.RenameFolder)
		r.Delete("/api/folders/{id}", filesH.DeleteFolder)
		r.Get("/api/folders/{id}/permissions", filesH.ListPermissions)
		r.Post("/api/folders/{id}/permissions", filesH.AddPermission)
		r.Delete("/api/folders/{id}/permissions/{permId}", filesH.DeletePermission)
		r.Post("/api/folders/{folderId}/files", filesH.UploadFile)
		r.Get("/api/files/{id}/download-token", filesH.HandleDownloadToken)
		r.Put("/api/files/{id}", filesH.RenameFile)
		r.Delete("/api/files/{id}", filesH.DeleteFile)

		// Kalender
		r.Get("/api/kalender", gameH.ListGames)
		r.Get("/api/kalender/{id}", gameH.GetGame)

		// Trainings (read + RSVP for all authenticated users)
		r.Get("/api/training-sessions", trainingH.ListSessions)
		r.Get("/api/training-sessions/{id}", trainingH.GetSession)
		r.Post("/api/training-sessions/{id}/respond", trainingH.Respond)
		r.Get("/api/training-sessions/{id}/attendances", trainingH.GetAttendances)

		// Games RSVP (user-facing; /my must come before /{id})
		r.Get("/api/games/my", gameH.ListMyGames)
		r.Post("/api/games/{id}/respond", gameH.RespondToGame)
		r.Get("/api/games/{id}/responses", gameH.ListGameResponses)
		r.Get("/api/games/{id}/participants", gameH.GetParticipants)
		r.Post("/api/games/{id}/lineup", gameH.SaveLineup)

		// Teams
		r.Get("/api/teams", gameH.ListTeamsForUser)
		r.Get("/api/teams/names", gameH.ListTeamNames)
		r.Get("/api/teams/my", teamsH.ListMyTeams)
		r.Get("/api/teams/{id}/roster", teamsH.GetRoster)

		// Trainer + sportliche_leitung
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("trainer", "sportliche_leitung"))
			// Venues (read)
			r.Get("/api/venues", venueH.List)
			// Trainings management
			r.Get("/api/training-series", trainingH.ListSeries)
			r.Post("/api/training-series", trainingH.CreateSeries)
			r.Put("/api/training-series/{id}", trainingH.UpdateSeries)
			r.Delete("/api/training-series/{id}", trainingH.DeleteSeries)
			r.Post("/api/training-sessions", trainingH.CreateSession)
			r.Put("/api/training-sessions/{id}", trainingH.UpdateSession)
			r.Delete("/api/training-sessions/{id}", trainingH.DeleteSession)
			r.Post("/api/training-sessions/{id}/attendances", trainingH.SaveAttendances)
			r.Post("/api/duty-slots", dutyH.CreateSlot)
			r.Put("/api/duty-slots/{id}", dutyH.UpdateSlot)
			r.Delete("/api/duty-slots/{id}", dutyH.DeleteSlot)
			r.Post("/api/duty-assignments/{id}/fulfill", dutyH.Fulfill)
			r.Post("/api/duty-assignments/{id}/cash-substitute", dutyH.CashSubstitute)
			r.Get("/api/membership-requests", authH.ListMembershipRequests)
			r.Post("/api/membership-requests/{id}/approve", authH.ApproveMembershipRequest)
			r.Post("/api/membership-requests/{id}/reject", authH.RejectMembershipRequest)
			r.Delete("/api/membership-requests/{id}", authH.DeleteMembershipRequest)
			r.Post("/api/auth/invite", authH.Invite)
		})

		// Vorstand + Trainer + sportliche_leitung
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			// Venues (write)
			r.Post("/api/venues", venueH.Create)
			r.Post("/api/venues/import", venueH.Import)
			r.Delete("/api/venues", venueH.DeleteAll)
			r.Put("/api/venues/{id}", venueH.Update)
			r.Delete("/api/venues/{id}", venueH.Delete)
			r.Post("/api/kalender", gameH.CreateGame)
			r.Put("/api/kalender/{id}", gameH.UpdateGame)
			r.Delete("/api/kalender/{id}", gameH.DeleteGame)
			r.Post("/api/kalender/{id}/regenerate", gameH.RegenerateSlots)
			r.Post("/api/kalender/regenerate-day", gameH.RegenerateDaySlots)
			r.Post("/api/members/{id}/change-drafts/{draftId}/accept", membH.AcceptChangeRequestHandler)
			r.Delete("/api/members/{id}/change-drafts/{draftId}", membH.RejectChangeRequestHandler)
			r.Get("/api/age-class-rules", cfgH.GetAgeClassRulesHandler)
		})

		// Admin only
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin"))
			r.Post("/api/impersonate/{id}", authH.Impersonate)
		})

		// Vorstand
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand"))
			r.Get("/api/members", membH.List)
			r.Get("/api/members/export", membH.Export)
			r.Get("/api/members/{id}", membH.Get)
			r.Post("/api/members", membH.Create)
			r.Put("/api/members/{id}", membH.Update)
			r.Put("/api/members/{id}/status", membH.UpdateStatus)
			r.Get("/api/club", cfgH.GetClub)
			r.Put("/api/club", cfgH.UpdateClub)
			r.Post("/api/seasons", cfgH.CreateSeason)
			r.Put("/api/seasons/{id}", cfgH.UpdateSeason)
			r.Put("/api/seasons/{id}/activate", cfgH.ActivateSeason)
			r.Delete("/api/seasons/{id}", cfgH.DeleteSeason)
			r.Put("/api/seasons/{id}/duty-targets", dutyH.SetSeasonTargets)
			r.Post("/api/teams", cfgH.CreateTeam)
			r.Put("/api/teams/{id}", cfgH.UpdateTeam)
			r.Get("/api/users", authH.ListUsers)
			r.Post("/api/users", authH.CreateUser)
			r.Put("/api/users/{id}", authH.UpdateUser)
			r.Put("/api/users/{id}/role", authH.UpdateUserRole)
			r.Delete("/api/users/{id}", authH.DeleteUser)
			r.Get("/api/invitations", authH.ListInvitations)
			r.Delete("/api/invitations/{id}", authH.DeleteInvitation)
			r.Post("/api/invitations/import-csv", authH.ImportCSV)
			r.Post("/api/invitations/{id}/send", authH.SendInvitation)
			r.Put("/api/invitations/{id}/member", authH.LinkInvitationMember)
			r.Post("/api/members/import", membH.Import)
			r.Delete("/api/members/{id}", membH.DeleteMember)
			r.Put("/api/members/{id}/user", membH.LinkUser)
			r.Post("/api/members/{id}/proxy-account", membH.CreateProxyAccount)
			r.Post("/api/members/{id}/welcome-email", welcomeH.Send)
			r.Get("/api/members/{id}/parents", membH.GetMemberParents)
			r.Post("/api/users/{id}/create-member", membH.CreateMemberFromUser)
			r.Post("/api/family-links", membH.CreateFamilyLink)
			r.Delete("/api/family-links", membH.DeleteFamilyLink)
			r.Post("/api/duty-types", dutyH.CreateType)
			r.Put("/api/duty-types/{id}", dutyH.UpdateType)
			r.Delete("/api/duty-types/{id}", dutyH.DeleteType)
			r.Get("/api/duty-accounts/export", dutyH.ExportAccounts)
			r.Post("/api/duty-templates", gameH.CreateTemplate)
			r.Put("/api/duty-templates/{id}", gameH.UpdateTemplate)
			r.Delete("/api/duty-templates/{id}", gameH.DeleteTemplate)
			r.Post("/api/upload/member-photo/{id}", uploadH.UploadMemberPhoto)
			r.Delete("/api/upload/member-photo/{id}", uploadH.DeleteMemberPhoto)
			r.Post("/api/upload/sepa-mandat/{id}", uploadH.UploadSepaMandat)
			r.Put("/api/age-class-rules/{ageClass}", cfgH.UpdateAgeClassRuleHandler)
		})

		// Vorstand + Trainer + sportliche_leitung
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Get("/api/duty-types", dutyH.ListTypes)
			r.Get("/api/duty-templates", gameH.ListTemplates)
			r.Get("/api/duty-templates/{id}", gameH.GetTemplateByID)
			r.Get("/api/duty-templates/{id}/preview", gameH.PreviewSlots)
			r.Get("/api/seasons", cfgH.ListSeasons)
			// Kader (season-based teams)
			r.Get("/api/kader", kaderH.ListKader)
			r.Post("/api/kader", kaderH.InitializeKader)
			r.Get("/api/kader/{id}", kaderH.GetKader)
			r.Put("/api/kader/{id}", kaderH.UpdateKader)
			r.Delete("/api/kader/{id}", kaderH.DeleteKader)
			r.Get("/api/kader/{id}/member-suggestions", kaderH.MemberSuggestions)
			r.Get("/api/kader/{id}/extended-member-suggestions", kaderH.ExtendedMemberSuggestions)
			r.Patch("/api/kader/{id}/games-per-season", kaderH.PatchGamesPerSeason)
			r.Post("/api/kader/copy-from-season", kaderH.CopyFromSeason)
			r.Post("/api/kader/auto-assign", kaderH.AutoAssign)
		})
	})

	// SPA fallback — serve embedded React build
	distFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		log.Fatalf("embed: %v", err)
	}
	r.Get("/*", spaHandler(distFS))

	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}

func runGenVapid() {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("gen-vapid: %v", err)
	}
	fmt.Printf("VAPID_PRIVATE_KEY=%s\nVAPID_PUBLIC_KEY=%s\n", priv, pub)
}

func runPushTest() {
	fs := flag.NewFlagSet("push-test", flag.ExitOnError)
	userID := fs.Int("user", 0, "User-ID (Pflicht)")
	title := fs.String("title", "TeamWERK Test", "Titel der Notification")
	body := fs.String("body", "Das ist eine Test-Notification.", "Text der Notification")
	url := fs.String("url", "/", "Ziel-URL beim Klick")
	envFile := fs.String("env", "", "Pfad zur Env-Datei (default: .env, VPS: /etc/teamwerk/env)")
	dbPath := fs.String("db", "", "Pfad zur SQLite-Datenbank (default: aus DB_PATH)")
	_ = fs.Parse(os.Args[2:])

	if *envFile != "" {
		_ = godotenv.Load(*envFile)
	} else {
		_ = godotenv.Load()
	}
	if *dbPath == "" {
		*dbPath = getEnvOrDefault("DB_PATH", "./teamwerk.db")
	}

	if *userID == 0 {
		fmt.Fprintln(os.Stderr, "Verwendung: teamwerk push-test --user=<id> [--title=...] [--body=...] [--url=...] [--db=...]")
		os.Exit(1)
	}

	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("push-test: load config: %v", err)
	}
	if cfg.VAPIDPrivateKey == "" {
		log.Fatal("push-test: VAPID_PRIVATE_KEY nicht gesetzt")
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("push-test: open db: %v", err)
	}
	defer database.Close()

	rows, err := database.Query(`SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?`, *userID)
	if err != nil {
		log.Fatalf("push-test: query: %v", err)
	}
	defer rows.Close()

	type sub struct {
		id       int
		endpoint string
		p256dh   string
		auth     string
	}
	var subs []sub
	for rows.Next() {
		var s sub
		rows.Scan(&s.id, &s.endpoint, &s.p256dh, &s.auth)
		subs = append(subs, s)
	}
	if len(subs) == 0 {
		log.Fatalf("push-test: keine Subscriptions für User %d gefunden", *userID)
	}

	payload, _ := json.Marshal(map[string]string{"title": *title, "body": *body, "url": *url})

	for _, s := range subs {
		fmt.Printf("  Sub %d: %s…\n", s.id, s.endpoint[:60])
		resp, err := webpush.SendNotification(payload, &webpush.Subscription{
			Endpoint: s.endpoint,
			Keys:     webpush.Keys{P256dh: s.p256dh, Auth: s.auth},
		}, &webpush.Options{
			VAPIDPublicKey:  cfg.VAPIDPublicKey,
			VAPIDPrivateKey: cfg.VAPIDPrivateKey,
			Subscriber:      cfg.VAPIDEmail,
			TTL:             3600,
		})
		if err != nil {
			fmt.Printf("  → Fehler: %v\n", err)
			continue
		}
		body2, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("  → HTTP %d  %s\n", resp.StatusCode, strings.TrimSpace(string(body2)))
		if resp.StatusCode == http.StatusGone {
			database.Exec(`DELETE FROM push_subscriptions WHERE id = ?`, s.id)
			fmt.Printf("  → Subscription %d gelöscht (expired)\n", s.id)
		}
	}
}

func runCreateAdmin() {
	_ = godotenv.Load()
	fs := flag.NewFlagSet("create-admin", flag.ExitOnError)
	email := fs.String("email", "", "E-Mail-Adresse (Pflicht)")
	name := fs.String("name", "Admin", "Anzeigename")
	password := fs.String("password", "", "Passwort (Pflicht)")
	dbPath := fs.String("db", getEnvOrDefault("DB_PATH", "./vereinswerk.db"), "Pfad zur SQLite-Datenbank")
	_ = fs.Parse(os.Args[2:])

	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Verwendung: vereinswerk create-admin --email=... --password=... [--name=...] [--db=...]")
		os.Exit(1)
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("create-admin: open db: %v", err)
	}
	defer database.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("create-admin: bcrypt: %v", err)
	}

	_, err = database.Exec(
		`INSERT INTO users (email, name, password, role) VALUES (?, ?, ?, 'admin')`,
		*email, *name, string(hash),
	)
	if err != nil {
		log.Fatalf("create-admin: insert user: %v", err)
	}
	fmt.Printf("Admin-Nutzer '%s' (%s) wurde angelegt.\n", *name, *email)
}

func runScheduler() {
	_ = godotenv.Load()
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("scheduler: load config: %v", err)
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("scheduler: open db: %v", err)
	}
	defer database.Close()
	scheduler.New(database, cfg, mailer.New(cfg.SMTP, cfg.BaseURL, cfg.MailerDisabled)).Run()
}

func runMigrateForce() {
	_ = godotenv.Load()
	dbPath := getEnvOrDefault("DB_PATH", "./teamwerk.db")
	var version int
	for i, arg := range os.Args {
		if arg == "--db" && i+1 < len(os.Args) {
			dbPath = os.Args[i+1]
		}
		if arg == "force" && i+1 < len(os.Args) {
			fmt.Sscan(os.Args[i+1], &version)
		}
	}
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("migrate force: open db: %v", err)
	}
	defer database.Close()
	if err := db.MigrateForce(database, db.MigrationsFS, version); err != nil {
		log.Fatalf("migrate force: %v", err)
	}
	log.Printf("forced migration version to %d", version)
}

func runMigrate() {
	_ = godotenv.Load()
	dbPath := getEnvOrDefault("DB_PATH", "./teamwerk.db")
	for i, arg := range os.Args {
		if arg == "--db" && i+1 < len(os.Args) {
			dbPath = os.Args[i+1]
		}
	}
	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("migrate: open db: %v", err)
	}
	defer database.Close()
	if err := db.Migrate(database, db.MigrationsFS); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrations applied")
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

func spaHandler(static fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(static))
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = path[1:] // strip leading /
		}
		if _, err := fs.Stat(static, path); err != nil {
			// not a real file — serve index.html for SPA routing
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
