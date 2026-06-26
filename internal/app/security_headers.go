package app

import "net/http"

// contentSecurityPolicy ist auf den realen Build abgestimmt:
//   - script-src 'self': der Vite-Build referenziert nur same-origin-Module
//     (keine Inline-<script>), daher kein 'unsafe-inline' nötig.
//   - style-src 'unsafe-inline' + fonts.googleapis.com: Tailwind-CSS ist
//     same-origin, der Google-Fonts-Stylesheet extern; React nutzt inline
//     style-Attribute.
//   - font-src fonts.gstatic.com: Hanken Grotesk (woff2).
//   - img-src data:/blob:: Avatare same-origin, Crop-Vorschau via blob:.
//   - connect-src 'self': SSE (/api/events) und alle API-Calls same-origin.
//   - frame-ancestors 'none' + object-src 'none' + base-uri 'self': Clickjacking-
//     und Injection-Härtung.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"img-src 'self' data: blob:; " +
	"connect-src 'self'; " +
	"frame-ancestors 'none'; " +
	"object-src 'none'; " +
	"base-uri 'self'"

// securityHeaders setzt Browser-Härtungsheader auf allen Antworten. Die Header
// werden in der Go-Kette gesetzt, damit sie unabhängig vom Reverse-Proxy wirken.
// HSTS wird nur bei aktivem TLS gesendet (hstsEnabled), um Aussperrung vor der
// Zertifikatsaufschaltung zu vermeiden.
func securityHeaders(hstsEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Content-Security-Policy", contentSecurityPolicy)
			if hstsEnabled {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}
