package db_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TestMigration065_DutyReminderOffsets prüft den Tabellen-Rebuild von
// duty_reminder_log (neue Spalte days_before, neuer PK) und die neue Tabelle
// duty_board_reminder_log für die Vorstands-Übersicht (duty-reminder-eskalation).
// Bestandszeilen aus der Zeit vor der Migration waren ausschließlich der alte
// 2-Tage-Reminder — der Backfill muss sie auf days_before=2 setzen, nicht auf 0
// oder NULL, sonst würde der neue Scheduler-Code sie fälschlich nie wiederfinden.
func TestMigration065_DutyReminderOffsets(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(64); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 64: %v", err)
	}

	if _, err := sqlDB.Exec(`
		INSERT INTO users (id, email, login_name, first_name, last_name, can_login, role)
		VALUES (1, 'a@b', 'a', 'A', 'B', 1, 'standard');
		INSERT INTO duty_reminder_log (user_id, event_date) VALUES (1, '2026-07-10');
	`); err != nil {
		t.Fatalf("seed Bestandszeile: %v", err)
	}

	if err := m.Migrate(65); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 65: %v", err)
	}

	var daysBefore int
	if err := sqlDB.QueryRow(
		`SELECT days_before FROM duty_reminder_log WHERE user_id = 1 AND event_date = '2026-07-10'`,
	).Scan(&daysBefore); err != nil {
		t.Fatalf("Bestandszeile nach Migration lesen: %v", err)
	}
	if daysBefore != 2 {
		t.Errorf("days_before = %d, erwartet 2 (Backfill des alten Einzel-Reminders)", daysBefore)
	}

	// Neuer PK erlaubt mehrere Offsets für denselben (user_id, event_date).
	if _, err := sqlDB.Exec(
		`INSERT INTO duty_reminder_log (user_id, event_date, days_before) VALUES (1, '2026-07-10', 7)`,
	); err != nil {
		t.Fatalf("zweiten Offset für dieselbe event_date einfügen: %v", err)
	}
	var count int
	if err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM duty_reminder_log WHERE user_id = 1 AND event_date = '2026-07-10'`,
	).Scan(&count); err != nil {
		t.Fatalf("Zeilen zählen: %v", err)
	}
	if count != 2 {
		t.Fatalf("erwartet 2 Zeilen (Offset 2 und 7) für dieselbe event_date, got %d", count)
	}

	// Neue Tabelle für die Vorstands-Übersicht existiert und ist eigenständig.
	if _, err := sqlDB.Exec(
		`INSERT INTO duty_board_reminder_log (user_id, event_date, days_before) VALUES (1, '2026-07-10', 3)`,
	); err != nil {
		t.Fatalf("duty_board_reminder_log insert: %v", err)
	}

	if err := m.Migrate(64); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 64: %v", err)
	}

	// Rollback: duty_board_reminder_log ist wieder weg.
	if _, err := sqlDB.Exec(`SELECT 1 FROM duty_board_reminder_log`); err == nil {
		t.Error("duty_board_reminder_log sollte nach dem Rollback nicht mehr existieren")
	}

	// Rollback: duty_reminder_log ist zurück auf den alten Schlüssel — nur die
	// days_before=2-Zeile überlebt, die 7-Tage-Zeile ist Datenverlust auf dem
	// Weg zurück in ein Schema, das sie nie kannte.
	var oldCount int
	if err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM duty_reminder_log WHERE user_id = 1 AND event_date = '2026-07-10'`,
	).Scan(&oldCount); err != nil {
		t.Fatalf("Zeilen nach Rollback zählen: %v", err)
	}
	if oldCount != 1 {
		t.Errorf("erwartet 1 überlebende Zeile (days_before=2) nach Rollback, got %d", oldCount)
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO duty_reminder_log (user_id, event_date) VALUES (1, '2026-08-01')`,
	); err != nil {
		t.Errorf("alte Schema-Form (ohne days_before) sollte nach Rollback wieder insertierbar sein: %v", err)
	}
}
