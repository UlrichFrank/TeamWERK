package chat_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Chat-Alben (chat-mehrere-bilder): bis zu 10 Bilder je Nachricht bzw.
// Mitteilung. Quelle der Wahrheit sind message_media/broadcast_media;
// messages.media_id/broadcasts.media_id tragen das Bild auf Position 0.

type albumItem struct {
	ID     int    `json:"id"`
	URL    string `json:"url"`
	Width  *int   `json:"width"`
	Height *int   `json:"height"`
}

type albumMsg struct {
	ID       int         `json:"id"`
	MediaID  *int        `json:"mediaId"`
	MediaURL *string     `json:"mediaUrl"`
	Media    []albumItem `json:"media"`
}

func newAlbumServer(t *testing.T, db *sql.DB, eh *hub.EventHub) (*chat.Handler, *httptest.Server) {
	t.Helper()
	h := chat.NewHandler(db, eh, testutil.TestConfig())
	h.SetPushFn(func(*sql.DB, *appconfig.Config, int, string, string, string, int) {})
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/messages", h.SendMessage)
		r.Get("/api/chat/conversations/{id}/messages", h.ListMessages)
		r.Get("/api/chat/messages/{id}", h.GetMessage)
		r.Delete("/api/chat/messages/{id}", h.DeleteMessage)
		r.Post("/api/chat/broadcasts", h.SendBroadcast)
		r.Get("/api/chat/broadcasts", h.ListBroadcasts)
	})
	return h, srv
}

func insertMediaN(t *testing.T, db *sql.DB, uploader, n int) []int {
	t.Helper()
	ids := make([]int, n)
	for i := range ids {
		ids[i] = insertMedia(t, db, uploader, fmt.Sprintf("album-%d-%d-%d.png", uploader, n, i))
	}
	return ids
}

func albumPositions(t *testing.T, db *sql.DB, table, keyCol string, ownerID int) []int {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf(`SELECT media_id FROM %s WHERE %s = ? ORDER BY position`, table, keyCol), ownerID)
	if err != nil {
		t.Fatalf("album lesen: %v", err)
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	return ids
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func postStatus(t *testing.T, srv *httptest.Server, path, tok string, body map[string]any) (int, map[string]any) {
	t.Helper()
	res := testutil.Post(t, srv, path, tok, body)
	defer res.Body.Close()
	var out map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// ── Senden: Nachricht ───────────────────────────────────────────────────────

func TestSendMessage_AlbumMitDreiBildern(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, member)
	ids := insertMediaN(t, db, owner, 3)
	album := []int{ids[2], ids[0], ids[1]} // Reihenfolge des Requests, nicht der IDs

	eh := hub.NewHub()
	h, srv := newAlbumServer(t, db, eh)
	calls := make(chan pushCall, 4)
	h.SetPushFn(func(_ *sql.DB, _ *appconfig.Config, userID int, title, body, url string, badge int) {
		calls <- pushCall{userID, title, body, url, badge}
	})
	ch := eh.SubscribeUser(member)
	defer eh.UnsubscribeUser(member, ch)

	status, out := postStatus(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages",
		testutil.Token(t, owner, "standard", nil), map[string]any{"body": "", "mediaIds": album})
	if status != http.StatusCreated {
		t.Fatalf("erwartet 201, bekommen %d", status)
	}
	msgID := int(out["id"].(float64))

	if got := albumPositions(t, db, "message_media", "message_id", msgID); !sameInts(got, album) {
		t.Errorf("message_media = %v, erwartet %v", got, album)
	}
	var first sql.NullInt64
	db.QueryRow(`SELECT media_id FROM messages WHERE id = ?`, msgID).Scan(&first)
	if !first.Valid || int(first.Int64) != album[0] {
		t.Errorf("messages.media_id = %v, erwartet Position 0 = %d", first, album[0])
	}

	wantEvent := fmt.Sprintf("chat:new-message:%d", convID)
	if ev, ok := recvWithinRR(ch, time.Second); !ok || ev != wantEvent {
		t.Fatalf("Event = %q ok=%v, erwartet %q", ev, ok, wantEvent)
	}
	if ev, ok := recvWithinRR(ch, 200*time.Millisecond); ok {
		t.Errorf("zweites Event %q — ein Album ist genau eine Nachricht", ev)
	}
	pushes := collectPollPushes(t, calls, 1)
	if pushes[0].body != "3 Bilder" {
		t.Errorf("Push-Vorschau = %q, erwartet \"3 Bilder\"", pushes[0].body)
	}
	select {
	case extra := <-calls:
		t.Errorf("zweite Push %+v — erwartet genau eine je Empfänger", extra)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestSendMessage_ElfBilderAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)
	ids := insertMediaN(t, db, owner, 11)
	_, srv := newAlbumServer(t, db, hub.NewHub())

	status, out := postStatus(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages",
		testutil.Token(t, owner, "standard", nil), map[string]any{"body": "x", "mediaIds": ids})
	if status != http.StatusBadRequest || out["error"] != "invalid_media" {
		t.Fatalf("erwartet 400 invalid_media, bekommen %d %v", status, out)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM messages`); n != 0 {
		t.Errorf("%d Nachrichten angelegt, erwartet 0", n)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM message_media`); n != 0 {
		t.Errorf("%d Zuordnungen angelegt, erwartet 0", n)
	}
}

func TestSendMessage_FremdeMediaIDAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	other := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, other)
	own := insertMedia(t, db, owner, "own.png")
	foreign := insertMedia(t, db, other, "foreign.png")
	_, srv := newAlbumServer(t, db, hub.NewHub())
	tok := testutil.Token(t, owner, "standard", nil)
	path := "/api/chat/conversations/" + itoa(convID) + "/messages"

	for name, body := range map[string]map[string]any{
		"Liste":    {"body": "", "mediaIds": []int{own, foreign}},
		"Kurzform": {"body": "", "mediaId": foreign},
	} {
		if status, _ := postStatus(t, srv, path, tok, body); status != http.StatusBadRequest {
			t.Errorf("%s: erwartet 400, bekommen %d", name, status)
		}
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM messages`); n != 0 {
		t.Errorf("%d Nachrichten angelegt, erwartet 0", n)
	}
}

func TestSendMessage_BereitsVerwendeteMediaIDAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)
	mediaID := insertMedia(t, db, owner, "reuse.png")
	_, srv := newAlbumServer(t, db, hub.NewHub())
	tok := testutil.Token(t, owner, "standard", nil)
	path := "/api/chat/conversations/" + itoa(convID) + "/messages"

	if status, _ := postStatus(t, srv, path, tok, map[string]any{"body": "", "mediaId": mediaID}); status != http.StatusCreated {
		t.Fatalf("erste Verwendung: erwartet 201, bekommen %d", status)
	}
	if status, _ := postStatus(t, srv, path, tok, map[string]any{"body": "nochmal", "mediaIds": []int{mediaID}}); status != http.StatusBadRequest {
		t.Fatalf("zweite Verwendung: erwartet 400, bekommen %d", status)
	}
	// Auch nicht an einer Mitteilung.
	admin := testutil.CreateUser(t, db, "admin")
	adminMedia := insertMedia(t, db, admin, "admin-reuse.png")
	adminTok := testutil.Token(t, admin, "admin", nil)
	if status, _ := postStatus(t, srv, "/api/chat/broadcasts", adminTok,
		map[string]any{"body": "", "mediaId": adminMedia, "targets": clubWide("users")}); status != http.StatusCreated {
		t.Fatalf("Mitteilung: erwartet 201, bekommen %d", status)
	}
	adminConv := createGroupConv(t, db, "A", admin)
	if status, _ := postStatus(t, srv, "/api/chat/conversations/"+itoa(adminConv)+"/messages", adminTok,
		map[string]any{"body": "", "mediaId": adminMedia}); status != http.StatusBadRequest {
		t.Fatalf("an Mitteilung hängendes Bild: erwartet 400, bekommen %d", status)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM messages`); n != 1 {
		t.Errorf("%d Nachrichten, erwartet 1", n)
	}
}

func TestSendMessage_MediaIdUndMediaIdsZugleich(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)
	ids := insertMediaN(t, db, owner, 2)
	_, srv := newAlbumServer(t, db, hub.NewHub())

	status, _ := postStatus(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages",
		testutil.Token(t, owner, "standard", nil), map[string]any{"body": "", "mediaId": ids[0], "mediaIds": []int{ids[1]}})
	if status != http.StatusBadRequest {
		t.Fatalf("erwartet 400, bekommen %d", status)
	}
}

func TestSendMessage_DoppelteIDImAlbum(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)
	mediaID := insertMedia(t, db, owner, "dup.png")
	_, srv := newAlbumServer(t, db, hub.NewHub())

	status, _ := postStatus(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages",
		testutil.Token(t, owner, "standard", nil), map[string]any{"body": "", "mediaIds": []int{mediaID, mediaID}})
	if status != http.StatusBadRequest {
		t.Fatalf("erwartet 400, bekommen %d", status)
	}
}

func TestSendMessage_LeereListeOhneTextAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)
	_, srv := newAlbumServer(t, db, hub.NewHub())

	status, _ := postStatus(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages",
		testutil.Token(t, owner, "standard", nil), map[string]any{"body": "", "mediaIds": []int{}})
	if status != http.StatusBadRequest {
		t.Fatalf("erwartet 400, bekommen %d", status)
	}
}

// ── Senden: Mitteilung ──────────────────────────────────────────────────────

func TestCreateBroadcast_AlbumMitBildern(t *testing.T) {
	db := testutil.NewDB(t)
	admin := testutil.CreateUser(t, db, "admin")
	empf := testutil.CreateUser(t, db, "standard")
	ids := insertMediaN(t, db, admin, 4)
	h, srv := newAlbumServer(t, db, hub.NewHub())
	calls := make(chan pushCall, 4)
	h.SetPushFn(func(_ *sql.DB, _ *appconfig.Config, userID int, title, body, url string, badge int) {
		calls <- pushCall{userID, title, body, url, badge}
	})

	status, out := postStatus(t, srv, "/api/chat/broadcasts", testutil.Token(t, admin, "admin", nil),
		map[string]any{"body": "", "mediaIds": ids, "targets": clubWide("users")})
	if status != http.StatusCreated {
		t.Fatalf("erwartet 201, bekommen %d", status)
	}
	bcID := int(out["id"].(float64))
	if got := albumPositions(t, db, "broadcast_media", "broadcast_id", bcID); !sameInts(got, ids) {
		t.Errorf("broadcast_media = %v, erwartet %v", got, ids)
	}
	var first sql.NullInt64
	db.QueryRow(`SELECT media_id FROM broadcasts WHERE id = ?`, bcID).Scan(&first)
	if !first.Valid || int(first.Int64) != ids[0] {
		t.Errorf("broadcasts.media_id = %v, erwartet %d", first, ids[0])
	}
	pushes := collectPollPushes(t, calls, 1)
	if pushes[0].userID != empf || pushes[0].body != "4 Bilder" {
		t.Errorf("Push = %+v, erwartet an %d mit \"4 Bilder\"", pushes[0], empf)
	}
}

func TestCreateBroadcast_FremdeMediaIDAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	admin := testutil.CreateUser(t, db, "admin")
	other := testutil.CreateUser(t, db, "standard")
	foreign := insertMedia(t, db, other, "bc-foreign.png")
	_, srv := newAlbumServer(t, db, hub.NewHub())
	tok := testutil.Token(t, admin, "admin", nil)

	for name, body := range map[string]map[string]any{
		"Liste":    {"body": "x", "mediaIds": []int{foreign}, "targets": clubWide("users")},
		"Kurzform": {"body": "x", "mediaId": foreign, "targets": clubWide("users")},
	} {
		if status, _ := postStatus(t, srv, "/api/chat/broadcasts", tok, body); status != http.StatusBadRequest {
			t.Errorf("%s: erwartet 400, bekommen %d", name, status)
		}
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM broadcasts`); n != 0 {
		t.Errorf("%d Mitteilungen angelegt, erwartet 0", n)
	}
}

func TestCreateBroadcast_ElfBilderAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	admin := testutil.CreateUser(t, db, "admin")
	ids := insertMediaN(t, db, admin, 11)
	_, srv := newAlbumServer(t, db, hub.NewHub())

	status, _ := postStatus(t, srv, "/api/chat/broadcasts", testutil.Token(t, admin, "admin", nil),
		map[string]any{"body": "", "mediaIds": ids, "targets": clubWide("users")})
	if status != http.StatusBadRequest {
		t.Fatalf("erwartet 400, bekommen %d", status)
	}
	if n := countRows(t, db, `SELECT COUNT(*) FROM broadcasts`); n != 0 {
		t.Errorf("%d Mitteilungen angelegt, erwartet 0", n)
	}
}

// ── Lesen ───────────────────────────────────────────────────────────────────

func listAlbumMsgs(t *testing.T, srv *httptest.Server, convID int, tok string) []albumMsg {
	t.Helper()
	res := testutil.Get(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages", tok)
	defer res.Body.Close()
	var msgs []albumMsg
	if err := json.NewDecoder(res.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return msgs
}

func mediaIDsOf(items []albumItem) []int {
	ids := make([]int, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

func TestListMessages_AlbumReihenfolgeUndAltfelder(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, member)
	a := insertMediaWithDims(t, db, owner, "a.png", 800, 600)
	b := insertMedia(t, db, owner, "b.png")
	c := insertMedia(t, db, owner, "c.png")
	_, srv := newAlbumServer(t, db, hub.NewHub())

	postStatus(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages",
		testutil.Token(t, owner, "standard", nil), map[string]any{"body": "Spieltag", "mediaIds": []int{c, a, b}})
	postStatus(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages",
		testutil.Token(t, owner, "standard", nil), map[string]any{"body": "nur Text"})

	msgs := listAlbumMsgs(t, srv, convID, testutil.Token(t, member, "standard", nil))
	if len(msgs) != 2 {
		t.Fatalf("erwartet 2 Nachrichten, bekommen %d", len(msgs))
	}
	m := msgs[0]
	if got := mediaIDsOf(m.Media); !sameInts(got, []int{c, a, b}) {
		t.Errorf("media = %v, erwartet [%d %d %d]", got, c, a, b)
	}
	if m.Media[0].URL != "/media/"+itoa(c) {
		t.Errorf("url = %q", m.Media[0].URL)
	}
	if m.Media[1].Width == nil || *m.Media[1].Width != 800 || m.Media[0].Width != nil {
		t.Errorf("Dimensionen falsch zugeordnet: %+v", m.Media)
	}
	if m.MediaID == nil || *m.MediaID != c || m.MediaURL == nil || *m.MediaURL != "/media/"+itoa(c) {
		t.Errorf("Altfelder = %v/%v, erwartet erstes Bild %d", m.MediaID, m.MediaURL, c)
	}
	if msgs[1].Media == nil || len(msgs[1].Media) != 0 {
		t.Errorf("Textnachricht: media = %v, erwartet leere Liste", msgs[1].Media)
	}
}

// Bestandsbild aus der Zeit vor den Alben: Migration 071 übernimmt es als
// Position 0; hier entspricht das einer Nachricht mit media_id plus der einen
// message_media-Zeile, wie die Migration sie schreibt.
func TestListMessages_BestandsbildAlsEinElementAlbum(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)
	mediaID := insertMedia(t, db, owner, "legacy.png")
	res, err := db.Exec(`INSERT INTO messages (conversation_id, sender_id, body, media_id) VALUES (?, ?, '', ?)`, convID, owner, mediaID)
	if err != nil {
		t.Fatalf("messages: %v", err)
	}
	msgID, _ := res.LastInsertId()
	db.Exec(`INSERT INTO message_media (message_id, media_id, position) SELECT id, media_id, 0 FROM messages WHERE id = ?`, msgID)
	_, srv := newAlbumServer(t, db, hub.NewHub())

	msgs := listAlbumMsgs(t, srv, convID, testutil.Token(t, owner, "standard", nil))
	if len(msgs) != 1 || !sameInts(mediaIDsOf(msgs[0].Media), []int{mediaID}) {
		t.Fatalf("erwartet Ein-Element-Album [%d], bekommen %+v", mediaID, msgs)
	}
}

func TestListMessages_GeloeschteNachrichtOhneMedia(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)
	ids := insertMediaN(t, db, owner, 2)
	_, srv := newAlbumServer(t, db, hub.NewHub())
	tok := testutil.Token(t, owner, "standard", nil)

	_, out := postStatus(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages", tok,
		map[string]any{"body": "", "mediaIds": ids})
	msgID := int(out["id"].(float64))
	testutil.Delete(t, srv, "/api/chat/messages/"+itoa(msgID), tok).Body.Close()

	msgs := listAlbumMsgs(t, srv, convID, tok)
	if len(msgs) != 1 || msgs[0].Media == nil || len(msgs[0].Media) != 0 {
		t.Fatalf("gelöschte Nachricht: media = %+v, erwartet leere Liste", msgs)
	}
}

func TestGetMessage_Album(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)
	ids := insertMediaN(t, db, owner, 3)
	_, srv := newAlbumServer(t, db, hub.NewHub())
	tok := testutil.Token(t, owner, "standard", nil)

	_, out := postStatus(t, srv, "/api/chat/conversations/"+itoa(convID)+"/messages", tok,
		map[string]any{"body": "hallo", "mediaIds": ids})
	msgID := int(out["id"].(float64))

	res := testutil.Get(t, srv, "/api/chat/messages/"+itoa(msgID), tok)
	defer res.Body.Close()
	var got albumMsg
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !sameInts(mediaIDsOf(got.Media), ids) {
		t.Errorf("media = %v, erwartet %v", mediaIDsOf(got.Media), ids)
	}
	if got.MediaID == nil || *got.MediaID != ids[0] {
		t.Errorf("mediaId = %v, erwartet %d", got.MediaID, ids[0])
	}
}

func TestListBroadcasts_Album(t *testing.T) {
	db := testutil.NewDB(t)
	admin := testutil.CreateUser(t, db, "admin")
	empf := testutil.CreateUser(t, db, "standard")
	a := insertMedia(t, db, admin, "bc-a.png")
	b := insertMediaWithDims(t, db, admin, "bc-b.png", 640, 480)
	c := insertMedia(t, db, admin, "bc-c.png")
	_, srv := newAlbumServer(t, db, hub.NewHub())

	postStatus(t, srv, "/api/chat/broadcasts", testutil.Token(t, admin, "admin", nil),
		map[string]any{"body": "Saisonabschluss", "mediaIds": []int{b, c, a}, "targets": clubWide("users")})
	postStatus(t, srv, "/api/chat/broadcasts", testutil.Token(t, admin, "admin", nil),
		map[string]any{"body": "ohne Bild", "targets": clubWide("users")})

	res := testutil.Get(t, srv, "/api/chat/broadcasts", testutil.Token(t, empf, "standard", nil))
	defer res.Body.Close()
	var list []albumMsg
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("erwartet 2 Mitteilungen, bekommen %d", len(list))
	}
	var withImg, without albumMsg
	for _, bc := range list {
		if len(bc.Media) > 0 {
			withImg = bc
		} else {
			without = bc
		}
	}
	if !sameInts(mediaIDsOf(withImg.Media), []int{b, c, a}) {
		t.Errorf("media = %v, erwartet [%d %d %d]", mediaIDsOf(withImg.Media), b, c, a)
	}
	if withImg.Media[0].Width == nil || *withImg.Media[0].Width != 640 {
		t.Errorf("Dimension fehlt: %+v", withImg.Media[0])
	}
	if withImg.MediaID == nil || *withImg.MediaID != b {
		t.Errorf("mediaId = %v, erwartet %d", withImg.MediaID, b)
	}
	if without.Media == nil {
		t.Error("Mitteilung ohne Bild: media fehlt, erwartet leere Liste")
	}
}
