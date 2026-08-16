package eventlog_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/eventlog"
	_ "modernc.org/sqlite"
)

// newTestDB opens an in-memory SQLite with all migrations applied. Inlined
// (rather than internal/testutil) to avoid a foundation → domain import.
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

// eventIDsFor returns the ids of a user's rows, ordered like ListForUser
// (created_at DESC, id DESC), for assertions that don't need the full struct.
func eventIDsFor(t *testing.T, db *sql.DB, userID int) []int {
	t.Helper()
	events, err := eventlog.ListForUser(context.Background(), db, userID, 100)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	ids := make([]int, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	return ids
}

// TestRecord_MultiRowInsert verifies that Record writes exactly one row per
// recipient, each carrying the identical category/title/body/url.
func TestRecord_MultiRowInsert(t *testing.T) {
	db := newTestDB(t)
	u1 := insertUser(t, db, "u1@test.local")
	u2 := insertUser(t, db, "u2@test.local")
	u3 := insertUser(t, db, "u3@test.local")

	eventlog.Record(db, []int{u1, u2, u3}, "games", "Titel", "Text", "/ziel")

	for _, uid := range []int{u1, u2, u3} {
		events, err := eventlog.ListForUser(context.Background(), db, uid, 10)
		if err != nil {
			t.Fatalf("ListForUser(%d): %v", uid, err)
		}
		if len(events) != 1 {
			t.Fatalf("user %d: got %d events, want 1", uid, len(events))
		}
		e := events[0]
		if e.Category != "games" || e.Title != "Titel" || e.Body != "Text" || e.URL != "/ziel" {
			t.Errorf("user %d: unexpected event %+v", uid, e)
		}
	}
}

// TestRecord_GrosserFanout verifies the multi-row-VALUES path holds up for a
// club-wide fan-out (~180 recipients is the documented ballpark; 200 gives
// headroom). No chunking is expected or required: modernc.org/sqlite's
// SQLITE_MAX_VARIABLE_NUMBER (32766) comfortably covers 200 recipients × 5
// bound params each.
func TestRecord_GrosserFanout(t *testing.T) {
	db := newTestDB(t)
	const n = 200
	ids := make([]int, n)
	for i := 0; i < n; i++ {
		ids[i] = insertUser(t, db, fmt.Sprintf("fanout%d@test.local", i))
	}

	eventlog.Record(db, ids, "membership", "Titel", "Text", "/ziel")

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_events`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != n {
		t.Fatalf("got %d rows, want %d", count, n)
	}
}

// TestRecord_EmptyRecipients_NoInsert verifies that an empty recipient list
// writes nothing and does not error.
func TestRecord_EmptyRecipients_NoInsert(t *testing.T) {
	db := newTestDB(t)
	eventlog.Record(db, nil, "games", "Titel", "Text", "/ziel")

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_events`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("got %d rows, want 0", count)
	}
}

// TestListForUser_OrderNewestFirst verifies descending order by created_at,
// with id as the deterministic tiebreaker (SQLite CURRENT_TIMESTAMP has only
// second resolution).
func TestListForUser_OrderNewestFirst(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")

	// Insert three rows sharing the same created_at second, ids ascending.
	for i := 0; i < 3; i++ {
		eventlog.Record(db, []int{uid}, "games", "T", "B", "/x")
	}

	events, err := eventlog.ListForUser(context.Background(), db, uid, 10)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	// Newest (highest id) first.
	if events[0].ID <= events[1].ID || events[1].ID <= events[2].ID {
		t.Errorf("expected descending id order, got %d, %d, %d", events[0].ID, events[1].ID, events[2].ID)
	}
}

// TestListForUser_Limit verifies the caller-provided limit caps the result.
func TestListForUser_Limit(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	for i := 0; i < 5; i++ {
		eventlog.Record(db, []int{uid}, "games", "T", "B", "/x")
	}

	events, err := eventlog.ListForUser(context.Background(), db, uid, 2)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (limit)", len(events))
	}
}

// TestListForUser_OtherUsersInvisible verifies user_id scoping.
func TestListForUser_OtherUsersInvisible(t *testing.T) {
	db := newTestDB(t)
	mine := insertUser(t, db, "mine@test.local")
	other := insertUser(t, db, "other@test.local")

	eventlog.Record(db, []int{other}, "games", "Fremd", "B", "/x")

	events, err := eventlog.ListForUser(context.Background(), db, mine, 10)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events for user with none, want 0", len(events))
	}
}

// TestMarkSeen_OnlyGivenIDs verifies that MarkSeen stamps exactly the passed
// IDs and never a whole user_id — the guarantee that undelivered rows (beyond
// a caller's cap) never get stamped alongside delivered ones.
func TestMarkSeen_OnlyGivenIDs(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	eventlog.Record(db, []int{uid}, "games", "A", "B", "/x")
	eventlog.Record(db, []int{uid}, "games", "C", "D", "/y")

	ids := eventIDsFor(t, db, uid)
	if len(ids) != 2 {
		t.Fatalf("setup: got %d rows, want 2", len(ids))
	}
	// Stamp only the first (newest) id.
	if err := eventlog.MarkSeen(context.Background(), db, []int{ids[0]}); err != nil {
		t.Fatalf("MarkSeen: %v", err)
	}

	var seenCount, unseenCount int
	db.QueryRow(`SELECT COUNT(*) FROM user_events WHERE id = ? AND seen_at IS NOT NULL`, ids[0]).Scan(&seenCount)
	db.QueryRow(`SELECT COUNT(*) FROM user_events WHERE id = ? AND seen_at IS NULL`, ids[1]).Scan(&unseenCount)
	if seenCount != 1 {
		t.Errorf("expected id %d to be stamped, seenCount=%d", ids[0], seenCount)
	}
	if unseenCount != 1 {
		t.Errorf("expected id %d to remain unstamped, unseenCount=%d", ids[1], unseenCount)
	}
}

// TestMarkSeen_DoesNotOverwriteExisting verifies that a second MarkSeen call
// leaves an already-stamped seen_at unchanged — the retention clock must not
// move on repeated reads.
func TestMarkSeen_DoesNotOverwriteExisting(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	eventlog.Record(db, []int{uid}, "games", "A", "B", "/x")
	ids := eventIDsFor(t, db, uid)

	if err := eventlog.MarkSeen(context.Background(), db, ids); err != nil {
		t.Fatalf("MarkSeen (1st): %v", err)
	}
	var firstStamp string
	db.QueryRow(`SELECT seen_at FROM user_events WHERE id = ?`, ids[0]).Scan(&firstStamp)

	// Force the clock forward so a re-stamp (if it happened) would be visibly different.
	db.Exec(`UPDATE user_events SET seen_at = datetime(seen_at, '-1 day') WHERE id = ?`, ids[0])
	var backdated string
	db.QueryRow(`SELECT seen_at FROM user_events WHERE id = ?`, ids[0]).Scan(&backdated)

	if err := eventlog.MarkSeen(context.Background(), db, ids); err != nil {
		t.Fatalf("MarkSeen (2nd): %v", err)
	}
	var secondStamp string
	db.QueryRow(`SELECT seen_at FROM user_events WHERE id = ?`, ids[0]).Scan(&secondStamp)

	if secondStamp != backdated {
		t.Errorf("MarkSeen overwrote an existing seen_at: was %q, now %q", backdated, secondStamp)
	}
}

// TestMarkSeen_EmptyIDs_NoOp verifies an empty id slice is a safe no-op.
func TestMarkSeen_EmptyIDs_NoOp(t *testing.T) {
	db := newTestDB(t)
	if err := eventlog.MarkSeen(context.Background(), db, nil); err != nil {
		t.Fatalf("MarkSeen(nil): %v", err)
	}
}

// seedEventAt inserts a single row with explicit created_at/seen_at
// timestamps (relative offsets in days from now), bypassing Record so the
// Purge boundary tests can control ages precisely.
func seedEventAt(t *testing.T, db *sql.DB, userID int, createdDaysAgo int, seenDaysAgo *int) {
	t.Helper()
	createdAt := time.Now().AddDate(0, 0, -createdDaysAgo).Format("2006-01-02 15:04:05")
	var seenAt any
	if seenDaysAgo != nil {
		seenAt = time.Now().AddDate(0, 0, -*seenDaysAgo).Format("2006-01-02 15:04:05")
	}
	if _, err := db.Exec(
		`INSERT INTO user_events (user_id, category, title, body, url, created_at, seen_at)
		 VALUES (?, 'games', 'T', 'B', '/x', ?, ?)`,
		userID, createdAt, seenAt); err != nil {
		t.Fatalf("seedEventAt: %v", err)
	}
}

func days(n int) *int { return &n }

// TestPurge_SeenOlderThanThreeDays_Deleted verifies the fachliche Regel: a
// row seen more than three days ago is purged.
func TestPurge_SeenOlderThanThreeDays_Deleted(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	seedEventAt(t, db, uid, 4, days(4))

	n, err := eventlog.Purge(db)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if n != 1 {
		t.Errorf("Purge returned %d, want 1", n)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM user_events`).Scan(&count)
	if count != 0 {
		t.Errorf("expected row deleted, %d remain", count)
	}
}

// TestPurge_SeenWithinThreeDays_Kept verifies a freshly-seen row survives.
func TestPurge_SeenWithinThreeDays_Kept(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	seedEventAt(t, db, uid, 2, days(2))

	n, err := eventlog.Purge(db)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if n != 0 {
		t.Errorf("Purge returned %d, want 0", n)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM user_events`).Scan(&count)
	if count != 1 {
		t.Errorf("expected row kept, got %d rows", count)
	}
}

// TestPurge_UnseenSurvivesVacation verifies an unseen row (no seen_at), even
// 30 days old, is never purged by the three-day rule — "wer im Urlaub war,
// verliert nichts".
func TestPurge_UnseenSurvivesVacation(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	seedEventAt(t, db, uid, 30, nil)

	n, err := eventlog.Purge(db)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if n != 0 {
		t.Errorf("Purge returned %d, want 0", n)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM user_events`).Scan(&count)
	if count != 1 {
		t.Errorf("expected unseen row kept, got %d rows", count)
	}
}

// TestPurge_UnseenSafetyCapAtNinetyDays verifies the operational safety cap:
// an unseen row older than 90 days is purged regardless.
func TestPurge_UnseenSafetyCapAtNinetyDays(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	seedEventAt(t, db, uid, 91, nil)

	n, err := eventlog.Purge(db)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if n != 1 {
		t.Errorf("Purge returned %d, want 1", n)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM user_events`).Scan(&count)
	if count != 0 {
		t.Errorf("expected row purged past the 90-day cap, %d remain", count)
	}
}
