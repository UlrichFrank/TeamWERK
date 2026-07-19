package chat_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func recvWithinRR(ch chan string, d time.Duration) (string, bool) {
	select {
	case ev := <-ch:
		return ev, true
	case <-time.After(d):
		return "", false
	}
}

type readerEntry struct {
	UserID int    `json:"userId"`
	Name   string `json:"name"`
	ReadAt string `json:"readAt"`
}

func mountReads(h *chat.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/api/chat/messages/{id}/reads", h.MessageReads)
	}
}

// 3.1 — Absender ruft /reads ab und bekommt die Leserliste.
func TestGetMessageReads_Sender_OK(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	r1 := testutil.CreateUser(t, db, "standard")
	r2 := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", sender, r1, r2)
	msg := insertMessage(t, db, conv, sender, "hallo")
	markRead(t, db, msg, r1)
	markRead(t, db, msg, r2)

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, mountReads(h))

	res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/messages/%d/reads", msg), testutil.Token(t, sender, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var readers []readerEntry
	json.NewDecoder(res.Body).Decode(&readers)
	if len(readers) != 2 {
		t.Fatalf("expected 2 readers, got %d", len(readers))
	}
	for _, rd := range readers {
		if rd.ReadAt == "" {
			t.Errorf("reader %d missing readAt", rd.UserID)
		}
		if rd.UserID == sender {
			t.Errorf("sender must not appear in reader list")
		}
	}
}

// 3.2 — Ein anderer Konversations-User (nicht der Absender) bekommt 403.
func TestGetMessageReads_ForeignUser_403(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	other := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", sender, other)
	msg := insertMessage(t, db, conv, sender, "hallo")

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, mountReads(h))

	res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/messages/%d/reads", msg), testutil.Token(t, other, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// 3.3 — Unbekannte/gelöschte Nachricht liefert 404.
func TestGetMessageReads_MessageMissing_404(t *testing.T) {
	db := testutil.NewDB(t)
	user := testutil.CreateUser(t, db, "standard")
	sender := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", sender, user)
	deleted := insertMessage(t, db, conv, sender, "weg")
	db.Exec(`UPDATE messages SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`, deleted)

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, mountReads(h))
	token := testutil.Token(t, sender, "standard", nil)

	// Nicht existierende ID.
	res := testutil.Get(t, srv, "/api/chat/messages/999999/reads", token)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing message: expected 404, got %d", res.StatusCode)
	}
	// Gelöschte Nachricht (auch für den Absender) → 404.
	res2 := testutil.Get(t, srv, fmt.Sprintf("/api/chat/messages/%d/reads", deleted), token)
	res2.Body.Close()
	if res2.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted message: expected 404, got %d", res2.StatusCode)
	}
}

// 3.4 — MarkRead feuert pro Absender genau ein coalesced read-receipt-Event mit
// dem höchsten neu gelesenen message_id als upToMessageId.
func TestMarkRead_BroadcastsReadReceiptToSenders(t *testing.T) {
	db := testutil.NewDB(t)
	senderA := testutil.CreateUser(t, db, "standard")
	senderB := testutil.CreateUser(t, db, "standard")
	reader := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", senderA, senderB, reader)

	insertMessage(t, db, conv, senderA, "a1")
	a2 := insertMessage(t, db, conv, senderA, "a2") // höchste von A
	b1 := insertMessage(t, db, conv, senderB, "b1") // einzige von B

	sharedHub := hub.NewHub()
	h := chat.NewHandler(db, sharedHub, testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/read", h.MarkRead)
	})

	chA := sharedHub.SubscribeUser(senderA)
	chB := sharedHub.SubscribeUser(senderB)
	defer sharedHub.UnsubscribeUser(senderA, chA)
	defer sharedHub.UnsubscribeUser(senderB, chB)

	res := testutil.Post(t, srv, fmt.Sprintf("/api/chat/conversations/%d/read", conv), testutil.Token(t, reader, "standard", nil), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	wantA := fmt.Sprintf("chat:read-receipt:%d:%d:%d", conv, reader, a2)
	wantB := fmt.Sprintf("chat:read-receipt:%d:%d:%d", conv, reader, b1)
	if ev, ok := recvWithinRR(chA, time.Second); !ok || ev != wantA {
		t.Errorf("senderA event = %q ok=%v, want %q", ev, ok, wantA)
	}
	if ev, ok := recvWithinRR(chB, time.Second); !ok || ev != wantB {
		t.Errorf("senderB event = %q ok=%v, want %q", ev, ok, wantB)
	}
	// Kein zweites Event pro Absender (coalesced).
	if ev, ok := recvWithinRR(chA, 200*time.Millisecond); ok {
		t.Errorf("senderA got a second event: %q", ev)
	}
	if ev, ok := recvWithinRR(chB, 200*time.Millisecond); ok {
		t.Errorf("senderB got a second event: %q", ev)
	}
}

// 3.5 — Die Message-Liste liefert readCount/readTotal/read pro Nachricht.
func TestListMessages_IncludesReadCounters(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	m1 := testutil.CreateUser(t, db, "standard")
	m2 := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", sender, m1, m2)
	msg := insertMessage(t, db, conv, sender, "hallo")
	markRead(t, db, msg, m1) // nur ein Leser

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/conversations/{id}/messages", h.ListMessages)
	})

	res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/conversations/%d/messages", conv), testutil.Token(t, sender, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var msgs []struct {
		ID        int  `json:"id"`
		ReadCount int  `json:"readCount"`
		ReadTotal int  `json:"readTotal"`
		Read      bool `json:"read"`
	}
	json.NewDecoder(res.Body).Decode(&msgs)
	var found bool
	for _, m := range msgs {
		if m.ID != msg {
			continue
		}
		found = true
		if m.ReadCount != 1 {
			t.Errorf("readCount = %d, want 1", m.ReadCount)
		}
		if m.ReadTotal != 2 {
			t.Errorf("readTotal = %d, want 2 (aktive Mitglieder außer Sender)", m.ReadTotal)
		}
		if !m.Read {
			t.Errorf("read = false, want true (readCount>0)")
		}
	}
	if !found {
		t.Fatalf("message %d not in response", msg)
	}
}
