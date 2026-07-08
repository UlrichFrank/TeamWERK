package chat_test

import (
	"database/sql"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// collectPushes wartet bis want Push-Calls eingegangen sind (oder Timeout) und
// gibt die Empfänger-IDs zurück.
func collectPushes(t *testing.T, calls <-chan pushCall, want int) map[int]bool {
	t.Helper()
	got := map[int]bool{}
	deadline := time.After(2 * time.Second)
	for len(got) < want {
		select {
		case c := <-calls:
			got[c.userID] = true
		case <-deadline:
			t.Fatalf("nur %d/%d Push-Calls empfangen: %v", len(got), want, got)
		}
	}
	// Kurz nachfassen: kein weiterer (unerwarteter) Call darf folgen.
	select {
	case c := <-calls:
		t.Fatalf("unerwarteter zusätzlicher Push an %d", c.userID)
	case <-time.After(100 * time.Millisecond):
	}
	return got
}

// TestSendMessage_GroupFansOutToAllNonSenders — Nachricht in einer Gruppe mit
// drei aktiven Mitgliedern ⇒ Push an beide Nicht-Sender, nicht an den Sender.
func TestSendMessage_GroupFansOutToAllNonSenders(t *testing.T) {
	db := testutil.NewDB(t)
	me := testutil.CreateUser(t, db, "standard")
	b := testutil.CreateUser(t, db, "standard")
	c := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "Team", me, b, c)

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	calls := make(chan pushCall, 8)
	h.SetPushFn(func(_ *sql.DB, _ *appconfig.Config, userID int, _, _, _ string, _ int) {
		calls <- pushCall{userID: userID}
	})
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/messages", h.SendMessage)
	})

	token := testutil.Token(t, me, "standard", nil)
	res := testutil.Post(t, srv, "/api/chat/conversations/"+strconv.Itoa(conv)+"/messages", token,
		map[string]string{"body": "hallo team"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", res.StatusCode)
	}

	got := collectPushes(t, calls, 2)
	if !got[b] || !got[c] {
		t.Fatalf("Push-Empfänger = %v, want {%d, %d}", got, b, c)
	}
	if got[me] {
		t.Fatal("Sender darf keinen Push bekommen")
	}
}

// TestSendBroadcast_FansOutToNonSenderRecipients — ein 'all'-Broadcast pusht an
// alle Nutzer außer dem Sender.
func TestSendBroadcast_FansOutToNonSenderRecipients(t *testing.T) {
	db := testutil.NewDB(t)
	admin := testutil.CreateUser(t, db, "admin")
	b := testutil.CreateUser(t, db, "standard")
	c := testutil.CreateUser(t, db, "standard")

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	calls := make(chan pushCall, 8)
	h.SetPushFn(func(_ *sql.DB, _ *appconfig.Config, userID int, _, _, _ string, _ int) {
		calls <- pushCall{userID: userID}
	})
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/broadcasts", h.SendBroadcast)
	})

	token := testutil.Token(t, admin, "admin", nil)
	res := testutil.Post(t, srv, "/api/chat/broadcasts", token,
		map[string]any{"body": "Ansage", "targetType": "all"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", res.StatusCode)
	}

	got := collectPushes(t, calls, 2)
	if !got[b] || !got[c] {
		t.Fatalf("Broadcast-Push-Empfänger = %v, want {%d, %d}", got, b, c)
	}
	if got[admin] {
		t.Fatal("Sender (admin) darf keinen Broadcast-Push bekommen")
	}
}
