package members_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// draftCountFor returns how many change-drafts exist for a member and field.
func draftCountFor(t *testing.T, db *sql.DB, memberID int, field string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM member_change_drafts WHERE member_id=? AND field_name=?`,
		memberID, field).Scan(&n); err != nil {
		t.Fatalf("draftCountFor: %v", err)
	}
	return n
}

// B-1: GET /api/members/{id}/change-drafts erzwingt Mitglieds-Ownership.
func TestGetChangeDrafts_OwnershipGate(t *testing.T) {
	db := testutil.NewDB(t)
	ownerID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, ownerID)
	srv := prodserver.New(t, db)
	path := fmt.Sprintf("/api/members/%d/change-drafts", memberID)

	// Eigentümer → 200
	res := testutil.Get(t, srv, path, testutil.Token(t, ownerID, "standard", nil))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Eigentümer: erwartet 200, bekam %d", res.StatusCode)
	}

	// Fremder Spieler → 403
	strangerID := testutil.CreateUser(t, db, "standard")
	res = testutil.Get(t, srv, path, testutil.Token(t, strangerID, "standard", []string{"spieler"}))
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("fremder Spieler: erwartet 403, bekam %d", res.StatusCode)
	}

	// Vorstand → 200 (darf alle Mitglieder lesen)
	vorstandID := testutil.CreateUser(t, db, "standard")
	res = testutil.Get(t, srv, path, testutil.Token(t, vorstandID, "standard", []string{"vorstand"}))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("Vorstand: erwartet 200, bekam %d", res.StatusCode)
	}
}

// B-1: Elternteil liest die Anträge des eigenen Kindes (nicht 403).
func TestGetChangeDrafts_ParentReadsChild(t *testing.T) {
	db := testutil.NewDB(t)
	parentID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`,
		parentID, childMemberID); err != nil {
		t.Fatalf("family_link: %v", err)
	}
	srv := prodserver.New(t, db)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d/change-drafts", childMemberID),
		testutil.Token(t, parentID, "standard", nil))
	res.Body.Close()
	if res.StatusCode == http.StatusForbidden {
		t.Errorf("Elternteil auf eigenes Kind: unerwartet 403")
	}
}

// B-1: POST /api/members/{id}/change-request erzwingt Ownership (Nicht-Bankfeld).
func TestCreateChangeRequest_OwnershipGate(t *testing.T) {
	db := testutil.NewDB(t)
	ownerID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, ownerID)
	srv := prodserver.New(t, db)
	path := fmt.Sprintf("/api/members/%d/change-request", memberID)
	body := map[string]any{
		"field_name": "name",
		"new_value":  map[string]string{"first_name": "Neu", "last_name": "Name"},
	}

	// Eigentümer → 201
	res := testutil.Post(t, srv, path, testutil.Token(t, ownerID, "standard", nil), body)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Errorf("Eigentümer: erwartet 201, bekam %d", res.StatusCode)
	}

	// Fremder Spieler → 403, kein Draft
	strangerID := testutil.CreateUser(t, db, "standard")
	res = testutil.Post(t, srv, path, testutil.Token(t, strangerID, "standard", []string{"spieler"}), body)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("fremder Spieler: erwartet 403, bekam %d", res.StatusCode)
	}
}

// B-3: Bankdaten-Antrag nur durch Eigentümer/Eltern — fremde dürfen keinen Envelope unterschieben.
func TestCreateChangeRequest_BankdatenOwnerOnly(t *testing.T) {
	db := testutil.NewDB(t)
	ownerID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, ownerID)
	srv := prodserver.New(t, db)
	path := fmt.Sprintf("/api/members/%d/change-request", memberID)
	bankBody := map[string]any{
		"field_name": "bankdaten",
		"new_value":  map[string]string{"bank_ciphertext": testCiphertext, "bank_dek_enc": testDekEnc},
	}

	// Eigentümer → 201
	res := testutil.Post(t, srv, path, testutil.Token(t, ownerID, "standard", nil), bankBody)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Errorf("Eigentümer: erwartet 201, bekam %d", res.StatusCode)
	}
	if n := draftCountFor(t, db, memberID, "bankdaten"); n != 1 {
		t.Errorf("nach Eigentümer-Antrag: erwartet 1 bankdaten-Draft, bekam %d", n)
	}

	// Fremder Spieler → 403, kein neuer/überschriebener Draft
	strangerID := testutil.CreateUser(t, db, "standard")
	res = testutil.Post(t, srv, path, testutil.Token(t, strangerID, "standard", []string{"spieler"}), bankBody)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("fremder Spieler: erwartet 403, bekam %d", res.StatusCode)
	}

	// Kassierer → 403 (Korrektur läuft über bank-details, nicht den Antragsweg)
	kassiererID := testutil.CreateUser(t, db, "standard")
	res = testutil.Post(t, srv, path, testutil.Token(t, kassiererID, "standard", []string{"kassierer"}), bankBody)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("Kassierer: erwartet 403, bekam %d", res.StatusCode)
	}

	// Es existiert weiterhin genau der eine (Eigentümer-)Draft.
	if n := draftCountFor(t, db, memberID, "bankdaten"); n != 1 {
		t.Errorf("bankdaten-Draft wurde durch fremde Anträge verändert: %d Drafts", n)
	}
}

// B-1: Ein bestehender legitimer pending-Antrag wird durch einen fremden POST nicht verdrängt.
func TestCreateChangeRequest_ForeignDoesNotDisplaceDraft(t *testing.T) {
	db := testutil.NewDB(t)
	ownerID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, ownerID)
	srv := prodserver.New(t, db)
	path := fmt.Sprintf("/api/members/%d/change-request", memberID)

	// Eigentümer legt legitimen Antrag an
	ownerBody := map[string]any{
		"field_name": "name",
		"new_value":  map[string]string{"first_name": "Echt", "last_name": "Owner"},
	}
	res := testutil.Post(t, srv, path, testutil.Token(t, ownerID, "standard", nil), ownerBody)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Eigentümer-Antrag: erwartet 201, bekam %d", res.StatusCode)
	}

	// Fremder versucht zu verdrängen
	strangerID := testutil.CreateUser(t, db, "standard")
	strangerBody := map[string]any{
		"field_name": "name",
		"new_value":  map[string]string{"first_name": "Boese", "last_name": "Stranger"},
	}
	res = testutil.Post(t, srv, path, testutil.Token(t, strangerID, "standard", []string{"spieler"}), strangerBody)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("fremder Verdränger: erwartet 403, bekam %d", res.StatusCode)
	}

	// Der Draft trägt weiterhin den Wert des Eigentümers.
	var newValue string
	if err := db.QueryRow(
		`SELECT new_value FROM member_change_drafts WHERE member_id=? AND field_name='name'`,
		memberID).Scan(&newValue); err != nil {
		t.Fatalf("Draft lesen: %v", err)
	}
	if !strings.Contains(newValue, "Echt") || strings.Contains(newValue, "Boese") {
		t.Errorf("Draft wurde verdrängt: %q", newValue)
	}
}
