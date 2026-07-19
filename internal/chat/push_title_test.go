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

func setUserName(t *testing.T, db *sql.DB, uid int, first, last string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE users SET first_name=?, last_name=? WHERE id=?`, first, last, uid); err != nil {
		t.Fatalf("setUserName: %v", err)
	}
}

// TestSendMessage_GroupChatTitleContainsGroupName — In Gruppenchats hängt der
// Push-Titel den Gruppennamen in Klammern an den Autor an; der Body bleibt
// der reine Nachrichtentext.
func TestSendMessage_GroupChatTitleContainsGroupName(t *testing.T) {
	db := testutil.NewDB(t)
	me := testutil.CreateUser(t, db, "standard")
	you := testutil.CreateUser(t, db, "standard")
	setUserName(t, db, me, "Ulrich", "Frank")
	conv := createGroupConv(t, db, "Vorstand", me, you)

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	calls := make(chan pushCall, 4)
	h.SetPushFn(func(_ *sql.DB, _ *appconfig.Config, userID int, title, body, url string, badge int) {
		calls <- pushCall{userID, title, body, url, badge}
	})
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/messages", h.SendMessage)
	})

	token := testutil.Token(t, me, "standard", nil)
	res := testutil.Post(t, srv, "/api/chat/conversations/"+strconv.Itoa(conv)+"/messages", token,
		map[string]string{"body": "hallo"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", res.StatusCode)
	}

	select {
	case c := <-calls:
		if want := "Ulrich Frank (Vorstand)"; c.title != want {
			t.Fatalf("title = %q, want %q", c.title, want)
		}
		if want := "hallo"; c.body != want {
			t.Fatalf("body = %q, want %q", c.body, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("kein Push-Call innerhalb 2s")
	}
}

// TestSendMessage_DirectChatBodyHasNoGroupName — Direktchats zeigen den
// Nachrichtentext ohne vorangestellten Gruppennamen.
func TestSendMessage_DirectChatBodyHasNoGroupName(t *testing.T) {
	db := testutil.NewDB(t)
	me := testutil.CreateUser(t, db, "standard")
	you := testutil.CreateUser(t, db, "standard")
	setUserName(t, db, me, "Ulrich", "Frank")
	res0, err := db.Exec(`INSERT INTO conversations (type, created_by) VALUES ('direct', ?)`, me)
	if err != nil {
		t.Fatalf("insert direct conv: %v", err)
	}
	cid, _ := res0.LastInsertId()
	conv := int(cid)
	for _, uid := range []int{me, you} {
		if _, err := db.Exec(`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, conv, uid); err != nil {
			t.Fatalf("add member: %v", err)
		}
	}

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	calls := make(chan pushCall, 4)
	h.SetPushFn(func(_ *sql.DB, _ *appconfig.Config, userID int, title, body, url string, badge int) {
		calls <- pushCall{userID, title, body, url, badge}
	})
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/messages", h.SendMessage)
	})

	token := testutil.Token(t, me, "standard", nil)
	res := testutil.Post(t, srv, "/api/chat/conversations/"+strconv.Itoa(conv)+"/messages", token,
		map[string]string{"body": "hi"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, want 201", res.StatusCode)
	}

	select {
	case c := <-calls:
		if want := "Ulrich Frank"; c.title != want {
			t.Fatalf("title = %q, want %q", c.title, want)
		}
		if want := "hi"; c.body != want {
			t.Fatalf("body = %q, want %q", c.body, want)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("kein Push-Call innerhalb 2s")
	}
}
