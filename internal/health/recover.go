package health

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync/atomic"
)

// panicsTotal zählt abgefangene HTTP-Handler-Panics seit Prozessstart. Es ist das
// neutrale Tier-3-Signal (exponiert als teamwerk_panics_total); die App alarmiert
// selbst nicht.
var panicsTotal atomic.Int64

// PanicsTotal liefert die Anzahl bisher abgefangener Panics.
func PanicsTotal() int64 { return panicsTotal.Load() }

// Recoverer ersetzt chi.Recoverer: fängt Panics ab, loggt sie strukturiert
// (slog, event="panic", mit Stacktrace), inkrementiert teamwerk_panics_total und
// antwortet mit 500 — ohne den Prozess zu beenden und ohne anbieter-spezifische
// Benachrichtigung. http.ErrAbortHandler wird wie in chi durchgereicht.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}
			panicsTotal.Add(1)
			slog.Error("panic recovered",
				"event", "panic",
				"error", fmt.Sprint(rec),
				"method", r.Method,
				"path", r.URL.Path,
				"stack", string(debug.Stack()),
			)
			w.WriteHeader(http.StatusInternalServerError)
		}()
		next.ServeHTTP(w, r)
	})
}
