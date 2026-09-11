package upload_test

import (
	"bytes"
	"database/sql"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/upload"
)

// Sicherheits-Härtung des Upload-Pfads (security-haertung-welle-1, Befund
// „Stored XSS über den Foto-Upload"): Der Typ einer hochgeladenen Datei kommt
// ausschließlich aus ihren Bytes, die gespeicherte Endung aus dem erkannten Typ,
// und die Auslieferung setzt den Content-Type selbst — eine `.html` darf nie als
// `text/html` unter dem App-Origin herauskommen.

// hardeningServer baut einen Test-Server mit eigenem Upload-Verzeichnis (damit
// die abgelegten Dateien prüfbar sind) und mountet Upload + Auslieferung hinter
// der regulären Bearer-Auth.
func hardeningServer(t *testing.T, db *sql.DB) (*httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	h := upload.NewHandler(db, dir, testutil.TestJWTSecret, hub.NewHub())
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(testutil.TestJWTSecret))
		r.Post("/api/upload/user-photo", h.UploadUserPhoto)
		r.Get("/api/uploads/*", h.ServeUpload)
	})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, dir
}

// postFilePart sendet einen Multipart-Upload mit frei wählbarem Dateinamen und
// frei wählbarem Teil-`Content-Type` — genau die beiden Angaben, denen der Server
// nicht mehr glauben darf.
func postFilePart(t *testing.T, srv *httptest.Server, path, token, filename, partType string, content []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	hdr.Set("Content-Type", partType)
	part, err := mw.CreatePart(hdr)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, &buf)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return res
}

// countFiles zählt alle Dateien unterhalb von dir (rekursiv).
func countFiles(t *testing.T, dir string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return n
}

// validPNG erzeugt ein echtes 2×2-PNG (kein bloßer Magic-Byte-Prefix).
func validPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 1, color.RGBA{B: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

// (a) HTML-Bytes, als image/png deklariert, Dateiname x.html → 400, nichts gespeichert.
func TestUploadUserPhoto_HTMLMitBildHeader_400(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	tok := testutil.Token(t, userID, "standard", nil)
	srv, dir := hardeningServer(t, db)

	htmlBytes := []byte("<html><body><script>alert(document.domain)</script></body></html>")
	res := postFilePart(t, srv, "/api/upload/user-photo", tok, "x.html", "image/png", htmlBytes)
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("HTML als image/png deklariert: erwartet 400, bekam %d", res.StatusCode)
	}
	if n := countFiles(t, dir); n != 0 {
		t.Errorf("abgelehnter Upload darf nichts ablegen, fand %d Datei(en) unter %s", n, dir)
	}
	var photoPath sql.NullString
	if err := db.QueryRow(`SELECT photo_path FROM users WHERE id=?`, userID).Scan(&photoPath); err != nil {
		t.Fatalf("read users.photo_path: %v", err)
	}
	if photoPath.Valid && photoPath.String != "" {
		t.Errorf("users.photo_path darf nicht gesetzt sein, bekam %q", photoPath.String)
	}
}

// (b) Gültiges PNG mit Dateiname foto.html → gespeichert als .png, ausgeliefert
// als image/png und inline (kein attachment).
func TestUploadUserPhoto_EndungFolgtDemInhalt(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	tok := testutil.Token(t, userID, "standard", nil)
	srv, _ := hardeningServer(t, db)

	res := postFilePart(t, srv, "/api/upload/user-photo", tok, "foto.html", "text/html", validPNG(t))
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		t.Fatalf("gültiges PNG: erwartet 2xx, bekam %d", res.StatusCode)
	}

	var photoPath sql.NullString
	if err := db.QueryRow(`SELECT photo_path FROM users WHERE id=?`, userID).Scan(&photoPath); err != nil {
		t.Fatalf("read users.photo_path: %v", err)
	}
	if !photoPath.Valid || !strings.HasSuffix(photoPath.String, ".png") {
		t.Fatalf("gespeicherte Datei muss auf .png enden, bekam %q", photoPath.String)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/uploads/"+photoPath.String, nil)
	req.Header.Set("Authorization", tok)
	got, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET Upload: %v", err)
	}
	defer got.Body.Close()
	if got.StatusCode != http.StatusOK {
		t.Fatalf("Auslieferung: erwartet 200, bekam %d", got.StatusCode)
	}
	if ct := got.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type: erwartet image/png, bekam %q", ct)
	}
	if cd := got.Header.Get("Content-Disposition"); cd != "" {
		t.Errorf("Bilder werden inline ausgeliefert, bekam Content-Disposition %q", cd)
	}
}

// (c) Ein Nicht-Bild (PDF) wird als Download ausgeliefert, nie inline.
func TestServeUpload_NichtBildTraegtAttachment(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	tok := testutil.Token(t, userID, "standard", nil)
	srv, dir := hardeningServer(t, db)

	pdf := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Root 1 0 R >>\n%%EOF\n")
	if err := os.WriteFile(filepath.Join(dir, "mandat.pdf"), pdf, 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/uploads/mandat.pdf", nil)
	req.Header.Set("Authorization", tok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", res.StatusCode)
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment") {
		t.Errorf("Nicht-Bild muss attachment tragen, bekam %q", cd)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type: erwartet application/pdf, bekam %q", ct)
	}
}

// Bestandsdateien mit falscher Endung sind über die Auslieferung entschärft: eine
// bereits abgelegte .html darf nie als text/html unter dem App-Origin erscheinen.
func TestServeUpload_BestandsHTMLNiemalsTextHTML(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	tok := testutil.Token(t, userID, "standard", nil)
	srv, dir := hardeningServer(t, db)

	if err := os.WriteFile(filepath.Join(dir, "alt.html"),
		[]byte("<html><script>alert(1)</script></html>"), 0o644); err != nil {
		t.Fatalf("write html: %v", err)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/uploads/alt.html", nil)
	req.Header.Set("Authorization", tok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type darf nie text/html sein, bekam %q", ct)
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment") {
		t.Errorf("Nicht-Bild muss attachment tragen, bekam %q", cd)
	}
}
