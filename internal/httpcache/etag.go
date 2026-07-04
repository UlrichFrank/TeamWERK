// Package httpcache bündelt die ETag-/304-Behandlung für quasi-statische
// Referenz-Routen. Foundation-Package: importiert keine Domain-Packages
// (siehe internal/arch/arch_test.go).
//
// Zwei Einstiegspunkte:
//
//   - Serve: für Routen mit günstig vorab berechenbarem Fingerprint (z. B.
//     Hash des Key-Materials bei /api/encryption-pubkey). Bei einem
//     If-None-Match-Treffer wird der Body gar nicht erst gebaut.
//   - ServeJSON: für Referenz-Listen ohne updated_at-Spalte — der schwache
//     ETag wird aus dem serialisierten Response-Body abgeleitet. Das deckt
//     Insert/Update/Delete gleichermaßen ab und ist für nutzergefilterte
//     Antworten (z. B. /api/teams) inhärent korrekt, weil jeder Nutzer den
//     ETag seiner eigenen Antwort bekommt. Die Ersparnis eines 304 ist die
//     Payload, nicht der Round-Trip (Design efficient-data-loading-quickwins).
package httpcache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ETagFor liefert einen schwachen ETag aus einem beliebigen Fingerprint
// (z. B. Key-Material oder serialisierter Body).
func ETagFor(fingerprint []byte) string {
	sum := sha256.Sum256(fingerprint)
	return fmt.Sprintf(`W/"%x"`, sum[:8])
}

// noneMatch meldet, ob der If-None-Match-Header der Anfrage den ETag trifft.
// Unterstützt kommaseparierte Listen und den Wildcard "*" (RFC 9110 §13.1.2).
func noneMatch(r *http.Request, etag string) bool {
	header := r.Header.Get("If-None-Match")
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

// Serve setzt ETag und (falls nicht leer) Cache-Control, beantwortet einen
// passenden If-None-Match mit 304 Not Modified (leerer Body) und ruft nur bei
// einem Miss body() auf, dessen Ergebnis als JSON geschrieben wird.
func Serve(w http.ResponseWriter, r *http.Request, etag, cacheControl string, body func() any) {
	writeCacheHeaders(w, etag, cacheControl)
	if noneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body())
}

// ServeJSON serialisiert v, leitet den schwachen ETag aus dem Payload ab und
// antwortet mit 304 bei passendem If-None-Match, sonst mit dem vollen Body.
func ServeJSON(w http.ResponseWriter, r *http.Request, cacheControl string, v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	etag := ETagFor(payload)
	writeCacheHeaders(w, etag, cacheControl)
	if noneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

func writeCacheHeaders(w http.ResponseWriter, etag, cacheControl string) {
	w.Header().Set("ETag", etag)
	if cacheControl != "" {
		w.Header().Set("Cache-Control", cacheControl)
	}
}
