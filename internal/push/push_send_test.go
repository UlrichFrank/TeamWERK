package push

import (
	"database/sql"
	"io"
	"net/http"
	"strings"
	"testing"

	webpush "github.com/SherClockHolmes/webpush-go"
	_ "modernc.org/sqlite"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
)

// newSubDB opens an in-memory SQLite with just the push_subscriptions table.
// The push send path only touches this table by id, so no full migration chain
// (and thus no testutil, which would create an import cycle) is needed here.
func newSubDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE push_subscriptions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		endpoint TEXT NOT NULL UNIQUE,
		p256dh TEXT NOT NULL,
		auth TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func seedSub(t *testing.T, db *sql.DB, userID int) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth) VALUES (?, ?, ?, ?)`,
		userID, "https://push.test.local/1", "p", "a")
	if err != nil {
		t.Fatalf("seedSub: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func subExists(t *testing.T, db *sql.DB, id int) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM push_subscriptions WHERE id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("subExists: %v", err)
	}
	return n == 1
}

// testCfg carries a non-empty VAPID key so the send guard does not short-circuit;
// the actual webpush call is replaced via the sendNotification seam.
func testCfg() *appconfig.Config {
	return &appconfig.Config{
		VAPIDPublicKey:  "test-public",
		VAPIDPrivateKey: "test-private",
		VAPIDEmail:      "mailto:test@test.local",
	}
}

// stubSend replaces the sendNotification seam with one returning the given
// status code and restores the original on cleanup.
func stubSend(t *testing.T, status int) {
	t.Helper()
	orig := sendNotification
	sendNotification = func(_ []byte, _ *webpush.Subscription, _ *webpush.Options) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	t.Cleanup(func() { sendNotification = orig })
}

// TestSendToUsers_DeletesOnPermanentFailure: 410/404 mean the endpoint is gone
// for good → the subscription is removed.
func TestSendToUsers_DeletesOnPermanentFailure(t *testing.T) {
	for _, status := range []int{http.StatusGone, http.StatusNotFound} {
		db := newSubDB(t)
		subID := seedSub(t, db, 1)
		stubSend(t, status)

		SendToUsers(db, testCfg(), []int{1}, "t", "b", "/x")

		if subExists(t, db, subID) {
			t.Fatalf("status %d: subscription should have been deleted", status)
		}
	}
}

// TestSendToUsers_KeepsOnTransientFailure: 400/401/5xx are transient (VAPID
// signing / payload faults) → the subscription MUST be retained (Regression
// Defekt 2 — früher löschte 400/401 ein gültiges Abo).
func TestSendToUsers_KeepsOnTransientFailure(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError} {
		db := newSubDB(t)
		subID := seedSub(t, db, 1)
		stubSend(t, status)

		SendToUsers(db, testCfg(), []int{1}, "t", "b", "/x")

		if !subExists(t, db, subID) {
			t.Fatalf("status %d: subscription must be kept, not deleted", status)
		}
	}
}

// TestSendToUserWithBadge_DeleteMatrix mirrors the matrix for the badge variant.
func TestSendToUserWithBadge_DeleteMatrix(t *testing.T) {
	cases := []struct {
		status   int
		wantKept bool
	}{
		{http.StatusGone, false},
		{http.StatusNotFound, false},
		{http.StatusBadRequest, true},
		{http.StatusUnauthorized, true},
		{http.StatusInternalServerError, true},
	}
	for _, c := range cases {
		db := newSubDB(t)
		subID := seedSub(t, db, 1)
		stubSend(t, c.status)

		SendToUserWithBadge(db, testCfg(), 1, "t", "b", "/x", 5)

		if got := subExists(t, db, subID); got != c.wantKept {
			t.Fatalf("status %d: kept=%v, want %v", c.status, got, c.wantKept)
		}
	}
}
