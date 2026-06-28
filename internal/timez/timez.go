// Package timez centralises TeamWERK's wall-clock interpretation of stored event
// dates and times.
//
// Events (games, trainings, duty slots) are stored as a naive DATE column plus a
// "HH:MM" TEXT column and are ALWAYS meant as Europe/Berlin wall-clock time —
// there is no timezone metadata in the database and no multi-timezone use case
// (see OpenSpec change timezone-correct-event-reminders). Any code that needs to
// turn such a stored value into an absolute instant (e.g. the reminder scheduler
// deciding whether an event is "in 3h") MUST go through here, so the Berlin
// interpretation lives in exactly one place. This was extracted from the iCal
// export (internal/calendar) without changing its behaviour.
package timez

import (
	"sync"
	"time"
	// Embed the IANA tz database so Berlin() works even on a minimal VPS image
	// that ships no system zoneinfo, and regardless of which other packages are
	// linked into the binary (the scheduler runs as a subcommand of the same
	// binary but must not rely on internal/calendar being on its import path).
	_ "time/tzdata"
)

var (
	berlinOnce sync.Once
	berlinLoc  *time.Location
)

// Berlin returns the Europe/Berlin location, loaded once and cached. If the
// system tz database is unavailable it falls back to UTC (must not happen on the
// VPS, but keeps the helper total rather than panicking).
func Berlin() *time.Location {
	berlinOnce.Do(func() {
		loc, err := time.LoadLocation("Europe/Berlin")
		if err != nil {
			loc = time.UTC
		}
		berlinLoc = loc
	})
	return berlinLoc
}

// ParseDT interprets a stored date and a "HH:MM" time string as wall-clock time
// in loc and returns the corresponding instant.
//
// It tolerates two quirks of the persisted data:
//   - modernc.org/sqlite returns DATE columns as full ISO timestamps
//     ("2026-08-15T00:00:00Z"), not "2026-08-15" — the date part is normalised.
//   - an empty time string is treated as midnight ("00:00").
func ParseDT(date, timeStr string, loc *time.Location) time.Time {
	if len(date) > 10 {
		date = date[:10]
	}
	if timeStr == "" {
		timeStr = "00:00"
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+timeStr, loc)
	if err != nil {
		t, _ = time.ParseInLocation("2006-01-02", date, loc)
	}
	return t
}
