package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serveThroughSecurityHeaders(t *testing.T, hsts bool) http.Header {
	t.Helper()
	h := securityHeaders(hsts)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/healthz", nil))
	return rec.Result().Header
}

// B-4: alle Härtungsheader sind auf der Antwort vorhanden.
func TestSecurityHeaders_Present(t *testing.T) {
	hdr := serveThroughSecurityHeaders(t, false)

	if got := hdr.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options: erwartet DENY, bekam %q", got)
	}
	if got := hdr.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options: erwartet nosniff, bekam %q", got)
	}
	if got := hdr.Get("Referrer-Policy"); got == "" {
		t.Error("Referrer-Policy fehlt")
	}
	csp := hdr.Get("Content-Security-Policy")
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("CSP ohne frame-ancestors 'none': %q", csp)
	}
	if !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "object-src 'none'") || !strings.Contains(csp, "base-uri 'self'") {
		t.Errorf("CSP unvollständig: %q", csp)
	}
}

// B-4: HSTS wird nur bei aktivem Flag gesetzt.
func TestSecurityHeaders_HSTSGated(t *testing.T) {
	if got := serveThroughSecurityHeaders(t, false).Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS sollte ohne Flag fehlen, war %q", got)
	}
	if got := serveThroughSecurityHeaders(t, true).Get("Strict-Transport-Security"); !strings.Contains(got, "max-age=") {
		t.Errorf("HSTS mit Flag: erwartet max-age, bekam %q", got)
	}
}
