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
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/upload"
)

// postSepaMandat lädt einen (clientseitig verschlüsselten) Blob unter "file" hoch und
// setzt optional das Feld dek_enc.
func postSepaMandat(t *testing.T, url, token, filename string, body []byte, dekEnc string) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", "application/octet-stream")
	fw, err := mw.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(body)
	if dekEnc != "" {
		mw.WriteField("dek_enc", dekEnc)
	}
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

// Modell B: Upload speichert den clientseitig verschlüsselten Blob roh + den gewrappten
// DEK; der Download liefert den Ciphertext-Blob unverändert (Server entschlüsselt nie).
func TestSepaUploadDownload_ZeroKnowledge(t *testing.T) {
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

	// Clientseitig verschlüsselter Blob (Magic "TWENC1\n" ‖ Ciphertext) + gewrappter DEK.
	clientBlob := append([]byte("TWENC1\n"), []byte("\x00\x01verschluesselter mandatsinhalt")...)
	const dekEnc = "d3JhcHBlZERFSw=="

	// Klartext-PDF (ohne Client-Magic) wird abgelehnt.
	plain := []byte("%PDF-1.4\nKlartext\n%%EOF")
	if res := postSepaMandat(t, srv.URL+"/api/upload/sepa-mandat/"+strconv.Itoa(id), tok, "m.pdf", plain, dekEnc); res.StatusCode != http.StatusBadRequest {
		res.Body.Close()
		t.Fatalf("Klartext-Upload: status %d, want 400", res.StatusCode)
	}
	// Fehlender dek_enc wird abgelehnt.
	if res := postSepaMandat(t, srv.URL+"/api/upload/sepa-mandat/"+strconv.Itoa(id), tok, "m.bin", clientBlob, ""); res.StatusCode != http.StatusBadRequest {
		res.Body.Close()
		t.Fatalf("ohne dek_enc: status %d, want 400", res.StatusCode)
	}

	// Gültiger Upload.
	res := postSepaMandat(t, srv.URL+"/api/upload/sepa-mandat/"+strconv.Itoa(id), tok, "m.bin", clientBlob, dekEnc)
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("upload status %d: %s", res.StatusCode, body)
	}
	res.Body.Close()

	// Datei auf der Platte == hochgeladener Ciphertext (Server hat nichts ver-/entschlüsselt);
	// dek_enc ist gespeichert.
	var rel, dbDek string
	db.QueryRow(`SELECT sepa_mandat_path, COALESCE(sepa_mandat_dek_enc,'') FROM members WHERE id=?`, id).Scan(&rel, &dbDek)
	onDisk, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatalf("read stored file: %v", err)
	}
	if !bytes.Equal(onDisk, clientBlob) {
		t.Errorf("gespeicherte Datei != hochgeladener Blob")
	}
	if dbDek != dekEnc {
		t.Errorf("dek_enc nicht gespeichert: %q", dbDek)
	}

	// Download-Token liefert token + dek_enc.
	tokRes := testutil.Get(t, srv, "/api/members/"+strconv.Itoa(id)+"/sepa-mandat/download-token", tok)
	var td struct {
		Token  string `json:"token"`
		DekEnc string `json:"dek_enc"`
	}
	json.NewDecoder(tokRes.Body).Decode(&td)
	tokRes.Body.Close()
	if td.Token == "" || td.DekEnc != dekEnc {
		t.Fatalf("download-token: token=%q dek_enc=%q", td.Token, td.DekEnc)
	}

	// Download liefert den Ciphertext-Blob unverändert (kein Server-Decrypt).
	dl := testutil.Get(t, srv, "/api/members/"+strconv.Itoa(id)+"/sepa-mandat/download?token="+td.Token, tok)
	got, _ := io.ReadAll(dl.Body)
	dl.Body.Close()
	if !bytes.Equal(got, clientBlob) {
		t.Errorf("Download liefert nicht den Ciphertext-Blob")
	}
}
