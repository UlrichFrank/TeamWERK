package db_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TestMigration071_ChatAlbum prüft die Zuordnungstabellen der Chat-Alben
// (chat-mehrere-bilder): Bestandsbilder aus messages.media_id/broadcasts.media_id
// erscheinen als Position 0, eine media-ID hängt an höchstens einer Nachricht
// (Unique-Index), und das harte Löschen einer Konversation nimmt die
// Zuordnungszeilen mit (ON DELETE CASCADE).
func TestMigration071_ChatAlbum(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(70); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 70: %v", err)
	}

	if _, err := sqlDB.Exec(`
		INSERT INTO users (id, email, login_name, first_name, last_name, can_login) VALUES (1, 'a@b', 'a', 'A', 'B', 1);
		INSERT INTO media (id, disk_name, mime_type, size, uploaded_by) VALUES
			(10, 'a.png', 'image/png', 1, 1), (11, 'b.png', 'image/png', 1, 1), (12, 'c.png', 'image/png', 1, 1);
		INSERT INTO conversations (id, type, name, created_by) VALUES (1, 'group', 'G', 1);
		INSERT INTO messages (id, conversation_id, sender_id, body, media_id) VALUES (1, 1, 1, '', 10);
		INSERT INTO messages (id, conversation_id, sender_id, body) VALUES (2, 1, 1, 'nur Text');
		INSERT INTO broadcasts (id, sender_id, body, media_id) VALUES (1, 1, '', 11);
	`); err != nil {
		t.Fatalf("seed Bestand: %v", err)
	}

	if err := m.Migrate(71); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 71: %v", err)
	}

	var msgID, pos int
	if err := sqlDB.QueryRow(`SELECT message_id, position FROM message_media WHERE media_id = 10`).Scan(&msgID, &pos); err != nil {
		t.Fatalf("Bestandsbild der Nachricht nicht übernommen: %v", err)
	}
	if msgID != 1 || pos != 0 {
		t.Errorf("message_media = (%d, %d), erwartet (1, 0)", msgID, pos)
	}
	var n int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM message_media`).Scan(&n)
	if n != 1 {
		t.Errorf("message_media hat %d Zeilen, erwartet 1 (Textnachricht ohne Bild bekommt keine)", n)
	}
	var bcID int
	if err := sqlDB.QueryRow(`SELECT broadcast_id, position FROM broadcast_media WHERE media_id = 11`).Scan(&bcID, &pos); err != nil {
		t.Fatalf("Bestandsbild der Mitteilung nicht übernommen: %v", err)
	}
	if bcID != 1 || pos != 0 {
		t.Errorf("broadcast_media = (%d, %d), erwartet (1, 0)", bcID, pos)
	}

	// Unique-Index: dieselbe media-ID an einer zweiten Nachricht scheitert.
	if _, err := sqlDB.Exec(`INSERT INTO message_media (message_id, media_id, position) VALUES (2, 10, 0)`); err == nil {
		t.Error("zweite Zuordnung derselben media_id sollte am Unique-Index scheitern")
	}
	if _, err := sqlDB.Exec(`INSERT INTO broadcast_media (broadcast_id, media_id, position) VALUES (1, 11, 1)`); err == nil {
		t.Error("zweite Zuordnung derselben media_id an eine Mitteilung sollte scheitern")
	}
	// Position außerhalb 0–9 scheitert am CHECK.
	if _, err := sqlDB.Exec(`INSERT INTO message_media (message_id, media_id, position) VALUES (1, 12, 10)`); err == nil {
		t.Error("position 10 sollte am CHECK scheitern")
	}

	// Kaskade: Löschen der Konversation entfernt messages und deren message_media.
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("pragma on: %v", err)
	}
	if _, err := sqlDB.Exec(`DELETE FROM conversations WHERE id = 1`); err != nil {
		t.Fatalf("Konversation löschen: %v", err)
	}
	sqlDB.QueryRow(`SELECT COUNT(*) FROM message_media`).Scan(&n)
	if n != 0 {
		t.Errorf("message_media hat nach Konversations-Löschung %d Zeilen, erwartet 0", n)
	}
	if _, err := sqlDB.Exec(`DELETE FROM broadcasts WHERE id = 1`); err != nil {
		t.Fatalf("Mitteilung löschen: %v", err)
	}
	sqlDB.QueryRow(`SELECT COUNT(*) FROM broadcast_media`).Scan(&n)
	if n != 0 {
		t.Errorf("broadcast_media hat nach Mitteilungs-Löschung %d Zeilen, erwartet 0", n)
	}
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("pragma off: %v", err)
	}

	if err := m.Migrate(70); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 70: %v", err)
	}
	if tableExists(t, sqlDB, "message_media") || tableExists(t, sqlDB, "broadcast_media") {
		t.Error("Zuordnungstabellen sollten nach dem Rollback fehlen")
	}
}
