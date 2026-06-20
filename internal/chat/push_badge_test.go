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

type pushCall struct {
	userID int
	title  string
	body   string
	url    string
	badge  int
}

func TestPostChatMessage_TriggersPushWithBadge(t *testing.T) {
	db := testutil.NewDB(t)
	me := testutil.CreateUser(t, db, "standard")
	you := testutil.CreateUser(t, db, "standard")

	convA := createConv(t, db, me, you)
	insertMessage(t, db, convA, me, "first")
	insertMessage(t, db, convA, me, "second")

	convB := createConv(t, db, me, you)

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	calls := make(chan pushCall, 4)
	h.SetPushFn(func(_ *sql.DB, _ *appconfig.Config, userID int, title, body, url string, badge int) {
		calls <- pushCall{userID, title, body, url, badge}
	})

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/messages", h.SendMessage)
	})

	token := testutil.Token(t, me, "standard", nil)
	res := testutil.Post(t, srv, "/api/chat/conversations/"+strconv.Itoa(convB)+"/messages", token,
		map[string]string{"body": "hi"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	select {
	case c := <-calls:
		if c.userID != you {
			t.Fatalf("expected push to user %d, got %d", you, c.userID)
		}
		if c.badge != 3 {
			t.Fatalf("expected badge=3 (2 old + 1 new), got %d", c.badge)
		}
		if want := "/chat?conv=" + strconv.Itoa(convB); c.url != want {
			t.Fatalf("expected url=%s, got %q", want, c.url)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("push call was not triggered within 2s")
	}
}
