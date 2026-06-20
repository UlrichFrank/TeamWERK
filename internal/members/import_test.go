package members_test

import (
	"bytes"
	"database/sql"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// postImport lädt eine CSV als multipart/form-data an POST /api/members/import hoch.
func postImport(t *testing.T, srv string, token, csv, mode string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "members.csv")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write([]byte(csv)); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	if err := mw.WriteField("mode", mode); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, srv+"/api/members/import", &buf)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", token) // token enthält bereits "Bearer "
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("import request: %v", err)
	}
	return res
}

func memberNumberOf(t *testing.T, db *sql.DB, firstName string) (string, bool) {
	t.Helper()
	var n sql.NullString
	err := db.QueryRow(`SELECT member_number FROM members WHERE first_name=?`, firstName).Scan(&n)
	if err == sql.ErrNoRows {
		return "", false
	}
	if err != nil {
		t.Fatalf("query member_number: %v", err)
	}
	return n.String, n.Valid
}

// Regressionstest: Beim CSV-Import eines NEUEN Mitglieds (append) muss die
// Mitgliedsnummer aus der Spalte "Mitgliedsnummer" persistiert werden.
func TestImport_SetsMemberNumber(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	csv := "Vorname;Name;Mitgliedsnummer;Status\n" +
		"Petra;Test;TS-0001;aktiv\n"

	res := postImport(t, srv.URL, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	num, ok := memberNumberOf(t, db, "Petra")
	if !ok {
		t.Fatal("Mitglied Petra wurde nicht angelegt")
	}
	if num != "TS-0001" {
		t.Errorf("member_number = %q, want %q", num, "TS-0001")
	}
}

// insertBareMember legt ein Bestandsmitglied OHNE Mitgliedsnummer an.
func insertBareMember(t *testing.T, db *sql.DB, first, last string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status) VALUES (?,?, 'aktiv')`,
		first, last); err != nil {
		t.Fatalf("insertBareMember: %v", err)
	}
}

// Dokumentiert die Falle: Im Standard-Modus "append" bleiben BESTANDS-Mitglieder
// unangetastet — eine per CSV gelieferte Mitgliedsnummer wird NICHT nachgezogen.
func TestImport_AppendLeavesExistingMemberNumberUnset(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test")

	csv := "Vorname;Name;Mitgliedsnummer;Status\nPetra;Test;TS-0001;aktiv\n"
	res := postImport(t, srv.URL, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	num, _ := memberNumberOf(t, db, "Petra")
	if num != "" {
		t.Errorf("append sollte Bestandsmitglied nicht anfassen, member_number=%q", num)
	}
}

// Der korrekte Weg, Bestandsmitglieder mit Nummern zu versorgen: "enrich"
// füllt leere Felder, ohne vorhandene Werte zu überschreiben.
func TestImport_EnrichFillsEmptyMemberNumber(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test")

	csv := "Vorname;Name;Mitgliedsnummer;Status\nPetra;Test;TS-0001;aktiv\n"
	res := postImport(t, srv.URL, token, csv, "enrich")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	num, ok := memberNumberOf(t, db, "Petra")
	if !ok {
		t.Fatal("Mitglied Petra fehlt")
	}
	if num != "TS-0001" {
		t.Errorf("enrich sollte leere member_number füllen, got %q want %q", num, "TS-0001")
	}
}
