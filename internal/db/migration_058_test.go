package db_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TestMigration058_BackfillKaderId: der Backfill über (team_id, season_id) muss
// jedem Bestands-Training genau den Kader seines Teams in seiner Saison geben —
// nicht den eines anderen Teams und nicht den derselben Mannschaft aus einer
// anderen Saison. `testutil.NewDB` migriert eine leere Datenbank und kann das
// nicht prüfen; hier laufen echte Bestandszeilen durch den Rebuild.
//
// Der Test deckt zugleich die zweite Zusage der Migration ab: `team_id` bleibt
// bei Mannschaftstrainings gesetzt (es ist die Projektion, nicht Ballast), und
// die an den Rebuild angehängten Daten (`training_responses`,
// `training_attendances`, `member_series_unavailabilities`) überleben ihn mit
// unveränderten IDs.
func TestMigration058_BackfillKaderId(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(57); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 57: %v", err)
	}

	// Zwei Saisons × zwei Teams — vier Kader. Wäre der Backfill nur auf team_id
	// oder nur auf season_id gekeyt, verwechselte er hier sichtbar.
	if _, err := sqlDB.Exec(`
		INSERT INTO seasons (id, name, start_date, end_date, is_active) VALUES
		    (1, '2024/25', '2024-09-01', '2025-06-30', 0),
		    (2, '2025/26', '2025-09-01', '2026-06-30', 1);
		INSERT INTO teams (id, name, age_class, gender) VALUES
		    (1, 'Team A', 'Erwachsene', 'mixed'),
		    (2, 'Team B', 'Erwachsene', 'mixed');
		INSERT INTO kader (id, season_id, age_class, gender, team_id, team_number) VALUES
		    (10, 1, 'Erwachsene', 'mixed', 1, 1),
		    (11, 1, 'Erwachsene', 'mixed', 2, 2),
		    (12, 2, 'Erwachsene', 'mixed', 1, 1),
		    (13, 2, 'Erwachsene', 'mixed', 2, 2);
		INSERT INTO users (id, email, login_name, first_name, last_name, can_login, role)
		VALUES (1, 'a@b', 'a', 'A', 'B', 1, 'admin');
		INSERT INTO members (id, first_name, last_name, status, user_id)
		VALUES (1, 'M', 'Eins', 'aktiv', 1);
	`); err != nil {
		t.Fatalf("seed Stammdaten: %v", err)
	}

	if _, err := sqlDB.Exec(`
		INSERT INTO training_series (id, team_id, season_id, name, day_of_week,
		    start_time, end_time, valid_from, valid_until, created_by) VALUES
		    (100, 1, 1, 'A alt', 2, '18:00', '20:00', '2024-10-01', '2025-06-30', 1),
		    (101, 1, 2, 'A neu', 2, '18:00', '20:00', '2025-10-01', '2026-06-30', 1),
		    (102, 2, 2, 'B neu', 3, '18:00', '20:00', '2025-10-01', '2026-06-30', 1);
		INSERT INTO training_sessions (id, series_id, team_id, season_id, date,
		    start_time, end_time, title) VALUES
		    (200, 100, 1, 1, '2024-10-08', '18:00', '20:00', 'A alt'),
		    (201, 101, 1, 2, '2025-10-07', '18:00', '20:00', 'A neu'),
		    (202, NULL, 2, 2, '2025-10-09', '18:00', '20:00', 'B einzeln');
		INSERT INTO training_responses (id, training_id, member_id, responded_by, status)
		VALUES (300, 201, 1, 1, 'confirmed');
		INSERT INTO training_attendances (id, training_id, member_id, present)
		VALUES (400, 201, 1, 1);
		INSERT INTO member_series_unavailabilities (id, member_id, training_series_id, created_by)
		VALUES (500, 1, 101, 1);
	`); err != nil {
		t.Fatalf("seed Trainings: %v", err)
	}

	if err := m.Migrate(58); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 58: %v", err)
	}

	// Jedes Training trägt genau den Kader seines (team_id, season_id).
	wantSeriesKader := map[int]int{100: 10, 101: 12, 102: 13}
	for id, want := range wantSeriesKader {
		var got int
		if err := sqlDB.QueryRow(`SELECT kader_id FROM training_series WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatalf("training_series %d lesen: %v", id, err)
		}
		if got != want {
			t.Errorf("training_series %d: kader_id = %d, erwartet %d", id, got, want)
		}
	}
	wantSessionKader := map[int]int{200: 10, 201: 12, 202: 13}
	for id, want := range wantSessionKader {
		var got int
		if err := sqlDB.QueryRow(`SELECT kader_id FROM training_sessions WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatalf("training_sessions %d lesen: %v", id, err)
		}
		if got != want {
			t.Errorf("training_sessions %d: kader_id = %d, erwartet %d", id, got, want)
		}
	}

	// team_id bleibt bei Mannschaftstrainings gesetzt — es ist die Projektion,
	// über die attendance/calendar/dashboard/videos weiterhin auflösen.
	var nullTeams int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM training_sessions WHERE team_id IS NULL`).Scan(&nullTeams); err != nil {
		t.Fatalf("team_id zählen: %v", err)
	}
	if nullTeams != 0 {
		t.Errorf("%d Mannschaftstrainings haben team_id NULL verloren", nullTeams)
	}

	// Die angehängten Daten überleben den Rebuild mit unveränderten IDs.
	for _, c := range []struct{ table, where string }{
		{"training_responses", "id = 300 AND training_id = 201 AND member_id = 1"},
		{"training_attendances", "id = 400 AND training_id = 201 AND member_id = 1"},
		{"member_series_unavailabilities", "id = 500 AND training_series_id = 101"},
	} {
		var n int
		if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM ` + c.table + ` WHERE ` + c.where).Scan(&n); err != nil {
			t.Fatalf("%s lesen: %v", c.table, err)
		}
		if n != 1 {
			t.Errorf("%s: erwartet 1 Zeile mit %s, gefunden %d", c.table, c.where, n)
		}
	}

	// Eine Übungsgruppe kann ab jetzt Besitzer sein: kader_id gesetzt,
	// team_id NULL — genau die Zeile, die vor 058 nicht existieren konnte.
	if _, err := sqlDB.Exec(
		`INSERT INTO kader (id, season_id, kind, name) VALUES (20, 2, 'practice', 'Torwarttraining')`); err != nil {
		t.Fatalf("Übungsgruppe anlegen: %v", err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO training_sessions (id, kader_id, season_id, date, start_time, end_time, title)
		VALUES (203, 20, 2, '2025-11-01', '18:00', '20:00', 'TW')`); err != nil {
		t.Fatalf("Übungsgruppen-Termin anlegen: %v", err)
	}
	var teamID any
	if err := sqlDB.QueryRow(`SELECT team_id FROM training_sessions WHERE id = 203`).Scan(&teamID); err != nil {
		t.Fatalf("team_id lesen: %v", err)
	}
	if teamID != nil {
		t.Errorf("team_id = %v, erwartet NULL", teamID)
	}
}
