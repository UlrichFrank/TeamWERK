package media_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/media"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// pngBytes ist eine minimale PNG-Signatur; http.DetectContentType erkennt sie
// als image/png (kein voller Decode nötig).
var pngBytes = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x00}

// pdfBytes ist eine PDF-Signatur (nicht erlaubt).
var pdfBytes = []byte("%PDF-1.4\n%âãÏÓ\n")

func newMediaServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	h := media.NewHandler(db, t.TempDir())
	return testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/media/upload", h.Upload)
		r.Get("/api/media/{id}", h.Serve)
	})
}

func multipartImage(t *testing.T, url, token, field string, data []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile(field, "img.bin")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	return req
}

func upload(t *testing.T, url, token string, data []byte) *http.Response {
	t.Helper()
	res, err := http.DefaultClient.Do(multipartImage(t, url, token, "image", data))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	return res
}

func TestUpload_OK(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)
	tok := testutil.Token(t, uid, "standard", nil)

	res := upload(t, srv.URL+"/api/media/upload", tok, pngBytes)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var body struct {
		MediaID int    `json:"mediaId"`
		URL     string `json:"url"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.MediaID == 0 {
		t.Errorf("expected non-zero mediaId")
	}
	if body.URL != "/media/"+strconv.Itoa(body.MediaID) {
		t.Errorf("expected url /media/%d, got %q", body.MediaID, body.URL)
	}
	// media-Zeile vorhanden?
	var cnt int
	db.QueryRow(`SELECT COUNT(*) FROM media WHERE id = ?`, body.MediaID).Scan(&cnt)
	if cnt != 1 {
		t.Errorf("expected media row, got %d", cnt)
	}
}

func TestUpload_BadMime(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)
	tok := testutil.Token(t, uid, "standard", nil)

	res := upload(t, srv.URL+"/api/media/upload", tok, pdfBytes)
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for PDF, got %d", res.StatusCode)
	}
}

func TestUpload_TooLarge(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)
	tok := testutil.Token(t, uid, "standard", nil)

	big := make([]byte, (1<<20)+(200<<10)) // ~1.2 MB > Limit
	copy(big, pngBytes)
	res := upload(t, srv.URL+"/api/media/upload", tok, big)
	defer res.Body.Close()
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", res.StatusCode)
	}
}

func TestUpload_Unauth(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMediaServer(t, db)
	res := upload(t, srv.URL+"/api/media/upload", "", pngBytes)
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestServe_OK(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)
	tok := testutil.Token(t, uid, "standard", nil)

	res := upload(t, srv.URL+"/api/media/upload", tok, pngBytes)
	var body struct {
		MediaID int `json:"mediaId"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()

	getRes := testutil.Get(t, srv, "/api/media/"+strconv.Itoa(body.MediaID), tok)
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRes.StatusCode)
	}
	if ct := getRes.Header.Get("Content-Type"); ct != "image/png" {
		t.Errorf("expected image/png, got %q", ct)
	}
}

func TestServe_NotFound(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)
	tok := testutil.Token(t, uid, "standard", nil)

	getRes := testutil.Get(t, srv, "/api/media/99999", tok)
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", getRes.StatusCode)
	}
}

func TestServe_Unauth(t *testing.T) {
	db := testutil.NewDB(t)
	srv := newMediaServer(t, db)
	getRes := testutil.Get(t, srv, "/api/media/1", "")
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", getRes.StatusCode)
	}
}
