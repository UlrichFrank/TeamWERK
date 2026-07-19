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
