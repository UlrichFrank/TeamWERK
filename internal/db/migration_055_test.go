package db_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TC: Migration 055 löst broadcasts.target_type durch die Zeilentabelle
// broadcast_targets ab. Wie bei 049 ist das Schema nicht der kritische Teil,
// sondern was der Tabellen-Rebuild NICHT anfasst: an broadcast_reads hängt die
// gesamte Zustellung. Ginge dort beim DROP TABLE eine Zeile verloren, wären
// Bestands-Mitteilungen für ihre Empfänger stumm verschwunden — ohne Fehler.
// testutil.NewDB migriert eine LEERE DB und fängt das nicht.
func TestMigration055_BestandszielUndReadsUeberleben(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(54); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 54: %v", err)
	}

	if _, err := sqlDB.Exec(`INSERT INTO users (id, email, login_name, first_name, last_name, can_login)
		VALUES (1, 'a@b', 'a', 'A', 'B', 1)`); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	seed := map[int]string{1: "users", 2: "spieler", 3: "legacy"}
	for id, target := range seed {
		if _, err := sqlDB.Exec(
			`INSERT INTO broadcasts (id, sender_id, target_type, body) VALUES (?, 1, ?, 'text')`,
			id, target); err != nil {
			t.Fatalf("seed broadcast %d: %v", id, err)
		}
		if _, err := sqlDB.Exec(`INSERT INTO broadcast_reads (broadcast_id, user_id) VALUES (?, 1)`, id); err != nil {
			t.Fatalf("seed broadcast_read %d: %v", id, err)
		}
	}

	if err := m.Migrate(55); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 55: %v", err)
	}

	if hasColumn(t, sqlDB, "broadcasts", "target_type") {
		t.Error("broadcasts.target_type existiert nach 055 up noch")
	}

	// Jede Bestandszeile wurde zu genau einem Ziel ohne Team.
	for id, target := range seed {
		var kind string
		var teamID any
		if err := sqlDB.QueryRow(
			`SELECT kind, team_id FROM broadcast_targets WHERE broadcast_id = ?`, id).Scan(&kind, &teamID); err != nil {
			t.Fatalf("read target of broadcast %d: %v", id, err)
		}
		if kind != target {
			t.Errorf("broadcast %d: kind = %q, want %q", id, kind, target)
		}
		if teamID != nil {
			t.Errorf("broadcast %d: team_id = %v, want NULL", id, teamID)
		}
	}

	// Die eigentliche Invariante: kein Cascade-Delete auf broadcast_reads.
	var reads int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM broadcast_reads`).Scan(&reads); err != nil {
		t.Fatalf("count broadcast_reads: %v", err)
	}
	if reads != 3 {
		t.Errorf("broadcast_reads nach 055 up: %d Zeilen, want 3 (Zustellung ginge sonst still verloren)", reads)
	}

	// Der CHECK bindet team_id an die Art des Ziels — in beide Richtungen.
	if _, err := sqlDB.Exec(
		`INSERT INTO broadcast_targets (broadcast_id, kind, team_id) VALUES (1, 'team_spieler', NULL)`); err == nil {
		t.Error("team_spieler ohne team_id wurde akzeptiert, CHECK greift nicht")
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO broadcast_targets (broadcast_id, kind, team_id) VALUES (1, 'members', 7)`); err == nil {
		t.Error("vereinsweites Ziel mit team_id wurde akzeptiert, CHECK greift nicht")
	}

	// Der Unique-Index greift auch bei team_id IS NULL — als PRIMARY KEY über die
	// drei Spalten täte er das nicht (NULLs sind im Index paarweise verschieden).
	if _, err := sqlDB.Exec(
		`INSERT INTO broadcast_targets (broadcast_id, kind, team_id) VALUES (1, 'users', NULL)`); err == nil {
		t.Error("doppeltes vereinsweites Ziel wurde akzeptiert, COALESCE-Unique-Index greift nicht")
	}

	// Down: target_type zurück, broadcast_reads weiterhin unangetastet. Die
	// Mehrfachziel-Zeile fällt dabei bewusst auf 'legacy'.
	if _, err := sqlDB.Exec(
		`INSERT INTO broadcast_targets (broadcast_id, kind, team_id) VALUES (2, 'eltern', NULL)`); err != nil {
		t.Fatalf("seed zweites Ziel: %v", err)
	}
	if err := m.Migrate(54); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 54: %v", err)
	}
	if !hasColumn(t, sqlDB, "broadcasts", "target_type") {
		t.Error("broadcasts.target_type fehlt nach 055 down")
	}
	wantAfterDown := map[int]string{1: "users", 2: "legacy", 3: "legacy"}
	for id, want := range wantAfterDown {
		var got string
		if err := sqlDB.QueryRow(`SELECT target_type FROM broadcasts WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatalf("read broadcast %d nach down: %v", id, err)
		}
		if got != want {
			t.Errorf("broadcast %d nach down: target_type = %q, want %q", id, got, want)
		}
	}
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM broadcast_reads`).Scan(&reads); err != nil {
		t.Fatalf("count broadcast_reads nach down: %v", err)
	}
	if reads != 3 {
		t.Errorf("broadcast_reads nach 055 down: %d Zeilen, want 3", reads)
	}
}
