package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/db"
	_ "modernc.org/sqlite"
)

func TestNormalizeLoginName(t *testing.T) {
	cases := []struct {
		first, last, want string
	}{
		{"Lena", "Schmidt", "Lena.Schmidt"},
		{"Lena", "Müller", "Lena.Mueller"},            // Umlaut transliteriert
		{"Anna Lena", "Schmidt", "Anna-Lena.Schmidt"}, // Doppelname → Bindestrich
		{"Jörg", "Weiß", "Joerg.Weiss"},               // ö und ß
		{"  Tim  ", " Bauer ", "Tim.Bauer"},           // Trimming
		{"Lea-Sophie", "Groß-Klein", "Lea-Sophie.Gross-Klein"},
		{"Tom!", "O'Brien", "Tom.OBrien"}, // Sonderzeichen entfernt
	}
	for _, c := range cases {
		if got := normalizeLoginName(c.first, c.last); got != c.want {
			t.Errorf("normalizeLoginName(%q,%q) = %q, want %q", c.first, c.last, got, c.want)
		}
	}
}

func TestNormalizeLoginNameEmpty(t *testing.T) {
	// Nur Sonderzeichen → unbrauchbar → leerer String (Handler liefert dann Fehler).
	if got := normalizeLoginName("???", "Schmidt"); got != "" {
		t.Errorf("expected empty for unusable first name, got %q", got)
	}
	if got := normalizeLoginName("Lena", "###"); got != "" {
		t.Errorf("expected empty for unusable last name, got %q", got)
	}
}

func TestGenerateUniqueLoginNameNoCollision(t *testing.T) {
	database := newLoginTestDB(t)
	got, err := generateUniqueLoginName(context.Background(), database, "Lena", "Schmidt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Lena.Schmidt" {
		t.Errorf("got %q, want Lena.Schmidt", got)
	}
}

func TestGenerateUniqueLoginNameCollisionSuffix(t *testing.T) {
	database := newLoginTestDB(t)
	insertLoginUser(t, database, "Lena.Schmidt", 1) // aktives Konto belegt den Namen
	got, err := generateUniqueLoginName(context.Background(), database, "Lena", "Schmidt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Lena.Schmidt2" {
		t.Errorf("got %q, want Lena.Schmidt2", got)
	}
}

func TestGenerateUniqueLoginNameCollisionWithInactiveAccount(t *testing.T) {
	database := newLoginTestDB(t)
	// Zwei noch inaktive (can_login=0) Konten dürfen nicht denselben Namen ziehen.
	insertLoginUser(t, database, "Lena.Schmidt", 0)
	insertLoginUser(t, database, "Lena.Schmidt2", 0)
	got, err := generateUniqueLoginName(context.Background(), database, "Lena", "Schmidt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Lena.Schmidt3" {
		t.Errorf("got %q, want Lena.Schmidt3", got)
	}
}

func TestGenerateUniqueLoginNameCaseInsensitive(t *testing.T) {
	database := newLoginTestDB(t)
	insertLoginUser(t, database, "lena.schmidt", 1) // andere Schreibweise belegt
	got, err := generateUniqueLoginName(context.Background(), database, "Lena", "Schmidt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.EqualFold(got, "Lena.Schmidt2") {
		t.Errorf("got %q, want Lena.Schmidt2 (case-insensitive)", got)
	}
}

func TestGenerateUniqueLoginNameEmptyError(t *testing.T) {
	database := newLoginTestDB(t)
	if _, err := generateUniqueLoginName(context.Background(), database, "???", "###"); err == nil {
		t.Error("expected error for unusable name, got nil")
	}
}

var loginDBCounter atomic.Uint64

// newLoginTestDB opens a fresh in-memory SQLite database with all migrations
// applied. Lives in package auth (not testutil) to avoid an import cycle, since
// these tests exercise unexported helpers.
func newLoginTestDB(t *testing.T) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("loginnametestdb_%d", loginDBCounter.Add(1))
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys=on", name)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Migrate(database, db.MigrationsFS); err != nil {
		database.Close()
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// insertLoginUser legt ein Konto mit gesetztem login_name und can_login an.
func insertLoginUser(t *testing.T, database *sql.DB, loginName string, canLogin int) {
	t.Helper()
	if _, err := database.Exec(
		`INSERT INTO users (email, login_name, password, role, can_login) VALUES (NULL, ?, '', 'standard', ?)`,
		loginName, canLogin,
	); err != nil {
		t.Fatalf("insertLoginUser(%q): %v", loginName, err)
	}
}
