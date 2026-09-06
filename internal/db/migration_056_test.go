package db_test

import (
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TC: Migration 056 nimmt der System-Rolle 'presseteam' die Grundlage. Der
// kritische Teil ist nicht der CHECK, sondern die Reihenfolge: würden die
// Bestände nicht VOR dem Tabellen-Rebuild umgeschrieben, scheiterte der
// INSERT SELECT am neuen Constraint — und die Migration ließe eine halb
// aufgebaute Datenbank zurück. testutil.NewDB migriert eine leere DB und
// fängt das nicht.
func TestMigration056_PresseteamWirdStandard(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(55); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 55: %v", err)
	}

	if _, err := sqlDB.Exec(`INSERT INTO users (id, email, login_name, first_name, last_name, can_login, role)
		VALUES (1, 'presse@b', 'presse', 'P', 'T', 1, 'presseteam'),
		       (2, 'admin@b', 'admin', 'A', 'D', 1, 'admin'),
		       (3, 'std@b', 'std', 'S', 'T', 1, 'standard')`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO invitation_tokens (id, email, role, token, expires_at)
		VALUES (1, 'neu@b', 'presseteam', 'tok-1', '2030-01-01')`); err != nil {
		t.Fatalf("seed invitation_token: %v", err)
	}

	if err := m.Migrate(56); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 56: %v", err)
	}

	// Die Presseteam-Zeile ist standard; die übrigen Rollen bleiben, wie sie waren.
	want := map[int]string{1: "standard", 2: "admin", 3: "standard"}
	for id, expected := range want {
		var role string
		if err := sqlDB.QueryRow(`SELECT role FROM users WHERE id = ?`, id).Scan(&role); err != nil {
			t.Fatalf("read role of user %d: %v", id, err)
		}
		if role != expected {
			t.Errorf("user %d: role = %q, want %q", id, role, expected)
		}
	}
	var tokenRole string
	if err := sqlDB.QueryRow(`SELECT role FROM invitation_tokens WHERE id = 1`).Scan(&tokenRole); err != nil {
		t.Fatalf("read invitation_token role: %v", err)
	}
	if tokenRole != "standard" {
		t.Errorf("invitation_tokens.role = %q, want standard", tokenRole)
	}

	// Der Wert ist danach kein zulässiger Zustand mehr — in beiden Tabellen.
	_, err := sqlDB.Exec(`INSERT INTO users (id, email, login_name, first_name, last_name, can_login, role)
		VALUES (4, 'x@b', 'x', 'X', 'Y', 1, 'presseteam')`)
	if err == nil || !strings.Contains(err.Error(), "CHECK") {
		t.Errorf("users: INSERT mit role='presseteam' muss am CHECK scheitern, err = %v", err)
	}
	_, err = sqlDB.Exec(`INSERT INTO invitation_tokens (id, email, role, token, expires_at)
		VALUES (2, 'y@b', 'presseteam', 'tok-2', '2030-01-01')`)
	if err == nil || !strings.Contains(err.Error(), "CHECK") {
		t.Errorf("invitation_tokens: INSERT mit role='presseteam' muss am CHECK scheitern, err = %v", err)
	}

	// Der Rebuild darf die an users hängenden Daten nicht mitnehmen.
	var users int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 3 {
		t.Errorf("users nach 056 up: %d Zeilen, want 3", users)
	}
}
