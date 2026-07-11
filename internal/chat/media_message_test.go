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

func newMediaMsgServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	return testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/messages", h.SendMessage)
		r.Get("/api/chat/conversations/{id}/messages", h.ListMessages)
		r.Post("/api/chat/broadcasts", h.SendBroadcast)
		r.Get("/api/chat/broadcasts", h.ListBroadcasts)
	})
}

func insertMedia(t *testing.T, db *sql.DB, uploader int, diskName string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO media (disk_name, mime_type, size, uploaded_by) VALUES (?, 'image/png', 10, ?)`,
		diskName, uploader)
	if err != nil {
		t.Fatalf("insertMedia: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

type mediaMsg struct {
	ID       int     `json:"id"`
	Preview  string  `json:"preview"`
	MediaID  *int    `json:"mediaId"`
	MediaURL *string `json:"mediaUrl"`
}

// Invariante: Eine reine Bildnachricht (leerer body + mediaId) wird gespeichert.
func TestSendMessage_ImageOnly(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, member)
	mediaID := insertMedia(t, db, member, "img-only.png")

	srv := newMediaMsgServer(t, db)
	tok := testutil.Token(t, member, "standard", nil)

	res := testutil.Post(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages", tok,
		map[string]any{"body": "", "mediaId": mediaID})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var gotBody string
	var gotMedia sql.NullInt64
	db.QueryRow(`SELECT body, media_id FROM messages WHERE conversation_id = ?`, convID).Scan(&gotBody, &gotMedia)
	if gotBody != "" {
		t.Errorf("expected empty body, got %q", gotBody)
	}
	if !gotMedia.Valid || int(gotMedia.Int64) != mediaID {
		t.Errorf("expected media_id %d, got %v", mediaID, gotMedia)
	}
}

// Fehlerfall: leerer body ohne mediaId → 400.
func TestSendMessage_EmptyNoMedia(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)

	srv := newMediaMsgServer(t, db)
	tok := testutil.Token(t, owner, "standard", nil)

	res := testutil.Post(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages", tok,
		map[string]any{"body": "   "})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

// Invariante: ListMessages liefert mediaId + mediaUrl (/media/<id>).
func TestListMessages_Media(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, member)
	mediaID := insertMedia(t, db, member, "list-media.png")

	srv := newMediaMsgServer(t, db)
	tok := testutil.Token(t, member, "standard", nil)

	post := testutil.Post(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages", tok,
		map[string]any{"body": "schau mal", "mediaId": mediaID})
	post.Body.Close()

	res := testutil.Get(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages", tok)
	defer res.Body.Close()
	var msgs []mediaMsg
	if err := json.NewDecoder(res.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.MediaID == nil || *m.MediaID != mediaID {
		t.Errorf("expected mediaId %d, got %v", mediaID, m.MediaID)
	}
	want := "/media/" + itoa(mediaID)
	if m.MediaURL == nil || *m.MediaURL != want {
		t.Errorf("expected mediaUrl %q, got %v", want, m.MediaURL)
	}
}

// Invariante: Eine reine Bild-Mitteilung (leerer body + mediaId) wird gespeichert.
func TestSendBroadcast_ImageOnly(t *testing.T) {
	db := testutil.NewDB(t)
	admin := testutil.CreateUser(t, db, "admin")
	mediaID := insertMedia(t, db, admin, "bc-img.png")

	srv := newMediaMsgServer(t, db)
	tok := testutil.Token(t, admin, "admin", nil)

	res := testutil.Post(t, srv, "/api/chat/broadcasts", tok,
		map[string]any{"body": "", "mediaId": mediaID, "targetType": "all"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var gotBody string
	var gotMedia sql.NullInt64
	db.QueryRow(`SELECT body, media_id FROM broadcasts`).Scan(&gotBody, &gotMedia)
	if gotBody != "" {
		t.Errorf("expected empty body, got %q", gotBody)
	}
	if !gotMedia.Valid || int(gotMedia.Int64) != mediaID {
		t.Errorf("expected media_id %d, got %v", mediaID, gotMedia)
	}
}

// Fehlerfall: Broadcast mit leerem body ohne mediaId → 400.
func TestSendBroadcast_EmptyNoMedia(t *testing.T) {
	db := testutil.NewDB(t)
	admin := testutil.CreateUser(t, db, "admin")

	srv := newMediaMsgServer(t, db)
	tok := testutil.Token(t, admin, "admin", nil)

	res := testutil.Post(t, srv, "/api/chat/broadcasts", tok,
		map[string]any{"body": "", "targetType": "all"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}
