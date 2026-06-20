package chat_test

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func createConv(t *testing.T, db *sql.DB, creator int, members ...int) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO conversations (type, name, created_by) VALUES ('group', ?, ?)`, "test", creator)
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

func insertMessage(t *testing.T, db *sql.DB, convID, senderID int, body string) int {
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

func markRead(t *testing.T, db *sql.DB, msgID, userID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT OR IGNORE INTO message_reads (message_id, user_id) VALUES (?, ?)`,
		msgID, userID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
}

func insertBroadcast(t *testing.T, db *sql.DB, senderID int, body string, recipients []int) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO broadcasts (sender_id, target_type, body) VALUES (?, 'all', ?)`,
		senderID, body)
	if err != nil {
		t.Fatalf("insert broadcast: %v", err)
	}
	bid, _ := res.LastInsertId()
	for _, uid := range recipients {
		var readAt any
		if uid == senderID {
			readAt = "now"
			if _, err := db.Exec(
				`INSERT OR IGNORE INTO broadcast_reads (broadcast_id, user_id, read_at) VALUES (?, ?, CURRENT_TIMESTAMP)`,
				bid, uid); err != nil {
				t.Fatalf("broadcast_reads sender: %v", err)
			}
			_ = readAt
			continue
		}
		if _, err := db.Exec(
			`INSERT OR IGNORE INTO broadcast_reads (broadcast_id, user_id) VALUES (?, ?)`,
			bid, uid); err != nil {
			t.Fatalf("broadcast_reads recipient: %v", err)
		}
	}
	return int(bid)
}

func TestComputeUnreadForUser_ZeroWhenNothing(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")

	n, err := chat.ComputeUnreadForUser(db, uid)
	if err != nil {
		t.Fatalf("ComputeUnreadForUser: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestComputeUnreadForUser_ConversationsAndBroadcasts(t *testing.T) {
	db := testutil.NewDB(t)
	me := testutil.CreateUser(t, db, "standard")
	other := testutil.CreateUser(t, db, "standard")

	convA := createConv(t, db, other, me)
	insertMessage(t, db, convA, other, "hi 1")
	insertMessage(t, db, convA, other, "hi 2")

	convB := createConv(t, db, other, me)
	insertMessage(t, db, convB, other, "hello")

	insertBroadcast(t, db, other, "b1", []int{me, other})
	insertBroadcast(t, db, other, "b2", []int{me, other})

	insertBroadcast(t, db, me, "self", []int{me, other})

	got, err := chat.ComputeUnreadForUser(db, me)
	if err != nil {
		t.Fatalf("ComputeUnreadForUser: %v", err)
	}
	if got != 5 {
		t.Fatalf("expected 5 (3 messages + 2 broadcasts, own broadcast excluded), got %d", got)
	}
}

func TestComputeUnreadForUser_ConversationFullyRead(t *testing.T) {
	db := testutil.NewDB(t)
	me := testutil.CreateUser(t, db, "standard")
	other := testutil.CreateUser(t, db, "standard")

	conv := createConv(t, db, other, me)
	m1 := insertMessage(t, db, conv, other, "one")
	m2 := insertMessage(t, db, conv, other, "two")
	markRead(t, db, m1, me)
	markRead(t, db, m2, me)

	got, err := chat.ComputeUnreadForUser(db, me)
	if err != nil {
		t.Fatalf("ComputeUnreadForUser: %v", err)
	}
	if got != 0 {
		t.Fatalf("expected 0 after marking all read, got %d", got)
	}
}

func TestComputeUnreadForUser_OwnBroadcastNotCounted(t *testing.T) {
	db := testutil.NewDB(t)
	me := testutil.CreateUser(t, db, "standard")
	other := testutil.CreateUser(t, db, "standard")

	conv := createConv(t, db, other, me)
	insertMessage(t, db, conv, other, "ping")

	insertBroadcast(t, db, me, "from me", []int{me, other})

	got, err := chat.ComputeUnreadForUser(db, me)
	if err != nil {
		t.Fatalf("ComputeUnreadForUser: %v", err)
	}
	if got != 1 {
		t.Fatalf("expected 1 (only the conv message), got %d", got)
	}
}
