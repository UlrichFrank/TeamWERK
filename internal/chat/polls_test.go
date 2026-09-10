package chat_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// --- Response-Typen (design.md §4) ----------------------------------------

type pollVoterResp struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type pollOptionResp struct {
	ID     int             `json:"id"`
	Label  string          `json:"label"`
	Count  int             `json:"count"`
	Voted  bool            `json:"voted"`
	Voters []pollVoterResp `json:"voters"`
}

type pollResp struct {
	AllowMultiple bool             `json:"allowMultiple"`
	ClosedAt      *string          `json:"closedAt"`
	VoterCount    int              `json:"voterCount"`
	Options       []pollOptionResp `json:"options"`
}

type createdMsgResp struct {
	ID int `json:"id"`
}

// --- Server- und Request-Helfer -------------------------------------------

func newPollServer(t *testing.T, db *sql.DB, hb *hub.EventHub) (*chat.Handler, *httptest.Server) {
	t.Helper()
	h := chat.NewHandler(db, hb, testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations/{id}/polls", h.CreatePoll)
		r.Put("/api/chat/messages/{id}/poll/vote", h.VotePoll)
		r.Post("/api/chat/messages/{id}/poll/close", h.ClosePoll)
		r.Get("/api/chat/messages/{id}/poll", h.GetPoll)
		r.Get("/api/chat/conversations/{id}/messages", h.ListMessages)
		r.Post("/api/chat/conversations/{id}/messages", h.SendMessage)
		r.Put("/api/chat/messages/{id}", h.EditMessage)
		r.Get("/api/chat/conversations", h.ListConversations)
	})
	return h, srv
}

func createPoll(t *testing.T, srv *httptest.Server, token string, convID int, question string, options []string, allowMultiple bool) (int, int) {
	t.Helper()
	res := testutil.Post(t, srv, fmt.Sprintf("/api/chat/conversations/%d/polls", convID), token,
		map[string]any{"question": question, "options": options, "allowMultiple": allowMultiple})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		return res.StatusCode, 0
	}
	var out createdMsgResp
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode create-poll response: %v", err)
	}
	return res.StatusCode, out.ID
}

func getPoll(t *testing.T, srv *httptest.Server, token string, msgID int) (int, *pollResp) {
	t.Helper()
	res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/messages/%d/poll", msgID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return res.StatusCode, nil
	}
	var pv pollResp
	if err := json.NewDecoder(res.Body).Decode(&pv); err != nil {
		t.Fatalf("decode poll: %v", err)
	}
	return res.StatusCode, &pv
}

func votePoll(t *testing.T, srv *httptest.Server, token string, msgID int, optionIDs []int) int {
	t.Helper()
	if optionIDs == nil {
		optionIDs = []int{}
	}
	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/chat/messages/%d/poll/vote", msgID), token,
		map[string]any{"optionIds": optionIDs})
	defer res.Body.Close()
	return res.StatusCode
}

func closePoll(t *testing.T, srv *httptest.Server, token string, msgID int) int {
	t.Helper()
	res := testutil.Post(t, srv, fmt.Sprintf("/api/chat/messages/%d/poll/close", msgID), token, nil)
	defer res.Body.Close()
	return res.StatusCode
}

// collectPollPushes wartet, bis want Push-Calls eingegangen sind (oder
// Timeout), und liefert sie vollständig (Titel/Body/Badge), anders als
// collectPushes (push_fanout_test.go), das nur die Empfänger-IDs meldet.
func collectPollPushes(t *testing.T, calls <-chan pushCall, want int) []pushCall {
	t.Helper()
	var got []pushCall
	deadline := time.After(2 * time.Second)
	for len(got) < want {
		select {
		case c := <-calls:
			got = append(got, c)
		case <-deadline:
			t.Fatalf("nur %d/%d Push-Calls empfangen", len(got), want)
		}
	}
	return got
}

// --- CreatePoll -------------------------------------------------------------

func TestCreatePoll_Group_OK(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, member)

	_, srv := newPollServer(t, db, hub.NewHub())
	token := testutil.Token(t, owner, "standard", nil)

	status, msgID := createPoll(t, srv, token, conv, "Wer fährt am Samstag?",
		[]string{"Ich", "Ich nicht", "Nur Hinfahrt"}, false)
	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d", status)
	}
	if msgID == 0 {
		t.Fatal("expected message id in response")
	}

	var body string
	if err := db.QueryRow(`SELECT body FROM messages WHERE id = ?`, msgID).Scan(&body); err != nil {
		t.Fatalf("query message: %v", err)
	}
	if body != "Wer fährt am Samstag?" {
		t.Errorf("message body = %q, want the question", body)
	}

	var allowMultiple int
	if err := db.QueryRow(`SELECT allow_multiple FROM chat_polls WHERE message_id = ?`, msgID).Scan(&allowMultiple); err != nil {
		t.Fatalf("expected chat_polls row: %v", err)
	}
	if allowMultiple != 0 {
		t.Errorf("allow_multiple = %d, want 0", allowMultiple)
	}

	rows, err := db.Query(`SELECT label FROM chat_poll_options WHERE message_id = ? ORDER BY position`, msgID)
	if err != nil {
		t.Fatalf("query options: %v", err)
	}
	defer rows.Close()
	var labels []string
	for rows.Next() {
		var l string
		rows.Scan(&l)
		labels = append(labels, l)
	}
	want := []string{"Ich", "Ich nicht", "Nur Hinfahrt"}
	if len(labels) != len(want) {
		t.Fatalf("options = %v, want %v", labels, want)
	}
	for i := range want {
		if labels[i] != want[i] {
			t.Errorf("option[%d] = %q, want %q (Reihenfolge muss dem Request entsprechen)", i, labels[i], want[i])
		}
	}
}

func TestCreatePoll_DirectConversation_400(t *testing.T) {
	db := testutil.NewDB(t)
	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")

	res, err := db.Exec(`INSERT INTO conversations (type, created_by) VALUES ('direct', ?)`, userA)
	if err != nil {
		t.Fatalf("insert direct conv: %v", err)
	}
	convID64, _ := res.LastInsertId()
	convID := int(convID64)
	db.Exec(`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, userA)
	db.Exec(`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, userB)

	_, srv := newPollServer(t, db, hub.NewHub())
	token := testutil.Token(t, userA, "standard", nil)

	status, _ := createPoll(t, srv, token, convID, "Frage?", []string{"A", "B"}, false)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conversation_id = ?`, convID).Scan(&count)
	if count != 0 {
		t.Errorf("expected no message to be created, got %d", count)
	}
}

func TestCreatePoll_OptionCount_400(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner)

	_, srv := newPollServer(t, db, hub.NewHub())
	token := testutil.Token(t, owner, "standard", nil)

	for _, tc := range []struct {
		name    string
		options []string
	}{
		{"eine Option", []string{"Nur eine"}},
		{"elf Optionen", []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}},
	} {
		status, _ := createPoll(t, srv, token, conv, "Frage?", tc.options, false)
		if status != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", tc.name, status)
		}
	}
}

func TestCreatePoll_DuplicateOptions_400(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner)

	_, srv := newPollServer(t, db, hub.NewHub())
	token := testutil.Token(t, owner, "standard", nil)

	status, _ := createPoll(t, srv, token, conv, "Pizza?", []string{"Pizza", " pizza "}, false)
	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestCreatePoll_LeftMember_403(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, memberA)
	if _, err := db.Exec(
		`UPDATE conversation_members SET left_at = CURRENT_TIMESTAMP WHERE conversation_id = ? AND user_id = ?`,
		conv, memberA); err != nil {
		t.Fatalf("mark left: %v", err)
	}

	_, srv := newPollServer(t, db, hub.NewHub())
	token := testutil.Token(t, memberA, "standard", nil)

	status, _ := createPoll(t, srv, token, conv, "Frage?", []string{"A", "B"}, false)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", status)
	}
}

func TestCreatePoll_PushAndEvent(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	b := testutil.CreateUser(t, db, "standard")
	c := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, b, c)

	sharedHub := hub.NewHub()
	h, srv := newPollServer(t, db, sharedHub)
	calls := make(chan pushCall, 8)
	h.SetPushFn(func(_ *sql.DB, _ *appconfig.Config, userID int, title, body, url string, badge int) {
		calls <- pushCall{userID, title, body, url, badge}
	})

	chB := sharedHub.SubscribeUser(b)
	chC := sharedHub.SubscribeUser(c)
	defer sharedHub.UnsubscribeUser(b, chB)
	defer sharedHub.UnsubscribeUser(c, chC)

	token := testutil.Token(t, owner, "standard", nil)
	status, _ := createPoll(t, srv, token, conv, "Pizza oder Nudeln?", []string{"Pizza", "Nudeln"}, false)
	if status != http.StatusCreated {
		t.Fatalf("expected 201, got %d", status)
	}

	pushes := collectPollPushes(t, calls, 2)
	wantBody := "Umfrage: Pizza oder Nudeln?"
	recipients := map[int]bool{}
	for _, p := range pushes {
		recipients[p.userID] = true
		if p.body != wantBody {
			t.Errorf("push body = %q, want %q", p.body, wantBody)
		}
	}
	if !recipients[b] || !recipients[c] {
		t.Fatalf("push recipients = %v, want {%d, %d}", recipients, b, c)
	}
	if recipients[owner] {
		t.Error("Ersteller darf keinen Push für die eigene Umfrage bekommen")
	}

	wantEvent := fmt.Sprintf("chat:new-message:%d", conv)
	if ev, ok := recvWithinRR(chB, time.Second); !ok || ev != wantEvent {
		t.Errorf("chB event = %q ok=%v, want %q", ev, ok, wantEvent)
	}
	if ev, ok := recvWithinRR(chC, time.Second); !ok || ev != wantEvent {
		t.Errorf("chC event = %q ok=%v, want %q", ev, ok, wantEvent)
	}
}

// --- VotePoll -----------------------------------------------------------

func TestVotePoll_SingleChoice_ReplacesVote(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Wer fährt?", []string{"Ich", "Nur Hinfahrt"}, false)
	_, pv := getPoll(t, srv, ownerToken, msgID)
	ich, hinfahrt := pv.Options[0].ID, pv.Options[1].ID

	if status := votePoll(t, srv, bobToken, msgID, []int{ich}); status != http.StatusNoContent {
		t.Fatalf("first vote: expected 204, got %d", status)
	}
	if status := votePoll(t, srv, bobToken, msgID, []int{hinfahrt}); status != http.StatusNoContent {
		t.Fatalf("second vote: expected 204, got %d", status)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM chat_poll_votes WHERE user_id = ?`, bob).Scan(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 vote for bob, got %d", count)
	}
	var optionID int
	db.QueryRow(`SELECT option_id FROM chat_poll_votes WHERE user_id = ?`, bob).Scan(&optionID)
	if optionID != hinfahrt {
		t.Errorf("expected vote on 'Nur Hinfahrt', got option %d", optionID)
	}
}

func TestVotePoll_MultiChoice_OK(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Wann?", []string{"Samstag", "Sonntag"}, true)
	_, pv := getPoll(t, srv, ownerToken, msgID)
	sat, sun := pv.Options[0].ID, pv.Options[1].ID

	if status := votePoll(t, srv, bobToken, msgID, []int{sat, sun}); status != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", status)
	}

	_, pv2 := getPoll(t, srv, bobToken, msgID)
	if pv2.VoterCount != 1 {
		t.Errorf("voterCount = %d, want 1 (Bob zählt einmal, nicht zweimal)", pv2.VoterCount)
	}
	for _, opt := range pv2.Options {
		if opt.Count != 1 || !opt.Voted {
			t.Errorf("option %q: count=%d voted=%v, want count=1 voted=true", opt.Label, opt.Count, opt.Voted)
		}
	}
}

func TestVotePoll_EmptyWithdraws(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)
	_, pv := getPoll(t, srv, ownerToken, msgID)
	optA := pv.Options[0].ID

	if status := votePoll(t, srv, bobToken, msgID, []int{optA}); status != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", status)
	}
	if status := votePoll(t, srv, bobToken, msgID, []int{}); status != http.StatusNoContent {
		t.Fatalf("expected 204 on withdraw, got %d", status)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM chat_poll_votes WHERE user_id = ?`, bob).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 votes after withdraw, got %d", count)
	}
}

func TestVotePoll_TwoOptionsOnSingleChoice_400(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)
	_, pv := getPoll(t, srv, ownerToken, msgID)
	optA, optB := pv.Options[0].ID, pv.Options[1].ID

	if status := votePoll(t, srv, bobToken, msgID, []int{optA, optB}); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM chat_poll_votes WHERE user_id = ?`, bob).Scan(&count)
	if count != 0 {
		t.Errorf("expected no votes recorded after rejected request, got %d", count)
	}
}

func TestVotePoll_ForeignOption_400(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID1 := createPoll(t, srv, ownerToken, conv, "Frage 1?", []string{"A", "B"}, false)
	_, msgID2 := createPoll(t, srv, ownerToken, conv, "Frage 2?", []string{"C", "D"}, false)
	_, pv2 := getPoll(t, srv, ownerToken, msgID2)
	foreignOption := pv2.Options[0].ID

	if status := votePoll(t, srv, bobToken, msgID1, []int{foreignOption}); status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", status)
	}
}

func TestVotePoll_Closed_409(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)
	_, pv := getPoll(t, srv, ownerToken, msgID)
	optA := pv.Options[0].ID

	if status := closePoll(t, srv, ownerToken, msgID); status != http.StatusNoContent {
		t.Fatalf("close: expected 204, got %d", status)
	}

	if status := votePoll(t, srv, bobToken, msgID, []int{optA}); status != http.StatusConflict {
		t.Fatalf("expected 409, got %d", status)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM chat_poll_votes WHERE user_id = ?`, bob).Scan(&count)
	if count != 0 {
		t.Errorf("expected no votes after 409, got %d", count)
	}
}

func TestVotePoll_DeletedMessage_404(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)
	_, pv := getPoll(t, srv, ownerToken, msgID)
	optA := pv.Options[0].ID

	if _, err := db.Exec(`UPDATE messages SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`, msgID); err != nil {
		t.Fatalf("mark deleted: %v", err)
	}

	if status := votePoll(t, srv, bobToken, msgID, []int{optA}); status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

func TestVotePoll_NonMember_403(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	outsider := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	outsiderToken := testutil.Token(t, outsider, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)
	_, pv := getPoll(t, srv, ownerToken, msgID)
	optA := pv.Options[0].ID

	if status := votePoll(t, srv, outsiderToken, msgID, []int{optA}); status != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", status)
	}
}

func TestVotePoll_NoPushEvent(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	sharedHub := hub.NewHub()
	h, srv := newPollServer(t, db, sharedHub)
	calls := make(chan pushCall, 8)
	h.SetPushFn(func(_ *sql.DB, _ *appconfig.Config, userID int, title, body, url string, badge int) {
		calls <- pushCall{userID, title, body, url, badge}
	})

	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)
	_, pv := getPoll(t, srv, ownerToken, msgID)
	optA := pv.Options[0].ID

	// Den Push der Umfrage-Erstellung selbst abfangen, um den Vote-Schritt zu isolieren.
	collectPollPushes(t, calls, 1)

	chOwner := sharedHub.SubscribeUser(owner)
	defer sharedHub.UnsubscribeUser(owner, chOwner)

	if status := votePoll(t, srv, bobToken, msgID, []int{optA}); status != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", status)
	}

	wantEvent := fmt.Sprintf("chat:poll-updated:%d:%d", conv, msgID)
	if ev, ok := recvWithinRR(chOwner, time.Second); !ok || ev != wantEvent {
		t.Errorf("owner event = %q ok=%v, want %q", ev, ok, wantEvent)
	}

	select {
	case c := <-calls:
		t.Fatalf("unerwarteter Push beim Abstimmen: %+v", c)
	case <-time.After(200 * time.Millisecond):
	}
}

// --- ClosePoll ------------------------------------------------------------

func TestClosePoll_Creator_OK(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	sharedHub := hub.NewHub()
	_, srv := newPollServer(t, db, sharedHub)
	ownerToken := testutil.Token(t, owner, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)

	chBob := sharedHub.SubscribeUser(bob)
	defer sharedHub.UnsubscribeUser(bob, chBob)

	if status := closePoll(t, srv, ownerToken, msgID); status != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", status)
	}

	var closedAt sql.NullString
	if err := db.QueryRow(`SELECT closed_at FROM chat_polls WHERE message_id = ?`, msgID).Scan(&closedAt); err != nil {
		t.Fatalf("query closed_at: %v", err)
	}
	if !closedAt.Valid {
		t.Error("expected closed_at to be set")
	}

	wantEvent := fmt.Sprintf("chat:poll-updated:%d:%d", conv, msgID)
	if ev, ok := recvWithinRR(chBob, time.Second); !ok || ev != wantEvent {
		t.Errorf("bob event = %q ok=%v, want %q", ev, ok, wantEvent)
	}
}

func TestClosePoll_OtherUserAndAdmin_403(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	admin := testutil.CreateUser(t, db, "admin")
	conv := createGroupConv(t, db, "G", owner, bob, admin)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)

	for _, tc := range []struct {
		name  string
		token string
	}{
		{"anderes Mitglied", testutil.Token(t, bob, "standard", nil)},
		{"admin (kein Bypass)", testutil.Token(t, admin, "admin", nil)},
	} {
		if status := closePoll(t, srv, tc.token, msgID); status != http.StatusForbidden {
			t.Errorf("%s: expected 403, got %d", tc.name, status)
		}
	}

	var closedAt sql.NullString
	db.QueryRow(`SELECT closed_at FROM chat_polls WHERE message_id = ?`, msgID).Scan(&closedAt)
	if closedAt.Valid {
		t.Error("expected poll to remain open")
	}
}

func TestClosePoll_Idempotent(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner)

	sharedHub := hub.NewHub()
	_, srv := newPollServer(t, db, sharedHub)
	ownerToken := testutil.Token(t, owner, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)

	if status := closePoll(t, srv, ownerToken, msgID); status != http.StatusNoContent {
		t.Fatalf("first close: expected 204, got %d", status)
	}
	var closedAt string
	db.QueryRow(`SELECT closed_at FROM chat_polls WHERE message_id = ?`, msgID).Scan(&closedAt)

	chOwner := sharedHub.SubscribeUser(owner)
	defer sharedHub.UnsubscribeUser(owner, chOwner)

	if status := closePoll(t, srv, ownerToken, msgID); status != http.StatusNoContent {
		t.Fatalf("second close: expected 204, got %d", status)
	}
	var closedAtAfter string
	db.QueryRow(`SELECT closed_at FROM chat_polls WHERE message_id = ?`, msgID).Scan(&closedAtAfter)
	if closedAt != closedAtAfter {
		t.Errorf("closed_at changed on repeat close: %q -> %q", closedAt, closedAtAfter)
	}

	// Kein Event beim zweiten (wirkungslosen) Beenden.
	if ev, ok := recvWithinRR(chOwner, 200*time.Millisecond); ok {
		t.Errorf("unexpected event on idempotent close: %q", ev)
	}
}

// --- GetPoll ----------------------------------------------------------------

func TestGetPoll_Member_OK(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	setUserName(t, db, bob, "Bob", "Beispiel")
	conv := createGroupConv(t, db, "G", owner, bob)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Pizza oder Nudeln?", []string{"Pizza", "Nudeln"}, false)
	_, pv0 := getPoll(t, srv, ownerToken, msgID)
	pizza := pv0.Options[0].ID

	if status := votePoll(t, srv, bobToken, msgID, []int{pizza}); status != http.StatusNoContent {
		t.Fatalf("vote: expected 204, got %d", status)
	}

	status, pv := getPoll(t, srv, ownerToken, msgID)
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if pv.VoterCount != 1 {
		t.Errorf("voterCount = %d, want 1", pv.VoterCount)
	}
	var found bool
	for _, opt := range pv.Options {
		if opt.Label == "Pizza" {
			found = true
			if opt.Count != 1 {
				t.Errorf("pizza count = %d, want 1", opt.Count)
			}
			if len(opt.Voters) != 1 || opt.Voters[0].Name != "Bob Beispiel" {
				t.Errorf("pizza voters = %+v, want [Bob Beispiel]", opt.Voters)
			}
			if opt.Voted {
				t.Error("owner did not vote — voted must reflect the caller, not any voter")
			}
		}
	}
	if !found {
		t.Fatal("expected 'Pizza' option in response")
	}

	// Aus Bobs Sicht ist seine eigene Stimme voted=true.
	_, pvBob := getPoll(t, srv, bobToken, msgID)
	for _, opt := range pvBob.Options {
		if opt.Label == "Pizza" && !opt.Voted {
			t.Error("bob's own vote should be voted=true from his perspective")
		}
	}
}

func TestGetPoll_NonMember_403(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	outsider := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	outsiderToken := testutil.Token(t, outsider, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)

	status, _ := getPoll(t, srv, outsiderToken, msgID)
	if status != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", status)
	}
}

func TestGetPoll_NotAPoll_404(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner)
	msgID := insertMessage(t, db, conv, owner, "keine Umfrage")

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)

	status, _ := getPoll(t, srv, ownerToken, msgID)
	if status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", status)
	}
}

// --- EditMessage (chat-message-edit, modifiziert) ---------------------------

func TestEditMessage_Poll_409(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)

	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/chat/messages/%d", msgID), ownerToken,
		map[string]string{"body": "Geänderte Frage?"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", res.StatusCode)
	}

	var body string
	db.QueryRow(`SELECT body FROM messages WHERE id = ?`, msgID).Scan(&body)
	if body != "Frage?" {
		t.Errorf("message body changed to %q, want unchanged", body)
	}
}

// --- ListMessages / ListConversations ---------------------------------------

func TestListMessages_IncludesPoll(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, member)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/conversations/%d/messages", conv), ownerToken)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var msgs []struct {
		ID   int       `json:"id"`
		Poll *pollResp `json:"poll"`
	}
	if err := json.NewDecoder(res.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var found bool
	for _, m := range msgs {
		if m.ID == msgID {
			found = true
			if m.Poll == nil {
				t.Fatal("expected poll to be set")
			}
			if len(m.Poll.Options) != 2 {
				t.Errorf("expected 2 options, got %d", len(m.Poll.Options))
			}
		}
	}
	if !found {
		t.Fatal("poll message not found in list")
	}
}

func TestListMessages_DeletedPollHasNoPoll(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)
	if _, err := db.Exec(`UPDATE messages SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?`, msgID); err != nil {
		t.Fatalf("mark deleted: %v", err)
	}

	res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/conversations/%d/messages", conv), ownerToken)
	defer res.Body.Close()
	var msgs []struct {
		ID   int       `json:"id"`
		Poll *pollResp `json:"poll"`
	}
	if err := json.NewDecoder(res.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, m := range msgs {
		if m.ID == msgID && m.Poll != nil {
			t.Fatal("expected poll to be nil for a deleted message")
		}
	}
}

func TestListConversations_LastMessageIsPoll(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, member)
	priorID := insertMessage(t, db, conv, owner, "vorher")
	// sent_at hat Sekundengranularität (CURRENT_TIMESTAMP) — explizit in die
	// Vergangenheit setzen, damit die Umfrage unabhängig vom Test-Timing als
	// zeitlich später gilt (kein Gleichstand-Flake).
	if _, err := db.Exec(`UPDATE messages SET sent_at = '2020-01-01 00:00:00' WHERE id = ?`, priorID); err != nil {
		t.Fatalf("pin prior message timestamp: %v", err)
	}

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)

	if status, _ := createPoll(t, srv, ownerToken, conv, "Pizza oder Nudeln?", []string{"Pizza", "Nudeln"}, false); status != http.StatusCreated {
		t.Fatalf("createPoll: expected 201, got %d", status)
	}

	res := testutil.Get(t, srv, "/api/chat/conversations", ownerToken)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var convs []struct {
		ID          int `json:"id"`
		LastMessage *struct {
			Body   string `json:"body"`
			IsPoll bool   `json:"isPoll"`
		} `json:"lastMessage"`
	}
	if err := json.NewDecoder(res.Body).Decode(&convs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	var found bool
	for _, c := range convs {
		if c.ID == conv {
			found = true
			if c.LastMessage == nil || !c.LastMessage.IsPoll {
				t.Errorf("expected lastMessage.isPoll=true, got %+v", c.LastMessage)
			} else if c.LastMessage.Body != "Pizza oder Nudeln?" {
				t.Errorf("lastMessage.body = %q, want the question", c.LastMessage.Body)
			}
		}
	}
	if !found {
		t.Fatal("conversation not found in list")
	}
}

// --- Invariante 4.3: parallele VotePoll-Requests desselben Nutzers ----------

// TestVotePoll_ConcurrentSameUserSingleChoice_ExactlyOneVote — mehrere
// gleichzeitige Stimmabgaben desselben Nutzers auf dieselbe Option einer
// Einfachauswahl-Umfrage dürfen nie mehr als eine Stimme hinterlassen. Jeder
// VotePoll-Aufruf löscht die Stimmen des Nutzers und fügt sie innerhalb
// derselben Transaktion neu ein (design.md §2); SQLite serialisiert
// nebenläufige Schreib-Transaktionen, sodass das Ergebnis unabhängig von der
// Ausführungsreihenfolge immer genau eine Zeile ist.
func TestVotePoll_ConcurrentSameUserSingleChoice_ExactlyOneVote(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	bob := testutil.CreateUser(t, db, "standard")
	conv := createGroupConv(t, db, "G", owner, bob)

	_, srv := newPollServer(t, db, hub.NewHub())
	ownerToken := testutil.Token(t, owner, "standard", nil)
	bobToken := testutil.Token(t, bob, "standard", nil)

	_, msgID := createPoll(t, srv, ownerToken, conv, "Frage?", []string{"A", "B"}, false)
	_, pv := getPoll(t, srv, ownerToken, msgID)
	optA := pv.Options[0].ID

	const n = 10
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			votePoll(t, srv, bobToken, msgID, []int{optA})
		}()
	}
	wg.Wait()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM chat_poll_votes WHERE user_id = ?`, bob).Scan(&count); err != nil {
		t.Fatalf("count votes: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 vote after %d concurrent requests, got %d", n, count)
	}
}
