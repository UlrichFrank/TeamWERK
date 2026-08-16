package notify

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/eventlog"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
	"github.com/teamstuttgart/teamwerk/internal/push"
	_ "modernc.org/sqlite"
)

// capturedMail is one message intercepted via the sendMail seam.
type capturedMail struct {
	to, subject, body string
}

// stubMail replaces the sendMail seam with a thread-safe capture and restores
// the original on cleanup. Returns a func yielding the captured messages.
func stubMail(t *testing.T) func() []capturedMail {
	t.Helper()
	var mu sync.Mutex
	var got []capturedMail
	orig := sendMail
	sendMail = func(_ *mailer.Mailer, to, subject, body string) error {
		mu.Lock()
		got = append(got, capturedMail{to, subject, body})
		mu.Unlock()
		return nil
	}
	t.Cleanup(func() { sendMail = orig })
	return func() []capturedMail {
		mu.Lock()
		defer mu.Unlock()
		out := make([]capturedMail, len(got))
		copy(out, got)
		return out
	}
}

// capturedPush is one call intercepted via the push.SendToUsers seam.
type capturedPush struct {
	userIDs []int
	title   string
}

// stubPush replaces push.SendToUsers with a thread-safe capture and restores
// the original on cleanup. Returns a func yielding the captured calls.
func stubPush(t *testing.T) func() []capturedPush {
	t.Helper()
	var mu sync.Mutex
	var got []capturedPush
	orig := push.SendToUsers
	push.SendToUsers = func(_ *sql.DB, _ *appconfig.Config, userIDs []int, title, _, _ string) {
		mu.Lock()
		ids := make([]int, len(userIDs))
		copy(ids, userIDs)
		got = append(got, capturedPush{ids, title})
		mu.Unlock()
	}
	t.Cleanup(func() { push.SendToUsers = orig })
	return func() []capturedPush {
		mu.Lock()
		defer mu.Unlock()
		out := make([]capturedPush, len(got))
		copy(out, got)
		return out
	}
}

func setPushPref(t *testing.T, db *sql.DB, uid int, category string, enabled bool) {
	t.Helper()
	p := 0
	if enabled {
		p = 1
	}
	if _, err := db.Exec(
		`INSERT INTO notification_preferences (user_id, category, push_enabled, email_enabled)
		 VALUES (?, ?, ?, 0)`, uid, category, p); err != nil {
		t.Fatalf("setPushPref: %v", err)
	}
}

// eventRowsFor returns the user_events rows for a user, for assertions.
func eventRowsFor(t *testing.T, db *sql.DB, userID int) []eventlog.Event {
	t.Helper()
	events, err := eventlog.ListForUser(context.Background(), db, userID, 100)
	if err != nil {
		t.Fatalf("ListForUser: %v", err)
	}
	return events
}

func setEmailPref(t *testing.T, db *sql.DB, uid int, category string, email bool) {
	t.Helper()
	e := 0
	if email {
		e = 1
	}
	if _, err := db.Exec(
		`INSERT INTO notification_preferences (user_id, category, push_enabled, email_enabled)
		 VALUES (?, ?, 1, ?)`, uid, category, e); err != nil {
		t.Fatalf("setEmailPref: %v", err)
	}
}

// TestSendCategoryEmail_DirektlinkAppended — der Body bekommt eine Direktlink-
// Zeile mit BaseURL+url; Aufruf ist synchron (nicht über Send-Goroutine).
func TestSendCategoryEmail_DirektlinkAppended(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "mail@test.local")
	cfg := &appconfig.Config{BaseURL: "https://tw.test"}
	mails := stubMail(t)

	sendCategoryEmail(db, cfg, uid, "Titel", "Text", "/ziel")

	got := mails()
	if len(got) != 1 {
		t.Fatalf("got %d mails, want 1", len(got))
	}
	if got[0].to != "mail@test.local" {
		t.Errorf("to = %q", got[0].to)
	}
	if !strings.Contains(got[0].body, "Direktlink: https://tw.test/ziel") {
		t.Errorf("body ohne Direktlink: %q", got[0].body)
	}
}

// TestSendCategoryEmail_NoEmail_Skips — fehlende Adresse ⇒ kein Versand.
func TestSendCategoryEmail_NoEmail_Skips(t *testing.T) {
	db := newTestDB(t)
	res, _ := db.Exec(`INSERT INTO users (email, password, role) VALUES ('', '', 'standard')`)
	id, _ := res.LastInsertId()
	cfg := &appconfig.Config{BaseURL: "https://tw.test"}
	mails := stubMail(t)

	sendCategoryEmail(db, cfg, int(id), "Titel", "Text", "/ziel")

	if got := mails(); len(got) != 0 {
		t.Fatalf("got %d mails, want 0 (keine Adresse)", len(got))
	}
}

// TestSend_EmailOnlyToEmailEnabled — Send schickt Email nur an Nutzer mit
// email_enabled=1; ein reiner Push-Nutzer (Default) bekommt keine Mail.
func TestSend_EmailOnlyToEmailEnabled(t *testing.T) {
	db := newTestDB(t)
	pushOnly := insertUser(t, db, "pushonly@test.local") // kein Row ⇒ email=false
	emailUser := insertUser(t, db, "emailon@test.local")
	setEmailPref(t, db, emailUser, "duties", true)
	cfg := &appconfig.Config{BaseURL: "https://tw.test"} // VAPID leer ⇒ Push no-op
	mails := stubMail(t)

	Send(db, cfg, []int{pushOnly, emailUser}, "duties", "Titel", "Text", "/x")

	// Email läuft als Goroutine — kurz auf die erwartete Zustellung warten.
	var got []capturedMail
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if got = mails(); len(got) >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(got) != 1 {
		t.Fatalf("got %d mails, want 1 (nur email-enabled)", len(got))
	}
	if got[0].to != "emailon@test.local" {
		t.Errorf("Mail an %q, want emailon@test.local", got[0].to)
	}
}

// TestSend_EmptyList_NoSend — leere Empfängerliste löst nichts aus.
func TestSend_EmptyList_NoSend(t *testing.T) {
	db := newTestDB(t)
	cfg := &appconfig.Config{BaseURL: "https://tw.test"}
	mails := stubMail(t)

	Send(db, cfg, nil, "duties", "Titel", "Text", "/x")

	time.Sleep(50 * time.Millisecond)
	if got := mails(); len(got) != 0 {
		t.Fatalf("got %d mails, want 0", len(got))
	}
}

// TestSend_SchreibtLogFuerAlleEmpfaenger verifies the fan-out happens over
// the UNFILTERED userIDs — one user_events row per recipient, regardless of
// any preference.
func TestSend_SchreibtLogFuerAlleEmpfaenger(t *testing.T) {
	db := newTestDB(t)
	u1 := insertUser(t, db, "u1@test.local")
	u2 := insertUser(t, db, "u2@test.local")
	u3 := insertUser(t, db, "u3@test.local")
	cfg := &appconfig.Config{BaseURL: "https://tw.test"} // VAPID leer ⇒ Push no-op
	stubMail(t)

	Send(db, cfg, []int{u1, u2, u3}, "duties", "Titel", "Text", "/x")

	for _, uid := range []int{u1, u2, u3} {
		rows := eventRowsFor(t, db, uid)
		if len(rows) != 1 {
			t.Fatalf("user %d: got %d user_events rows, want 1", uid, len(rows))
		}
		if rows[0].Category != "duties" || rows[0].Title != "Titel" || rows[0].Body != "Text" || rows[0].URL != "/x" {
			t.Errorf("user %d: unexpected row %+v", uid, rows[0])
		}
	}
}

// TestSend_LogUnabhaengigVonPushPraeferenz verifies that a user with
// push_enabled=0 gets no push, but still gets a user_events row.
func TestSend_LogUnabhaengigVonPushPraeferenz(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	setPushPref(t, db, uid, "duties", false)
	cfg := &appconfig.Config{BaseURL: "https://tw.test"}
	stubMail(t)
	pushes := stubPush(t)

	Send(db, cfg, []int{uid}, "duties", "Titel", "Text", "/x")

	for _, call := range pushes() {
		for _, id := range call.userIDs {
			if id == uid {
				t.Fatalf("user %d received push despite push_enabled=0", uid)
			}
		}
	}
	if rows := eventRowsFor(t, db, uid); len(rows) != 1 {
		t.Fatalf("got %d user_events rows, want 1 despite disabled push", len(rows))
	}
}

// TestSend_LogUnabhaengigVonPushSubscription verifies that a user without any
// row in push_subscriptions still gets a user_events row (Send doesn't gate
// the log write on subscription existence — that's push.SendToUsers's own
// concern, orthogonal to the log).
func TestSend_LogUnabhaengigVonPushSubscription(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local") // no push_subscriptions row
	cfg := &appconfig.Config{BaseURL: "https://tw.test"}
	stubMail(t)

	Send(db, cfg, []int{uid}, "duties", "Titel", "Text", "/x")

	if rows := eventRowsFor(t, db, uid); len(rows) != 1 {
		t.Fatalf("got %d user_events rows, want 1 despite missing subscription", len(rows))
	}
}

// TestSend_NoEmailUnterdruecktNurEmail verifies that NoEmail() suppresses
// only the email branch — push and the event log fire unchanged.
func TestSend_NoEmailUnterdruecktNurEmail(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	setEmailPref(t, db, uid, "duties", true) // would normally receive email
	cfg := &appconfig.Config{BaseURL: "https://tw.test"}
	mails := stubMail(t)
	pushes := stubPush(t)

	Send(db, cfg, []int{uid}, "duties", "Titel", "Text", "/x", NoEmail())

	time.Sleep(50 * time.Millisecond)
	if got := mails(); len(got) != 0 {
		t.Fatalf("got %d mails, want 0 (NoEmail)", len(got))
	}

	found := false
	for _, call := range pushes() {
		for _, id := range call.userIDs {
			if id == uid {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected push to still fire for user %d despite NoEmail()", uid)
	}

	if rows := eventRowsFor(t, db, uid); len(rows) != 1 {
		t.Fatalf("got %d user_events rows, want 1 (NoEmail must not affect the log)", len(rows))
	}
}

// TestSend_SkipPushPrefIgnoriertPraeferenz verifies that SkipPushPref() sends
// push regardless of push_enabled=0, while email stays preference-gated.
func TestSend_SkipPushPrefIgnoriertPraeferenz(t *testing.T) {
	db := newTestDB(t)
	uid := insertUser(t, db, "u@test.local")
	setPushPref(t, db, uid, "sonstiges", false) // push disabled
	cfg := &appconfig.Config{BaseURL: "https://tw.test"}
	mails := stubMail(t)
	pushes := stubPush(t)

	Send(db, cfg, []int{uid}, "sonstiges", "Titel", "Text", "/x", SkipPushPref())

	found := false
	for _, call := range pushes() {
		for _, id := range call.userIDs {
			if id == uid {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected push to fire for user %d despite push_enabled=0 (SkipPushPref)", uid)
	}

	// Email bleibt präferenzgesteuert: kein email_enabled=1 gesetzt ⇒ keine Mail.
	time.Sleep(50 * time.Millisecond)
	if got := mails(); len(got) != 0 {
		t.Fatalf("got %d mails, want 0 (email stays preference-gated under SkipPushPref)", len(got))
	}
}

// TestSend_LeereEmpfaengerlisteSchreibtNichts verifies an empty recipient
// list writes no user_events row and does not error.
func TestSend_LeereEmpfaengerlisteSchreibtNichts(t *testing.T) {
	db := newTestDB(t)
	cfg := &appconfig.Config{BaseURL: "https://tw.test"}
	stubMail(t)

	Send(db, cfg, nil, "duties", "Titel", "Text", "/x")

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_events`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("got %d user_events rows, want 0", count)
	}
}

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
