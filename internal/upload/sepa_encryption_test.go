package upload_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/crypto"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/upload"
)

// postSingleFile lädt eine Datei unter dem Formularfeld "file" hoch.
func postSingleFile(t *testing.T, url, token, filename string, body []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "application/pdf")
	fw, err := mw.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(body)
	mw.Close()
	req, _ := http.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestSepaUploadDownload_EncryptedAtRest (7.7): Upload speichert das PDF
// verschlüsselt auf der Platte; der Download liefert das Original zurück.
func TestSepaUploadDownload_EncryptedAtRest(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	dir := t.TempDir()
	h := upload.NewHandler(db, dir, testutil.TestJWTSecret, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/upload/sepa-mandat/{id}", h.UploadSepaMandat)
		r.Get("/api/members/{id}/sepa-mandat/download-token", h.SepaDownloadToken)
		r.Get("/api/members/{id}/sepa-mandat/download", h.SepaDownload)
	})
	tok := testutil.Token(t, 1, "admin", nil)

	original := []byte("%PDF-1.4\nMandat Originalinhalt\n%%EOF")
	res := postSingleFile(t, srv.URL+"/api/upload/sepa-mandat/"+strconv.Itoa(id), tok, "mandat.pdf", original)
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("upload status %d: %s", res.StatusCode, body)
	}
	res.Body.Close()

	// Datei auf der Platte ist verschlüsselt (Magic-Header), nicht das Klartext-PDF.
	var rel string
	db.QueryRow(`SELECT sepa_mandat_path FROM members WHERE id=?`, id).Scan(&rel)
	onDisk, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("read stored file: %v", err)
	}
	if !crypto.IsEncryptedBytes(onDisk) {
		t.Errorf("gespeicherte Datei nicht verschlüsselt")
	}
	if bytes.HasPrefix(onDisk, []byte("%PDF")) {
		t.Errorf("gespeicherte Datei beginnt mit %%PDF (Klartext)")
	}

	// Download liefert das Original-PDF (entschlüsselt).
	tokRes := testutil.Get(t, srv, "/api/members/"+strconv.Itoa(id)+"/sepa-mandat/download-token", tok)
	var td struct {
		Token string `json:"token"`
	}
	json.NewDecoder(tokRes.Body).Decode(&td)
	tokRes.Body.Close()
	if td.Token == "" {
		t.Fatal("kein Download-Token erhalten")
	}

	// In Produktion ist die Download-Route public; im Test umschließt
	// testutil.NewServer alle Routen mit auth.Middleware, daher ein gültiger
	// Bearer zusätzlich zum Query-Token (den der Handler prüft).
	dl := testutil.Get(t, srv, "/api/members/"+strconv.Itoa(id)+"/sepa-mandat/download?token="+td.Token, tok)
	got, _ := io.ReadAll(dl.Body)
	dl.Body.Close()
	if !bytes.Equal(got, original) {
		t.Errorf("Download liefert nicht das Original-PDF:\n got %q\nwant %q", got, original)
	}
}
