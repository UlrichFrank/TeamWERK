package db_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TC: Migration 028 fügt media + media_id (messages/broadcasts) hinzu und lockert
// den body-CHECK. Kritisch: der Rebuild darf Bestandszeilen NICHT verlieren und
// insbesondere keine message_reactions/broadcast_reads kaskadierend löschen
// (Migrationslauf mit foreign_keys=OFF). testutil.NewDB migriert eine LEERE DB,
// fängt Datenverlust also nicht — daher dieser dedizierte Seed-und-Migrate-Test.
func TestMigration028_PreservesRows(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(27); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 27: %v", err)
	}

	// Bestandsdaten vor 028 seeden.
	if _, err := sqlDB.Exec(`INSERT INTO users (id, email, login_name, first_name, last_name, can_login)
		VALUES (1, 'a@b', 'a', 'A', 'B', 1)`); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO conversations (id, type, name, created_by) VALUES (1, 'group', 'G', 1)`); err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO messages (id, conversation_id, sender_id, body) VALUES (1, 1, 1, 'hallo')`); err != nil {
		t.Fatalf("seed message: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO message_reactions (message_id, user_id, emoji) VALUES (1, 1, '👍')`); err != nil {
		t.Fatalf("seed reaction: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO broadcasts (id, sender_id, target_type, body) VALUES (1, 1, 'all', 'info')`); err != nil {
		t.Fatalf("seed broadcast: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO broadcast_reads (broadcast_id, user_id) VALUES (1, 1)`); err != nil {
		t.Fatalf("seed broadcast_read: %v", err)
	}

	if err := m.Migrate(28); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 28: %v", err)
	}

	// Schema: neue Spalten + media-Tabelle.
	if !hasColumn(t, sqlDB, "messages", "media_id") {
		t.Error("messages.media_id fehlt nach 028 up")
	}
	if !hasColumn(t, sqlDB, "broadcasts", "media_id") {
		t.Error("broadcasts.media_id fehlt nach 028 up")
	}
	if !tableExists(t, sqlDB, "media") {
		t.Error("Tabelle media fehlt nach 028 up")
	}

	// Bestandszeilen erhalten, media_id NULL.
	var body string
	var mediaID *int
	if err := sqlDB.QueryRow(`SELECT body, media_id FROM messages WHERE id=1`).Scan(&body, &mediaID); err != nil {
		t.Fatalf("read message: %v", err)
	}
	if body != "hallo" || mediaID != nil {
		t.Errorf("message: erwartet ('hallo', NULL), bekam (%q, %v)", body, mediaID)
	}
	// KEIN Cascade-Delete auf message_reactions.
	var reactions int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM message_reactions WHERE message_id=1`).Scan(&reactions)
	if reactions != 1 {
		t.Errorf("message_reactions: erwartet 1 (kein Cascade-Delete), bekam %d", reactions)
	}
	var bBody string
	sqlDB.QueryRow(`SELECT body FROM broadcasts WHERE id=1`).Scan(&bBody)
	if bBody != "info" {
		t.Errorf("broadcast: erwartet 'info', bekam %q", bBody)
	}
	var reads int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM broadcast_reads WHERE broadcast_id=1`).Scan(&reads)
	if reads != 1 {
		t.Errorf("broadcast_reads: erwartet 1 (kein Cascade-Delete), bekam %d", reads)
	}

	// Gelockerter CHECK: reine Bildnachricht (leerer body + media_id) erlaubt.
	if _, err := sqlDB.Exec(`INSERT INTO media (id, disk_name, mime_type, size, uploaded_by) VALUES (1, 'x.png', 'image/png', 10, 1)`); err != nil {
		t.Fatalf("seed media: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO messages (conversation_id, sender_id, body, media_id) VALUES (1, 1, '', 1)`); err != nil {
		t.Errorf("reine Bildnachricht sollte erlaubt sein, bekam: %v", err)
	}
	// Weiterhin verboten: leerer body OHNE media.
	if _, err := sqlDB.Exec(`INSERT INTO messages (conversation_id, sender_id, body) VALUES (1, 1, '')`); err == nil {
		t.Error("erwartet CHECK-Verletzung für leeren body ohne media_id")
	}

	// Down: media_id + media weg, Zeilen erhalten (leerer body → Platzhalter).
	if err := m.Migrate(27); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate down to 27: %v", err)
	}
	if hasColumn(t, sqlDB, "messages", "media_id") {
		t.Error("messages.media_id sollte nach 028 down weg sein")
	}
	if tableExists(t, sqlDB, "media") {
		t.Error("media sollte nach 028 down weg sein")
	}
	var afterDown int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM messages WHERE id=1`).Scan(&afterDown)
	if afterDown != 1 {
		t.Errorf("message 1 sollte nach down erhalten sein, COUNT=%d", afterDown)
	}
	// Platzhalter für die zuvor eingefügte reine Bildnachricht.
	var placeholders int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM messages WHERE body='[Bild]'`).Scan(&placeholders)
	if placeholders < 1 {
		t.Errorf("erwartet Platzhalter '[Bild]' für reine Bildzeile nach down, bekam %d", placeholders)
	}
}

// TC: Sanity — 028 up unmittelbar nach voller Migration lässt sich sauber
// nach 27 und wieder nach 28 bewegen (Down/Up-Idempotenz des Rebuilds).
func TestMigration028_DownUp_Roundtrip(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(28); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 28: %v", err)
	}
	if err := m.Migrate(27); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("down to 27: %v", err)
	}
	if err := m.Migrate(28); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("up to 28 again: %v", err)
	}
	if !hasColumn(t, sqlDB, "messages", "media_id") {
		t.Error("messages.media_id fehlt nach Roundtrip")
	}
	// idx_messages_conv muss nach dem Rebuild wieder existieren.
	var idx int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_messages_conv'`).Scan(&idx)
	if idx != 1 {
		t.Errorf("idx_messages_conv fehlt nach Rebuild, COUNT=%d", idx)
	}
}
