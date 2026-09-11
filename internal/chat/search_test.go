// Diese Datei liegt bewusst in package chat (Whitebox-Test), nicht im sonst
// üblichen chat_test — der Unit-Test für die unexportierte snippet-Funktion
// (weiter unten) braucht direkten Zugriff. Die HTTP-Helfer sind deshalb
// eigene, schlanke Kopien statt der chat_test-Helfer aus handler_test.go/
// unread_test.go (die aus einem anderen Package nicht erreichbar sind).
package chat

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newSearchTestConv(t *testing.T, db *sql.DB, creator int, members ...int) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO conversations (type, name, created_by) VALUES ('group', 'G', ?)`, creator)
	if err != nil {
		t.Fatalf("create conv: %v", err)
	}
	convID, _ := res.LastInsertId()
	for _, uid := range append([]int{creator}, members...) {
		if _, err := db.Exec(
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`,
			convID, uid); err != nil {
			t.Fatalf("add member %d: %v", uid, err)
		}
	}
	return int(convID)
}

func newSearchTestMessage(t *testing.T, db *sql.DB, convID, senderID int, body string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`,
		convID, senderID, body)
	if err != nil {
		t.Fatalf("insert message: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func newSearchTestBroadcast(t *testing.T, db *sql.DB, senderID int, body string, recipients ...int) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO broadcasts (sender_id, body) VALUES (?, ?)`, senderID, body)
	if err != nil {
		t.Fatalf("insert broadcast: %v", err)
	}
	bid, _ := res.LastInsertId()
	for _, uid := range recipients {
		if _, err := db.Exec(
			`INSERT OR IGNORE INTO broadcast_reads (broadcast_id, user_id) VALUES (?, ?)`,
			bid, uid); err != nil {
			t.Fatalf("broadcast_reads: %v", err)
		}
	}
	return int(bid)
}

func newSearchServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	h := NewHandler(db, hub.NewHub(), testutil.TestConfig())
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/chat/search", h.Search)
		r.Get("/api/chat/conversations/{id}/messages", h.ListMessages)
	})
}

type searchResp struct {
	Items []SearchHit `json:"items"`
	Total int         `json:"total"`
}

func doSearch(t *testing.T, srv *httptest.Server, q, token string) (int, searchResp) {
	t.Helper()
	path := "/api/chat/search?" + (url.Values{"q": {q}}).Encode()
	res := testutil.Get(t, srv, path, token)
	defer res.Body.Close()
	var body searchResp
	if res.StatusCode == http.StatusOK {
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode search response: %v", err)
		}
	}
	return res.StatusCode, body
}

type aroundResp struct {
	Items []struct {
		ID int `json:"id"`
	} `json:"items"`
	HasOlder bool `json:"hasOlder"`
	HasNewer bool `json:"hasNewer"`
}

// TestSearch_TrefferInEigenerKonversation: ein Suchbegriff, der im Body einer
// Nachricht einer aktiven Konversation des Anfragenden vorkommt, liefert genau
// diesen Treffer mit Konversation, Absender, Snippet und Zeitpunkt.
func TestSearch_TrefferInEigenerKonversation(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	convID := newSearchTestConv(t, db, owner, member)
	newSearchTestMessage(t, db, convID, owner, "Die Hallenadresse ist Musterweg 7")

	srv := newSearchServer(t, db)
	status, body := doSearch(t, srv, "Hallenadresse", testutil.Token(t, member, "standard", nil))

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body.Total != 1 {
		t.Fatalf("expected total 1, got %d", body.Total)
	}
	if len(body.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(body.Items))
	}
	hit := body.Items[0]
	if hit.Kind != "message" {
		t.Errorf("expected kind message, got %q", hit.Kind)
	}
	if hit.ConversationID == nil || *hit.ConversationID != convID {
		t.Errorf("expected conversationId %d, got %v", convID, hit.ConversationID)
	}
	if !strings.Contains(hit.Snippet, "Hallenadresse") {
		t.Errorf("expected snippet to contain search term, got %q", hit.Snippet)
	}
	if hit.SenderName == "" {
		t.Errorf("expected non-empty senderName")
	}
}

// TestSearch_TrefferInMitteilung: ein Suchbegriff, der im Body einer für den
// Nutzer sichtbaren Mitteilung vorkommt, liefert einen kind=broadcast-Treffer.
func TestSearch_TrefferInMitteilung(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	recipient := testutil.CreateUser(t, db, "standard")
	newSearchTestBroadcast(t, db, sender, "Sammelbestellung Trikots bis Freitag", sender, recipient)

	srv := newSearchServer(t, db)
	status, body := doSearch(t, srv, "Trikots", testutil.Token(t, recipient, "standard", nil))

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body.Total != 1 {
		t.Fatalf("expected total 1, got %d", body.Total)
	}
	hit := body.Items[0]
	if hit.Kind != "broadcast" {
		t.Errorf("expected kind broadcast, got %q", hit.Kind)
	}
	if hit.ConversationName != "Mitteilung" {
		t.Errorf("expected conversationName 'Mitteilung', got %q", hit.ConversationName)
	}
	if hit.ConversationID != nil {
		t.Errorf("expected no conversationId for a broadcast hit, got %v", *hit.ConversationID)
	}
}

// TestSearch_FremdeKonversationNichtGefunden: ein Treffer in einer
// Konversation, in der der Anfragende nie Mitglied war, sowie in einer, die
// er verlassen hat (left_at gesetzt), erscheint nicht in den Ergebnissen.
func TestSearch_FremdeKonversationNichtGefunden(t *testing.T) {
	db := testutil.NewDB(t)
	me := testutil.CreateUser(t, db, "standard")
	owner := testutil.CreateUser(t, db, "standard")
	other := testutil.CreateUser(t, db, "standard")

	foreignConv := newSearchTestConv(t, db, owner, other)
	newSearchTestMessage(t, db, foreignConv, owner, "Geheimzahl 424242 fremd")

	leftConv := newSearchTestConv(t, db, owner, me)
	newSearchTestMessage(t, db, leftConv, owner, "Geheimzahl 424242 verlassen")
	if _, err := db.Exec(
		`UPDATE conversation_members SET left_at = CURRENT_TIMESTAMP WHERE conversation_id = ? AND user_id = ?`,
		leftConv, me); err != nil {
		t.Fatalf("set left_at: %v", err)
	}

	srv := newSearchServer(t, db)
	status, body := doSearch(t, srv, "424242", testutil.Token(t, me, "standard", nil))

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body.Total != 0 {
		t.Fatalf("expected total 0, got %d", body.Total)
	}
}

// TestSearch_GeloeschteNachrichtNichtGefunden: der body einer gelöschten
// Nachricht bleibt in der DB stehen (DeleteMessage maskiert nur die
// Projektion) — die Suche muss deleted_at trotzdem explizit ausschließen.
func TestSearch_GeloeschteNachrichtNichtGefunden(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := newSearchTestConv(t, db, owner)
	msgID := newSearchTestMessage(t, db, convID, owner, "Einzigartigkeitswort Zyxwvut")
	if _, err := db.Exec(`UPDATE messages SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`, msgID); err != nil {
		t.Fatalf("soft-delete message: %v", err)
	}

	srv := newSearchServer(t, db)
	status, body := doSearch(t, srv, "Zyxwvut", testutil.Token(t, owner, "standard", nil))

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body.Total != 0 {
		t.Fatalf("expected total 0 for a deleted message, got %d", body.Total)
	}
}

// TestSearch_LeeresQ400: ohne q bzw. mit ausschließlich Whitespace antwortet
// die Suche mit 400 statt einer ungefilterten Volltabelle.
func TestSearch_LeeresQ400(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	srv := newSearchServer(t, db)
	token := testutil.Token(t, owner, "standard", nil)

	for _, q := range []string{"", "   "} {
		status, _ := doSearch(t, srv, q, token)
		if status != http.StatusBadRequest {
			t.Errorf("q=%q: expected 400, got %d", q, status)
		}
	}

	// Ganz ohne q-Parameter.
	res := testutil.Get(t, srv, "/api/chat/search", token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("without q param: expected 400, got %d", res.StatusCode)
	}
}

// TestSearch_ProzentUndUnterstrichLiteral: %/_ im Suchbegriff sind literal zu
// behandeln, nicht als SQLite-LIKE-Wildcards (design.md Entscheidung 4).
func TestSearch_ProzentUndUnterstrichLiteral(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := newSearchTestConv(t, db, owner)
	newSearchTestMessage(t, db, convID, owner, "50% Rabatt")
	newSearchTestMessage(t, db, convID, owner, "50 Prozent Rabatt")
	newSearchTestMessage(t, db, convID, owner, "a_c")
	newSearchTestMessage(t, db, convID, owner, "abc")

	srv := newSearchServer(t, db)
	token := testutil.Token(t, owner, "standard", nil)

	status, body := doSearch(t, srv, "50%", token)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body.Total != 1 {
		t.Fatalf("expected exactly 1 hit for the literal '50%%', got %d", body.Total)
	}

	status, body = doSearch(t, srv, "a_c", token)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if body.Total != 1 {
		t.Fatalf("expected exactly 1 hit for the literal 'a_c' (not matching 'abc'), got %d", body.Total)
	}
	if body.Items[0].Snippet != "a_c" {
		t.Errorf("expected snippet 'a_c', got %q", body.Items[0].Snippet)
	}
}

// TestListMessages_AroundLiefertFensterUmZiel: der around-Cursor liefert bis
// zu messagePageSize/2 Nachrichten vor und nach der Ziel-id, aufsteigend
// sortiert, mit korrekten hasOlder/hasNewer-Flags am Rand des Fensters.
func TestListMessages_AroundLiefertFensterUmZiel(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	convID := newSearchTestConv(t, db, owner, member)

	ids := make([]int, 0, 30)
	for i := 0; i < 30; i++ {
		ids = append(ids, newSearchTestMessage(t, db, convID, owner, "m"))
	}
	target := ids[14] // 15. Nachricht

	srv := newSearchServer(t, db)
	token := testutil.Token(t, member, "standard", nil)

	res := testutil.Get(t, srv,
		"/api/chat/conversations/"+strconv.Itoa(convID)+"/messages?around="+strconv.Itoa(target), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var resp aroundResp
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 30 {
		t.Fatalf("expected all 30 messages in the window, got %d", len(resp.Items))
	}
	for i := 1; i < len(resp.Items); i++ {
		if resp.Items[i].ID <= resp.Items[i-1].ID {
			t.Fatalf("expected ascending ids, got %d after %d", resp.Items[i].ID, resp.Items[i-1].ID)
		}
	}
	found := false
	for _, m := range resp.Items {
		if m.ID == target {
			found = true
		}
	}
	if !found {
		t.Errorf("expected target id %d in the window", target)
	}
	if resp.HasOlder {
		t.Errorf("expected hasOlder=false, all 30 messages fit in the 50+50 window")
	}
	if resp.HasNewer {
		t.Errorf("expected hasNewer=false, all 30 messages fit in the 50+50 window")
	}

	// Zweiter Teil: 130 Nachrichten, around = id der 65. → Fenster kappt auf
	// beiden Seiten bei 50.
	convID2 := newSearchTestConv(t, db, owner, member)
	ids2 := make([]int, 0, 130)
	for i := 0; i < 130; i++ {
		ids2 = append(ids2, newSearchTestMessage(t, db, convID2, owner, "m"))
	}
	target2 := ids2[64] // 65. Nachricht

	res2 := testutil.Get(t, srv,
		"/api/chat/conversations/"+strconv.Itoa(convID2)+"/messages?around="+strconv.Itoa(target2), token)
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res2.StatusCode)
	}
	var resp2 aroundResp
	if err := json.NewDecoder(res2.Body).Decode(&resp2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp2.Items) != 100 {
		t.Fatalf("expected exactly 100 messages (50+50), got %d", len(resp2.Items))
	}
	if !resp2.HasOlder {
		t.Errorf("expected hasOlder=true")
	}
	if !resp2.HasNewer {
		t.Errorf("expected hasNewer=true")
	}
}

// TestListMessages_AroundMitAfterOderBefore400: around ist zu after/before
// mutually exclusive, jede Kombination antwortet mit 400.
func TestListMessages_AroundMitAfterOderBefore400(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := newSearchTestConv(t, db, owner)
	msgID := newSearchTestMessage(t, db, convID, owner, "m")

	srv := newSearchServer(t, db)
	token := testutil.Token(t, owner, "standard", nil)
	base := "/api/chat/conversations/" + strconv.Itoa(convID) + "/messages"

	for _, q := range []string{
		"?around=" + strconv.Itoa(msgID) + "&after=" + strconv.Itoa(msgID),
		"?around=" + strconv.Itoa(msgID) + "&before=" + strconv.Itoa(msgID),
	} {
		res := testutil.Get(t, srv, base+q, token)
		defer res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", q, res.StatusCode)
		}
	}
}

// TestSnippet ist der isolierte Unit-Test für den rune-genauen
// Ausschnitt-Helfer (design.md Entscheidung 5): Fundstelle mittig, am Anfang,
// kein Treffer, und ein Umlaut-Fall, der keine Rune zerschneiden darf.
func TestSnippet(t *testing.T) {
	// Fundstelle mittig in langem Text → beide Ellipsen.
	mid := strings.Repeat("x", 100) + "GEFUNDEN" + strings.Repeat("y", 100)
	got := snippet(mid, "gefunden")
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "…") {
		t.Errorf("expected both ellipses for a mid-text hit, got %q", got)
	}
	if !strings.Contains(got, "GEFUNDEN") {
		t.Errorf("expected the snippet to contain the match, got %q", got)
	}

	// Fundstelle am Anfang → nur hintere Ellipse.
	start := "GEFUNDEN am Anfang " + strings.Repeat("z", 200)
	got = snippet(start, "gefunden")
	if strings.HasPrefix(got, "…") {
		t.Errorf("expected no leading ellipsis for a match at the start, got %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected a trailing ellipsis (text longer than the window), got %q", got)
	}

	// Kein Treffer → Anfang des Texts + Ellipse.
	noMatch := strings.Repeat("a", 200)
	got = snippet(noMatch, "nichtvorhanden")
	if strings.HasPrefix(got, "…") {
		t.Errorf("expected no leading ellipsis without a match, got %q", got)
	}
	withoutEllipsis := strings.TrimSuffix(got, "…")
	if withoutEllipsis == got {
		t.Errorf("expected a trailing ellipsis for a truncated no-match text, got %q", got)
	}
	if utf8.RuneCountInString(withoutEllipsis) != searchSnippetLen {
		t.Errorf("expected exactly %d runes before the ellipsis, got %d", searchSnippetLen, utf8.RuneCountInString(withoutEllipsis))
	}

	// Umlaute im Text dürfen keine Rune zerschneiden.
	umlaut := "Vorbereitung für Ünterricht: " + strings.Repeat("ä", 100) + "TREFFER" + strings.Repeat("ö", 100)
	got = snippet(umlaut, "treffer")
	if !utf8.ValidString(got) {
		t.Errorf("expected valid UTF-8 (no rune cut), got %q", got)
	}
	if !strings.Contains(got, "TREFFER") {
		t.Errorf("expected the snippet to contain the match despite umlauts, got %q", got)
	}
}
