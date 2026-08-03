package members_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Geburtsdatum im Kontakt-Tab: Self-Service-Kopie auf users.date_of_birth,
// analog zu street/zip/city (siehe openspec/changes/profil-kontakt-geburtsdatum).

// Happy-Path: PUT /api/profile/me übernimmt date_of_birth sofort in users.
func TestUpdateProfile_DateOfBirth_PersistsChange(t *testing.T) {
	database := testutil.NewDB(t)
	userID := testutil.CreateUser(t, database, "standard")
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, "/api/profile/me",
		testutil.Token(t, userID, "standard", nil),
		map[string]string{"first_name": "Klara", "last_name": "Muster", "date_of_birth": "1990-05-12"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var dob *string
	if err := database.QueryRow(`SELECT date_of_birth FROM users WHERE id=?`, userID).Scan(&dob); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if dob == nil || *dob != "1990-05-12" {
		t.Errorf("expected users.date_of_birth='1990-05-12', got %v", dob)
	}
}

// Fehlerfall: ohne gültigen Bearer-Token → 401, kein Write.
func TestUpdateProfile_DateOfBirth_Unauthenticated401(t *testing.T) {
	database := testutil.NewDB(t)
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, "/api/profile/me", "",
		map[string]string{"date_of_birth": "1990-05-12"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

// GET /api/profile/me liefert sowohl den Self-Service-Wert (date_of_birth)
// als auch das Mitglieder-Record-Datum (own_member.date_of_birth) — Grundlage
// für den Frontend-Default-Fallback.
func TestGetProfile_IncludesDateOfBirth(t *testing.T) {
	database := testutil.NewDB(t)
	userID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, userID)
	database.Exec(`UPDATE members SET date_of_birth='2008-04-15' WHERE id=?`, memberID)
	database.Exec(`UPDATE users SET date_of_birth='2008-04-16' WHERE id=?`, userID)
	srv := newMembersServer(t, database)

	res := testutil.Get(t, srv, "/api/profile/me", testutil.Token(t, userID, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		DateOfBirth string `json:"date_of_birth"`
		OwnMember   struct {
			DateOfBirth string `json:"date_of_birth"`
		} `json:"own_member"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DateOfBirth != "2008-04-16" {
		t.Errorf("expected top-level date_of_birth (users) = '2008-04-16', got %q", body.DateOfBirth)
	}
	if body.OwnMember.DateOfBirth == "" {
		t.Error("expected own_member.date_of_birth to be set")
	}
}

// Happy-Path: PUT /api/profile/kind/{id}/account übernimmt date_of_birth sofort
// in users des Kindes, wenn das Kind einen eigenen Account hat.
func TestUpdateChildAccount_DateOfBirth_PersistsChange(t *testing.T) {
	database := testutil.NewDB(t)
	parentID := testutil.CreateUser(t, database, "standard")
	childUserID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, childUserID)
	if _, err := database.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentID, memberID); err != nil {
		t.Fatalf("family_link: %v", err)
	}
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/profile/kind/%d/account", memberID),
		testutil.Token(t, parentID, "standard", nil),
		map[string]string{"first_name": "Max", "last_name": "Muster", "date_of_birth": "2016-03-01"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	var dob *string
	if err := database.QueryRow(`SELECT date_of_birth FROM users WHERE id=?`, childUserID).Scan(&dob); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if dob == nil || *dob != "2016-03-01" {
		t.Errorf("expected users.date_of_birth='2016-03-01' for child, got %v", dob)
	}
}

// Fehlerfall: kein family_links-Eintrag zwischen Aufrufer und Kind → 403.
func TestUpdateChildAccount_NoFamilyLink403(t *testing.T) {
	database := testutil.NewDB(t)
	strangerID := testutil.CreateUser(t, database, "standard")
	childUserID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, childUserID)
	srv := newMembersServer(t, database)

	res := testutil.Do(t, srv, http.MethodPut, fmt.Sprintf("/api/profile/kind/%d/account", memberID),
		testutil.Token(t, strangerID, "standard", nil),
		map[string]string{"date_of_birth": "2016-03-01"})
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

// GET /api/profile/kind/{id} liefert date_of_birth im user_contact-Objekt,
// wenn das Kind einen eigenen Account hat.
func TestGetChildProfile_IncludesDateOfBirthInUserContact(t *testing.T) {
	database := testutil.NewDB(t)
	parentID := testutil.CreateUser(t, database, "standard")
	childUserID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, childUserID)
	database.Exec(`UPDATE users SET date_of_birth='2016-03-02' WHERE id=?`, childUserID)
	if _, err := database.Exec(
		`INSERT INTO family_links (parent_user_id, member_id) VALUES (?, ?)`, parentID, memberID); err != nil {
		t.Fatalf("family_link: %v", err)
	}
	srv := newMembersServer(t, database)

	res := testutil.Get(t, srv, fmt.Sprintf("/api/profile/kind/%d", memberID), testutil.Token(t, parentID, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		UserContact struct {
			DateOfBirth string `json:"date_of_birth"`
		} `json:"user_contact"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.UserContact.DateOfBirth != "2016-03-02" {
		t.Errorf("expected user_contact.date_of_birth='2016-03-02', got %q", body.UserContact.DateOfBirth)
	}
}

// Änderungsantrag (field_name=profil) trägt date_of_birth; Annehmen übernimmt
// den Wert auf members.date_of_birth — identisch zum Adress-Verhalten.
func TestProfilDraft_UebernimmtDateOfBirth(t *testing.T) {
	database := testutil.NewDB(t)
	userID := testutil.CreateUser(t, database, "standard")
	memberID := testutil.CreateMember(t, database, userID)
	database.Exec(`UPDATE members SET date_of_birth='2008-04-15' WHERE id=?`, memberID)
	h := members.NewHandler(database, hub.NewHub())

	newValue, _ := json.Marshal(map[string]string{
		"first_name":    "Klara",
		"last_name":     "Muster",
		"street":        "Musterstr. 1",
		"zip":           "70173",
		"city":          "Stuttgart",
		"date_of_birth": "2008-04-16",
	})
	draft, err := h.CreateOrUpdateDraft(memberID, userID, members.ChangeRequest{
		FieldName: "profil",
		NewValue:  json.RawMessage(newValue),
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateDraft: %v", err)
	}
	if err := h.AcceptDraft(draft.ID); err != nil {
		t.Fatalf("AcceptDraft: %v", err)
	}

	// members.date_of_birth ist als DATE deklariert — SQLite normalisiert auf
	// ISO-Timestamp (siehe Gotcha „SQLite DATE-Felder"), daher Vergleich nur auf
	// den Datumsanteil.
	var dob string
	if err := database.QueryRow(`SELECT substr(date_of_birth,1,10) FROM members WHERE id=?`, memberID).Scan(&dob); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if dob != "2008-04-16" {
		t.Errorf("expected members.date_of_birth='2008-04-16' after accept, got %q", dob)
	}
}
