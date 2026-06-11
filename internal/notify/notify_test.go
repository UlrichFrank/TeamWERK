package notify

import (
	"database/sql"
	"sort"
	"testing"

	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	_ "modernc.org/sqlite"
)

// newTestDB opens an in-memory SQLite with all migrations applied.
// Inlined to avoid a notify → testutil → auth → notify import cycle.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys=on")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := appdb.Migrate(database, appdb.MigrationsFS); err != nil {
		database.Close()
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func insertUser(t *testing.T, db *sql.DB, email string) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (email, password, role) VALUES (?, '', 'standard')`, email)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func TestFilterByEmailPref(t *testing.T) {
	db := newTestDB(t)

	uNoRow := insertUser(t, db, "noprefs@test.local")
	uEmailOn := insertUser(t, db, "emailon@test.local")
	uEmailOff := insertUser(t, db, "emailoff@test.local")
	uOtherCat := insertUser(t, db, "othercat@test.local")

	_, err := db.Exec(
		`INSERT INTO notification_preferences (user_id, category, push_enabled, email_enabled) VALUES
			(?, 'duties', 1, 1),
			(?, 'duties', 1, 0),
			(?, 'games',  1, 1)`,
		uEmailOn, uEmailOff, uOtherCat,
	)
	if err != nil {
		t.Fatalf("seed preferences: %v", err)
	}

	got := filterByEmailPref(db, []int{uNoRow, uEmailOn, uEmailOff, uOtherCat}, "duties")
	sort.Ints(got)

	want := []int{uEmailOn}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("filterByEmailPref(duties) = %v, want %v (uNoRow=%d uEmailOn=%d uEmailOff=%d uOtherCat=%d)",
			got, want, uNoRow, uEmailOn, uEmailOff, uOtherCat)
	}
}

func TestFilterByEmailPref_EmptyInput(t *testing.T) {
	db := newTestDB(t)
	if got := filterByEmailPref(db, nil, "duties"); got != nil {
		t.Fatalf("empty input: got %v, want nil", got)
	}
}
