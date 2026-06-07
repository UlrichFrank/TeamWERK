package testutil

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/db"
	_ "modernc.org/sqlite"
)

// NewDB opens a fresh in-memory SQLite database with all migrations applied.
// The connection is closed automatically when the test ends.
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys=on")
	if err != nil {
		t.Fatalf("testutil.NewDB open: %v", err)
	}
	if err := db.Migrate(database, db.MigrationsFS); err != nil {
		database.Close()
		t.Fatalf("testutil.NewDB migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}
