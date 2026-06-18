package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// spaTestFS mirrors a realistic web/dist layout for spaFallback.
func spaTestFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":               {Data: []byte("<!doctype html><html></html>")},
		"sw.js":                     {Data: []byte("// service worker")},
		"manifest.webmanifest":      {Data: []byte("{}")},
		"assets/index-AbCd1234.js":  {Data: []byte("console.log(1)")},
		"assets/index-AbCd1234.css": {Data: []byte("body{}")},
		"icons/icon-192.png":        {Data: []byte("png")},
	}
}

func TestSpaHandler_CacheHeaders(t *testing.T) {
	const immutable = "public, max-age=31536000, immutable"
	const revalidate = "no-cache, must-revalidate"

	cases := []struct {
		path      string
		wantCache string
		wantETag  bool
	}{
		{"/", revalidate, true},
		{"/index.html", revalidate, true},
		{"/sw.js", revalidate, true},
		{"/manifest.webmanifest", revalidate, true},
		{"/assets/index-AbCd1234.js", immutable, false},
		{"/assets/index-AbCd1234.css", immutable, false},
		{"/icons/icon-192.png", revalidate, true},
		// Unknown route → SPA fallback to index.html → revalidate + ETag.
		{"/dashboard", revalidate, true},
	}

	h := spaFallback(spaTestFS(), "testhash")
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			h(rec, req)

			if got := rec.Header().Get("Cache-Control"); got != tc.wantCache {
				t.Errorf("Cache-Control = %q, want %q", got, tc.wantCache)
			}
			etag := rec.Header().Get("ETag")
			if tc.wantETag && etag == "" {
				t.Errorf("expected ETag header, got none")
			}
			if !tc.wantETag && etag != "" {
				t.Errorf("expected no ETag header, got %q", etag)
			}
		})
	}
}

func TestSpaHandler_ETag_304(t *testing.T) {
	h := spaFallback(spaTestFS(), "testhash")

	// First request: capture the ETag and a 200 with body. We use "/" rather
	// than "/index.html" because http.FileServer redirects the explicit
	// index.html path to "/" (301); both resolve to the same served path.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	h(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first GET status = %d, want 200", rec1.Code)
	}
	etag := rec1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("first GET returned no ETag")
	}
	if rec1.Body.Len() == 0 {
		t.Fatal("first GET returned empty body")
	}

	// Second request with matching If-None-Match → 304, empty body.
	req2 := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	req2.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	h(rec2, req2)

	if rec2.Code != http.StatusNotModified {
		t.Fatalf("conditional GET status = %d, want 304", rec2.Code)
	}
	if rec2.Body.Len() != 0 {
		t.Fatalf("304 response should have empty body, got %d bytes", rec2.Body.Len())
	}
}

func TestSpaHandler_ETag_Changes_With_BuildHash(t *testing.T) {
	etagFor := func(hash string) string {
		h := spaFallback(spaTestFS(), hash)
		req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
		rec := httptest.NewRecorder()
		h(rec, req)
		return rec.Header().Get("ETag")
	}

	devETag := etagFor("dev")
	newETag := etagFor("x123")

	if devETag == "" || newETag == "" {
		t.Fatal("expected non-empty ETags")
	}
	if devETag == newETag {
		t.Errorf("ETag did not change with buildHash: both %q", devETag)
	}
}
