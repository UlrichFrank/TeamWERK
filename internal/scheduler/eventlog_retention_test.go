package scheduler

import (
	"database/sql"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// seedUserEvent legt eine user_events-Zeile mit expliziten created_at/seen_at-
// Zeitstempeln an (relative Tages-Offsets ab jetzt), damit die Purge-Grenzen
// präzise gesteuert werden können — analog zu den Retention-Tests der übrigen
// Scheduler-Jobs (vgl. training_diary_retention_test.go).
func seedUserEvent(t *testing.T, db *sql.DB, userID int, createdDaysAgo int, seenDaysAgo *int) int {
	t.Helper()
	createdAt := time.Now().AddDate(0, 0, -createdDaysAgo).Format("2006-01-02 15:04:05")
	var seenAt any
	if seenDaysAgo != nil {
		s := time.Now().AddDate(0, 0, -*seenDaysAgo).Format("2006-01-02 15:04:05")
		seenAt = s
	}
	res, err := db.Exec(
		`INSERT INTO user_events (user_id, category, title, body, url, created_at, seen_at)
		 VALUES (?, 'games', 'Titel', 'Text', '/x', ?, ?)`,
		userID, createdAt, seenAt)
	if err != nil {
		t.Fatalf("seedUserEvent: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("seedUserEvent LastInsertId: %v", err)
	}
	return int(id)
}

func userEventExists(t *testing.T, db *sql.DB, id int) bool {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_events WHERE id = ?`, id).Scan(&count); err != nil {
		t.Fatalf("userEventExists: %v", err)
	}
	return count > 0
}

func daysAgo(n int) *int { return &n }

// Szenario „Gesehene Zeile verfällt": seen_at vor vier Tagen → der
// Retention-Lauf löscht sie (spec.md „Retention drei Tage nach Ansicht").
func TestEventLogRetention_SeenFourDaysAgo_Purged(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	id := seedUserEvent(t, db, userID, 4, daysAgo(4))

	New(db, testutil.TestConfig(), nil).purgeEventLog()

	if userEventExists(t, db, id) {
		t.Error("vor vier Tagen gesehene Zeile hätte gelöscht werden müssen")
	}
}

// Szenario „Frisch gesehene Zeile bleibt": seen_at vor zwei Tagen → bleibt
// erhalten (Retention-Fenster sind drei Tage).
func TestEventLogRetention_SeenTwoDaysAgo_Kept(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	id := seedUserEvent(t, db, userID, 2, daysAgo(2))

	New(db, testutil.TestConfig(), nil).purgeEventLog()

	if !userEventExists(t, db, id) {
		t.Error("vor zwei Tagen gesehene Zeile wurde fälschlich gelöscht")
	}
}

// Szenario „Ungesehene Zeile überlebt den Urlaub": 30 Tage alt, seen_at NULL
// → bleibt erhalten (weit unterhalb der 90-Tage-Sicherheitskappe).
func TestEventLogRetention_UnseenThirtyDays_Kept(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	id := seedUserEvent(t, db, userID, 30, nil)

	New(db, testutil.TestConfig(), nil).purgeEventLog()

	if !userEventExists(t, db, id) {
		t.Error("30 Tage alte ungesehene Zeile wurde fälschlich gelöscht")
	}
}

// Szenario „Sicherheitskappe greift": 91 Tage alt, seen_at NULL → der
// Retention-Lauf löscht sie trotzdem (Betriebs-Sicherung gegen unbegrenztes
// Wachstum durch nie wieder eingeloggte Accounts).
func TestEventLogRetention_UnseenNinetyOneDays_Purged(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	id := seedUserEvent(t, db, userID, 91, nil)

	New(db, testutil.TestConfig(), nil).purgeEventLog()

	if userEventExists(t, db, id) {
		t.Error("91 Tage alte ungesehene Zeile hätte durch die Sicherheitskappe gelöscht werden müssen")
	}
}

// Zweiter Lauf verändert nichts und wirft nicht — ein DELETE ist von Natur
// aus wiederholbar, kein Idempotenzschutz nötig (design.md Decision 5).
func TestEventLogRetention_Idempotent(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	seedUserEvent(t, db, userID, 4, daysAgo(4))
	keptID := seedUserEvent(t, db, userID, 2, daysAgo(2))

	s := New(db, testutil.TestConfig(), nil)
	s.purgeEventLog()
	s.purgeEventLog()

	if !userEventExists(t, db, keptID) {
		t.Error("frisch gesehene Zeile hätte den zweiten Lauf überleben müssen")
	}
}
