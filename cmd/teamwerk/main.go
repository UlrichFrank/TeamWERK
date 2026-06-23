package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"

	webpush "github.com/SherClockHolmes/webpush-go"
	"golang.org/x/crypto/bcrypt"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/teamstuttgart/teamwerk/internal/absences"
	"github.com/teamstuttgart/teamwerk/internal/app"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/beitragslauf"
	"github.com/teamstuttgart/teamwerk/internal/beitragssaetze"
	"github.com/teamstuttgart/teamwerk/internal/calendar"
	"github.com/teamstuttgart/teamwerk/internal/carpooling"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/dashboard"
	"github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/files"
	"github.com/teamstuttgart/teamwerk/internal/games"
	"github.com/teamstuttgart/teamwerk/internal/health"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/kader"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/metrics"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
	"github.com/teamstuttgart/teamwerk/internal/scheduler"
	"github.com/teamstuttgart/teamwerk/internal/stammvereine"
	"github.com/teamstuttgart/teamwerk/internal/teams"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
	"github.com/teamstuttgart/teamwerk/internal/upload"
	"github.com/teamstuttgart/teamwerk/internal/venues"
)

//go:embed all:web/dist
var webFS embed.FS

var buildHash = "dev"

// newLogHandler baut den slog-Handler aus dem konfigurierten Format. "text"
// (lokale DX) → menschenlesbar auf stderr; sonst JSON auf stdout (Prod-Default,
// neutrale Schnittstelle für beliebige Log-Collector).
func newLogHandler(format string, w io.Writer) slog.Handler {
	if format == "text" {
		return slog.NewTextHandler(w, nil)
	}
	return slog.NewJSONHandler(w, nil)
}

// setupLogger setzt den Default-Logger gemäß LOG_FORMAT.
func setupLogger(format string) {
	if format == "text" {
		slog.SetDefault(slog.New(newLogHandler("text", os.Stderr)))
		return
	}
	slog.SetDefault(slog.New(newLogHandler("json", os.Stdout)))
}

// fatal loggt strukturiert und beendet den Prozess mit Exit-Code 1
// (Ersatz für log.Fatal/Fatalf).
func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func main() {
	_ = godotenv.Load()
	setupLogger(getEnvOrDefault("LOG_FORMAT", "json"))

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
	if len(os.Args) > 1 && os.Args[1] == "metrics" {
		runMetrics()
		return
	}

	serve()
}

func runMetrics() {
	gate := false
	for _, a := range os.Args[2:] {
		if a == "--gate" {
			gate = true
		}
	}
	os.Exit(metrics.Run(metrics.Options{Gate: gate}))
}

func serve() {
	cfg, err := appconfig.Load()
	if err != nil {
		fatal("config load failed", "error", err)
	}

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		fatal("open db failed", "error", err)
	}
	defer database.Close()

	m := mailer.New(cfg.SMTP, cfg.BaseURL, cfg.MailerDisabled)
	hubInstance := hub.NewHub()
	handlers := &app.Handlers{
		Auth:           auth.NewHandler(database, cfg, cfg.JWTSecret, m, cfg.BaseURL, hubInstance),
		Config:         appconfig.NewHandler(database, hubInstance),
		Members:        members.NewHandler(database, hubInstance),
		WelcomeEmail:   members.NewWelcomeEmailHandler(database, m),
		Duties:         duties.NewHandler(database, cfg, hubInstance),
		Dashboard:      dashboard.NewHandler(database),
		Games:          games.NewHandler(database, cfg, hubInstance),
		Kader:          kader.NewHandler(database, hubInstance),
		Upload:         upload.NewHandler(database, cfg.UploadDir, cfg.JWTSecret, hubInstance),
		Files:          files.NewHandler(database, cfg.FilesDir, cfg.JWTSecret),
		Carpool:        carpooling.NewHandler(database, cfg, hubInstance),
		Chat:           chat.NewHandler(database, hubInstance, cfg),
		Notif:          notifications.NewHandler(database, cfg),
		Training:       trainings.NewHandler(database, cfg, hubInstance),
		Absences:       absences.NewHandler(database, hubInstance),
		Teams:          teams.NewHandler(database),
		Venues:         venues.NewHandler(database, hubInstance),
		Beitragssaetze: beitragssaetze.NewHandler(database, hubInstance),
		Beitragslauf:   beitragslauf.NewHandler(database, hubInstance, cfg.BeitragslaufDir),
		Stammvereine:   stammvereine.NewHandler(database, hubInstance),
		Calendar:       calendar.NewHandler(database),
		Health:         health.NewHandler(database, cfg.DBPath, cfg.MetricsToken),
		Hub:            hub.NewHandler(hubInstance, buildHash),
		JWTSecret:      cfg.JWTSecret,
		Database:       database,
		BaseURL:        cfg.BaseURL,
		BuildHash:      buildHash,
	}

	distFS, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		fatal("embed failed", "error", err)
	}

	root := chi.NewRouter()
	root.Use(middleware.Logger)
	root.Mount("/", app.BuildRouter(handlers, distFS))

	slog.Info("listening", "port", cfg.Port)
	fatal("http server stopped", "error", http.ListenAndServe(":"+cfg.Port, root))
}

func runGenVapid() {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		fatal("gen-vapid failed", "error", err)
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
		fatal("push-test: load config failed", "error", err)
	}
	if cfg.VAPIDPrivateKey == "" {
		fatal("push-test: VAPID_PRIVATE_KEY nicht gesetzt")
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		fatal("push-test: open db failed", "error", err)
	}
	defer database.Close()

	rows, err := database.Query(`SELECT id, endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = ?`, *userID)
	if err != nil {
		fatal("push-test: query failed", "error", err)
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
		fatal("push-test: keine Subscriptions gefunden", "user", *userID)
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
		fatal("create-admin: open db failed", "error", err)
	}
	defer database.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		fatal("create-admin: bcrypt failed", "error", err)
	}

	_, err = database.Exec(
		`INSERT INTO users (email, name, password, role) VALUES (?, ?, ?, 'admin')`,
		*email, *name, string(hash),
	)
	if err != nil {
		fatal("create-admin: insert user failed", "error", err)
	}
	fmt.Printf("Admin-Nutzer '%s' (%s) wurde angelegt.\n", *name, *email)
}

func runScheduler() {
	_ = godotenv.Load()
	cfg, err := appconfig.Load()
	if err != nil {
		fatal("scheduler: load config failed", "error", err)
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		fatal("scheduler: open db failed", "error", err)
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
		fatal("migrate force: open db failed", "error", err)
	}
	defer database.Close()
	if err := db.MigrateForce(database, db.MigrationsFS, version); err != nil {
		fatal("migrate force failed", "error", err)
	}
	slog.Info("forced migration version", "version", version)
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
		fatal("migrate: open db failed", "error", err)
	}
	defer database.Close()
	if err := db.Migrate(database, db.MigrationsFS); err != nil {
		fatal("migrate failed", "error", err)
	}
	slog.Info("migrations applied")
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
