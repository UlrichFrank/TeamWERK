package members_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// importReport spiegelt den ImportReport für die JSON-Dekodierung in Tests.
type importReport struct {
	Total     int `json:"total"`
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Skipped   int `json:"skipped"`
	Errors    int `json:"errors"`
	NotFound  int `json:"not_found"`
	Rows      []struct {
		Line    int      `json:"line"`
		Status  string   `json:"status"`
		Name    string   `json:"name"`
		Changes []string `json:"changes"`
	} `json:"rows"`
}

func decodeReport(t *testing.T, res *http.Response) importReport {
	t.Helper()
	var rep importReport
	if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	return rep
}

// gültige Test-IBAN (MOD-97 = 1, 22 Zeichen DE)
const testIBAN = "DE89370400440532013000"

// statusOf liefert den status eines Mitglieds anhand des Vornamens.
func statusOf(t *testing.T, db *sql.DB, firstName string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(`SELECT status FROM members WHERE first_name=?`, firstName).Scan(&s); err != nil {
		t.Fatalf("query status: %v", err)
	}
	return s
}

// postImport lädt eine CSV als multipart/form-data an POST /api/members/import hoch.
func postImport(t *testing.T, srv string, token, csv, mode string) *http.Response {
	return postImportOpts(t, srv, token, csv, mode, nil)
}

// postImportOpts wie postImport, aber mit zusätzlichen Formfeldern (z.B. "fields",
// "apply_lines", "preview").
func postImportOpts(t *testing.T, srv string, token, csv, mode string, extra map[string]string) *http.Response {
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
	for k, v := range extra {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatalf("WriteField %s: %v", k, err)
		}
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

// memberNumberOf liefert (member_number, existiert). Der bool bedeutet, ob das
// Mitglied existiert — NICHT, ob die Nummer gesetzt ist (NULL → "" mit true).
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
	return n.String, true
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

// Realdaten-nah: Bestandsmitglied MIT Geburtsdatum, CSV mit "geboren am".
// Reproduziert den Fall, in dem enrich die Nummer NICHT füllt, weil der
// Datums-Match (COALESCE(date_of_birth,”)=?) scheitert.
func TestImport_EnrichFillsMemberNumber_WithDOB(t *testing.T) {
	cases := []struct {
		name       string
		dbDOB      string // wie date_of_birth in der DB steht
		csvGeboren string // "geboren am"-Wert in der CSV
	}{
		{"iso_date", "2007-10-14", "14.10.2007"},
		{"iso_timestamp", "2007-10-14T00:00:00Z", "14.10.2007"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			srv := newMembersServer(t, db)
			token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
			if _, err := db.Exec(
				`INSERT INTO members (first_name, last_name, status, date_of_birth) VALUES ('Petra','Test','aktiv',?)`,
				tc.dbDOB); err != nil {
				t.Fatalf("insert member: %v", err)
			}

			csv := "Vorname;Name;Mitgliedsnummer;geboren am;Status\n" +
				"Petra;Test;TS-0001;" + tc.csvGeboren + ";aktiv\n"
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
				t.Errorf("enrich (dbDOB=%q) füllte member_number nicht: got %q want %q", tc.dbDOB, num, "TS-0001")
			}
		})
	}
}

// Realfall "Götz" (Mitglied id 25): DB-Geburtsdatum 1967-12-06, CSV liefert
// "geboren am" als 2-stelliges Jahr "06.12.67". Vor dem Fix wurde 67 als 2067
// interpretiert → kein Match → Mitgliedsnummer wurde nicht ergänzt.
func TestImport_EnrichFillsMemberNumber_TwoDigitYear1967(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	if _, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status, date_of_birth) VALUES ('Götz-Bernhard','Haase','passiv','1967-12-06')`); err != nil {
		t.Fatalf("insert member: %v", err)
	}

	csv := "Name,Vorname,Mitgliedsnummer,geboren am,Status\n" +
		"Haase,Götz-Bernhard,61,06.12.67,passiv\n"
	res := postImport(t, srv.URL, token, csv, "enrich")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	num, ok := memberNumberOf(t, db, "Götz-Bernhard")
	if !ok {
		t.Fatal("Mitglied Götz-Bernhard fehlt")
	}
	if num != "61" {
		t.Errorf("enrich füllte member_number nicht: got %q want %q", num, "61")
	}
}

// Fall B (Realfall Janosch Jäger, id 97): Bestandsmitglied OHNE Geburtsdatum,
// CSV liefert eines. Bei eindeutigem Namen matcht enrich über den Namen und
// füllt Mitgliedsnummer (und Geburtsdatum).
func TestImport_EnrichFillsMemberNumber_DBohneGeburtsdatum(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	// Janosch ohne dob; zusätzlich ein gleichnamiger Nachname (Felix) MIT dob,
	// um zu zeigen, dass der Vorname trotzdem eindeutig trennt.
	if _, err := db.Exec(`INSERT INTO members (first_name, last_name, status) VALUES ('Janosch','Jäger','aktiv')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO members (first_name, last_name, status, date_of_birth) VALUES ('Felix','Jäger','aktiv','2011-07-05')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	csv := "Name,Vorname,Mitgliedsnummer,geboren am,Status\n" +
		"Jäger,Janosch,259,23.11.2015,aktiv\n"
	res := postImport(t, srv.URL, token, csv, "enrich")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	num, ok := memberNumberOf(t, db, "Janosch")
	if !ok {
		t.Fatal("Janosch fehlt")
	}
	if num != "259" {
		t.Errorf("enrich füllte member_number nicht: got %q want %q", num, "259")
	}
	// Geburtsdatum wurde mitgefüllt (Datumsanteil; Speicherung kann Timestamp sein).
	var dob sql.NullString
	db.QueryRow(`SELECT substr(date_of_birth,1,10) FROM members WHERE first_name='Janosch'`).Scan(&dob)
	if dob.String != "2015-11-23" {
		t.Errorf("Geburtsdatum nicht ergänzt: got %q want %q", dob.String, "2015-11-23")
	}
}

// Eindeutigkeits-Schutz: zwei gleichnamige Bestandsmitglieder OHNE Geburtsdatum
// → kein Fall-B-Match, Fehler statt willkürlicher Befüllung.
func TestImport_EnrichAmbiguousNoDOB_NichtBefuellt(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	for range 2 {
		if _, err := db.Exec(`INSERT INTO members (first_name, last_name, status) VALUES ('Max','Muster','aktiv')`); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	csv := "Name,Vorname,Mitgliedsnummer,geboren am,Status\n" +
		"Muster,Max,900,01.01.10,aktiv\n"
	res := postImport(t, srv.URL, token, csv, "enrich")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	var withNum int
	db.QueryRow(`SELECT COUNT(*) FROM members WHERE first_name='Max' AND member_number IS NOT NULL`).Scan(&withNum)
	if withNum != 0 {
		t.Errorf("mehrdeutiger Fall sollte niemanden befüllen, befüllt: %d", withNum)
	}
}

// Feld-Whitelist: fields=iban aktualisiert nur die IBAN, nicht den Status.
func TestImport_FieldsWhitelist_NurIBAN(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test") // status aktiv, iban leer

	csv := "Vorname;Name;Status;IBAN\nPetra;Test;pausiert;" + testIBAN + "\n"
	res := postImportOpts(t, srv.URL, token, csv, "update", map[string]string{"fields": "iban"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	var iban sql.NullString
	db.QueryRow(`SELECT iban FROM members WHERE first_name='Petra'`).Scan(&iban)
	if iban.String != testIBAN {
		t.Errorf("iban = %q, want %q", iban.String, testIBAN)
	}
	if got := statusOf(t, db, "Petra"); got != "aktiv" {
		t.Errorf("status = %q, want unverändert %q (nicht in fields)", got, "aktiv")
	}
}

// Feld-Whitelist: ohne "status" in fields bleibt das abgeleitete beitragsfrei unverändert.
func TestImport_FieldsWhitelist_StatusSteuertBeitragsfrei(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test") // status aktiv, beitragsfrei 0

	// CSV-Status "beitragsfrei" würde beitragsfrei=1 ableiten — darf ohne "status" in fields nicht greifen.
	csv := "Vorname;Name;Status;IBAN\nPetra;Test;beitragsfrei;" + testIBAN + "\n"
	res := postImportOpts(t, srv.URL, token, csv, "update", map[string]string{"fields": "iban"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	var beitragsfrei int
	db.QueryRow(`SELECT COALESCE(beitragsfrei,0) FROM members WHERE first_name='Petra'`).Scan(&beitragsfrei)
	if beitragsfrei != 0 {
		t.Errorf("beitragsfrei = %d, want 0 (status nicht in fields)", beitragsfrei)
	}
	if got := statusOf(t, db, "Petra"); got != "aktiv" {
		t.Errorf("status = %q, want unverändert %q", got, "aktiv")
	}
}

// Regression: ohne fields werden wie bisher alle abweichenden Felder übernommen.
func TestImport_LeeresFields_AlleFelder(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test")

	csv := "Vorname;Name;Status\nPetra;Test;pausiert\n"
	res := postImport(t, srv.URL, token, csv, "update")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}
	if got := statusOf(t, db, "Petra"); got != "pausiert" {
		t.Errorf("status = %q, want %q (ohne fields alle Felder)", got, "pausiert")
	}
}

// Feld-Whitelist gilt nur für Bestandsmitglieder: ein neu angelegtes Mitglied
// bekommt trotz fields=iban alle CSV-Felder.
func TestImport_FieldsWhitelist_NeuesMitgliedAllFelder(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)

	csv := "Vorname;Name;Status;Passnummer\nMax;Neu;pausiert;P123\n"
	res := postImportOpts(t, srv.URL, token, csv, "update", map[string]string{"fields": "iban"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	var status string
	var pass sql.NullString
	if err := db.QueryRow(`SELECT status, pass_number FROM members WHERE first_name='Max'`).Scan(&status, &pass); err != nil {
		t.Fatalf("Mitglied Max nicht angelegt: %v", err)
	}
	if status != "pausiert" {
		t.Errorf("neues Mitglied: status = %q, want %q (Whitelist gilt nicht beim Anlegen)", status, "pausiert")
	}
	if pass.String != "P123" {
		t.Errorf("neues Mitglied: pass_number = %q, want %q", pass.String, "P123")
	}
}

// apply_lines: nur die ausgewählte Zeile wird geschrieben.
func TestImport_ApplyLines_NurAusgewaehlteZeile(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test") // Zeile 2
	insertBareMember(t, db, "Hans", "Meier") // Zeile 3

	csv := "Vorname;Name;Status\nPetra;Test;pausiert\nHans;Meier;verletzt\n"
	res := postImportOpts(t, srv.URL, token, csv, "update", map[string]string{"apply_lines": "2"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	if got := statusOf(t, db, "Petra"); got != "pausiert" {
		t.Errorf("Petra (Zeile 2, ausgewählt): status = %q, want %q", got, "pausiert")
	}
	if got := statusOf(t, db, "Hans"); got != "aktiv" {
		t.Errorf("Hans (Zeile 3, nicht ausgewählt): status = %q, want unverändert %q", got, "aktiv")
	}
}

// apply_lines: abgewählte Zeile mit Änderungen wird als skipped gemeldet, nicht geschrieben.
func TestImport_ApplyLines_AbgewaehltIstSkipped(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test") // Zeile 2

	// apply_lines verweist auf eine nicht vorhandene Zeile → Zeile 2 ist abgewählt.
	csv := "Vorname;Name;Status\nPetra;Test;pausiert\n"
	res := postImportOpts(t, srv.URL, token, csv, "update", map[string]string{"apply_lines": "999"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	rep := decodeReport(t, res)
	if rep.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", rep.Skipped)
	}
	if rep.Updated != 0 {
		t.Errorf("updated = %d, want 0", rep.Updated)
	}
	if len(rep.Rows) != 1 || rep.Rows[0].Status != "skipped" {
		t.Errorf("row status = %+v, want skipped", rep.Rows)
	}
	if got := statusOf(t, db, "Petra"); got != "aktiv" {
		t.Errorf("status = %q, want unverändert %q", got, "aktiv")
	}
}

// apply_lines wird im Dry-Run ignoriert: alle Zeilen werden als updated gemeldet, nichts geschrieben.
func TestImport_ApplyLines_DryRunIgnoriert(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test")

	csv := "Vorname;Name;Status\nPetra;Test;pausiert\n"
	res := postImportOpts(t, srv.URL, token, csv, "preview", map[string]string{"apply_lines": "999"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	rep := decodeReport(t, res)
	if rep.Updated != 1 {
		t.Errorf("updated = %d, want 1 (Dry-Run ignoriert apply_lines)", rep.Updated)
	}
	if rep.Skipped != 0 {
		t.Errorf("skipped = %d, want 0 im Dry-Run", rep.Skipped)
	}
	if got := statusOf(t, db, "Petra"); got != "aktiv" {
		t.Errorf("Dry-Run schrieb status = %q, want unverändert %q", got, "aktiv")
	}
}
