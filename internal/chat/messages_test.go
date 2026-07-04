package chat_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newMessagesServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/conversations/{id}/messages", h.ListMessages)
	})
}

// insertMessage wird aus unread_test.go mitgenutzt (gleiches Package).

type msgResp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

func getMessages(t *testing.T, srv *httptest.Server, path, token string) (int, []msgResp) {
	t.Helper()
	res := testutil.Get(t, srv, path, token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return res.StatusCode, nil
	}
	var msgs []msgResp
	if err := json.NewDecoder(res.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	return res.StatusCode, msgs
}

// Invariante: ?after=<msgId> liefert genau die Nachrichten mit id > msgId,
// aufsteigend; ohne Neueres eine leere Liste.
func TestMessagesAfter_ReturnsOnlyNewer(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, member)

	ids := make([]int, 0, 5)
	for _, body := range []string{"m1", "m2", "m3", "m4", "m5"} {
		ids = append(ids, insertMessage(t, db, convID, owner, body))
	}

	srv := newMessagesServer(t, db)
	token := testutil.Token(t, member, "standard", nil)

	status, msgs := getMessages(t, srv,
		"/api/chat/conversations/"+itoa(convID)+"/messages?after="+itoa(ids[2]), token)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 newer messages, got %d", len(msgs))
	}
	if msgs[0].ID != ids[3] || msgs[1].ID != ids[4] {
		t.Errorf("expected ids [%d %d] ascending, got [%d %d]", ids[3], ids[4], msgs[0].ID, msgs[1].ID)
	}

	// Kein Neueres → leere Liste (kein Fehler)
	status, msgs = getMessages(t, srv,
		"/api/chat/conversations/"+itoa(convID)+"/messages?after="+itoa(ids[4]), token)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for empty delta, got %d", status)
	}
	if len(msgs) != 0 {
		t.Errorf("expected empty list, got %d messages", len(msgs))
	}
}

// Invariante: ?before=<msgId> liefert die Seite der Nachrichten unmittelbar
// vor msgId (aufsteigend), begrenzt auf die Seitengröße; msgId selbst und
// Neueres fehlen.
func TestMessagesBefore_ReturnsOlderPage(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, member)

	// 105 ältere + 1 neueste Nachricht → die Seite vor der neuesten muss die
	// letzten 100 der älteren enthalten (unmittelbare Vorgänger, nicht die
	// ältesten 100).
	ids := make([]int, 0, 106)
	for i := 0; i < 106; i++ {
		ids = append(ids, insertMessage(t, db, convID, owner, "m"))
	}
	newest := ids[105]

	srv := newMessagesServer(t, db)
	token := testutil.Token(t, member, "standard", nil)

	status, msgs := getMessages(t, srv,
		"/api/chat/conversations/"+itoa(convID)+"/messages?before="+itoa(newest), token)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if len(msgs) != 100 {
		t.Fatalf("expected page of 100 older messages, got %d", len(msgs))
	}
	if msgs[0].ID != ids[5] || msgs[99].ID != ids[104] {
		t.Errorf("expected ids %d..%d ascending, got %d..%d", ids[5], ids[104], msgs[0].ID, msgs[99].ID)
	}
	for _, m := range msgs {
		if m.ID >= newest {
			t.Errorf("message %d is not older than before-cursor %d", m.ID, newest)
		}
	}

	// Seite vor der ältesten Nachricht → leer
	status, msgs = getMessages(t, srv,
		"/api/chat/conversations/"+itoa(convID)+"/messages?before="+itoa(ids[0]), token)
	if status != http.StatusOK {
		t.Fatalf("expected 200 for empty history page, got %d", status)
	}
	if len(msgs) != 0 {
		t.Errorf("expected empty list before oldest message, got %d", len(msgs))
	}
}

// Ohne Cursor-Parameter bleibt das Verhalten unverändert: letzte 100
// Nachrichten, älteste zuerst.
func TestMessages_NoParamUnchanged(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, member)

	ids := make([]int, 0, 3)
	for _, body := range []string{"m1", "m2", "m3"} {
		ids = append(ids, insertMessage(t, db, convID, owner, body))
	}

	srv := newMessagesServer(t, db)
	token := testutil.Token(t, member, "standard", nil)

	status, msgs := getMessages(t, srv,
		"/api/chat/conversations/"+itoa(convID)+"/messages", token)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	if msgs[0].ID != ids[0] || msgs[2].ID != ids[2] {
		t.Errorf("expected oldest-first ordering %v, got [%d %d %d]", ids, msgs[0].ID, msgs[1].ID, msgs[2].ID)
	}
}

// Fehlerfälle: ungültige Cursor-Werte und after+before zusammen → 400.
func TestMessagesAfter_InvalidCursorRejected(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)

	srv := newMessagesServer(t, db)
	token := testutil.Token(t, owner, "standard", nil)
	base := "/api/chat/conversations/" + itoa(convID) + "/messages"

	for _, tc := range []struct{ name, query string }{
		{"non-numeric after", "?after=abc"},
		{"negative after", "?after=-1"},
		{"non-numeric before", "?before=xyz"},
		{"after and before combined", "?after=1&before=2"},
	} {
		status, _ := getMessages(t, srv, base+tc.query, token)
		if status != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", tc.name, status)
		}
	}
}

// Autorisierung: Nicht-Mitglieder bekommen auch mit Cursor keine Nachrichten.
func TestMessagesAfter_NonMemberForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	outsider := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)
	insertMessage(t, db, convID, owner, "geheim")

	srv := newMessagesServer(t, db)

	status, _ := getMessages(t, srv,
		"/api/chat/conversations/"+itoa(convID)+"/messages?after=0",
		testutil.Token(t, outsider, "standard", nil))
	if status != http.StatusForbidden {
		t.Errorf("expected 403 for non-member, got %d", status)
	}
}
