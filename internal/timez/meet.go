package timez

import (
	"database/sql"
	"time"
)

// ParseHHMM parses a "HH:MM" wall-clock string into minutes since midnight.
// ok is false for anything that is not a valid 24-hour time.
func ParseHHMM(s string) (minutes int, ok bool) {
	t, err := time.Parse("15:04", s)
	if err != nil || len(s) != 5 {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

// MeetTime derives a game's meeting time from its kick-off and the stored
// offset (games.meet_offset_minutes, spiel-treffpunkt). The meeting time is
// never stored as a clock time — it is always "kick-off minus offset", so a
// rescheduled kick-off moves it along. This is the single place that does the
// arithmetic; every package that shows a meeting time goes through here.
//
// date may be a full ISO timestamp (SQLite DATE quirk) and is truncated.
// meetDate differs from the game date exactly when the meeting falls before
// midnight of the game day. ok is false when no offset is set or the inputs
// do not parse.
func MeetTime(date, startHHMM string, offset sql.NullInt64) (meetDate, meetHHMM string, ok bool) {
	if !offset.Valid {
		return "", "", false
	}
	if len(date) > 10 {
		date = date[:10]
	}
	start, okStart := ParseHHMM(startHHMM)
	day, err := time.Parse("2006-01-02", date)
	if !okStart || err != nil {
		return "", "", false
	}
	// Naive minute arithmetic in UTC: the result is a wall-clock label, not an
	// instant, so a DST switch on the game day must not shift it.
	meet := day.Add(time.Duration(start-int(offset.Int64)) * time.Minute)
	return meet.Format("2006-01-02"), meet.Format("15:04"), true
}
