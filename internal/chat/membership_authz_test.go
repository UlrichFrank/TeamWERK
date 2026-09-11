package chat_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestMarkRead_NichtMitglied403: eine Lesebestätigung ist eine Schreiboperation
// in fremden Nachrichten. Ohne aktive Mitgliedschaft → 403, und es entsteht
// keine message_reads-Zeile (vorher konnte jeder Eingeloggte für jede convID
// Read-Receipts erzeugen).
func TestMarkRead_NichtMitglied403(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	outsider := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", sender, member)
	msgID := insertMessage(t, db, conv, sender, "hallo")

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/read", h.MarkRead)
	})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/chat/conversations/%d/read", conv),
		testutil.Token(t, outsider, "standard", nil), nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-member, got %d", res.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM message_reads WHERE message_id=? AND user_id=?`,
		msgID, outsider).Scan(&n)
	if n != 0 {
		t.Errorf("expected no message_reads row for the outsider, got %d", n)
	}
}

// TestMarkRead_AusgetretenesMitglied403: wer die Gruppe verlassen hat, liest den
// Verlauf weiter (isMember in ListMessages), setzt aber keine Lesebestätigung
// mehr — die Zusage „gelesen" gilt nur für aktive Mitglieder.
func TestMarkRead_AusgetretenesMitglied403(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	leaver := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", sender, leaver)
	insertMessage(t, db, conv, sender, "hallo")
	db.Exec(`UPDATE conversation_members SET left_at = CURRENT_TIMESTAMP
	         WHERE conversation_id=? AND user_id=?`, conv, leaver)

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/read", h.MarkRead)
	})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/chat/conversations/%d/read", conv),
		testutil.Token(t, leaver, "standard", nil), nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for a member who left, got %d", res.StatusCode)
	}
}

// TestLeave_NichtMitglied403KeineSystemnachricht: ein Fremder kann keine Gruppe
// „verlassen" — vorher landete dabei eine System-Nachricht „hat die Gruppe
// verlassen" im Verlauf einer Gruppe, in der er nie war.
func TestLeave_NichtMitglied403KeineSystemnachricht(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	outsider := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "Gruppe", owner, member)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/chat/conversations/"+itoa(convID)+"/members/me",
		testutil.Token(t, outsider, "standard", nil), nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-member, got %d", res.StatusCode)
	}

	if systemMessageExists(t, db, convID, outsider, "hat die Gruppe verlassen") {
		t.Error("expected no system message for a non-member")
	}
	var total int
	db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conversation_id=?`, convID).Scan(&total)
	if total != 0 {
		t.Errorf("expected no messages at all, got %d", total)
	}
}

// TestLeave_ZweitesVerlassen403: der zweite Austritt desselben Nutzers ist kein
// stiller No-op mehr (204 + zweite System-Nachricht), sondern 403 — die
// Mitgliedschaft ist bereits beendet.
func TestLeave_ZweitesVerlassen403(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "Gruppe", owner, memberA)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)
	token := testutil.Token(t, memberA, "standard", nil)
	path := "/api/chat/conversations/" + itoa(convID) + "/members/me"

	res := testutil.Do(t, srv, http.MethodDelete, path, token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("first leave: expected 204, got %d", res.StatusCode)
	}

	res2 := testutil.Do(t, srv, http.MethodDelete, path, token, nil)
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusForbidden {
		t.Fatalf("second leave: expected 403, got %d", res2.StatusCode)
	}

	var n int
	db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conversation_id=? AND is_system=1`,
		convID).Scan(&n)
	if n != 1 {
		t.Errorf("expected exactly one system message, got %d", n)
	}
}
