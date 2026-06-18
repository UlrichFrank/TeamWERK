package testutil

import (
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/db"
	_ "modernc.org/sqlite"
)

var dbCounter atomic.Uint64

// NewDB opens a fresh in-memory SQLite database with all migrations applied.
// Each test gets its own named shared-cache database so that multiple goroutines
// (e.g. HTTP handlers in httptest servers) can share the migrated schema without
// needing SetMaxOpenConns(1), which would serialize concurrent-claim tests.
// The connection is closed automatically when the test ends.
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("testdb_%d", dbCounter.Add(1))
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys=on", name)
	database, err := sql.Open("sqlite", dsn)
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
