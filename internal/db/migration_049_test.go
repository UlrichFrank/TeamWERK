package db_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TC: Migration 049 stellt broadcasts auf die vier vereinsweiten Zielgruppen um.
// Kritisch ist nicht das Schema, sondern was der Tabellen-Rebuild NICHT anfasst:
// an broadcast_reads hängt die gesamte Zustellung. Ginge dort beim DROP TABLE
// eine Zeile verloren, wären Bestands-Mitteilungen für ihre Empfänger stumm
// verschwunden — ohne Fehler. testutil.NewDB migriert eine LEERE DB und fängt
// das nicht, daher dieser dedizierte Seed-und-Migrate-Test.
func TestMigration049_ZielgruppenUndBestand(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(48); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 48: %v", err)
	}

	if _, err := sqlDB.Exec(`INSERT INTO users (id, email, login_name, first_name, last_name, can_login)
		VALUES (1, 'a@b', 'a', 'A', 'B', 1)`); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// Je eine Zeile pro Alt-Zielgruppe.
	if _, err := sqlDB.Exec(`INSERT INTO broadcasts (id, sender_id, target_type, body) VALUES (1, 1, 'all', 'an alle')`); err != nil {
		t.Fatalf("seed broadcast all: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO broadcasts (id, sender_id, target_type, target_id, body) VALUES (2, 1, 'team', 7, 'an team 7')`); err != nil {
		t.Fatalf("seed broadcast team: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO broadcasts (id, sender_id, target_type, target_role, body) VALUES (3, 1, 'role', 'spieler', 'an spieler')`); err != nil {
		t.Fatalf("seed broadcast role: %v", err)
	}
	for _, id := range []int{1, 2, 3} {
		if _, err := sqlDB.Exec(`INSERT INTO broadcast_reads (broadcast_id, user_id) VALUES (?, 1)`, id); err != nil {
			t.Fatalf("seed broadcast_read %d: %v", id, err)
		}
	}

	if err := m.Migrate(49); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 49: %v", err)
	}

	// Schema: die beiden Alt-Spalten sind weg.
	if hasColumn(t, sqlDB, "broadcasts", "target_id") {
		t.Error("broadcasts.target_id existiert nach 049 up noch")
	}
	if hasColumn(t, sqlDB, "broadcasts", "target_role") {
		t.Error("broadcasts.target_role existiert nach 049 up noch")
	}

	// Bestandswerte abgebildet: 'all' → 'users', alles andere → 'legacy'.
	want := map[int]string{1: "users", 2: "legacy", 3: "legacy"}
	for id, expected := range want {
		var got string
		if err := sqlDB.QueryRow(`SELECT target_type FROM broadcasts WHERE id=?`, id).Scan(&got); err != nil {
			t.Fatalf("read broadcast %d: %v", id, err)
		}
		if got != expected {
			t.Errorf("broadcast %d: target_type = %q, want %q", id, got, expected)
		}
	}

	// Die eigentliche Invariante: kein Cascade-Delete auf broadcast_reads.
	var reads int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM broadcast_reads`).Scan(&reads); err != nil {
		t.Fatalf("count broadcast_reads: %v", err)
	}
	if reads != 3 {
		t.Errorf("broadcast_reads nach 049 up: %d Zeilen, want 3 (Zustellung ginge sonst still verloren)", reads)
	}

	// Der neue CHECK ist scharf: ein Alt-Wert lässt sich nicht mehr schreiben.
	if _, err := sqlDB.Exec(`INSERT INTO broadcasts (id, sender_id, target_type, body) VALUES (4, 1, 'team', 'x')`); err == nil {
		t.Error("INSERT mit target_type='team' wurde nach 049 akzeptiert, CHECK greift nicht")
	}

	// Down: Alt-Spalten zurück, broadcast_reads weiterhin unangetastet.
	if err := m.Migrate(48); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 48: %v", err)
	}
	if !hasColumn(t, sqlDB, "broadcasts", "target_id") {
		t.Error("broadcasts.target_id fehlt nach 049 down")
	}
	if !hasColumn(t, sqlDB, "broadcasts", "target_role") {
		t.Error("broadcasts.target_role fehlt nach 049 down")
	}
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM broadcast_reads`).Scan(&reads); err != nil {
		t.Fatalf("count broadcast_reads nach down: %v", err)
	}
	if reads != 3 {
		t.Errorf("broadcast_reads nach 049 down: %d Zeilen, want 3", reads)
	}
}
