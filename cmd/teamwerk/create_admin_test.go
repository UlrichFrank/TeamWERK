package main

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/db"
)

// insertAdminUser schrieb bis zu diesem Test in eine Spalte `users.name`, die es
// seit Migration 019 nicht mehr gibt — `create-admin` (und damit
// `make create-admin-remote`) schlug auf jeder migrierten DB fehl. Der Test läuft
// gegen das echte, vollständig migrierte Schema und fängt genau diese Drift.
func TestInsertAdminUser(t *testing.T) {
	path := dbPathForTest(t)
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	if err := insertAdminUser(database, "admin@example.org", "Erika Musterfrau", "hash"); err != nil {
		t.Fatalf("insertAdminUser: %v", err)
	}

	var first, last, role string
	err = database.QueryRow(
		`SELECT first_name, last_name, role FROM users WHERE email = ?`,
		"admin@example.org",
	).Scan(&first, &last, &role)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if first != "Erika" || last != "Musterfrau" {
		t.Errorf("Name falsch zerlegt: first=%q last=%q", first, last)
	}
	if role != "admin" {
		t.Errorf("role = %q, erwartet admin", role)
	}
}

func TestSplitName(t *testing.T) {
	cases := []struct {
		in          string
		first, last string
	}{
		{"Erika Musterfrau", "Erika", "Musterfrau"},
		{"Anna Lena Müller", "Anna Lena", "Müller"},
		{"Admin", "Admin", ""},
		{"  Max   Mustermann  ", "Max", "Mustermann"},
		{"", "", ""},
	}
	for _, c := range cases {
		first, last := splitName(c.in)
		if first != c.first || last != c.last {
			t.Errorf("splitName(%q) = (%q, %q), erwartet (%q, %q)", c.in, first, last, c.first, c.last)
		}
	}
}
