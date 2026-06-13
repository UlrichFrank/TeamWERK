package chat_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newChatServer(t *testing.T, db *sql.DB) (*chat.Handler, func(r chi.Router)) {
	t.Helper()
	h := chat.NewHandler(db, hub.NewHub(), testutil.TestConfig())
	return h, func(r chi.Router) {
		r.Get("/api/chat/conversations", h.ListConversations)
		r.Post("/api/chat/conversations", h.CreateConversation)
		r.Put("/api/chat/conversations/{id}", h.UpdateConversation)
		r.Delete("/api/chat/conversations/{id}", h.DeleteConversation)
		r.Delete("/api/chat/conversations/{id}/members/me", h.LeaveConversation)
		r.Delete("/api/chat/conversations/{id}/members/{uid}", h.RemoveMember)
		r.Delete("/api/chat/conversations/{id}/everyone", h.DeleteConversationForEveryone)
		r.Post("/api/chat/conversations/{id}/members", h.AddMember)
		r.Post("/api/chat/conversations/{id}/transfer-ownership", h.TransferOwnership)
	}
}

// createGroupConv creates a group conversation with the given creator and
// extra active member user IDs. Returns the conversation ID.
func createGroupConv(t *testing.T, db *sql.DB, name string, creator int, members ...int) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO conversations (type, name, created_by) VALUES ('group', ?, ?)`, name, creator)
	if err != nil {
		t.Fatalf("createGroupConv conv: %v", err)
	}
	convID, _ := res.LastInsertId()
	ids := append([]int{creator}, members...)
	for _, uid := range ids {
		if _, err := db.Exec(
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`,
			convID, uid); err != nil {
			t.Fatalf("createGroupConv member %d: %v", uid, err)
		}
	}
	return int(convID)
}

func memberLeftAt(t *testing.T, db *sql.DB, convID, userID int) sql.NullString {
	t.Helper()
	var leftAt sql.NullString
	if err := db.QueryRow(
		`SELECT left_at FROM conversation_members WHERE conversation_id = ? AND user_id = ?`,
		convID, userID).Scan(&leftAt); err != nil {
		t.Fatalf("memberLeftAt: %v", err)
	}
	return leftAt
}

func systemMessageExists(t *testing.T, db *sql.DB, convID int, senderID int, bodyLike string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE conversation_id = ? AND sender_id = ? AND is_system = 1 AND body LIKE ?`,
		convID, senderID, "%"+bodyLike+"%").Scan(&n); err != nil {
		t.Fatalf("systemMessageExists: %v", err)
	}
	return n > 0
}

func convName(t *testing.T, db *sql.DB, convID int) string {
	t.Helper()
	var name sql.NullString
	if err := db.QueryRow(`SELECT name FROM conversations WHERE id = ?`, convID).Scan(&name); err != nil {
		t.Fatalf("convName: %v", err)
	}
	return name.String
}

func convCreatedBy(t *testing.T, db *sql.DB, convID int) int {
	t.Helper()
	var createdBy int
	if err := db.QueryRow(`SELECT created_by FROM conversations WHERE id = ?`, convID).Scan(&createdBy); err != nil {
		t.Fatalf("convCreatedBy: %v", err)
	}
	return createdBy
}

func convExists(t *testing.T, db *sql.DB, convID int) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversations WHERE id = ?`, convID).Scan(&n); err != nil {
		t.Fatalf("convExists: %v", err)
	}
	return n > 0
}

func TestRemoveMember_CreatorRemovesMember(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	memberB := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, memberA, memberB)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/chat/conversations/"+itoa(convID)+"/members/"+itoa(memberB),
		testutil.Token(t, owner, "standard", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if !memberLeftAt(t, db, convID, memberB).Valid {
		t.Errorf("expected left_at set for removed member")
	}
	if !systemMessageExists(t, db, convID, memberB, "wurde entfernt") {
		t.Errorf("expected 'wurde entfernt' system message")
	}
}

func TestRemoveMember_NonCreatorForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	memberB := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, memberA, memberB)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/chat/conversations/"+itoa(convID)+"/members/"+itoa(memberB),
		testutil.Token(t, memberA, "standard", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestRemoveMember_SelfRejected(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, memberA)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/chat/conversations/"+itoa(convID)+"/members/"+itoa(owner),
		testutil.Token(t, owner, "standard", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestUpdateConversation_RenameSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "Alt", owner, memberA)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/chat/conversations/"+itoa(convID),
		testutil.Token(t, owner, "standard", nil),
		map[string]string{"name": "Taktik"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if convName(t, db, convID) != "Taktik" {
		t.Errorf("expected name 'Taktik', got %q", convName(t, db, convID))
	}
	if !systemMessageExists(t, db, convID, owner, "in 'Taktik' umbenannt") {
		t.Errorf("expected rename system message")
	}
}

func TestUpdateConversation_EmptyNameRejected(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "Alt", owner)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodPut,
		"/api/chat/conversations/"+itoa(convID),
		testutil.Token(t, owner, "standard", nil),
		map[string]string{"name": "   "})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	if convName(t, db, convID) != "Alt" {
		t.Errorf("expected unchanged name, got %q", convName(t, db, convID))
	}
}

func TestTransferOwnership_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberB := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, memberB)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodPost,
		"/api/chat/conversations/"+itoa(convID)+"/transfer-ownership",
		testutil.Token(t, owner, "standard", nil),
		map[string]int{"newOwnerId": memberB})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if convCreatedBy(t, db, convID) != memberB {
		t.Errorf("expected created_by=%d, got %d", memberB, convCreatedBy(t, db, convID))
	}
	if !systemMessageExists(t, db, convID, owner, "hat die Verwaltung an") {
		t.Errorf("expected transfer system message")
	}
	if memberLeftAt(t, db, convID, owner).Valid {
		t.Errorf("old owner should still be active member")
	}
}

func TestTransferOwnership_RecipientNotMember(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	outsider := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodPost,
		"/api/chat/conversations/"+itoa(convID)+"/transfer-ownership",
		testutil.Token(t, owner, "standard", nil),
		map[string]int{"newOwnerId": outsider})
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
}

func TestTransferOwnership_NonCreatorForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	memberB := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, memberA, memberB)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodPost,
		"/api/chat/conversations/"+itoa(convID)+"/transfer-ownership",
		testutil.Token(t, memberA, "standard", nil),
		map[string]int{"newOwnerId": memberB})
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func TestDeleteForEveryone_HardDelete(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, memberA)

	// Add some messages
	db.Exec(`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`, convID, owner, "Hallo")
	db.Exec(`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`, convID, memberA, "Hi")

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/chat/conversations/"+itoa(convID)+"/everyone",
		testutil.Token(t, owner, "standard", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if convExists(t, db, convID) {
		t.Errorf("expected conversation row to be gone")
	}
	var msgCount int
	db.QueryRow(`SELECT COUNT(*) FROM messages WHERE conversation_id = ?`, convID).Scan(&msgCount)
	if msgCount != 0 {
		t.Errorf("expected messages cascaded, got %d", msgCount)
	}
	var memCount int
	db.QueryRow(`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ?`, convID).Scan(&memCount)
	if memCount != 0 {
		t.Errorf("expected conversation_members cascaded, got %d", memCount)
	}
}

func TestDeleteForEveryone_NonCreatorForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner, memberA)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/chat/conversations/"+itoa(convID)+"/everyone",
		testutil.Token(t, memberA, "standard", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
	if !convExists(t, db, convID) {
		t.Errorf("conversation should still exist after forbidden delete")
	}
}

func TestAddMember_EmitsSystemMessage(t *testing.T) {
	db := testutil.NewDB(t)
	// canContactUser non-admin path requires shared accessible teams, so
	// we make the owner admin and rely on the shortcut for admins/vorstand.
	owner := testutil.CreateUser(t, db, "admin")
	newGuy := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "G", owner)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodPost,
		"/api/chat/conversations/"+itoa(convID)+"/members",
		testutil.Token(t, owner, "admin", nil),
		map[string]int{"userId": newGuy})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if !systemMessageExists(t, db, convID, newGuy, "wurde hinzugefügt") {
		t.Errorf("expected 'wurde hinzugefügt' system message for added user")
	}
}

func itoa(n int) string {
	// small allocation-free decimal int formatter
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// directConvExists checks whether a direct conversation exists between two users.
func directConvExists(t *testing.T, db *sql.DB, userA, userB int) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM conversations c
		JOIN conversation_members m1 ON m1.conversation_id = c.id AND m1.user_id = ?
		JOIN conversation_members m2 ON m2.conversation_id = c.id AND m2.user_id = ?
		WHERE c.type = 'direct'`, userA, userB).Scan(&n); err != nil {
		t.Fatalf("directConvExists: %v", err)
	}
	return n > 0
}

// TC-CH-EXT01: Ein Mitglied verlässt eine Gruppe → left_at gesetzt, System-Nachricht.
func TestLeave_MemberLeavesGroup(t *testing.T) {
	db := testutil.NewDB(t)
	owner := testutil.CreateUser(t, db, "standard")
	memberA := testutil.CreateUser(t, db, "standard")
	convID := createGroupConv(t, db, "Gruppe", owner, memberA)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	res := testutil.Do(t, srv, http.MethodDelete,
		"/api/chat/conversations/"+itoa(convID)+"/members/me",
		testutil.Token(t, memberA, "standard", nil), nil)
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if !memberLeftAt(t, db, convID, memberA).Valid {
		t.Error("expected left_at set for leaving member")
	}
	if !systemMessageExists(t, db, convID, memberA, "hat die Gruppe verlassen") {
		t.Error("expected 'hat die Gruppe verlassen' system message")
	}
}

// TC-CH-EXT02: Direkt-Conversation kann nicht verlassen werden → 400.
func TestLeave_DirectConversationRejected(t *testing.T) {
	db := testutil.NewDB(t)
	userA := testutil.CreateUser(t, db, "standard")
	userB := testutil.CreateUser(t, db, "standard")

	// Create a direct conversation manually.
	res, err := db.Exec(`INSERT INTO conversations (type, created_by) VALUES ('direct', ?)`, userA)
	if err != nil {
		t.Fatalf("insert direct conv: %v", err)
	}
	convID64, _ := res.LastInsertId()
	convID := int(convID64)
	db.Exec(`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, userA)
	db.Exec(`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, userB)

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)

	leaveRes := testutil.Do(t, srv, http.MethodDelete,
		"/api/chat/conversations/"+itoa(convID)+"/members/me",
		testutil.Token(t, userA, "standard", nil), nil)
	defer leaveRes.Body.Close()

	if leaveRes.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for leaving direct conversation, got %d", leaveRes.StatusCode)
	}
}

// TC-CH-EXT03: CreateDirect bei bestehender Conversation gibt die vorhandene zurück.
func TestCreateDirect_DuplicateReturnsExisting(t *testing.T) {
	db := testutil.NewDB(t)
	// admin bypasses canContactUser check
	userA := testutil.CreateUser(t, db, "admin")
	userB := testutil.CreateUser(t, db, "standard")

	_, routes := newChatServer(t, db)
	srv := testutil.NewServer(t, routes)
	token := testutil.Token(t, userA, "admin", nil)

	// First create.
	r1 := testutil.Post(t, srv, "/api/chat/conversations", token,
		map[string]any{"type": "direct", "userId": userB})
	r1.Body.Close()
	if r1.StatusCode != http.StatusCreated && r1.StatusCode != http.StatusOK {
		t.Fatalf("first create: expected 200/201, got %d", r1.StatusCode)
	}

	// Second create with same partner — must not duplicate.
	r2 := testutil.Post(t, srv, "/api/chat/conversations", token,
		map[string]any{"type": "direct", "userId": userB})
	r2.Body.Close()
	if r2.StatusCode != http.StatusCreated && r2.StatusCode != http.StatusOK {
		t.Fatalf("second create: expected 200/201, got %d", r2.StatusCode)
	}

	var count int
	db.QueryRow(`
		SELECT COUNT(*) FROM conversations c
		JOIN conversation_members m1 ON m1.conversation_id = c.id AND m1.user_id = ?
		JOIN conversation_members m2 ON m2.conversation_id = c.id AND m2.user_id = ?
		WHERE c.type = 'direct'`, userA, userB).Scan(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 direct conversation, got %d", count)
	}
}
