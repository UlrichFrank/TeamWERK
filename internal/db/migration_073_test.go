package db_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TestMigration073_GameMeetingPoint prüft Treffzeit-Spalten und Debounce-Queue
// (spiel-treffpunkt): Bestandsspiele ohne Treffzeit, CHECKs greifen, Löschen
// des Spiels räumt die wartende Meldung mit auf, down entfernt alles.
func TestMigration073_GameMeetingPoint(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(72); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 72: %v", err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO seasons (id, name, start_date, end_date, is_active) VALUES (1, '26/27', '2026-07-01', '2027-06-30', 1);
		INSERT INTO games (id, season_id, opponent, date, time) VALUES (1, 1, 'TSV', '2026-10-11', '15:00');
	`); err != nil {
		t.Fatalf("seed Bestand: %v", err)
	}
	if err := m.Migrate(73); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 73: %v", err)
	}

	var off *int
	var place string
	if err := sqlDB.QueryRow(`SELECT meet_offset_minutes, meet_place FROM games WHERE id=1`).Scan(&off, &place); err != nil {
		t.Fatalf("read: %v", err)
	}
	if off != nil || place != "" {
		t.Fatalf("Bestandsspiel hat Treffzeit %v / %q, erwartet keine", off, place)
	}

	if _, err := sqlDB.Exec(`UPDATE games SET meet_offset_minutes=90, meet_place='Parkplatz' WHERE id=1`); err != nil {
		t.Fatalf("gültige Werte: %v", err)
	}
	if _, err := sqlDB.Exec(`UPDATE games SET meet_offset_minutes=721 WHERE id=1`); err == nil {
		t.Error("Abstand > 720 sollte am CHECK scheitern")
	}
	if _, err := sqlDB.Exec(`UPDATE games SET meet_offset_minutes=-1 WHERE id=1`); err == nil {
		t.Error("negativer Abstand sollte am CHECK scheitern")
	}
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'x'
	}
	if _, err := sqlDB.Exec(`UPDATE games SET meet_place=? WHERE id=1`, string(long)); err == nil {
		t.Error("Ort > 100 Zeichen sollte am CHECK scheitern")
	}

	if _, err := sqlDB.Exec(`INSERT INTO pending_game_meeting_push (game_id, prev_offset, prev_place, notify_after) VALUES (1, NULL, '', datetime('now'))`); err != nil {
		t.Fatalf("pending insert: %v", err)
	}
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("foreign_keys on: %v", err)
	}
	if _, err := sqlDB.Exec(`DELETE FROM games WHERE id = 1`); err != nil {
		t.Fatalf("delete game: %v", err)
	}
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("foreign_keys off: %v", err)
	}
	var n int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM pending_game_meeting_push`).Scan(&n)
	if n != 0 {
		t.Errorf("nach Löschen des Spiels %d wartende Meldungen, erwartet 0", n)
	}

	if err := m.Migrate(72); err != nil {
		t.Fatalf("migrate down to 72: %v", err)
	}
	if _, err := sqlDB.Exec(`SELECT meet_offset_minutes FROM games`); err == nil {
		t.Error("meet_offset_minutes existiert nach down noch")
	}
}
