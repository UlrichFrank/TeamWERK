package members_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
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
		Line        int      `json:"line"`
		Status      string   `json:"status"`
		Name        string   `json:"name"`
		Changes     []string `json:"changes"`
		Message     string   `json:"message"`
		DOB         string   `json:"dob"`
		IBANWarning string   `json:"iban_warning"`
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

	csv := "Vorname;Name;Mitgliedsnummer;Status TeamWERK\n" +
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

	csv := "Vorname;Name;Mitgliedsnummer;Status TeamWERK\nPetra;Test;TS-0001;aktiv\n"
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

	csv := "Vorname;Name;Mitgliedsnummer;Status TeamWERK\nPetra;Test;TS-0001;aktiv\n"
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

			csv := "Vorname;Name;Mitgliedsnummer;geboren am;Status TeamWERK\n" +
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

	csv := "Name,Vorname,Mitgliedsnummer,geboren am,Status TeamWERK\n" +
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

	csv := "Name,Vorname,Mitgliedsnummer,geboren am,Status TeamWERK\n" +
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

	csv := "Name,Vorname,Mitgliedsnummer,geboren am,Status TeamWERK\n" +
		"Muster,Max,900,01.01.10,aktiv\n"
	res := postImport(t, srv.URL, token, csv, "enrich")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	// Meldung B positiv pinnen (emptyCnt>=2-Zweig): beide DB-Mitglieder haben leeres
	// date_of_birth, die CSV liefert ein DOB → Zweig 1939-1948, Text "… ohne Geburtsdatum …".
	rep := decodeReport(t, res)
	if rep.Errors != 1 {
		t.Errorf("errors = %d, want 1", rep.Errors)
	}
	if len(rep.Rows) != 1 || !strings.Contains(rep.Rows[0].Message, "ohne Geburtsdatum") {
		t.Errorf("rows[0].Message = %q, want Substring %q", func() string {
			if len(rep.Rows) > 0 {
				return rep.Rows[0].Message
			}
			return "<keine Row>"
		}(), "ohne Geburtsdatum")
	}

	var withNum int
	db.QueryRow(`SELECT COUNT(*) FROM members WHERE first_name='Max' AND member_number IS NOT NULL`).Scan(&withNum)
	if withNum != 0 {
		t.Errorf("mehrdeutiger Fall sollte niemanden befüllen, befüllt: %d", withNum)
	}
}

// Modell B: Bankdaten werden per CSV NICHT importiert (Zero-Knowledge — der Server kann sie
// nicht verschlüsseln). Eine CSV-IBAN landet weder in members.iban noch in member_sensitive;
// die Feld-Whitelist (fields=iban) greift somit auf nichts und lässt den Status unverändert.
func TestImport_CSV_IgnoriertBankdaten(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test") // status aktiv, iban leer

	csv := "Vorname;Name;Status TeamWERK;IBAN\nPetra;Test;pausiert;" + testIBAN + "\n"
	res := postImportOpts(t, srv.URL, token, csv, "update", map[string]string{"fields": "iban"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	var iban sql.NullString
	db.QueryRow(`SELECT iban FROM members WHERE first_name='Petra'`).Scan(&iban)
	if iban.Valid && iban.String != "" {
		t.Errorf("CSV-IBAN gespeichert (%q) — darf nicht (kein Server-Bankimport)", iban.String)
	}
	var cnt int
	db.QueryRow(`SELECT COUNT(*) FROM member_sensitive WHERE member_id=(SELECT id FROM members WHERE first_name='Petra')`).Scan(&cnt)
	if cnt != 0 {
		t.Errorf("member_sensitive angelegt (%d) — CSV importiert keine Bankdaten", cnt)
	}
	if got := statusOf(t, db, "Petra"); got != "aktiv" {
		t.Errorf("status = %q, want unverändert %q (nicht in fields)", got, "aktiv")
	}
}

// Feld-Whitelist: ohne "beitragsfrei" in fields bleibt das Flag unverändert.
// Status und beitragsfrei sind seit Mapping-Umstellung getrennte Whitelist-Einträge.
func TestImport_FieldsWhitelist_BeitragsfreiSeparat(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test") // status aktiv, beitragsfrei 0

	// CSV liefert beitragsfrei=ja — darf ohne "beitragsfrei" in fields nicht greifen.
	csv := "Vorname;Name;beitragsfrei;IBAN\nPetra;Test;ja;" + testIBAN + "\n"
	res := postImportOpts(t, srv.URL, token, csv, "update", map[string]string{"fields": "iban"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	var beitragsfrei int
	db.QueryRow(`SELECT COALESCE(beitragsfrei,0) FROM members WHERE first_name='Petra'`).Scan(&beitragsfrei)
	if beitragsfrei != 0 {
		t.Errorf("beitragsfrei = %d, want 0 (beitragsfrei nicht in fields)", beitragsfrei)
	}
}

// Regression: ohne fields werden wie bisher alle abweichenden Felder übernommen.
func TestImport_LeeresFields_AlleFelder(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test")

	csv := "Vorname;Name;Status TeamWERK\nPetra;Test;pausiert\n"
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

	csv := "Vorname;Name;Status TeamWERK;Passnummer\nMax;Neu;pausiert;P123\n"
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

	csv := "Vorname;Name;Status TeamWERK\nPetra;Test;pausiert\nHans;Meier;verletzt\n"
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
	csv := "Vorname;Name;Status TeamWERK\nPetra;Test;pausiert\n"
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

	csv := "Vorname;Name;Status TeamWERK\nPetra;Test;pausiert\n"
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

// "Status TeamWERK" steuert members.status beim Anlegen neuer Mitglieder.
func TestImport_StatusTeamWERK_AppendNew(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)

	// "Status" (Freitext) wird ignoriert, "Status TeamWERK" entscheidet.
	csv := "Vorname;Name;Status;Status TeamWERK\nPetra;Test;Zweitspielrecht, beitragsfrei;passiv\n"
	res := postImport(t, srv.URL, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}
	if got := statusOf(t, db, "Petra"); got != "passiv" {
		t.Errorf("status = %q, want %q (Status TeamWERK steuert)", got, "passiv")
	}
	var beitragsfrei int
	db.QueryRow(`SELECT COALESCE(beitragsfrei,0) FROM members WHERE first_name='Petra'`).Scan(&beitragsfrei)
	if beitragsfrei != 0 {
		t.Errorf("beitragsfrei = %d, want 0 (Status-Spalte darf nichts mehr ableiten)", beitragsfrei)
	}
}

// CSV-Spalte "beitragsfrei" landet direkt im Flag (Append-Pfad).
func TestImport_BeitragsfreiSpalte_DirectMap(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)

	csv := "Vorname;Name;beitragsfrei\nPetra;Test;ja\n"
	res := postImport(t, srv.URL, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}
	var beitragsfrei int
	db.QueryRow(`SELECT COALESCE(beitragsfrei,0) FROM members WHERE first_name='Petra'`).Scan(&beitragsfrei)
	if beitragsfrei != 1 {
		t.Errorf("beitragsfrei = %d, want 1", beitragsfrei)
	}
}

// CSV-Spalte "Grund für Beitragsfreiheit" wird beim Anlegen übernommen.
func TestImport_BeitragsfreiGrund_Append(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)

	csv := "Vorname;Name;beitragsfrei;Grund für Beitragsfreiheit\nPetra;Test;ja;kein aktiver Sportler mehr\n"
	res := postImport(t, srv.URL, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}
	var grund sql.NullString
	db.QueryRow(`SELECT beitragsfrei_grund FROM members WHERE first_name='Petra'`).Scan(&grund)
	if !grund.Valid || grund.String != "kein aktiver Sportler mehr" {
		t.Errorf("beitragsfrei_grund = %v %q, want %q", grund.Valid, grund.String, "kein aktiver Sportler mehr")
	}
}

// Enrich-Modus überschreibt einen bereits gefüllten Grund nicht.
func TestImport_BeitragsfreiGrund_EnrichLeaves(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	if _, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status, beitragsfrei, beitragsfrei_grund) VALUES ('Petra','Test','aktiv',1,'Zweitspielrecht')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	csv := "Vorname;Name;beitragsfrei;Grund für Beitragsfreiheit\nPetra;Test;ja;kein aktiver Sportler mehr\n"
	res := postImport(t, srv.URL, token, csv, "enrich")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}
	var grund sql.NullString
	db.QueryRow(`SELECT beitragsfrei_grund FROM members WHERE first_name='Petra'`).Scan(&grund)
	if grund.String != "Zweitspielrecht" {
		t.Errorf("beitragsfrei_grund = %q, want unverändert %q", grund.String, "Zweitspielrecht")
	}
}

// Alte "Status"-Spalte wird ersatzlos ignoriert: weder Status noch beitragsfrei werden verändert.
func TestImport_AlteStatusSpalteWirdIgnoriert(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	if _, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status, beitragsfrei) VALUES ('Petra','Test','passiv',0)`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Nur die alte "Status"-Spalte vorhanden — kein "Status TeamWERK", kein "beitragsfrei".
	csv := "Vorname;Name;Status\nPetra;Test;beitragsfrei\n"
	res := postImport(t, srv.URL, token, csv, "update")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}

	if got := statusOf(t, db, "Petra"); got != "passiv" {
		t.Errorf("status = %q, want unverändert %q", got, "passiv")
	}
	var beitragsfrei int
	db.QueryRow(`SELECT COALESCE(beitragsfrei,0) FROM members WHERE first_name='Petra'`).Scan(&beitragsfrei)
	if beitragsfrei != 0 {
		t.Errorf("beitragsfrei = %d, want unverändert 0 (alte Status-Spalte darf nichts ableiten)", beitragsfrei)
	}
}

// "Status TeamWERK=gekündigt" mappt weiterhin auf members.status='ausgetreten'.
func TestImport_GekuendigtBleibtAlias(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	insertBareMember(t, db, "Petra", "Test") // status aktiv

	csv := "Vorname;Name;Status TeamWERK\nPetra;Test;gekündigt\n"
	res := postImport(t, srv.URL, token, csv, "update")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("import status %d, want 200", res.StatusCode)
	}
	if got := statusOf(t, db, "Petra"); got != "ausgetreten" {
		t.Errorf("status = %q, want %q (gekündigt → ausgetreten Alias)", got, "ausgetreten")
	}
}

// ── Charakterisierungstests: nageln das AKTUELLE Verhalten von Import fest ──────
// (test/members-import) Vor dem Refactor. Assertions locken das REALE Verhalten.

// lastNameOf liefert last_name eines Mitglieds anhand des Vornamens.
func lastNameOf(t *testing.T, db *sql.DB, firstName string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(`SELECT last_name FROM members WHERE first_name=?`, firstName).Scan(&s); err != nil {
		t.Fatalf("query last_name: %v", err)
	}
	return s
}

// adminServer richtet DB + Server + Admin-Token in einem Schritt ein.
func adminServer(t *testing.T) (*sql.DB, string, string) {
	t.Helper()
	db := testutil.NewDB(t)
	srv := newMembersServer(t, db)
	token := testutil.Token(t, testutil.CreateUser(t, db, "admin"), "admin", nil)
	return db, srv.URL, token
}

// read400 liest StatusCode + Body eines erwarteten 400-Fehlers (NICHT decoden).
func read400(t *testing.T, res *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}

// postImportNoFile lädt ein multipart/form-data OHNE "file"-Feld hoch (nur "mode").
func postImportNoFile(t *testing.T, srv, token string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("mode", "append"); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	mw.Close()
	req, err := http.NewRequest(http.MethodPost, srv+"/api/members/import", &buf)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("import request: %v", err)
	}
	return res
}

// ── BOM ─────────────────────────────────────────────────────────────────────

func TestImport_StripsUTF8BOM(t *testing.T) {
	db, srv, token := adminServer(t)
	csv := "\xef\xbb\xbfVorname;Name\nMax;Muster\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	if rep := decodeReport(t, res); rep.Created != 1 {
		t.Errorf("created = %d, want 1", rep.Created)
	}
	if got := lastNameOf(t, db, "Max"); got != "Muster" {
		t.Errorf("last_name = %q, want %q", got, "Muster")
	}
}

func TestImport_BOMWithCommaDelimiter(t *testing.T) {
	_, srv, token := adminServer(t)
	csv := "\xef\xbb\xbfVorname,Name\nMax,Muster\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	if rep := decodeReport(t, res); rep.Created != 1 {
		t.Errorf("created = %d, want 1", rep.Created)
	}
}

// ── Delimiter ─────────────────────────────────────────────────────────────────

func TestImport_DelimiterSemicolon(t *testing.T) {
	db, srv, token := adminServer(t)
	csv := "Vorname;Name\nMax;Muster\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	if rep := decodeReport(t, res); rep.Created != 1 {
		t.Errorf("created = %d, want 1", rep.Created)
	}
	if got := statusOf(t, db, "Max"); got != "aktiv" {
		t.Errorf("status = %q, want %q", got, "aktiv")
	}
}

func TestImport_DelimiterComma(t *testing.T) {
	db, srv, token := adminServer(t)
	csv := "Vorname,Name\nMax,Muster\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	if rep := decodeReport(t, res); rep.Created != 1 {
		t.Errorf("created = %d, want 1", rep.Created)
	}
	if got := statusOf(t, db, "Max"); got != "aktiv" {
		t.Errorf("status = %q, want %q", got, "aktiv")
	}
}

// Delimiter wird NUR aus der ersten Zeile bestimmt: Header ",", eine Datenzeile
// mit gequotetem ";" ändert den Delimiter nicht — das ";" landet im Nachnamen.
func TestImport_DelimiterDetectedFromFirstLineOnly(t *testing.T) {
	db, srv, token := adminServer(t)
	csv := "Vorname,Name\nMax,\"Muster;Test\"\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	if rep := decodeReport(t, res); rep.Created != 1 {
		t.Errorf("created = %d, want 1", rep.Created)
	}
	if got := lastNameOf(t, db, "Max"); got != "Muster;Test" {
		t.Errorf("last_name = %q, want %q (Delimiter bleibt \",\")", got, "Muster;Test")
	}
}

// ── Column-Aliase ─────────────────────────────────────────────────────────────

func TestImport_ColumnAliasName(t *testing.T) {
	db, srv, token := adminServer(t)
	csv := "Vorname;Name\nMax;Muster\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	if rep := decodeReport(t, res); rep.Created != 1 {
		t.Errorf("created = %d, want 1", rep.Created)
	}
	if got := lastNameOf(t, db, "Max"); got != "Muster" {
		t.Errorf("last_name = %q, want %q (Alias Name→Nachname)", got, "Muster")
	}
}

func TestImport_ColumnAliasGeborenAmUndMitgliedSeit(t *testing.T) {
	db, srv, token := adminServer(t)
	csv := "Vorname;Name;geboren am;Mitglied seit\nMax;Muster;14.10.2007;01.09.2020\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	var dob, joinDate sql.NullString
	if err := db.QueryRow(
		`SELECT substr(date_of_birth,1,10), substr(join_date,1,10) FROM members WHERE first_name='Max'`,
	).Scan(&dob, &joinDate); err != nil {
		t.Fatalf("query: %v", err)
	}
	if dob.String != "2007-10-14" {
		t.Errorf("date_of_birth = %q, want %q (Alias geboren am)", dob.String, "2007-10-14")
	}
	if joinDate.String != "2020-09-01" {
		t.Errorf("join_date = %q, want %q (Alias Mitglied seit)", joinDate.String, "2020-09-01")
	}
}

// ── Dedup ─────────────────────────────────────────────────────────────────────

func TestImport_CSVInternalDuplicate(t *testing.T) {
	_, srv, token := adminServer(t)
	csv := "Vorname;Name\nPetra;Test\nPetra;Test\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	rep := decodeReport(t, res)
	if rep.Errors != 2 {
		t.Errorf("errors = %d, want 2", rep.Errors)
	}
	if rep.Created != 0 {
		t.Errorf("created = %d, want 0", rep.Created)
	}
	if len(rep.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rep.Rows))
	}
	if rep.Rows[0].Message != "Mehrfach in CSV (auch Zeile 3)" {
		t.Errorf("rows[0].Message = %q, want %q", rep.Rows[0].Message, "Mehrfach in CSV (auch Zeile 3)")
	}
	if rep.Rows[1].Message != "Mehrfach in CSV (zuerst Zeile 2)" {
		t.Errorf("rows[1].Message = %q, want %q", rep.Rows[1].Message, "Mehrfach in CSV (zuerst Zeile 2)")
	}
}

// TestImport_CSVDuplicateCaseInsensitive pinnt, dass der Dedup-Key case-insensitiv ist
// (strings.ToLower auf Vor-/Nachname). Ohne diesen Test bliebe ein Refactor, der das ToLower
// aus dem dupKey entfernt, unentdeckt (beide Zeilen würden fälschlich als distinct gelten).
func TestImport_CSVDuplicateCaseInsensitive(t *testing.T) {
	_, srv, token := adminServer(t)
	csv := "Vorname;Name\nPetra;Test\npetra;test\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	rep := decodeReport(t, res)
	if rep.Errors != 2 {
		t.Errorf("errors = %d, want 2 (case-insensitiver Dedup-Key)", rep.Errors)
	}
	if rep.Created != 0 {
		t.Errorf("created = %d, want 0", rep.Created)
	}
}

func TestImport_CSVDuplicateDistinctByDOB(t *testing.T) {
	_, srv, token := adminServer(t)
	csv := "Vorname;Name;geboren am\nPetra;Test;14.10.2007\nPetra;Test;15.10.2007\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	rep := decodeReport(t, res)
	if rep.Errors != 0 {
		t.Errorf("errors = %d, want 0 (verschiedene Geburtsdaten → kein Dup)", rep.Errors)
	}
	if rep.Created != 2 {
		t.Errorf("created = %d, want 2", rep.Created)
	}
}

// ── 400-Fehler (kein decode, nur StatusCode + Body-Substring) ─────────────────

func TestImport_MissingRequiredColumnVorname(t *testing.T) {
	_, srv, token := adminServer(t)
	csv := "Name;geboren am\nMuster;14.10.2007\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
	if body := read400(t, res); !strings.Contains(body, "missing required column: Vorname") {
		t.Errorf("body = %q, want substring %q", body, "missing required column: Vorname")
	}
}

func TestImport_MissingRequiredColumnNachname(t *testing.T) {
	_, srv, token := adminServer(t)
	csv := "Vorname;geboren am\nMax;14.10.2007\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
	if body := read400(t, res); !strings.Contains(body, "missing required column: Nachname") {
		t.Errorf("body = %q, want substring %q", body, "missing required column: Nachname")
	}
}

func TestImport_BrokenCSVFieldCount(t *testing.T) {
	_, srv, token := adminServer(t)
	csv := "Vorname;Name\nMax;Muster;Extra\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
	if body := read400(t, res); !strings.Contains(body, "cannot parse CSV") {
		t.Errorf("body = %q, want substring %q", body, "cannot parse CSV")
	}
}

func TestImport_EmptyFileNoHeader(t *testing.T) {
	_, srv, token := adminServer(t)
	res := postImport(t, srv, token, "", "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
	if body := read400(t, res); !strings.Contains(body, "cannot read CSV header") {
		t.Errorf("body = %q, want substring %q", body, "cannot read CSV header")
	}
}

func TestImport_MissingFileField(t *testing.T) {
	_, srv, token := adminServer(t)
	res := postImportNoFile(t, srv, token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
	if body := read400(t, res); !strings.Contains(body, "missing file") {
		t.Errorf("body = %q, want substring %q", body, "missing file")
	}
}

// ── Row-Fehler ────────────────────────────────────────────────────────────────

func TestImport_EmptyNameCellIsRowError(t *testing.T) {
	_, srv, token := adminServer(t)
	csv := "Vorname;Name\n;Test\nMax;Muster\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	rep := decodeReport(t, res)
	if rep.Errors != 1 {
		t.Errorf("errors = %d, want 1", rep.Errors)
	}
	if rep.Created != 1 {
		t.Errorf("created = %d, want 1", rep.Created)
	}
	if len(rep.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rep.Rows))
	}
	if rep.Rows[0].Message != "Vorname und Nachname sind Pflichtfelder" {
		t.Errorf("rows[0].Message = %q, want %q", rep.Rows[0].Message, "Vorname und Nachname sind Pflichtfelder")
	}
	if rep.Rows[1].Status != "created" {
		t.Errorf("rows[1].Status = %q, want %q", rep.Rows[1].Status, "created")
	}
}

// ── not_found ─────────────────────────────────────────────────────────────────

func TestImport_EnrichNotFound(t *testing.T) {
	_, srv, token := adminServer(t)
	csv := "Vorname;Name;geboren am\nMax;Neu;14.10.2007\n"
	res := postImport(t, srv, token, csv, "enrich")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	rep := decodeReport(t, res)
	if rep.NotFound != 1 {
		t.Errorf("not_found = %d, want 1", rep.NotFound)
	}
	if rep.Created != 0 {
		t.Errorf("created = %d, want 0", rep.Created)
	}
	if len(rep.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rep.Rows))
	}
	if rep.Rows[0].Status != "not_found" {
		t.Errorf("rows[0].Status = %q, want %q", rep.Rows[0].Status, "not_found")
	}
	if rep.Rows[0].Name != "Neu, Max" {
		t.Errorf("rows[0].Name = %q, want %q", rep.Rows[0].Name, "Neu, Max")
	}
	if rep.Rows[0].DOB != "2007-10-14" {
		t.Errorf("rows[0].DOB = %q, want %q", rep.Rows[0].DOB, "2007-10-14")
	}
}

// ── Stufen-Guards ─────────────────────────────────────────────────────────────

// Enrich, gleichnamige Bestandsmitglieder MIT Geburtsdatum, CSV OHNE
// Geburtsdatum-Spalte → früher cnt>=2-Zweig (Meldung A, nicht B).
func TestImport_EnrichAmbiguousNoDOB_MeldungA(t *testing.T) {
	db, srv, token := adminServer(t)
	if _, err := db.Exec(`INSERT INTO members (first_name, last_name, status, date_of_birth) VALUES ('Max','Muster','aktiv','2007-10-14')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO members (first_name, last_name, status, date_of_birth) VALUES ('Max','Muster','aktiv','2008-01-01')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// KEINE Geburtsdatum-Spalte → dob=="" → früher enrich-Zweig.
	csv := "Vorname;Name\nMax;Muster\n"
	res := postImport(t, srv, token, csv, "enrich")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	rep := decodeReport(t, res)
	if rep.Errors != 1 {
		t.Errorf("errors = %d, want 1", rep.Errors)
	}
	if len(rep.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rep.Rows))
	}
	msg := rep.Rows[0].Message
	if !strings.Contains(msg, "Treffer") || !strings.Contains(msg, "Geburtsdatum in CSV fehlt") {
		t.Errorf("rows[0].Message = %q, want Meldung A (enthält \"Treffer\" und \"Geburtsdatum in CSV fehlt\")", msg)
	}
	// Abgrenzung zu Meldung B (Zweig 1939): darf NICHT getroffen sein.
	if strings.Contains(msg, "ohne Geburtsdatum") {
		t.Errorf("rows[0].Message = %q, das ist Meldung B — erwartet war Meldung A", msg)
	}
}

// Exakter changes[]-Vertrag (Reihenfolge + Format) bei mehreren geänderten Feldern.
func TestImport_UpdateChangesContract(t *testing.T) {
	db, srv, token := adminServer(t)
	if _, err := db.Exec(`INSERT INTO members (first_name, last_name, status, gender) VALUES ('Max','Muster','aktiv','f')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	csv := "Vorname;Name;Geschlecht;Position;Passnummer\nMax;Muster;m;TW;P123\n"
	res := postImportOpts(t, srv, token, csv, "update", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	rep := decodeReport(t, res)
	if len(rep.Rows) != 1 || rep.Rows[0].Status != "updated" {
		t.Fatalf("rows = %+v, want 1 updated row", rep.Rows)
	}
	want := []string{
		`Geschlecht: "f" → "m"`,
		`Passnummer: "" → "P123"`,
		`Position: "" → "TW"`,
	}
	got := rep.Rows[0].Changes
	if len(got) != len(want) {
		t.Fatalf("changes = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("changes[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// Append ohne "Status TeamWERK"-Spalte → Status-Fallback "aktiv".
func TestImport_AppendStatusFallbackAktiv(t *testing.T) {
	db, srv, token := adminServer(t)
	csv := "Vorname;Name\nMax;Muster\n"
	res := postImport(t, srv, token, csv, "append")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	// Erfolgspfad explizit: ohne dies bliebe der Test bei einem "status weglassen,
	// DB-DEFAULT nutzen"-Refactor grün (der Handler-Fallback verschwände unbemerkt).
	rep := decodeReport(t, res)
	if rep.Created != 1 {
		t.Fatalf("created = %d, want 1", rep.Created)
	}
	if got := statusOf(t, db, "Max"); got != "aktiv" {
		t.Errorf("status = %q, want %q (Fallback)", got, "aktiv")
	}
}
