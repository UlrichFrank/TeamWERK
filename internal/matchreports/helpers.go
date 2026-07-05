package matchreports

import (
	"net/http"
	"strconv"
)

// parsePathID liest r.PathValue(name) und interpretiert es als int > 0.
// Rückgabe (0, false) bedeutet: fehlend oder ungültig — Handler sollte 400 liefern.
func parsePathID(r *http.Request, name string) (int, bool) {
	raw := r.PathValue(name)
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
