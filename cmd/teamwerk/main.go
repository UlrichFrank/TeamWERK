package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/dashboard"
	"github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/games"
	"github.com/teamstuttgart/teamwerk/internal/kader"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/scheduler"
	"github.com/teamstuttgart/teamwerk/internal/upload"
)

//go:embed all:web/dist
var webFS embed.FS

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

	m := mailer.New(cfg.SMTP)
	authH := auth.NewHandler(database, cfg.JWTSecret, m, cfg.BaseURL)
	cfgH := appconfig.NewHandler(database)
	membH := members.NewHandler(database)
	dutyH := duties.NewHandler(database)
	dashH := dashboard.NewHandler(database)
	gameH := games.NewHandler(database)
	kaderH := kader.NewHandler(database)
	uploadH := upload.NewHandler(database, cfg.UploadDir)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(corsMiddleware(cfg.BaseURL))

	// Public routes
	r.Get("/api/uploads/*", uploadH.ServeUpload)
	r.Post("/api/auth/login", authH.Login)
	r.Post("/api/auth/refresh", authH.Refresh)
	r.Post("/api/auth/logout", authH.Logout)
	r.Post("/api/auth/request-membership", authH.RequestMembership)
	r.Post("/api/auth/register", authH.Register)
	r.Post("/api/auth/forgot-password", authH.ForgotPassword)
	r.Post("/api/auth/reset-password", authH.ResetPassword)
	r.Get("/api/profile/email/confirm", authH.ConfirmEmailChange)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(cfg.JWTSecret))

		// Members
		r.Get("/api/members", membH.List)
		r.Post("/api/members", membH.Create)
		r.Get("/api/members/export", membH.Export)
		r.Get("/api/members/{id}", membH.Get)
		r.Put("/api/members/{id}", membH.Update)
		r.Put("/api/members/{id}/status", membH.UpdateStatus)
		r.Get("/api/members/{id}/change-drafts", membH.GetChangeRequestsHandler)
		r.Post("/api/members/{id}/change-request", membH.CreateChangeRequestHandler)
		r.Post("/api/members/{id}/change-drafts/{draftId}/accept", membH.AcceptChangeRequestHandler)
		r.Delete("/api/members/{id}/change-drafts/{draftId}", membH.RejectChangeRequestHandler)
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
		r.Post("/api/upload/user-photo", uploadH.UploadUserPhoto)

		// Dashboard
		r.Get("/api/dashboard", dashH.Get)

		// Duties
		r.Get("/api/duty-board", dutyH.Board)
		r.Post("/api/duty-board/{slotId}/claim", dutyH.Claim)
		r.Delete("/api/duty-board/{slotId}/claim", dutyH.Unclaim)
		r.Get("/api/duty-accounts", dutyH.Accounts)
		r.Get("/api/duty-slots", dutyH.ListSlots)
		r.Get("/api/duty-slots/{id}/assignments", dutyH.ListAssignments)

		// Games
		r.Get("/api/games", gameH.ListGames)
		r.Get("/api/games/{id}", gameH.GetGame)

		// Teams (filtered by user role)
		r.Get("/api/teams", gameH.ListTeamsForUser)

		// Admin + Trainer
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin", "trainer"))
			r.Post("/api/duty-slots", dutyH.CreateSlot)
			r.Put("/api/duty-slots/{id}", dutyH.UpdateSlot)
			r.Delete("/api/duty-slots/{id}", dutyH.DeleteSlot)
			r.Post("/api/duty-assignments/{id}/fulfill", dutyH.Fulfill)
			r.Post("/api/duty-assignments/{id}/cash-substitute", dutyH.CashSubstitute)
			r.Get("/api/admin/membership-requests", authH.ListMembershipRequests)
			r.Post("/api/admin/membership-requests/{id}/approve", authH.ApproveMembershipRequest)
			r.Post("/api/admin/membership-requests/{id}/reject", authH.RejectMembershipRequest)
			r.Delete("/api/admin/membership-requests/{id}", authH.DeleteMembershipRequest)
			r.Post("/api/auth/invite", authH.Invite)
		})

		// Admin + Vorstand + Trainer
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin", "vorstand", "trainer"))
			r.Post("/api/admin/games", gameH.CreateGame)
			r.Put("/api/admin/games/{id}", gameH.UpdateGame)
			r.Delete("/api/admin/games/{id}", gameH.DeleteGame)
			r.Post("/api/admin/games/{id}/regenerate", gameH.RegenerateSlots)
			r.Post("/api/admin/games/regenerate-day", gameH.RegenerateDaySlots)
		})

		// Admin + Vorstand
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole("admin", "vorstand"))
			r.Get("/api/admin/club", cfgH.GetClub)
			r.Put("/api/admin/club", cfgH.UpdateClub)
			r.Get("/api/admin/seasons", cfgH.ListSeasons)
			r.Post("/api/admin/seasons", cfgH.CreateSeason)
			r.Put("/api/admin/seasons/{id}/activate", cfgH.ActivateSeason)
			r.Delete("/api/admin/seasons/{id}", cfgH.DeleteSeason)
			r.Put("/api/admin/seasons/{id}/duty-targets", dutyH.SetSeasonTargets)
			r.Get("/api/admin/teams", cfgH.ListTeams)
			r.Post("/api/admin/teams", cfgH.CreateTeam)
			r.Put("/api/admin/teams/{id}", cfgH.UpdateTeam)
			r.Post("/api/admin/teams/{id}/assign-trainer", cfgH.AssignTrainer)
			r.Get("/api/admin/users", authH.ListUsers)
			r.Put("/api/admin/users/{id}/role", authH.UpdateUserRole)
			r.Delete("/api/admin/users/{id}", authH.DeleteUser)
			r.Get("/api/admin/invitations", authH.ListInvitations)
			r.Delete("/api/admin/invitations/{id}", authH.DeleteInvitation)
			r.Post("/api/members/import", membH.Import)
			r.Delete("/api/admin/members/{id}", membH.DeleteMember)
			r.Put("/api/admin/members/{id}/user", membH.LinkUser)
			r.Get("/api/admin/members/{id}/parents", membH.GetMemberParents)
			r.Post("/api/admin/users/{id}/create-member", membH.CreateMemberFromUser)
			r.Post("/api/admin/family-links", membH.CreateFamilyLink)
			r.Delete("/api/admin/family-links", membH.DeleteFamilyLink)
			r.Get("/api/admin/duty-types", dutyH.ListTypes)
			r.Post("/api/admin/duty-types", dutyH.CreateType)
			r.Put("/api/admin/duty-types/{id}", dutyH.UpdateType)
			r.Delete("/api/admin/duty-types/{id}", dutyH.DeleteType)
			r.Get("/api/admin/duty-accounts/export", dutyH.ExportAccounts)
			r.Get("/api/admin/duty-templates", gameH.ListTemplates)
			r.Post("/api/admin/duty-templates", gameH.CreateTemplate)
			r.Get("/api/admin/duty-templates/{id}", gameH.GetTemplateByID)
			r.Put("/api/admin/duty-templates/{id}", gameH.UpdateTemplate)
			r.Delete("/api/admin/duty-templates/{id}", gameH.DeleteTemplate)
			r.Get("/api/admin/duty-templates/{id}/preview", gameH.PreviewSlots)
			r.Post("/api/upload/member-photo/{id}", uploadH.UploadMemberPhoto)
			r.Post("/api/upload/sepa-mandat/{id}", uploadH.UploadSepaMandat)
			// Kader (season-based teams)
			r.Get("/api/admin/kader", kaderH.ListKader)
			r.Post("/api/admin/kader", kaderH.InitializeKader)
			r.Get("/api/admin/kader/{id}", kaderH.GetKader)
			r.Put("/api/admin/kader/{id}", kaderH.UpdateKader)
			r.Delete("/api/admin/kader/{id}", kaderH.DeleteKader)
			r.Get("/api/admin/kader/{id}/member-suggestions", kaderH.MemberSuggestions)
			r.Post("/api/admin/kader/copy-from-season", kaderH.CopyFromSeason)
			r.Post("/api/admin/kader/auto-assign", kaderH.AutoAssign)
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
	database, err := db.Open(getEnvOrDefault("DB_PATH", "./teamwerk.db"))
	if err != nil {
		log.Fatalf("scheduler: open db: %v", err)
	}
	defer database.Close()
	scheduler.New(database).Run()
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
