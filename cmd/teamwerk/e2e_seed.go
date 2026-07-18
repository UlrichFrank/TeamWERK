package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/teamstuttgart/teamwerk/internal/db"
)

// runE2ESeed legt eine deterministische Test-Datenbank für die Playwright-E2E-Suite an.
// Verwendung: teamwerk e2e-seed --db=./e2e.db
//
// Die Datei wird bei jedem Lauf frisch erzeugt (idempotent: gleicher Aufruf → gleicher
// Zustand). Enthält 1 Admin + 3 Standard-Nutzer und eine kleine Chat-Konversation mit
// Textnachrichten. Bewusst schlank — die Suite wächst mit weiteren Killer-Cases.
func runE2ESeed() {
	fs := flag.NewFlagSet("e2e-seed", flag.ExitOnError)
	dbPath := fs.String("db", "", "Pfad zur SQLite-Datenbank (Pflicht)")
	_ = fs.Parse(os.Args[2:])
	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "Verwendung: teamwerk e2e-seed --db=<pfad>")
		os.Exit(1)
	}

	// Frische DB erzwingen (deterministisch). WAL-Seitendateien mit entfernen.
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(*dbPath + suffix)
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		fatal("e2e-seed: open db failed", "error", err)
	}
	defer database.Close()

	if err := db.Migrate(database, db.MigrationsFS); err != nil {
		fatal("e2e-seed: migrate failed", "error", err)
	}

	adminHash, err := bcrypt.GenerateFromPassword([]byte("E2ETestPassword!"), bcrypt.DefaultCost)
	if err != nil {
		fatal("e2e-seed: bcrypt failed", "error", err)
	}
	adminID := mustInsertUser(database, "e2e@test.local", string(adminHash), "admin", "E2E", "Admin")

	// Drei Standard-Nutzer (gleiches Passwort für Einfachheit).
	stdHash, _ := bcrypt.GenerateFromPassword([]byte("E2ETestPassword!"), bcrypt.DefaultCost)
	userIDs := []int{
		mustInsertUser(database, "user1@test.local", string(stdHash), "standard", "Uwe", "Eins"),
		mustInsertUser(database, "user2@test.local", string(stdHash), "standard", "Uta", "Zwei"),
		mustInsertUser(database, "user3@test.local", string(stdHash), "standard", "Udo", "Drei"),
	}

	// Eine Gruppen-Konversation Admin + User1 mit ein paar Textnachrichten.
	seedTextConversation(database, "E2E Chat", []int{adminID, userIDs[0]}, adminID, 8)

	fmt.Printf("e2e-seed: DB %s angelegt (admin=%d, users=%v)\n", *dbPath, adminID, userIDs)
}

func mustInsertUser(database *sql.DB, email, hash, role, first, last string) int {
	res, err := database.Exec(
		`INSERT INTO users (email, password, role, first_name, last_name) VALUES (?, ?, ?, ?, ?)`,
		email, hash, role, first, last)
	if err != nil {
		fatal("e2e-seed: insert user failed", "email", email, "error", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// seedTextConversation legt eine Gruppen-Konversation mit Mitgliedern + n Textnachrichten an.
func seedTextConversation(database *sql.DB, title string, memberIDs []int, senderID, n int) {
	res, err := database.Exec(
		`INSERT INTO conversations (type, name, created_by) VALUES ('group', ?, ?)`, title, senderID)
	if err != nil {
		fatal("e2e-seed: insert conversation failed", "error", err)
	}
	convID, _ := res.LastInsertId()
	for _, m := range memberIDs {
		if _, err := database.Exec(
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, m); err != nil {
			fatal("e2e-seed: insert conversation_member failed", "error", err)
		}
	}
	for i := 0; i < n; i++ {
		if _, err := database.Exec(
			`INSERT INTO messages (conversation_id, sender_id, body) VALUES (?, ?, ?)`,
			convID, senderID, fmt.Sprintf("E2E Nachricht %d", i+1)); err != nil {
			fatal("e2e-seed: insert message failed", "error", err)
		}
	}
}
