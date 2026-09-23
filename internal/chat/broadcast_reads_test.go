package chat_test

import (
	"database/sql"
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

// fanOutBroadcast legt eine Mitteilung samt Fan-out an, wie SendBroadcast es tut:
// eine broadcast_reads-Zeile je Empfänger, der Absender mit gesetztem read_at.
func fanOutBroadcast(t *testing.T, db *sql.DB, sender int, recipients ...int) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO broadcasts (sender_id, body) VALUES (?, 'Hallenordnung')`, sender)
	if err != nil {
		t.Fatalf("insert broadcast: %v", err)
	}
	id, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO broadcast_reads (broadcast_id, user_id, read_at) VALUES (?, ?, CURRENT_TIMESTAMP)`, id, sender); err != nil {
		t.Fatalf("insert sender read: %v", err)
	}
	for _, uid := range recipients {
		if _, err := db.Exec(`INSERT INTO broadcast_reads (broadcast_id, user_id) VALUES (?, ?)`, id, uid); err != nil {
			t.Fatalf("insert recipient read: %v", err)
		}
	}
	return int(id)
}

func markBroadcastReadAt(t *testing.T, db *sql.DB, broadcastID, userID int, readAt string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE broadcast_reads SET read_at = ? WHERE broadcast_id = ? AND user_id = ?`, readAt, broadcastID, userID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
}

func createUsers(t *testing.T, db *sql.DB, n int) []int {
	t.Helper()
	ids := make([]int, n)
	for i := range ids {
		ids[i] = testutil.CreateUser(t, db, "standard")
	}
	return ids
}

// listBroadcastsRaw liefert die Rohobjekte, damit fehlende Felder von 0 unterscheidbar sind.
func listBroadcastsRaw(t *testing.T, db *sql.DB, userID int) map[int]map[string]json.RawMessage {
	t.Helper()
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/broadcasts", h.ListBroadcasts)
	})
	res := testutil.Get(t, srv, "/api/chat/broadcasts", testutil.Token(t, userID, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var list []map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	out := map[int]map[string]json.RawMessage{}
	for _, obj := range list {
		var id int
		json.Unmarshal(obj["id"], &id)
		out[id] = obj
	}
	return out
}

func aggOf(t *testing.T, obj map[string]json.RawMessage) (count, total int) {
	t.Helper()
	rc, okC := obj["readCount"]
	rt, okT := obj["readTotal"]
	if !okC || !okT {
		t.Fatalf("readCount/readTotal fehlen im JSON: %v", obj)
	}
	json.Unmarshal(rc, &count)
	json.Unmarshal(rt, &total)
	return count, total
}

// 1.4 — Absender sieht 3/10; Absender zählt nicht mit, obwohl er read_at trägt.
func TestListBroadcasts_SenderSeesAggregate(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	recipients := createUsers(t, db, 10)
	b := fanOutBroadcast(t, db, sender, recipients...)
	for _, uid := range recipients[:3] {
		markBroadcastReadAt(t, db, b, uid, "2026-09-01 10:00:00")
	}

	obj, ok := listBroadcastsRaw(t, db, sender)[b]
	if !ok {
		t.Fatalf("broadcast %d fehlt in der Liste des Absenders", b)
	}
	if count, total := aggOf(t, obj); count != 3 || total != 10 {
		t.Errorf("aggregate = %d/%d, want 3/10", count, total)
	}
}

// 1.4 — Niemand hat gelesen → 0/10 (Felder vorhanden, nicht weggelassen).
func TestListBroadcasts_SenderSeesZeroWhenUnread(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	b := fanOutBroadcast(t, db, sender, createUsers(t, db, 10)...)

	obj := listBroadcastsRaw(t, db, sender)[b]
	if count, total := aggOf(t, obj); count != 0 || total != 10 {
		t.Errorf("aggregate = %d/%d, want 0/10", count, total)
	}
}

// 1.4 — Empfänger sieht weder readCount noch readTotal.
func TestListBroadcasts_RecipientSeesNoAggregate(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	recipients := createUsers(t, db, 3)
	b := fanOutBroadcast(t, db, sender, recipients...)
	markBroadcastReadAt(t, db, b, recipients[1], "2026-09-01 10:00:00")

	obj, ok := listBroadcastsRaw(t, db, recipients[0])[b]
	if !ok {
		t.Fatalf("broadcast %d fehlt in der Liste des Empfängers", b)
	}
	if _, has := obj["readCount"]; has {
		t.Errorf("readCount darf für fremde Mitteilung nicht im JSON stehen")
	}
	if _, has := obj["readTotal"]; has {
		t.Errorf("readTotal darf für fremde Mitteilung nicht im JSON stehen")
	}
}

// 1.5 — Der Nenner ist eingefroren: neue Nutzer nach dem Senden ändern readTotal nicht.
func TestListBroadcasts_ReadTotalIsFrozen(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	b := fanOutBroadcast(t, db, sender, createUsers(t, db, 10)...)
	createUsers(t, db, 5)

	obj := listBroadcastsRaw(t, db, sender)[b]
	if _, total := aggOf(t, obj); total != 10 {
		t.Errorf("readTotal = %d, want 10 (Snapshot vom Fan-out)", total)
	}
}

// 1.6 — Weggewischt ohne Öffnen bleibt im Nenner und zählt nicht als gelesen.
func TestListBroadcasts_HiddenUnreadStaysInTotal(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	recipients := createUsers(t, db, 4)
	b := fanOutBroadcast(t, db, sender, recipients...)
	markBroadcastReadAt(t, db, b, recipients[0], "2026-09-01 10:00:00")

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Delete("/api/chat/broadcasts/{id}", h.DeleteBroadcast)
	})
	res := testutil.Delete(t, srv, fmt.Sprintf("/api/chat/broadcasts/%d", b), testutil.Token(t, recipients[1], "standard", nil))
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("hide: expected 204, got %d", res.StatusCode)
	}

	obj := listBroadcastsRaw(t, db, sender)[b]
	if count, total := aggOf(t, obj); count != 1 || total != 4 {
		t.Errorf("aggregate = %d/%d, want 1/4", count, total)
	}
}

func mountBroadcastReads(h *chat.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/api/chat/broadcasts/{id}/reads", h.BroadcastReads)
	}
}

// 2.4 — Absender: 200, sortiert nach readAt, ohne ihn selbst, ohne Nichtleser.
func TestBroadcastReads_Sender_OK(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	recipients := createUsers(t, db, 3)
	b := fanOutBroadcast(t, db, sender, recipients...)
	markBroadcastReadAt(t, db, b, recipients[0], "2026-09-01 12:00:00")
	markBroadcastReadAt(t, db, b, recipients[1], "2026-09-01 09:00:00")

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, mountBroadcastReads(h))
	res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/broadcasts/%d/reads", b), testutil.Token(t, sender, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var readers []readerEntry
	json.NewDecoder(res.Body).Decode(&readers)
	if len(readers) != 2 {
		t.Fatalf("expected 2 readers, got %d: %+v", len(readers), readers)
	}
	if readers[0].UserID != recipients[1] || readers[1].UserID != recipients[0] {
		t.Errorf("order = [%d %d], want [%d %d] (readAt aufsteigend)",
			readers[0].UserID, readers[1].UserID, recipients[1], recipients[0])
	}
	for _, rd := range readers {
		if rd.UserID == sender {
			t.Errorf("Absender darf nicht in der Leserliste stehen")
		}
	}
}

// 2.4 — Empfänger derselben Mitteilung und Unbeteiligte: 403; unbekannte ID: 404; ohne Token: 401.
func TestBroadcastReads_Errors(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	recipient := testutil.CreateUser(t, db, "standard")
	outsider := testutil.CreateUser(t, db, "standard")
	b := fanOutBroadcast(t, db, sender, recipient)
	markBroadcastReadAt(t, db, b, recipient, "2026-09-01 10:00:00")

	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, mountBroadcastReads(h))
	url := fmt.Sprintf("/api/chat/broadcasts/%d/reads", b)

	cases := []struct {
		name  string
		url   string
		token string
		want  int
	}{
		{"Empfänger", url, testutil.Token(t, recipient, "standard", nil), http.StatusForbidden},
		{"Unbeteiligter", url, testutil.Token(t, outsider, "standard", nil), http.StatusForbidden},
		{"unbekannte ID", "/api/chat/broadcasts/999999/reads", testutil.Token(t, sender, "standard", nil), http.StatusNotFound},
		{"unauthentifiziert", url, "", http.StatusUnauthorized},
	}
	for _, c := range cases {
		res := testutil.Get(t, srv, c.url, c.token)
		res.Body.Close()
		if res.StatusCode != c.want {
			t.Errorf("%s: expected %d, got %d", c.name, c.want, res.StatusCode)
		}
	}
}

// 3.4 — Erstes Markieren: genau ein Event an den Absender. Zweites Markieren: 204,
// kein weiteres Event, readCount bleibt 1. Absender markiert selbst: kein Event.
func TestMarkBroadcastRead_EventOnlyOnFirstRead(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	recipient := testutil.CreateUser(t, db, "standard")
	b := fanOutBroadcast(t, db, sender, recipient)

	sharedHub := hub.NewHub()
	h := chat.NewHandler(db, sharedHub, testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/broadcasts/{id}/read", h.MarkBroadcastRead)
	})
	ch := sharedHub.SubscribeUser(sender)
	defer sharedHub.UnsubscribeUser(sender, ch)

	url := fmt.Sprintf("/api/chat/broadcasts/%d/read", b)
	want := fmt.Sprintf("chat:broadcast-read:%d", b)

	res := testutil.Post(t, srv, url, testutil.Token(t, recipient, "standard", nil), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("first read: expected 204, got %d", res.StatusCode)
	}
	if ev, ok := recvWithinRR(ch, time.Second); !ok || ev != want {
		t.Fatalf("sender event = %q ok=%v, want %q", ev, ok, want)
	}

	res = testutil.Post(t, srv, url, testutil.Token(t, recipient, "standard", nil), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("second read: expected 204, got %d", res.StatusCode)
	}
	if ev, ok := recvWithinRR(ch, 200*time.Millisecond); ok {
		t.Errorf("zweites Markieren erzeugte ein weiteres Event: %q", ev)
	}

	obj := listBroadcastsRaw(t, db, sender)[b]
	if count, _ := aggOf(t, obj); count != 1 {
		t.Errorf("readCount = %d, want 1", count)
	}

	// Absender markiert seine eigene Mitteilung — kein broadcast-read an sich selbst.
	db.Exec(`UPDATE broadcast_reads SET read_at = NULL WHERE broadcast_id = ? AND user_id = ?`, b, sender)
	res = testutil.Post(t, srv, url, testutil.Token(t, sender, "standard", nil), nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("sender read: expected 204, got %d", res.StatusCode)
	}
	for {
		ev, ok := recvWithinRR(ch, 200*time.Millisecond)
		if !ok {
			break
		}
		if ev == want {
			t.Errorf("Absender bekam broadcast-read für die eigene Mitteilung")
		}
	}
}
