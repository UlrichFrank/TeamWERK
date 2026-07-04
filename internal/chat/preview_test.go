package chat_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type msgListItem struct {
	ID        int    `json:"id"`
	Preview   string `json:"preview"`
	Truncated bool   `json:"truncated"`
	DeletedAt string `json:"deletedAt"`
}

// TestListMessages_BodyPreviewTruncated: Body > 280 Zeichen → gekürzter Preview
// (genau 280 Zeichen) + truncated=true; kurzer Body bleibt unverändert
// (truncated=false).
func TestListMessages_BodyPreviewTruncated(t *testing.T) {
	db := testutil.NewDB(t)
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)

	longBody := strings.Repeat("a", 500)
	shortBody := "kurz"
	resLong, err := db.Exec(`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?,?,?)`, convID, owner, longBody)
	if err != nil {
		t.Fatalf("insert long: %v", err)
	}
	longID64, _ := resLong.LastInsertId()
	if _, err := db.Exec(`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?,?,?)`, convID, owner, shortBody); err != nil {
		t.Fatalf("insert short: %v", err)
	}

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/conversations/{id}/messages", h.ListMessages)
	})
	token := testutil.Token(t, owner, "standard", nil)

	res := testutil.Get(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []msgListItem
	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(items))
	}

	byID := map[int]msgListItem{}
	for _, it := range items {
		byID[it.ID] = it
	}
	long := byID[int(longID64)]
	if !long.Truncated {
		t.Errorf("expected truncated=true for long body")
	}
	if len([]rune(long.Preview)) != 280 {
		t.Errorf("expected preview length 280 runes, got %d", len([]rune(long.Preview)))
	}
	// Der kurze Body wird nicht gekürzt.
	var foundShort bool
	for _, it := range items {
		if it.Preview == shortBody {
			foundShort = true
			if it.Truncated {
				t.Errorf("short body must not be truncated")
			}
		}
	}
	if !foundShort {
		t.Errorf("short message preview not found")
	}
}

// TestListMessages_DeletedNoBody: eine gelöschte Nachricht liefert weder Preview
// noch Body und truncated=false.
func TestListMessages_DeletedNoBody(t *testing.T) {
	db := testutil.NewDB(t)
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)

	res, err := db.Exec(
		`INSERT INTO messages (conversation_id, sender_id, body, deleted_at) VALUES (?,?,?,CURRENT_TIMESTAMP)`,
		convID, owner, strings.Repeat("x", 400))
	if err != nil {
		t.Fatalf("insert deleted: %v", err)
	}
	delID64, _ := res.LastInsertId()

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/conversations/{id}/messages", h.ListMessages)
		r.Get("/api/chat/messages/{id}", h.GetMessage)
	})
	token := testutil.Token(t, owner, "standard", nil)

	res2 := testutil.Get(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages", token)
	defer res2.Body.Close()
	var items []msgListItem
	json.NewDecoder(res2.Body).Decode(&items)
	if len(items) != 1 {
		t.Fatalf("expected 1 message, got %d", len(items))
	}
	if items[0].Preview != "" {
		t.Errorf("deleted message must have empty preview, got %q", items[0].Preview)
	}
	if items[0].Truncated {
		t.Errorf("deleted message must not be truncated")
	}
	if items[0].DeletedAt == "" {
		t.Errorf("expected deletedAt to be set for deleted message")
	}

	// Einzel-Pfad: gelöschte Nachricht liefert deleted=true und leeren Body.
	res3 := testutil.Get(t, srv, "/api/chat/messages/"+itoa(int(delID64)), token)
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("GetMessage: expected 200, got %d", res3.StatusCode)
	}
	var single struct {
		ID      int    `json:"id"`
		Body    string `json:"body"`
		Deleted bool   `json:"deleted"`
	}
	json.NewDecoder(res3.Body).Decode(&single)
	if single.Body != "" {
		t.Errorf("deleted single message must have empty body, got %q", single.Body)
	}
	if !single.Deleted {
		t.Errorf("expected deleted=true for deleted single message")
	}
}

// TestGetMessage_FullBody: Einzel-Pfad liefert den ungekürzten Volltext für ein
// Mitglied; ein Nicht-Mitglied bekommt 403.
func TestGetMessage_FullBody(t *testing.T) {
	db := testutil.NewDB(t)
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	owner := testutil.CreateUser(t, db, "standard")
	outsider := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)

	fullBody := strings.Repeat("z", 500)
	res, err := db.Exec(`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?,?,?)`, convID, owner, fullBody)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	msgID64, _ := res.LastInsertId()

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/messages/{id}", h.GetMessage)
	})

	// Mitglied liest den Volltext.
	ownerTok := testutil.Token(t, owner, "standard", nil)
	res2 := testutil.Get(t, srv, "/api/chat/messages/"+itoa(int(msgID64)), ownerTok)
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("member GetMessage: expected 200, got %d", res2.StatusCode)
	}
	var single struct {
		Body    string `json:"body"`
		Deleted bool   `json:"deleted"`
	}
	json.NewDecoder(res2.Body).Decode(&single)
	if single.Body != fullBody {
		t.Errorf("expected full body of %d chars, got %d", len(fullBody), len(single.Body))
	}
	if single.Deleted {
		t.Errorf("expected deleted=false")
	}

	// Nicht-Mitglied → 403 (Sichtbarkeit invariant).
	outTok := testutil.Token(t, outsider, "standard", nil)
	res3 := testutil.Get(t, srv, "/api/chat/messages/"+itoa(int(msgID64)), outTok)
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusForbidden {
		t.Fatalf("outsider GetMessage: expected 403, got %d", res3.StatusCode)
	}
}
