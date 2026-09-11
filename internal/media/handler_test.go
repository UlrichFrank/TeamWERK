package media_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/media"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// realPNG erzeugt ein echtes, decodierbares PNG mit den gewünschten
// Dimensionen — für Tests, die die Header-Probe (decodeDimensions) prüfen.
func realPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

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

func TestUpload_ReturnsDimensions(t *testing.T) {
	// Header-Probe muss beim Upload die tatsächlichen Pixel-Dimensionen
	// ermitteln, in der DB persistieren und in der Response mitschicken —
	// Grundlage dafür, dass das Frontend die aspect-ratio ab dem ersten
	// Frame kennt und nichts nach dem Bild-Load verspringt.
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)
	tok := testutil.Token(t, uid, "standard", nil)

	img := realPNG(t, 3, 2)
	res := upload(t, srv.URL+"/api/media/upload", tok, img)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var body struct {
		MediaID int `json:"mediaId"`
		Width   int `json:"width"`
		Height  int `json:"height"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Width != 3 || body.Height != 2 {
		t.Errorf("expected 3x2 in response, got %dx%d", body.Width, body.Height)
	}
	var w, h sql.NullInt64
	db.QueryRow(`SELECT width, height FROM media WHERE id = ?`, body.MediaID).Scan(&w, &h)
	if !w.Valid || !h.Valid || w.Int64 != 3 || h.Int64 != 2 {
		t.Errorf("expected width=3 height=2 in DB, got %v %v", w, h)
	}
}

func TestUpload_CorruptHeader_AcceptedWithoutDimensions(t *testing.T) {
	// pngBytes ist nur die PNG-Magic + Padding (DetectContentType erkennt es
	// als image/png, aber DecodeConfig scheitert am fehlenden IHDR). Wir
	// verifizieren die dokumentierte Toleranz: Upload durchlaufen lassen,
	// keine width/height in Response, NULL in der DB, kein 5xx.
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)
	tok := testutil.Token(t, uid, "standard", nil)

	res := upload(t, srv.URL+"/api/media/upload", tok, pngBytes)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}
	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := raw["width"]; ok {
		t.Errorf("expected no width in response, got %v", raw["width"])
	}
	if _, ok := raw["height"]; ok {
		t.Errorf("expected no height in response, got %v", raw["height"])
	}
	var w, h sql.NullInt64
	mediaID := int(raw["mediaId"].(float64))
	db.QueryRow(`SELECT width, height FROM media WHERE id = ?`, mediaID).Scan(&w, &h)
	if w.Valid || h.Valid {
		t.Errorf("expected NULL width/height in DB, got %v %v", w, h)
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

// ── Objekt-Gate (security-haertung-welle-1, Entscheidung 3) ───────────────────

// uploadMedia lädt ein PNG als uid hoch und gibt die media-ID zurück.
func uploadMedia(t *testing.T, srv *httptest.Server, db *sql.DB, uid int) int {
	t.Helper()
	res := upload(t, srv.URL+"/api/media/upload", testutil.Token(t, uid, "standard", nil), pngBytes)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d", res.StatusCode)
	}
	var body struct {
		MediaID int `json:"mediaId"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.MediaID
}

// TestServe_FremderNutzer404 hält die Kernzusage fest: wer weder hochgeladen
// hat noch das referenzierende Objekt sehen darf, bekommt 404 — nicht 403,
// damit die Existenz einer ID nicht erratbar ist.
func TestServe_FremderNutzer404(t *testing.T) {
	db := testutil.NewDB(t)
	uploader := testutil.CreateUser(t, db, "standard")
	fremder := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)

	mediaID := uploadMedia(t, srv, db, uploader)

	res := testutil.Get(t, srv, "/api/media/"+strconv.Itoa(mediaID),
		testutil.Token(t, fremder, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("Fremder: erwartet 404, bekommen %d", res.StatusCode)
	}
}

// TestServe_AdminOhneBezug404: eine Vereinsfunktion verschafft keinen Zugang
// zu fremden Chat-Bildern — auch der System-Admin nicht.
func TestServe_AdminOhneBezug404(t *testing.T) {
	db := testutil.NewDB(t)
	uploader := testutil.CreateUser(t, db, "standard")
	adminID := testutil.CreateUser(t, db, "admin")
	srv := newMediaServer(t, db)

	mediaID := uploadMedia(t, srv, db, uploader)

	res := testutil.Get(t, srv, "/api/media/"+strconv.Itoa(mediaID),
		testutil.Token(t, adminID, "admin", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("admin ohne Bezug: erwartet 404, bekommen %d", res.StatusCode)
	}
}

// TestServe_KonversationsmitgliedOK: das Bild hängt an einer Nachricht in einer
// Konversation, in der der Abrufende Mitglied ist → 200.
func TestServe_KonversationsmitgliedOK(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	empfaenger := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)

	mediaID := uploadMedia(t, srv, db, sender)

	res, err := db.Exec(`INSERT INTO conversations (type, name, created_by) VALUES ('group', 'Gruppe', ?)`, sender)
	if err != nil {
		t.Fatalf("conversations: %v", err)
	}
	convID, _ := res.LastInsertId()
	for _, uid := range []int{sender, empfaenger} {
		if _, err := db.Exec(
			`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, uid); err != nil {
			t.Fatalf("conversation_members: %v", err)
		}
	}
	if _, err := db.Exec(
		`INSERT INTO messages (conversation_id, sender_id, body, media_id) VALUES (?, ?, '', ?)`,
		convID, sender, mediaID); err != nil {
		t.Fatalf("messages: %v", err)
	}

	getRes := testutil.Get(t, srv, "/api/media/"+strconv.Itoa(mediaID),
		testutil.Token(t, empfaenger, "standard", nil))
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("Konversationsmitglied: erwartet 200, bekommen %d", getRes.StatusCode)
	}
}

// TestServe_AusgetretenesMitgliedOK: der Verlauf bleibt lesbar — wer die Gruppe
// verlassen hat (left_at gesetzt), sieht die Bilder seiner Zeit weiter.
func TestServe_AusgetretenesMitgliedOK(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	ausgetreten := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)

	mediaID := uploadMedia(t, srv, db, sender)

	res, err := db.Exec(`INSERT INTO conversations (type, name, created_by) VALUES ('group', 'Gruppe', ?)`, sender)
	if err != nil {
		t.Fatalf("conversations: %v", err)
	}
	convID, _ := res.LastInsertId()
	db.Exec(`INSERT INTO conversation_members (conversation_id, user_id) VALUES (?, ?)`, convID, sender)
	db.Exec(`INSERT INTO conversation_members (conversation_id, user_id, left_at) VALUES (?, ?, CURRENT_TIMESTAMP)`,
		convID, ausgetreten)
	if _, err := db.Exec(
		`INSERT INTO messages (conversation_id, sender_id, body, media_id) VALUES (?, ?, '', ?)`,
		convID, sender, mediaID); err != nil {
		t.Fatalf("messages: %v", err)
	}

	getRes := testutil.Get(t, srv, "/api/media/"+strconv.Itoa(mediaID),
		testutil.Token(t, ausgetreten, "standard", nil))
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("ausgetretenes Mitglied: erwartet 200, bekommen %d", getRes.StatusCode)
	}
}

// TestServe_MitteilungsempfaengerOK: das Bild hängt an einer Mitteilung, für
// die der Abrufende eine broadcast_reads-Zeile hat → 200.
func TestServe_MitteilungsempfaengerOK(t *testing.T) {
	db := testutil.NewDB(t)
	sender := testutil.CreateUser(t, db, "standard")
	empfaenger := testutil.CreateUser(t, db, "standard")
	fremder := testutil.CreateUser(t, db, "standard")
	srv := newMediaServer(t, db)

	mediaID := uploadMedia(t, srv, db, sender)

	res, err := db.Exec(`INSERT INTO broadcasts (sender_id, body, media_id) VALUES (?, '', ?)`, sender, mediaID)
	if err != nil {
		t.Fatalf("broadcasts: %v", err)
	}
	broadcastID, _ := res.LastInsertId()
	if _, err := db.Exec(
		`INSERT INTO broadcast_reads (broadcast_id, user_id) VALUES (?, ?)`, broadcastID, empfaenger); err != nil {
		t.Fatalf("broadcast_reads: %v", err)
	}

	okRes := testutil.Get(t, srv, "/api/media/"+strconv.Itoa(mediaID),
		testutil.Token(t, empfaenger, "standard", nil))
	defer okRes.Body.Close()
	if okRes.StatusCode != http.StatusOK {
		t.Fatalf("Mitteilungsempfänger: erwartet 200, bekommen %d", okRes.StatusCode)
	}

	// Gegenprobe: ohne broadcast_reads-Zeile bleibt es bei 404.
	denyRes := testutil.Get(t, srv, "/api/media/"+strconv.Itoa(mediaID),
		testutil.Token(t, fremder, "standard", nil))
	defer denyRes.Body.Close()
	if denyRes.StatusCode != http.StatusNotFound {
		t.Fatalf("Nicht-Empfänger: erwartet 404, bekommen %d", denyRes.StatusCode)
	}
}
