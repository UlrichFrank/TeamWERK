package upload_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

const bulkImportPath = "/api/members/sepa-mandates/import"

type bulkImportEntry struct {
	Filename   string  `json:"filename"`
	MemberID   *int    `json:"member_id,omitempty"`
	MemberName *string `json:"member_name,omitempty"`
	Reason     string  `json:"reason,omitempty"`
}

type bulkImportCandidate struct {
	MemberID   int    `json:"member_id"`
	MemberName string `json:"member_name"`
}

type bulkAmbiguousEntry struct {
	Filename   string                `json:"filename"`
	Candidates []bulkImportCandidate `json:"candidates"`
}

type bulkImportReport struct {
	Imported      []bulkImportEntry    `json:"imported"`
	AlreadyExists []bulkImportEntry    `json:"already_exists"`
	NoMatch       []bulkImportEntry    `json:"no_match"`
	Ambiguous     []bulkAmbiguousEntry `json:"ambiguous"`
}

// pdfBody returns a minimal valid PDF byte payload of approximately size bytes.
func pdfBody(size int) []byte {
	prefix := []byte("%PDF-1.4\n% bulk-test\n")
	if size <= len(prefix) {
		return prefix
	}
	buf := make([]byte, size)
	copy(buf, prefix)
	for i := len(prefix); i < size; i++ {
		buf[i] = 'x'
	}
	return buf
}

// postBulk posts a multipart body with the given (filename, contentType, body)
// tuples under the "files" form field. mimeType "" means "let server sniff".
func postBulk(t *testing.T, srv, token string, files []struct {
	Name        string
	ContentType string
	Body        []byte
}) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, f := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename=%q`, f.Name))
		if f.ContentType != "" {
			h.Set("Content-Type", f.ContentType)
		}
		fw, err := mw.CreatePart(h)
		if err != nil {
			t.Fatalf("CreatePart: %v", err)
		}
		if _, err := fw.Write(f.Body); err != nil {
			t.Fatalf("write part: %v", err)
		}
	}
	mw.Close()

	req, err := http.NewRequest(http.MethodPost, srv+bulkImportPath, &buf)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post bulk: %v", err)
	}
	return res
}

func decodeBulkReport(t *testing.T, res *http.Response) bulkImportReport {
	t.Helper()
	var rep bulkImportReport
	if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
		t.Fatalf("decode bulk report: %v", err)
	}
	res.Body.Close()
	return rep
}

func insertMember(t *testing.T, db *sql.DB, first, last string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO members (first_name, last_name, status) VALUES (?, ?, 'aktiv')`,
		first, last)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func setSepaPath(t *testing.T, db *sql.DB, memberID int, path string) {
	t.Helper()
	if _, err := db.Exec(`UPDATE members SET sepa_mandat_path=?, sepa_mandat=1 WHERE id=?`,
		path, memberID); err != nil {
		t.Fatalf("set sepa path: %v", err)
	}
}

func getSepaPath(t *testing.T, db *sql.DB, memberID int) (string, int) {
	t.Helper()
	var path sql.NullString
	var flag int
	if err := db.QueryRow(`SELECT sepa_mandat_path, sepa_mandat FROM members WHERE id=?`, memberID).
		Scan(&path, &flag); err != nil {
		t.Fatalf("query sepa: %v", err)
	}
	return path.String, flag
}

func vorstandToken(t *testing.T, db *sql.DB) string {
	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'vorstand')`,
		mid); err != nil {
		t.Fatalf("attach vorstand: %v", err)
	}
	return testutil.Token(t, uid, "standard", []string{"vorstand"})
}

func kassiererToken(t *testing.T, db *sql.DB) string {
	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	if _, err := db.Exec(
		`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'kassierer')`,
		mid); err != nil {
		t.Fatalf("attach kassierer: %v", err)
	}
	return testutil.Token(t, uid, "standard", []string{"kassierer"})
}

func spielerToken(t *testing.T, db *sql.DB) string {
	uid := testutil.CreateUser(t, db, "standard")
	return testutil.Token(t, uid, "standard", []string{"spieler"})
}

func TestBulkImport_HappyPath_MatchAndStore(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	memberID := insertMember(t, db, "Max", "Mustermann")
	token := vorstandToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"MaxMustermann.pdf", "application/pdf", pdfBody(200)},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", res.StatusCode)
	}
	rep := decodeBulkReport(t, res)
	if len(rep.Imported) != 1 || rep.Imported[0].MemberID == nil || *rep.Imported[0].MemberID != memberID {
		t.Fatalf("imported = %+v", rep.Imported)
	}
	path, flag := getSepaPath(t, db, memberID)
	if path == "" || !strings.HasPrefix(path, "sepa-mandats/") || flag != 1 {
		t.Fatalf("member state: path=%q flag=%d", path, flag)
	}
}

func TestBulkImport_SkipsExistingPath(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	memberID := insertMember(t, db, "Max", "Mustermann")
	setSepaPath(t, db, memberID, "sepa-mandats/existing.pdf")
	token := vorstandToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"MaxMustermann.pdf", "application/pdf", pdfBody(200)},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", res.StatusCode)
	}
	rep := decodeBulkReport(t, res)
	if len(rep.AlreadyExists) != 1 || len(rep.Imported) != 0 {
		t.Fatalf("unexpected: %+v", rep)
	}
	path, flag := getSepaPath(t, db, memberID)
	if path != "sepa-mandats/existing.pdf" || flag != 1 {
		t.Fatalf("path should be unchanged: %q flag=%d", path, flag)
	}
}

func TestBulkImport_NoMatchReported(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	insertMember(t, db, "Max", "Mustermann")
	token := vorstandToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"Unbekannt.pdf", "application/pdf", pdfBody(200)},
	})
	rep := decodeBulkReport(t, res)
	if len(rep.NoMatch) != 1 || len(rep.Imported) != 0 {
		t.Fatalf("unexpected: %+v", rep)
	}
}

func TestBulkImport_AmbiguousMatchSkipped(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	idA := insertMember(t, db, "Lukas", "Schmidt")
	idB := insertMember(t, db, "Lukas", "Schmidt")
	token := vorstandToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"LukasSchmidt.pdf", "application/pdf", pdfBody(200)},
	})
	rep := decodeBulkReport(t, res)
	if len(rep.Ambiguous) != 1 {
		t.Fatalf("expected 1 ambiguous, got %+v", rep)
	}
	if len(rep.Ambiguous[0].Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %+v", rep.Ambiguous[0])
	}
	pathA, _ := getSepaPath(t, db, idA)
	pathB, _ := getSepaPath(t, db, idB)
	if pathA != "" || pathB != "" {
		t.Fatalf("members must not have been touched: A=%q B=%q", pathA, pathB)
	}
}

func TestBulkImport_UmlautNormalization(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	memberID := insertMember(t, db, "Jürgen", "Müller")
	token := vorstandToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"JuergenMueller.pdf", "application/pdf", pdfBody(200)},
	})
	rep := decodeBulkReport(t, res)
	if len(rep.Imported) != 1 || rep.Imported[0].MemberID == nil || *rep.Imported[0].MemberID != memberID {
		t.Fatalf("expected umlaut match: %+v", rep)
	}
}

func TestBulkImport_ReverseNameOrder(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	memberID := insertMember(t, db, "Max", "Mustermann")
	token := vorstandToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"MustermannMax.pdf", "application/pdf", pdfBody(200)},
	})
	rep := decodeBulkReport(t, res)
	if len(rep.Imported) != 1 || rep.Imported[0].MemberID == nil || *rep.Imported[0].MemberID != memberID {
		t.Fatalf("expected reverse-order match: %+v", rep)
	}
}

func TestBulkImport_RejectsNonPDF(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	insertMember(t, db, "Max", "Mustermann")
	token := vorstandToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"MaxMustermann.jpg", "image/jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}},
	})
	rep := decodeBulkReport(t, res)
	if len(rep.NoMatch) != 1 || rep.NoMatch[0].Reason != "kein PDF" {
		t.Fatalf("expected reject as non-PDF: %+v", rep)
	}
	if len(rep.Imported) != 0 {
		t.Fatalf("must not import: %+v", rep.Imported)
	}
}

func TestBulkImport_FileTooLarge(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	memberID := insertMember(t, db, "Max", "Mustermann")
	token := vorstandToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"MaxMustermann.pdf", "application/pdf", pdfBody((10 << 20) + 1024)},
	})
	rep := decodeBulkReport(t, res)
	if len(rep.NoMatch) != 1 || rep.NoMatch[0].Reason != "zu groß (>10 MB)" {
		t.Fatalf("expected zu groß: %+v", rep)
	}
	path, _ := getSepaPath(t, db, memberID)
	if path != "" {
		t.Fatalf("must not have stored: %q", path)
	}
}

func TestBulkImport_ForbiddenForSpieler(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	insertMember(t, db, "Max", "Mustermann")
	token := spielerToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"MaxMustermann.pdf", "application/pdf", pdfBody(200)},
	})
	if res.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 403, got %d body=%s", res.StatusCode, string(body))
	}
}

func TestBulkImport_AllowedForKassierer(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	memberID := insertMember(t, db, "Max", "Mustermann")
	token := kassiererToken(t, db)

	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"MaxMustermann.pdf", "application/pdf", pdfBody(200)},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status: %d", res.StatusCode)
	}
	rep := decodeBulkReport(t, res)
	if len(rep.Imported) != 1 {
		t.Fatalf("expected 1 imported as kassierer: %+v", rep)
	}
	path, _ := getSepaPath(t, db, memberID)
	if path == "" {
		t.Fatalf("expected member to be updated")
	}
}

func TestBulkImport_BroadcastEmitted(t *testing.T) {
	db := testutil.NewDB(t)
	srv, hubInstance := prodserver.NewWithHub(t, db)
	insertMember(t, db, "Max", "Mustermann")
	token := vorstandToken(t, db)

	events := subscribeHub(t, hubInstance)
	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"MaxMustermann.pdf", "application/pdf", pdfBody(200)},
	})
	res.Body.Close()

	select {
	case ev := <-events:
		if ev != "members" {
			t.Fatalf("unexpected event: %q", ev)
		}
	case <-time.After(time.Second):
		t.Fatalf("no broadcast within 1s")
	}
}

func TestBulkImport_NoBroadcastWhenNothingImported(t *testing.T) {
	db := testutil.NewDB(t)
	srv, hubInstance := prodserver.NewWithHub(t, db)
	insertMember(t, db, "Max", "Mustermann")
	token := vorstandToken(t, db)

	events := subscribeHub(t, hubInstance)
	res := postBulk(t, srv.URL, token, []struct {
		Name        string
		ContentType string
		Body        []byte
	}{
		{"Unbekannt.pdf", "application/pdf", pdfBody(200)},
	})
	res.Body.Close()

	select {
	case ev := <-events:
		t.Fatalf("unexpected broadcast: %q", ev)
	case <-time.After(100 * time.Millisecond):
	}
}

func subscribeHub(t *testing.T, h *hub.EventHub) <-chan string {
	t.Helper()
	ch := h.Subscribe()
	t.Cleanup(func() { h.Unsubscribe(ch) })
	return ch
}
