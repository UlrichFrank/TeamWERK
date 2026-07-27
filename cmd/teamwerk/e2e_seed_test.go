package main

import (
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/teamstuttgart/teamwerk/internal/db"
)

// seedTempDB legt eine frische, migrierte SQLite-DB in einem Temp-Verzeichnis an,
// führt seedE2E aus (mit MEDIA_DIR im Temp-Ordner, damit keine echten Dateien
// entstehen) und gibt die offene DB zurück.
func seedTempDB(t *testing.T) *sql.DB {
	t.Helper()
	t.Setenv("MEDIA_DIR", t.TempDir())
	database, err := db.Open(filepath.Join(t.TempDir(), "e2e.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	seedE2E(database)
	return database
}

func seededRowCounts(t *testing.T, database *sql.DB) map[string]int {
	t.Helper()
	tables := []string{
		"users", "conversations", "conversation_members", "messages",
		"message_reads", "media", "duty_slots", "members",
	}
	out := map[string]int{}
	for _, tb := range tables {
		var n int
		if err := database.QueryRow("SELECT COUNT(*) FROM " + tb).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", tb, err)
		}
		out[tb] = n
	}
	return out
}

// Zwei frische Läufe des Seeds ergeben denselben Datenbank-Zustand (deterministisch,
// Voraussetzung für reproduzierbare E2E-Läufe).
func TestE2ESeed_Idempotent(t *testing.T) {
	first := seededRowCounts(t, seedTempDB(t))
	second := seededRowCounts(t, seedTempDB(t))
	if !reflect.DeepEqual(first, second) {
		t.Errorf("seed is not deterministic:\n first=%v\nsecond=%v", first, second)
	}
	// Sanity: der Seed ist nicht leer (sonst wäre die Gleichheit trivial).
	if first["users"] < 4 || first["media"] < 4 || first["messages"] == 0 {
		t.Errorf("seed looks empty: %v", first)
	}
}

// Nach dem Seed verifiziert das Admin-Passwort gegen den gespeicherten bcrypt-Hash
// — der Login-Flow (POST /api/auth/login) kann die Seed-Credentials nutzen.
func TestE2ESeed_LoginWorks(t *testing.T) {
	database := seedTempDB(t)
	var hash, role string
	if err := database.QueryRow(
		`SELECT password, role FROM users WHERE email = ?`, "e2e@test.local").Scan(&hash, &role); err != nil {
		t.Fatalf("admin user not seeded: %v", err)
	}
	if role != "admin" {
		t.Errorf("seed admin role = %q, want admin", role)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("E2ETestPassword!")); err != nil {
		t.Errorf("seed admin password does not verify: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("falsch")); err == nil {
		t.Error("a wrong password must not verify against the seeded hash")
	}
}

// adminUnreadCount spiegelt die Backend-Unread-Logik (chat/handler.go ListConversations):
// Nachrichten in der Konversation, deren Sender NICHT der Admin ist und für die keine
// message_reads-Zeile des Admins existiert.
func adminUnreadCount(t *testing.T, database *sql.DB, convID, adminID int) int {
	t.Helper()
	var n int
	err := database.QueryRow(`
		SELECT COUNT(*) FROM messages m
		WHERE m.conversation_id = ?
		  AND m.sender_id != ?
		  AND NOT EXISTS (
		    SELECT 1 FROM message_reads mr WHERE mr.message_id = m.id AND mr.user_id = ?
		  )`, convID, adminID, adminID).Scan(&n)
	if err != nil {
		t.Fatalf("adminUnreadCount: %v", err)
	}
	return n
}

func convIDByName(t *testing.T, database *sql.DB, name string) int {
	t.Helper()
	var id int
	if err := database.QueryRow(
		`SELECT id FROM conversations WHERE name = ?`, name).Scan(&id); err != nil {
		t.Fatalf("conversation %q not seeded: %v", name, err)
	}
	return id
}

func msgCount(t *testing.T, database *sql.DB, convID int) int {
	t.Helper()
	var n int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM messages WHERE conversation_id = ?`, convID).Scan(&n); err != nil {
		t.Fatalf("msgCount: %v", err)
	}
	return n
}

// adminID ermittelt die User-ID des Seed-Admins (e2e@test.local).
func adminID(t *testing.T, database *sql.DB) int {
	t.Helper()
	var id int
	if err := database.QueryRow(
		`SELECT id FROM users WHERE email = ?`, "e2e@test.local").Scan(&id); err != nil {
		t.Fatalf("admin not seeded: %v", err)
	}
	return id
}

// Die drei langen, bildlastigen Threads existieren mit den erwarteten Volumina und
// den exakten Admin-Unread-Zuständen (0 / 40 / 180) — die Kern-Regressionsfälle für
// Scroll-ans-Ende, Divider-auf-Seite und Chip (erste Ungelesene älter als die Seite).
func TestE2ESeed_LongThreads(t *testing.T) {
	database := seedTempDB(t)
	admin := adminID(t, database)

	cases := []struct {
		name       string
		wantMsgs   int
		wantUnread int
	}{
		{"E2E Chat lang gelesen", 150, 0},
		{"E2E Chat lang unread", 150, 40},
		{"E2E Chat viele ungelesen", 250, 180},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			convID := convIDByName(t, database, tc.name)
			if got := msgCount(t, database, convID); got != tc.wantMsgs {
				t.Errorf("message count = %d, want %d", got, tc.wantMsgs)
			}
			if got := adminUnreadCount(t, database, convID, admin); got != tc.wantUnread {
				t.Errorf("admin unread = %d, want %d", got, tc.wantUnread)
			}
		})
	}
}

// Die langen Threads enthalten Bilder sowohl mit als auch ohne Server-Dimensionen
// (width/height NULL) — der NULL-Dims-Pfad exerziert den AuthImage-Fallback im Frontend.
func TestE2ESeed_LongThreads_MixedImageDims(t *testing.T) {
	database := seedTempDB(t)

	var withDims, noDims int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM media WHERE width IS NOT NULL AND height IS NOT NULL`).Scan(&withDims); err != nil {
		t.Fatalf("count with-dims media: %v", err)
	}
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM media WHERE width IS NULL OR height IS NULL`).Scan(&noDims); err != nil {
		t.Fatalf("count no-dims media: %v", err)
	}
	if withDims == 0 {
		t.Error("expected at least one media row WITH dimensions")
	}
	if noDims == 0 {
		t.Error("expected at least one media row WITHOUT dimensions (AuthImage fallback path)")
	}
}
