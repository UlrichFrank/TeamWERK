package notify

import (
	"database/sql"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	appdb "github.com/teamstuttgart/teamwerk/internal/db"
	"github.com/teamstuttgart/teamwerk/internal/mailer"
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
