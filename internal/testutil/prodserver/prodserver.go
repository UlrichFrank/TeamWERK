// Package prodserver provides a test HTTP server backed by the production
// router (app.BuildRouter). It lives in a subpackage so that packages
// referenced from the production wiring (e.g. internal/files, internal/auth)
// can still import internal/testutil without creating an import cycle.
package prodserver

import (
	"database/sql"
	"net/http/httptest"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/absences"
	"github.com/teamstuttgart/teamwerk/internal/app"
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
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
	"github.com/teamstuttgart/teamwerk/internal/teams"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
	"github.com/teamstuttgart/teamwerk/internal/upload"
	"github.com/teamstuttgart/teamwerk/internal/venues"
)

// New starts a test HTTP server backed by the same router definition used
// in production (app.BuildRouter). All routes, middleware groups, and
// authorization checks are wired exactly as in main.go.
//
// The server is closed automatically when the test ends.
func New(t *testing.T, database *sql.DB) *httptest.Server {
	t.Helper()
	cfg := testutil.TestConfig()
	hubInstance := hub.NewHub()
	m := mailer.New(appconfig.SMTPConfig{}, "http://localhost", true)

	handlers := &app.Handlers{
		Auth:         auth.NewHandler(database, cfg, testutil.TestJWTSecret, m, "http://localhost"),
		Config:       appconfig.NewHandler(database, hubInstance),
		Members:      members.NewHandler(database, hubInstance),
		WelcomeEmail: members.NewWelcomeEmailHandler(database, m),
		Duties:       duties.NewHandler(database, cfg, hubInstance),
		Dashboard:    dashboard.NewHandler(database),
		Games:        games.NewHandler(database, cfg, hubInstance),
		Kader:        kader.NewHandler(database, hubInstance),
		Upload:       upload.NewHandler(database, t.TempDir(), testutil.TestJWTSecret),
		Files:        files.NewHandler(database, t.TempDir(), testutil.TestJWTSecret),
		Carpool:      carpooling.NewHandler(database, cfg, hubInstance),
		Chat:         chat.NewHandler(database, hubInstance, cfg),
		Notif:        notifications.NewHandler(database, cfg),
		Training:     trainings.NewHandler(database, cfg, hubInstance),
		Absences:     absences.NewHandler(database, hubInstance),
		Teams:        teams.NewHandler(database),
		Venues:       venues.NewHandler(database, hubInstance),
		Hub:          hub.NewHandler(hubInstance, "test"),
		JWTSecret:    testutil.TestJWTSecret,
		Database:     database,
		BaseURL:      "",
	}

	srv := httptest.NewServer(app.BuildRouter(handlers, nil))
	t.Cleanup(srv.Close)
	return srv
}
