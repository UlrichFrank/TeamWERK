package chat_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type convListItem struct {
	ID          int `json:"id"`
	LastMessage *struct {
		Body   string `json:"body"`
		SentAt string `json:"sentAt"`
	} `json:"lastMessage"`
}

// TestListConversations_SortedByLastActivity sichert die Invariante der
// Chat-Übersicht (chat-konversationen): Konversationen kommen absteigend nach
// letzter Aktivität zurück — die zuletzt aktive zuerst. Konversationen ohne
// Nachricht werden anhand von created_at einsortiert.
func TestListConversations_SortedByLastActivity(t *testing.T) {
	db := testutil.NewDB(t)
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	owner := testutil.CreateUser(t, db, "standard")

	convOld := createGroupConv(t, db, "Alt", owner)
	convNew := createGroupConv(t, db, "Neu", owner)
	convEmpty := createGroupConv(t, db, "Leer", owner)

	// Nachrichten mit explizitem sent_at → deterministische Aktivität.
	if _, err := db.Exec(
		`INSERT INTO messages (conversation_id, sender_id, body, sent_at) VALUES (?,?,?,?)`,
		convOld, owner, "alt", "2026-03-01 10:00:00"); err != nil {
		t.Fatalf("insert old: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO messages (conversation_id, sender_id, body, sent_at) VALUES (?,?,?,?)`,
		convNew, owner, "neu", "2026-05-01 10:00:00"); err != nil {
		t.Fatalf("insert new: %v", err)
	}
	// Leere Konversation ohne Nachricht: created_at zwischen alt und neu.
	if _, err := db.Exec(
		`UPDATE conversations SET created_at = ? WHERE id = ?`,
		"2026-04-01 10:00:00", convEmpty); err != nil {
		t.Fatalf("update empty created_at: %v", err)
	}

	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/conversations", h.ListConversations)
	})
	token := testutil.Token(t, owner, "standard", nil)

	res := testutil.Get(t, srv, "/api/chat/conversations", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var items []convListItem
	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 conversations, got %d", len(items))
	}

	// Erwartete Reihenfolge: convNew (Mai) → convEmpty (Apr) → convOld (März).
	want := []int{convNew, convEmpty, convOld}
	for i, id := range want {
		if items[i].ID != id {
			t.Errorf("index %d: expected conv %d, got %d (order: %d, %d, %d)",
				i, id, items[i].ID, items[0].ID, items[1].ID, items[2].ID)
		}
	}

	// Aktivste Konversation an Index 0 trägt ihre letzte Nachricht.
	if items[0].LastMessage == nil || items[0].LastMessage.Body != "neu" {
		t.Errorf("index 0: expected lastMessage 'neu', got %+v", items[0].LastMessage)
	}
	// Leere Konversation hat keine letzte Nachricht.
	if items[1].LastMessage != nil {
		t.Errorf("index 1 (empty conv): expected no lastMessage, got %+v", items[1].LastMessage)
	}
}
