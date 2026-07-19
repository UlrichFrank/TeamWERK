package testutil

import (
	"database/sql"
	"testing"
)

// Fixtures für Mannschafts-Aufgaben (Responsibilities) und -Strafen (Penalties).
// Stil wie fixtures.go: t.Helper(), Exec + t.Fatalf, LastInsertId für Autoincrement-PKs,
// INSERT OR IGNORE für Composite-PK-Junctions.

// AppointStrafenwart trägt einen Member als Strafenwart eines Kaders ein (kader_strafenwarte).
func AppointStrafenwart(t *testing.T, database *sql.DB, kaderID, memberID int) {
	t.Helper()
	_, err := database.Exec(
		`INSERT OR IGNORE INTO kader_strafenwarte (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID)
	if err != nil {
		t.Fatalf("AppointStrafenwart: %v", err)
	}
}

// AppointKassenwart trägt einen Member als Kassenwart eines Kaders ein (kader_kassenwarte).
func AppointKassenwart(t *testing.T, database *sql.DB, kaderID, memberID int) {
	t.Helper()
	_, err := database.Exec(
		`INSERT OR IGNORE INTO kader_kassenwarte (kader_id, member_id) VALUES (?, ?)`,
		kaderID, memberID)
	if err != nil {
		t.Fatalf("AppointKassenwart: %v", err)
	}
}

// CreateCashbookEntry legt eine Kassenbuchung an (amountCent signed) und gibt die ID zurück.
// enteredByMemberID=0 → NULL (entered_by_member_id).
func CreateCashbookEntry(t *testing.T, database *sql.DB, kaderID, memberID, amountCent int, note string) int {
	t.Helper()
	var byArg any
	if memberID > 0 {
		byArg = memberID
	}
	res, err := database.Exec(
		`INSERT INTO team_cashbook_entries (kader_id, amount_cent, note, entered_by_member_id) VALUES (?, ?, ?, ?)`,
		kaderID, amountCent, note, byArg)
	if err != nil {
		t.Fatalf("CreateCashbookEntry: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// SetPenaltyUnit setzt die Strafen-Einheit eines Kaders (penalty_settings, Upsert).
func SetPenaltyUnit(t *testing.T, database *sql.DB, kaderID int, unit string) {
	t.Helper()
	_, err := database.Exec(`
		INSERT INTO penalty_settings (kader_id, unit) VALUES (?, ?)
		ON CONFLICT(kader_id) DO UPDATE SET unit = excluded.unit`, kaderID, unit)
	if err != nil {
		t.Fatalf("SetPenaltyUnit: %v", err)
	}
}

// AddResponsibilityType legt einen Aufgaben-Catalog-Eintrag für einen Kader an und gibt die ID zurück.
func AddResponsibilityType(t *testing.T, database *sql.DB, kaderID int, label string) int {
	t.Helper()
	res, err := database.Exec(
		`INSERT INTO responsibility_types (kader_id, label) VALUES (?, ?)`,
		kaderID, label)
	if err != nil {
		t.Fatalf("AddResponsibilityType: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// AssignResponsibility weist einem Member eine Aufgabe (Snapshot-Label) zu und gibt die ID zurück.
func AssignResponsibility(t *testing.T, database *sql.DB, kaderID, memberID int, label string) int {
	t.Helper()
	res, err := database.Exec(
		`INSERT INTO member_responsibilities (kader_id, member_id, label) VALUES (?, ?, ?)`,
		kaderID, memberID, label)
	if err != nil {
		t.Fatalf("AssignResponsibility: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// AddPenaltyType legt einen Strafen-Catalog-Eintrag (Grund + Default-Betrag in Cent) an und gibt die ID zurück.
func AddPenaltyType(t *testing.T, database *sql.DB, kaderID int, reason string, defaultAmountCent int) int {
	t.Helper()
	res, err := database.Exec(
		`INSERT INTO penalty_types (kader_id, reason, default_amount_cent) VALUES (?, ?, ?)`,
		kaderID, reason, defaultAmountCent)
	if err != nil {
		t.Fatalf("AddPenaltyType: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// CreatePenalty vergibt eine Strafe (amount_cent in Cent) und gibt die ID zurück.
// createdByMemberID=0 → NULL (created_by_member_id).
func CreatePenalty(t *testing.T, database *sql.DB, kaderID, memberID, amountCent int, reason string, createdByMemberID int) int {
	t.Helper()
	var byArg any
	if createdByMemberID > 0 {
		byArg = createdByMemberID
	}
	res, err := database.Exec(
		`INSERT INTO team_penalties (kader_id, member_id, amount_cent, reason, created_by_member_id) VALUES (?, ?, ?, ?, ?)`,
		kaderID, memberID, amountCent, reason, byArg)
	if err != nil {
		t.Fatalf("CreatePenalty: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}
