package db_test

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/teamstuttgart/teamwerk/internal/db"
)

// newMigrator öffnet eine frische In-Memory-DB und liefert einen migrate-Instance,
// der wie db.Migrate() mit deaktivierter FK-Enforcement auf einer einzigen
// Connection arbeitet (PRAGMA foreign_keys ist innerhalb der von golang-migrate
// genutzten Transaktion ein No-op).
func newMigrator(t *testing.T) (*sql.DB, *migrate.Migrate) {
	t.Helper()
	sqlDB, err := sql.Open("sqlite-busy-counting", "file:migtest?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("pragma off: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	src, err := iofs.New(db.MigrationsFS, "migrations")
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	driver, err := sqlite3.WithInstance(sqlDB, &sqlite3.Config{})
	if err != nil {
		t.Fatalf("driver: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite3", driver)
	if err != nil {
		t.Fatalf("migrate init: %v", err)
	}
	t.Cleanup(func() { m.Close() })
	return sqlDB, m
}

func hasColumn(t *testing.T, sqlDB *sql.DB, table, col string) bool {
	t.Helper()
	rows, err := sqlDB.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatalf("pragma_table_info(%s): %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if name == col {
			return true
		}
	}
	return false
}

func tableExists(t *testing.T, sqlDB *sql.DB, table string) bool {
	t.Helper()
	var n int
	err := sqlDB.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n)
	if err != nil {
		t.Fatalf("sqlite_master: %v", err)
	}
	return n > 0
}

func TestMigration011_Up_AddsColumnsAndQueue(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(11); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 11: %v", err)
	}

	if !hasColumn(t, sqlDB, "games", "note") {
		t.Error("erwartet games.note nach 011 up")
	}
	if !tableExists(t, sqlDB, "pending_event_notes_push") {
		t.Error("erwartet Tabelle pending_event_notes_push nach 011 up")
	}
}

func TestMigration011_Up_CheckRejectsLongNote(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(11); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 11: %v", err)
	}

	// Minimal nötige Parent-Zeilen für die FK-Spalten von games.
	if _, err := sqlDB.Exec(`INSERT INTO seasons (id, name, start_date, end_date, is_active)
		VALUES (1, '24/25', '2024-08-01', '2025-07-31', 1)`); err != nil {
		t.Fatalf("seed season: %v", err)
	}

	long := strings.Repeat("x", 201)
	_, err := sqlDB.Exec(
		`INSERT INTO games (season_id, opponent, date, note) VALUES (1, 'X', '2025-01-01', ?)`, long)
	if err == nil {
		t.Fatal("erwartet CHECK-Verletzung für games.note > 200 Zeichen")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "constraint") {
		t.Errorf("erwartet constraint-Fehler, bekam: %v", err)
	}

	ok := strings.Repeat("y", 200)
	if _, err := sqlDB.Exec(
		`INSERT INTO games (season_id, opponent, date, note) VALUES (1, 'X', '2025-01-01', ?)`, ok); err != nil {
		t.Errorf("200-Zeichen-Note sollte erlaubt sein, bekam: %v", err)
	}
}

func TestMigration011_Down_RemovesColumnsAndQueue(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(11); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 11: %v", err)
	}
	if err := m.Migrate(10); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 10: %v", err)
	}

	if hasColumn(t, sqlDB, "games", "note") {
		t.Error("games.note sollte nach 011 down weg sein")
	}
	if tableExists(t, sqlDB, "pending_event_notes_push") {
		t.Error("pending_event_notes_push sollte nach 011 down weg sein")
	}
}

// TC: Migration 016 füllt ein eindeutig zuordenbares namenloses Kinderkonto aus
// membership_requests, lässt aber mehrdeutige Zuordnungen unangetastet.
func TestMigration016_BackfillChildNames(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(15); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 15: %v", err)
	}

	// (A) Eindeutig: ein namenloses Kinderkonto + genau ein passender Antrag.
	if _, err := sqlDB.Exec(
		`INSERT INTO users (id, email, login_name, first_name, last_name, can_login, recovery_email)
		 VALUES (100, NULL, 'Lena.Schmidt', '', '', 0, 'eltern@test.local')`); err != nil {
		t.Fatalf("seed child A: %v", err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO membership_requests (first_name, last_name, email, is_child, parent_email, status)
		 VALUES ('Lena', 'Schmidt', '', 1, 'eltern@test.local', 'approved')`); err != nil {
		t.Fatalf("seed request A: %v", err)
	}

	// (B) Mehrdeutig: zwei namensgleiche approved Anträge derselben Eltern-Adresse
	// → COUNT != 1 → Konto bleibt leer (kein Ratewerk).
	if _, err := sqlDB.Exec(
		`INSERT INTO users (id, email, login_name, first_name, last_name, can_login, recovery_email)
		 VALUES (101, NULL, 'Max.Mueller', '', '', 0, 'zwilling@test.local')`); err != nil {
		t.Fatalf("seed child B: %v", err)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO membership_requests (first_name, last_name, email, is_child, parent_email, status)
		 VALUES ('Max', 'Mueller', '', 1, 'zwilling@test.local', 'approved'),
		        ('Max', 'Mueller', '', 1, 'zwilling@test.local', 'approved')`); err != nil {
		t.Fatalf("seed requests B: %v", err)
	}

	if err := m.Migrate(16); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 16: %v", err)
	}

	var aFirst, aLast string
	if err := sqlDB.QueryRow(`SELECT first_name, last_name FROM users WHERE id=100`).Scan(&aFirst, &aLast); err != nil {
		t.Fatalf("read child A: %v", err)
	}
	if aFirst != "Lena" || aLast != "Schmidt" {
		t.Errorf("child A: erwartet 'Lena Schmidt', bekam %q %q", aFirst, aLast)
	}

	var bFirst, bLast string
	if err := sqlDB.QueryRow(`SELECT first_name, last_name FROM users WHERE id=101`).Scan(&bFirst, &bLast); err != nil {
		t.Fatalf("read child B: %v", err)
	}
	if bFirst != "" || bLast != "" {
		t.Errorf("child B (mehrdeutig): erwartet leer, bekam %q %q", bFirst, bLast)
	}

	// down (No-op) muss von golang-migrate fehlerfrei akzeptiert werden; die
	// bereits gefüllten Namen bleiben erhalten (kein Rollback des Backfills).
	if err := m.Migrate(15); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 15: %v", err)
	}
	sqlDB.QueryRow(`SELECT first_name, last_name FROM users WHERE id=100`).Scan(&aFirst, &aLast)
	if aFirst != "Lena" || aLast != "Schmidt" {
		t.Errorf("nach down: Name sollte unverändert sein, bekam %q %q", aFirst, aLast)
	}
}
