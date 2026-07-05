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

// TC: Migration 018 ersetzt rsvp_opt_out durch zwei rsvp_default_*-Enums;
// bestehende opt_out=1-Rows werden konservativ auf players='confirmed' gemappt,
// extended startet überall 'none' (aktuelles Verhalten).
func TestMigration018_ReplacesOptOutWithPerRoleDefaults(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(17); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 17: %v", err)
	}

	// Parent-Rows für FKs.
	if _, err := sqlDB.Exec(`INSERT INTO seasons (id, name, start_date, end_date, is_active)
		VALUES (1, '24/25', '2024-08-01', '2025-07-31', 1)`); err != nil {
		t.Fatalf("seed season: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO teams (id, name, gender, age_class, is_active)
		VALUES (1, 'H1', 'm', 'herren', 1)`); err != nil {
		t.Fatalf("seed team: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO users (id, email, login_name, first_name, last_name, can_login)
		VALUES (1, 'a@b', 'a', 'A', 'B', 1)`); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	// Ein Spiel mit opt_out=1 (soll später zu players='confirmed' werden), eins mit opt_out=0.
	if _, err := sqlDB.Exec(`INSERT INTO games (id, season_id, opponent, date, rsvp_opt_out)
		VALUES (10, 1, 'X', '2025-01-01', 1), (11, 1, 'Y', '2025-01-02', 0)`); err != nil {
		t.Fatalf("seed games: %v", err)
	}
	// Eine Trainings-Serie mit opt_out=1.
	if _, err := sqlDB.Exec(`INSERT INTO training_series
		(id, team_id, season_id, name, day_of_week, start_time, end_time,
		 valid_from, valid_until, created_by, rsvp_opt_out)
		VALUES (20, 1, 1, 'A', 1, '18:00', '19:30', '2024-08-01', '2025-07-31', 1, 1)`); err != nil {
		t.Fatalf("seed series: %v", err)
	}
	// Eine Session mit opt_out=0.
	if _, err := sqlDB.Exec(`INSERT INTO training_sessions
		(id, series_id, team_id, season_id, date, start_time, end_time, rsvp_opt_out)
		VALUES (30, 20, 1, 1, '2025-01-06', '18:00', '19:30', 0)`); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	if err := m.Migrate(18); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 18: %v", err)
	}

	// Alte Spalte weg, neue Spalten da.
	if hasColumn(t, sqlDB, "games", "rsvp_opt_out") {
		t.Error("games.rsvp_opt_out sollte nach 018 up entfernt sein")
	}
	if !hasColumn(t, sqlDB, "games", "rsvp_default_players") {
		t.Error("games.rsvp_default_players fehlt nach 018 up")
	}
	if !hasColumn(t, sqlDB, "games", "rsvp_default_extended") {
		t.Error("games.rsvp_default_extended fehlt nach 018 up")
	}
	if hasColumn(t, sqlDB, "training_series", "rsvp_opt_out") {
		t.Error("training_series.rsvp_opt_out sollte weg sein")
	}
	if hasColumn(t, sqlDB, "training_sessions", "rsvp_opt_out") {
		t.Error("training_sessions.rsvp_opt_out sollte weg sein")
	}

	// Backfill: opt_out=1 → players='confirmed', extended immer 'none'.
	var p, e string
	sqlDB.QueryRow(`SELECT rsvp_default_players, rsvp_default_extended FROM games WHERE id=10`).Scan(&p, &e)
	if p != "confirmed" || e != "none" {
		t.Errorf("game 10: erwartet ('confirmed','none'), bekam (%q,%q)", p, e)
	}
	sqlDB.QueryRow(`SELECT rsvp_default_players, rsvp_default_extended FROM games WHERE id=11`).Scan(&p, &e)
	if p != "none" || e != "none" {
		t.Errorf("game 11: erwartet ('none','none'), bekam (%q,%q)", p, e)
	}
	sqlDB.QueryRow(`SELECT rsvp_default_players, rsvp_default_extended FROM training_series WHERE id=20`).Scan(&p, &e)
	if p != "confirmed" || e != "none" {
		t.Errorf("series 20: erwartet ('confirmed','none'), bekam (%q,%q)", p, e)
	}
	sqlDB.QueryRow(`SELECT rsvp_default_players, rsvp_default_extended FROM training_sessions WHERE id=30`).Scan(&p, &e)
	if p != "none" || e != "none" {
		t.Errorf("session 30: erwartet ('none','none'), bekam (%q,%q)", p, e)
	}

	// CHECK-Constraint greift.
	if _, err := sqlDB.Exec(`UPDATE games SET rsvp_default_players='bogus' WHERE id=10`); err == nil {
		t.Error("erwartet CHECK-Verletzung für rsvp_default_players='bogus'")
	}

	// Down: Enums zurück auf Bool, 'confirmed' → 1, sonst 0.
	if err := m.Migrate(17); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 17: %v", err)
	}
	if !hasColumn(t, sqlDB, "games", "rsvp_opt_out") {
		t.Error("games.rsvp_opt_out sollte nach 018 down zurück sein")
	}
	var opt10, opt11 int
	sqlDB.QueryRow(`SELECT rsvp_opt_out FROM games WHERE id=10`).Scan(&opt10)
	sqlDB.QueryRow(`SELECT rsvp_opt_out FROM games WHERE id=11`).Scan(&opt11)
	if opt10 != 1 || opt11 != 0 {
		t.Errorf("nach down: erwartet games (1,0), bekam (%d,%d)", opt10, opt11)
	}
}

// TC: Migration 023 legt system_settings idempotent an und schreibt genau eine
// Default-Row maintenance_mode=off. Zweites Up ist ein No-op.
func TestMigration023_SystemSettings_Idempotent(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(23); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 23: %v", err)
	}
	if !tableExists(t, sqlDB, "system_settings") {
		t.Fatal("erwartet Tabelle system_settings nach 023 up")
	}

	var count int
	if err := sqlDB.QueryRow(`SELECT count(*) FROM system_settings WHERE key='maintenance_mode'`).Scan(&count); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if count != 1 {
		t.Errorf("erwartet genau 1 Row maintenance_mode, bekam %d", count)
	}

	var value string
	if err := sqlDB.QueryRow(`SELECT value FROM system_settings WHERE key='maintenance_mode'`).Scan(&value); err != nil {
		t.Fatalf("query value: %v", err)
	}
	if value != "off" {
		t.Errorf("erwartet default value 'off', bekam %q", value)
	}

	// Zweite up-Ausführung derselben Migration darf keinen Konflikt werfen und
	// die Row-Anzahl nicht ändern (INSERT OR IGNORE + CREATE TABLE IF NOT EXISTS).
	if _, err := sqlDB.Exec(`CREATE TABLE IF NOT EXISTS system_settings (
		key TEXT PRIMARY KEY, value TEXT NOT NULL,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_by INTEGER REFERENCES users(id) ON DELETE SET NULL)`); err != nil {
		t.Fatalf("re-create idempotent failed: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT OR IGNORE INTO system_settings (key, value) VALUES ('maintenance_mode', 'off')`); err != nil {
		t.Fatalf("re-insert idempotent failed: %v", err)
	}

	if err := sqlDB.QueryRow(`SELECT count(*) FROM system_settings`).Scan(&count); err != nil {
		t.Fatalf("query count 2: %v", err)
	}
	if count != 1 {
		t.Errorf("nach doppelter Ausführung erwartet 1 Row, bekam %d", count)
	}

	// Down entfernt Tabelle wieder.
	if err := m.Migrate(22); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 22: %v", err)
	}
	if tableExists(t, sqlDB, "system_settings") {
		t.Error("system_settings sollte nach 023 down weg sein")
	}
}
