package health

import (
	"errors"
	"net/http"
	"strings"
	"sync/atomic"

	"modernc.org/sqlite"
)

// httpInFlight zählt laufende HTTP-Requests. Über teamwerk_http_requests_in_flight
// exponiert; nützlich, um CPU-Spitzen gegen Traffic-Last zu deuten.
var httpInFlight atomic.Int64

// sqliteBusyTotal zählt im HTTP-Pfad beobachtete SQLITE_BUSY-Returns. Über
// teamwerk_sqlite_busy_total exponiert; Frühwarn-Signal für Schreibkonkurrenz.
// Scheduler-Pfade zählen nicht hier (separater Prozess) — sie emittieren statt-
// dessen ein slog.Warn(event="sqlite_busy", source="scheduler").
var sqliteBusyTotal atomic.Int64

// HTTPInFlight liefert die Anzahl gerade laufender HTTP-Requests.
func HTTPInFlight() int64 { return httpInFlight.Load() }

// SQLiteBusyTotal liefert die Anzahl beobachteter SQLITE_BUSY-Returns im HTTP-Pfad.
func SQLiteBusyTotal() int64 { return sqliteBusyTotal.Load() }

// RecordSQLiteBusy inkrementiert den BUSY-Counter unbedingt. Nutzbar für Tests
// und für Caller, die BUSY bereits selbst klassifiziert haben.
func RecordSQLiteBusy() { sqliteBusyTotal.Add(1) }

// IsSQLiteBusy prüft, ob err einen SQLITE_BUSY-Return des SQLite-Treibers
// repräsentiert. Reine Erkennung ohne Side-Effect — geeignet für Pfade, die
// BUSY anders signalisieren wollen (z. B. der Scheduler-Prozess via slog).
func IsSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	var sErr *sqlite.Error
	if errors.As(err, &sErr) {
		// SQLITE_BUSY = 5; SQLITE_BUSY_RECOVERY/SNAPSHOT/TIMEOUT haben den
		// Primary-Code 5 in den unteren 8 Bits.
		if sErr.Code()&0xff == 5 {
			return true
		}
	}
	// Fallback für Wrapped/Stringified-Fehler (z. B. fmt.Errorf("...: %w")).
	return strings.Contains(err.Error(), "SQLITE_BUSY") || strings.Contains(err.Error(), "database is locked")
}

// CheckSQLiteBusy ist die HTTP-Pfad-Variante von IsSQLiteBusy: erkennt BUSY und
// inkrementiert teamwerk_sqlite_busy_total. Rückgabe true bei BUSY.
//
// Konvention: Mutations-Handler rufen das im DB-Error-Pfad einmal auf, bevor
// der Error in die Response geht.
func CheckSQLiteBusy(err error) bool {
	if IsSQLiteBusy(err) {
		sqliteBusyTotal.Add(1)
		return true
	}
	return false
}

// InFlightMiddleware zählt laufende HTTP-Requests. Sie muss VOR der Recover-
// Middleware in die Chi-Kette eingehängt werden, damit Panic-Recovery den
// defer-Dekrement nicht überspringt.
func InFlightMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpInFlight.Add(1)
		defer httpInFlight.Add(-1)
		next.ServeHTTP(w, r)
	})
}
