package db_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TC: Migration 029 konsolidiert Profilbilder auf users.photo_path. Drei
// Konstellationen decken die Merge-Regeln ab:
//
//	A) Member mit User, beide Fotos gefüllt → users gewinnt (member-Datei
//	   wird obsolet; Datei-Cleanup läuft im Backfill).
//	B) Member mit User, nur members.photo_path → nach users kopieren.
//	C) Member ohne User, nur members.photo_path → Datei wird beim Column-Drop
//	   obsolet, Backfill entfernt sie später vom Disk.
//
// Zusätzlich prüfen wir photo_visible-Übernahme auf user_visibility und den
// Wegfall der Spalten selbst.
func TestMigration029_ConsolidatesPhotoOnUser(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(28); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 28: %v", err)
	}

	// Users seeden.
	if _, err := sqlDB.Exec(`INSERT INTO users (id, email, login_name, first_name, last_name, can_login, photo_path)
		VALUES
		  (1, 'a@b', 'a', 'A', 'A', 1, 'own-a.jpg'),
		  (2, 'b@b', 'b', 'B', 'B', 1, NULL)`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	// Members seeden (A: user_id=1 + photo, B: user_id=2 + photo, C: user_id=NULL + photo).
	if _, err := sqlDB.Exec(`INSERT INTO members (id, first_name, last_name, user_id, photo_path, photo_visible, status)
		VALUES
		  (10, 'Alpha', 'A', 1, 'member-a.jpg', 1, 'aktiv'),
		  (20, 'Bravo', 'B', 2, 'member-b.jpg', 1, 'aktiv'),
		  (30, 'Charlie', 'C', NULL, 'member-c.jpg', 1, 'aktiv')`); err != nil {
		t.Fatalf("seed members: %v", err)
	}
	// user_visibility: für User 1 gibt es schon eine Zeile mit photo_visible=0
	// (Update-Pfad); für User 2 gibt es noch keine (Insert-Pfad).
	if _, err := sqlDB.Exec(`INSERT INTO user_visibility (user_id, phones_visible, address_visible, photo_visible, email_visible, whatsapp_visible)
		VALUES (1, 1, 0, 0, 0, 0)`); err != nil {
		t.Fatalf("seed user_visibility: %v", err)
	}

	if err := m.Migrate(29); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 29: %v", err)
	}

	// Spalten sind weg.
	if hasColumn(t, sqlDB, "members", "photo_path") {
		t.Error("members.photo_path sollte nach 029 up entfernt sein")
	}
	if hasColumn(t, sqlDB, "members", "photo_visible") {
		t.Error("members.photo_visible sollte nach 029 up entfernt sein")
	}

	// A: users.photo_path bleibt bei 'own-a.jpg' (users gewinnt).
	var pA *string
	if err := sqlDB.QueryRow(`SELECT photo_path FROM users WHERE id=1`).Scan(&pA); err != nil {
		t.Fatalf("read user 1: %v", err)
	}
	if pA == nil || *pA != "own-a.jpg" {
		t.Errorf("user 1: erwartet 'own-a.jpg' (users gewinnt), bekam %v", pA)
	}

	// B: users.photo_path wurde von 'member-b.jpg' übernommen.
	var pB *string
	if err := sqlDB.QueryRow(`SELECT photo_path FROM users WHERE id=2`).Scan(&pB); err != nil {
		t.Fatalf("read user 2: %v", err)
	}
	if pB == nil || *pB != "member-b.jpg" {
		t.Errorf("user 2: erwartet 'member-b.jpg' (übernommen), bekam %v", pB)
	}

	// photo_visible: User 1 wurde von 0→1 aktualisiert.
	var pv1 int
	sqlDB.QueryRow(`SELECT photo_visible FROM user_visibility WHERE user_id=1`).Scan(&pv1)
	if pv1 != 1 {
		t.Errorf("user_visibility user 1: erwartet photo_visible=1, bekam %d", pv1)
	}
	// photo_visible: User 2 hat jetzt eine Zeile mit photo_visible=1.
	var pv2 int
	sqlDB.QueryRow(`SELECT photo_visible FROM user_visibility WHERE user_id=2`).Scan(&pv2)
	if pv2 != 1 {
		t.Errorf("user_visibility user 2: erwartet photo_visible=1 (Insert), bekam %d", pv2)
	}

	// Down: Spalten wieder da, aber leer (Datenverlust bewusst).
	if err := m.Migrate(28); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 28: %v", err)
	}
	if !hasColumn(t, sqlDB, "members", "photo_path") {
		t.Error("members.photo_path sollte nach 029 down wieder existieren")
	}
	if !hasColumn(t, sqlDB, "members", "photo_visible") {
		t.Error("members.photo_visible sollte nach 029 down wieder existieren")
	}
	// Keine Rückkopie: die Spalte ist NULL/0.
	var mp *string
	sqlDB.QueryRow(`SELECT photo_path FROM members WHERE id=10`).Scan(&mp)
	if mp != nil {
		t.Errorf("members.photo_path nach down erwartet NULL (keine Rückkopie), bekam %v", mp)
	}
}
