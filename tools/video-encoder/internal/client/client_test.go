package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeServer bildet die für das Tool relevanten TeamWERK-Routen nach: Login
// setzt ein Secure-Refresh-Cookie, Refresh rotiert es, der tus-Endpunkt
// verlangt einen gültigen Bearer-Token. Läuft über TLS, weil der Cookie-Jar
// Secure-Cookies (wie in Produktion) nur über https zurückschickt.
type fakeServer struct {
	t   *testing.T
	srv *httptest.Server

	mu            sync.Mutex
	tokens        map[string]bool
	refreshCookie string
	nextToken     int
	refreshStatus int // ≠0: Refresh antwortet mit diesem Status
	refreshCalls  int
	patchCalls    int
	headCalls     int
	patchAuth     []string
	size          int64
	offset        int64
	data          bytes.Buffer
	metadata      string
	location      string // Location-Antwort beim Anlegen (Default relativ)

	// beforePatch läuft vor jedem PATCH (1-basiert gezählt); true = Request erledigt.
	beforePatch func(call int, w http.ResponseWriter, r *http.Request) bool
}

func newFakeServer(t *testing.T) *fakeServer {
	f := &fakeServer{t: t, tokens: map[string]bool{}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/login", f.login)
	mux.HandleFunc("POST /api/auth/refresh", f.refresh)
	mux.HandleFunc("POST /api/videos/upload/", f.create)
	mux.HandleFunc("PATCH /api/videos/upload/abc", f.patch)
	mux.HandleFunc("HEAD /api/videos/upload/abc", f.head)
	f.srv = httptest.NewTLSServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeServer) client(t *testing.T) *Client {
	t.Helper()
	c, err := New(f.srv.URL, f.srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	c.chunkSize = 1024
	c.retryDelays = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	c.sleep = func(context.Context, time.Duration) error { return nil }
	return c
}

func (f *fakeServer) issueToken() string {
	f.nextToken++
	tok := "t" + strconv.Itoa(f.nextToken)
	f.tokens[tok] = true
	return tok
}

// expireTokens simuliert den nach 15 min abgelaufenen Access-Token.
func (f *fakeServer) expireTokens() {
	f.tokens = map[string]bool{}
}

func (f *fakeServer) authorized(r *http.Request) bool {
	tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return ok && f.tokens[tok]
}

func (f *fakeServer) login(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var req struct{ Email, Password string }
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Email == "locked@test" {
		http.Error(w, "too many attempts", http.StatusTooManyRequests)
		return
	}
	if req.Email != "trainer@test" || req.Password != "geheim" {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	f.refreshCookie = "r1"
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "r1", Path: "/", Secure: true, HttpOnly: true})
	_ = json.NewEncoder(w).Encode(map[string]string{"access_token": f.issueToken()})
}

func (f *fakeServer) refresh(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshCalls++
	ck, err := r.Cookie("refresh_token")
	if f.refreshStatus != 0 {
		http.Error(w, "unauthorized", f.refreshStatus)
		return
	}
	if err != nil || ck.Value != f.refreshCookie {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	f.refreshCookie = "r" + strconv.Itoa(f.refreshCalls+1)
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: f.refreshCookie, Path: "/", Secure: true, HttpOnly: true})
	_ = json.NewEncoder(w).Encode(map[string]string{"access_token": f.issueToken()})
}

func (f *fakeServer) create(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	f.size, _ = strconv.ParseInt(r.Header.Get("Upload-Length"), 10, 64)
	f.metadata = r.Header.Get("Upload-Metadata")
	loc := f.location
	if loc == "" {
		loc = "/api/videos/upload/abc"
	}
	w.Header().Set("Location", loc)
	w.WriteHeader(http.StatusCreated)
}

func (f *fakeServer) patch(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.patchCalls++
	call := f.patchCalls
	f.patchAuth = append(f.patchAuth, r.Header.Get("Authorization"))
	hook := f.beforePatch
	f.mu.Unlock()
	if hook != nil && hook(call, w, r) {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if off, _ := strconv.ParseInt(r.Header.Get("Upload-Offset"), 10, 64); off != f.offset {
		http.Error(w, "offset mismatch", http.StatusConflict)
		return
	}
	n, _ := io.Copy(&f.data, r.Body)
	f.offset += n
	w.Header().Set("Upload-Offset", strconv.FormatInt(f.offset, 10))
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeServer) head(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.headCalls++
	if !f.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Upload-Offset", strconv.FormatInt(f.offset, 10))
	w.Header().Set("Upload-Length", strconv.FormatInt(f.size, 10))
	w.WriteHeader(http.StatusOK)
}

func writeFile(t *testing.T, n int) (string, []byte) {
	t.Helper()
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i * 7)
	}
	p := filepath.Join(t.TempDir(), "spiel-720p.mp4")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p, data
}

func loggedIn(t *testing.T, f *fakeServer) *Client {
	t.Helper()
	c := f.client(t)
	if err := c.Login(context.Background(), "trainer@test", "geheim"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	return c
}

// 6.1: Access-Token läuft mitten im Upload ab → Refresh (mit Cookie aus dem
// Jar) und derselbe Chunk wird mit dem neuen Token wiederholt.
func TestUpload_RefreshesTokenOn401AndRetriesChunk(t *testing.T) {
	f := newFakeServer(t)
	c := loggedIn(t, f)
	path, data := writeFile(t, 3000) // 3 Chunks à 1024

	f.beforePatch = func(call int, _ http.ResponseWriter, _ *http.Request) bool {
		if call == 2 {
			f.mu.Lock()
			f.expireTokens()
			f.mu.Unlock()
		}
		return false
	}
	var created string
	err := c.Upload(context.Background(), Upload{Path: path, VideoID: 42, OnCreated: func(l string) { created = l }})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if !bytes.Equal(f.data.Bytes(), data) {
		t.Fatalf("server received %d bytes, want exact file content (%d)", f.data.Len(), len(data))
	}
	if f.refreshCalls != 1 {
		t.Fatalf("expected exactly 1 refresh, got %d", f.refreshCalls)
	}
	// 1 ok (t1), 1× 401 (t1 abgelaufen), Wiederholung + Rest mit t2.
	want := []string{"Bearer t1", "Bearer t1", "Bearer t2", "Bearer t2"}
	if strings.Join(f.patchAuth, ",") != strings.Join(want, ",") {
		t.Fatalf("PATCH auth sequence = %v, want %v", f.patchAuth, want)
	}
	if !strings.HasSuffix(created, "/api/videos/upload/abc") {
		t.Fatalf("OnCreated got %q", created)
	}
	// Metadaten wie beim Browser-Client: video_id bindet die Session an die Zeile.
	if !strings.Contains(f.metadata, "video_id "+base64.StdEncoding.EncodeToString([]byte("42"))) {
		t.Fatalf("Upload-Metadata missing video_id: %q", f.metadata)
	}
}

// 6.2: Refresh-Token abgelaufen → sauberer Abbruch mit ErrSessionExpired, kein
// weiterer Versuch, keine Schleife.
func TestUpload_RefreshFailureAborts(t *testing.T) {
	f := newFakeServer(t)
	c := loggedIn(t, f)
	path, _ := writeFile(t, 3000)
	f.mu.Lock()
	f.refreshStatus = http.StatusUnauthorized
	f.mu.Unlock()
	f.beforePatch = func(call int, _ http.ResponseWriter, _ *http.Request) bool {
		f.mu.Lock()
		f.expireTokens()
		f.mu.Unlock()
		return false
	}

	err := c.Upload(context.Background(), Upload{Path: path, VideoID: 42})
	if !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got %v", err)
	}
	if f.refreshCalls != 1 || f.patchCalls != 1 {
		t.Fatalf("expected 1 refresh + 1 PATCH, got %d refresh / %d PATCH", f.refreshCalls, f.patchCalls)
	}
}

// 5.5: Verbindungsabbruch mitten im Chunk → HEAD liefert den bestätigten Stand,
// der Upload setzt dort fort statt neu zu beginnen.
func TestUpload_ResumesAfterConnectionDrop(t *testing.T) {
	f := newFakeServer(t)
	c := loggedIn(t, f)
	path, data := writeFile(t, 3000)

	f.beforePatch = func(call int, w http.ResponseWriter, r *http.Request) bool {
		if call != 2 {
			return false
		}
		// Halben Chunk annehmen (tusd speichert Teil-Bodies), dann Verbindung kappen.
		f.mu.Lock()
		n, _ := io.CopyN(&f.data, r.Body, 512)
		f.offset += n
		f.mu.Unlock()
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack: %v", err)
			return true
		}
		conn.Close()
		return true
	}

	if err := c.Upload(context.Background(), Upload{Path: path, VideoID: 42}); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if !bytes.Equal(f.data.Bytes(), data) {
		t.Fatalf("server content differs after resume (%d bytes)", f.data.Len())
	}
	if f.headCalls == 0 {
		t.Fatal("expected a HEAD to query the confirmed offset")
	}
}

// Fortsetzen einer bestehenden Session (nach erneuter Anmeldung): kein neues
// Anlegen, Start beim Server-Offset.
func TestUpload_ContinuesExistingSession(t *testing.T) {
	f := newFakeServer(t)
	c := loggedIn(t, f)
	path, data := writeFile(t, 3000)
	f.mu.Lock()
	f.size = 3000
	f.data.Write(data[:2048])
	f.offset = 2048
	f.mu.Unlock()

	err := c.Upload(context.Background(), Upload{Path: path, VideoID: 42, Location: f.srv.URL + "/api/videos/upload/abc"})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if !bytes.Equal(f.data.Bytes(), data) || f.patchCalls != 1 || f.metadata != "" {
		t.Fatalf("expected one PATCH for the remaining chunk and no new session (patch=%d, meta=%q)", f.patchCalls, f.metadata)
	}
}

// Eine Location auf einem fremden Host bekommt den Bearer-Token nie zu sehen.
func TestUpload_RejectsForeignLocation(t *testing.T) {
	f := newFakeServer(t)
	c := loggedIn(t, f)
	path, _ := writeFile(t, 100)
	f.location = "https://evil.example/api/videos/upload/abc"

	err := c.Upload(context.Background(), Upload{Path: path, VideoID: 42})
	if err == nil || !strings.Contains(err.Error(), "fremde Adresse") {
		t.Fatalf("expected foreign-location error, got %v", err)
	}
	if f.patchCalls != 0 {
		t.Fatal("no PATCH must be sent")
	}
}

func TestLogin_Errors(t *testing.T) {
	f := newFakeServer(t)
	c := f.client(t)
	if err := c.Login(context.Background(), "trainer@test", "falsch"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password: got %v", err)
	}
	if err := c.Login(context.Background(), "locked@test", "x"); !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("locked account: got %v", err)
	}
	if c.accessToken() != "" {
		t.Fatal("failed login must not leave a token")
	}
}

func TestNew_RejectsInvalidURL(t *testing.T) {
	for _, u := range []string{"", "teamwerk.team-stuttgart.org", "ftp://x"} {
		if _, err := New(u, nil); err == nil {
			t.Errorf("New(%q) should fail", u)
		}
	}
}

// API-Routen jenseits des Uploads: eigener Server, weil sie einfache JSON-Antworten sind.
func TestAPI_CreateVideoSeasonsGames(t *testing.T) {
	var gotBody map[string]any
	seasonsStatus := http.StatusOK
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/videos", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		if gotBody["team_id"] == float64(99) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"video_id": 7, "upload_url": "/api/videos/upload/"})
	})
	mux.HandleFunc("GET /api/seasons", func(w http.ResponseWriter, r *http.Request) {
		if seasonsStatus != http.StatusOK {
			http.Error(w, "forbidden", seasonsStatus)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "is_active": false}, {"id": 3, "is_active": true}})
	})
	mux.HandleFunc("GET /api/games", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("season_id") != "3" {
			t.Errorf("games must be queried for the active season, got %q", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{
			{"id": 5, "date": "2026-03-01T00:00:00Z", "opponent": "TV Musterstadt", "teams": []map[string]any{{"id": 7}}},
			{"id": 6, "date": "2026-03-08T00:00:00Z", "opponent": "SV Beispiel", "teams": []map[string]any{{"id": 99}}},
		}})
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()
	c, _ := New(srv.URL, srv.Client())
	ctx := context.Background()

	season, err := c.ActiveSeasonID(ctx)
	if err != nil || season != 3 {
		t.Fatalf("ActiveSeasonID = %d, %v", season, err)
	}
	games, err := c.Games(ctx, 3, 7)
	if err != nil || len(games) != 1 || games[0].ID != 5 || games[0].Label() != "01.03.2026 · TV Musterstadt" {
		t.Fatalf("Games = %+v, %v", games, err)
	}

	game := 5
	id, err := c.CreateVideo(ctx, NewVideo{TeamID: 7, SeasonID: 3, GameID: &game, SizeBytes: 1234})
	if err != nil || id != 7 {
		t.Fatalf("CreateVideo = %d, %v", id, err)
	}
	if gotBody["size_bytes"] != float64(1234) || gotBody["game_id"] != float64(5) || gotBody["title"] != "" {
		t.Fatalf("unexpected body %v", gotBody)
	}
	if _, has := gotBody["description"]; has {
		t.Fatal("empty description must be omitted")
	}
	if _, err := c.CreateVideo(ctx, NewVideo{Title: "x", TeamID: 99, SeasonID: 3, SizeBytes: 1}); err == nil || !strings.Contains(err.Error(), "Berechtigung") {
		t.Fatalf("403 must become a permission error, got %v", err)
	}

	seasonsStatus = http.StatusForbidden
	if _, err := c.ActiveSeasonID(ctx); !errors.Is(err, ErrNoUploadPermission) {
		t.Fatalf("403 on seasons must mean no upload permission, got %v", err)
	}
}
