package testutil

import (
	"database/sql"
	"net/http"
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
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/notifications"
	"github.com/teamstuttgart/teamwerk/internal/teams"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
	"github.com/teamstuttgart/teamwerk/internal/upload"
	"github.com/teamstuttgart/teamwerk/internal/venues"
)

// BuildHandlers creates all HTTP handlers wired to the given database.
// Suitable for integration tests that need the full router (e.g. the permission matrix test).
func BuildHandlers(t *testing.T, database *sql.DB) *app.Handlers {
	t.Helper()
	cfg := TestConfig()
	h := hub.NewHub()
	return &app.Handlers{
		Auth:         auth.NewHandler(database, cfg, TestJWTSecret, nil, ""),
		Config:       appconfig.NewHandler(database, h),
		Members:      members.NewHandler(database, h),
		WelcomeEmail: members.NewWelcomeEmailHandler(database, nil),
		Duties:       duties.NewHandler(database, cfg, h),
		Dashboard:    dashboard.NewHandler(database),
		Games:        games.NewHandler(database, cfg, h),
		Kader:        kader.NewHandler(database, h),
		Upload:       upload.NewHandler(database, t.TempDir(), TestJWTSecret),
		Files:        files.NewHandler(database, t.TempDir(), TestJWTSecret),
		Carpool:      carpooling.NewHandler(database, cfg, h),
		Chat:         chat.NewHandler(database, h, cfg),
		Notif:        notifications.NewHandler(database, cfg),
		Training:     trainings.NewHandler(database, cfg, h),
		Absences:     absences.NewHandler(database, h),
		Teams:        teams.NewHandler(database),
		Venues:       venues.NewHandler(database, h),
		Hub:          hub.NewHandler(h, "test"),
		JWTSecret:    TestJWTSecret,
		Database:     database,
		BaseURL:      "",
	}
}

// BuildRouter creates the full application router for integration tests.
func BuildRouter(t *testing.T, database *sql.DB) http.Handler {
	t.Helper()
	return app.BuildRouter(BuildHandlers(t, database), nil)
}
