package httpcache_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/httpcache"
)

func TestServe_NoneMatchReturns304(t *testing.T) {
	etag := httpcache.ETagFor([]byte("key-material"))

	// Erster Abruf ohne If-None-Match: voller Body + ETag.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/encryption-pubkey", nil)
	httpcache.Serve(rec, req, etag, "public, max-age=86400", func() any {
		return map[string]string{"group_public_key": "PUB"}
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("erster Abruf: status %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("ETag"); got != etag {
		t.Errorf("ETag = %q, want %q", got, etag)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["group_public_key"] != "PUB" {
		t.Errorf("Body = %q (err %v), want group_public_key=PUB", rec.Body.String(), err)
	}

	// Zweiter Abruf mit If-None-Match: 304, leerer Body, body() wird nicht gebaut.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/encryption-pubkey", nil)
	req.Header.Set("If-None-Match", etag)
	called := false
	httpcache.Serve(rec, req, etag, "public, max-age=86400", func() any {
		called = true
		return nil
	})
	if rec.Code != http.StatusNotModified {
		t.Fatalf("revalidierter Abruf: status %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("304-Body nicht leer: %q", rec.Body.String())
	}
	if called {
		t.Errorf("body() wurde trotz If-None-Match-Treffer aufgerufen")
	}

	// Fehlerfall: fremder ETag im If-None-Match liefert weiterhin den vollen Body.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/encryption-pubkey", nil)
	req.Header.Set("If-None-Match", `W/"deadbeef"`)
	httpcache.Serve(rec, req, etag, "", func() any { return "x" })
	if rec.Code != http.StatusOK {
		t.Errorf("fremder ETag: status %d, want 200", rec.Code)
	}
}

func TestServe_SetsCacheControl(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/push/vapid-public-key", nil)
	httpcache.Serve(rec, req, httpcache.ETagFor([]byte("vapid")), "public, max-age=31536000, immutable", func() any {
		return map[string]string{"publicKey": "VAPID"}
	})
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("Cache-Control = %q, want immutable-Direktive", got)
	}

	// Leere Cache-Control-Angabe setzt keinen Header.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	httpcache.Serve(rec, req, httpcache.ETagFor([]byte("y")), "", func() any { return "y" })
	if got := rec.Header().Get("Cache-Control"); got != "" {
		t.Errorf("Cache-Control = %q, want leer", got)
	}
}

func TestServeJSON_ETagFollowsBody(t *testing.T) {
	// Gleicher Body → gleicher ETag → 304 bei Revalidierung.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/seasons", nil)
	httpcache.ServeJSON(rec, req, "private, no-cache", []string{"2025/26"})
	if rec.Code != http.StatusOK {
		t.Fatalf("erster Abruf: status %d, want 200", rec.Code)
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("kein ETag gesetzt")
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Errorf("Cache-Control = %q, want private, no-cache", got)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/seasons", nil)
	req.Header.Set("If-None-Match", etag)
	httpcache.ServeJSON(rec, req, "private, no-cache", []string{"2025/26"})
	if rec.Code != http.StatusNotModified || rec.Body.Len() != 0 {
		t.Errorf("unveränderter Body: status %d, body %q — want 304 + leer", rec.Code, rec.Body.String())
	}

	// Geänderter Body → anderer ETag → voller Body trotz altem If-None-Match.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/seasons", nil)
	req.Header.Set("If-None-Match", etag)
	httpcache.ServeJSON(rec, req, "private, no-cache", []string{"2025/26", "2026/27"})
	if rec.Code != http.StatusOK {
		t.Fatalf("geänderter Body: status %d, want 200", rec.Code)
	}
	if newTag := rec.Header().Get("ETag"); newTag == etag {
		t.Errorf("ETag nach Mutation unverändert: %q", newTag)
	}
}
