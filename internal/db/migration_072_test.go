package db_test

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// TestMigration072_ConversationSources prüft die Herkunftstabelle der
// Chat-Gruppen (chat-gruppe-aktualisieren): kein Backfill für Bestandsgruppen,
// PK dedupliziert, CHECKs greifen, Löschen der Konversation räumt mit auf.
func TestMigration072_ConversationSources(t *testing.T) {
	sqlDB, m := newMigrator(t)
	if err := m.Migrate(71); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 71: %v", err)
	}
	if _, err := sqlDB.Exec(`
		INSERT INTO users (id, email, login_name, first_name, last_name, can_login) VALUES (1, 'a@b', 'a', 'A', 'B', 1);
		INSERT INTO conversations (id, type, name, created_by) VALUES (1, 'group', 'G', 1);
	`); err != nil {
		t.Fatalf("seed Bestand: %v", err)
	}
	if err := m.Migrate(72); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 72: %v", err)
	}

	var n int
	sqlDB.QueryRow(`SELECT COUNT(*) FROM conversation_sources`).Scan(&n)
	if n != 0 {
		t.Fatalf("Bestandsgruppe hat %d Herkunftszeilen, erwartet 0 (kein Backfill)", n)
	}

	if _, err := sqlDB.Exec(`INSERT INTO conversation_sources (conversation_id, group_type, ref_id, kind) VALUES (1, 'team', 5, 'spieler')`); err != nil {
		t.Fatalf("gültige Zeile: %v", err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO conversation_sources (conversation_id, group_type, ref_id, kind) VALUES (1, 'team', 5, 'spieler')`); err == nil {
		t.Error("doppeltes Tripel sollte am PK scheitern")
	}
	if _, err := sqlDB.Exec(`INSERT INTO conversation_sources (conversation_id, group_type, ref_id, kind) VALUES (1, 'team', 5, 'foobar')`); err == nil {
		t.Error("unbekannter kind sollte am CHECK scheitern")
	}
	if _, err := sqlDB.Exec(`INSERT INTO conversation_sources (conversation_id, group_type, ref_id, kind) VALUES (1, 'verein', 5, 'spieler')`); err == nil {
		t.Error("unbekannter group_type sollte am CHECK scheitern")
	}

	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("foreign_keys on: %v", err)
	}
	if _, err := sqlDB.Exec(`DELETE FROM conversations WHERE id = 1`); err != nil {
		t.Fatalf("delete conversation: %v", err)
	}
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatalf("foreign_keys off: %v", err)
	}
	sqlDB.QueryRow(`SELECT COUNT(*) FROM conversation_sources`).Scan(&n)
	if n != 0 {
		t.Errorf("nach Löschen der Konversation %d Herkunftszeilen übrig, erwartet 0", n)
	}

	if err := m.Migrate(71); err != nil {
		t.Fatalf("migrate down to 71: %v", err)
	}
}
