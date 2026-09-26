package app

import (
	"net/http"
	"net/http/httptest"
	"os"
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

// B-4: HSTS wird nur bei aktivem Flag gesetzt, dann mit exaktem Wert
// (2 Jahre, includeSubDomains; kein preload, siehe design.md Entscheidung 5).
func TestSecurityHeaders_HSTSGated(t *testing.T) {
	if got := serveThroughSecurityHeaders(t, false).Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS sollte ohne Flag fehlen, war %q", got)
	}
	const wantHSTS = "max-age=63072000; includeSubDomains"
	if got := serveThroughSecurityHeaders(t, true).Get("Strict-Transport-Security"); got != wantHSTS {
		t.Errorf("HSTS mit Flag: erwartet %q, bekam %q", wantHSTS, got)
	}
}

// Das Google-Cast-Sender-SDK wird von www.gstatic.com geladen (web/src/lib/cast.ts).
// Fehlt der Ursprung in script-src, scheitert das SDK still und Chromecast ist
// in keinem Browser nutzbar.
func TestSecurityHeaders_CSPErlaubtCastSDK(t *testing.T) {
	csp := serveThroughSecurityHeaders(t, false).Get("Content-Security-Policy")
	for _, directive := range strings.Split(csp, ";") {
		fields := strings.Fields(directive)
		if len(fields) > 0 && fields[0] == "script-src" {
			for _, src := range fields[1:] {
				if src == "https://www.gstatic.com" {
					return
				}
			}
			t.Fatalf("script-src ohne https://www.gstatic.com: %q", directive)
		}
	}
	t.Fatalf("CSP ohne script-src: %q", csp)
}

// nginx setzt die CSP als zweite Schicht. Zwei CSP-Header wirken im Browser als
// Schnittmenge — weicht nginx ab, blockt es, was die Go-Middleware erlaubt.
func TestSecurityHeaders_NginxCSPDeckungsgleich(t *testing.T) {
	conf, err := os.ReadFile("../../deploy/nginx-teamwerk.conf")
	if err != nil {
		t.Fatalf("nginx-Konfiguration lesen: %v", err)
	}
	if !strings.Contains(string(conf), `add_header Content-Security-Policy "`+contentSecurityPolicy+`" always;`) {
		t.Errorf("CSP in deploy/nginx-teamwerk.conf weicht von contentSecurityPolicy ab")
	}
}
