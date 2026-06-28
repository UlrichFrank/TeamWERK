package timez

import (
	"testing"
	"time"
)

// The whole point of the reminder timezone fix: a stored "15:00" wall-clock time
// must map to the correct absolute instant for Europe/Berlin, which differs by
// DST. If this offset is wrong, every reminder fires off by 1–2h.
func TestParseDT_BerlinDSTOffset(t *testing.T) {
	berlin := Berlin()

	cases := []struct {
		name    string
		date    string
		time    string
		wantUTC string // expected instant in UTC
	}{
		// Summer: CEST = UTC+2 → 15:00 Berlin == 13:00Z
		{"summer CEST", "2026-07-15", "15:00", "2026-07-15T13:00:00Z"},
		// Winter: CET = UTC+1 → 15:00 Berlin == 14:00Z
		{"winter CET", "2026-01-15", "15:00", "2026-01-15T14:00:00Z"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseDT(c.date, c.time, berlin).UTC().Format(time.RFC3339)
			if got != c.wantUTC {
				t.Errorf("ParseDT(%q,%q) = %s UTC, want %s", c.date, c.time, got, c.wantUTC)
			}
		})
	}
}

// modernc.org/sqlite returns DATE columns as full ISO timestamps; the date part
// must be normalised. An empty time defaults to midnight.
func TestParseDT_Normalisation(t *testing.T) {
	berlin := Berlin()

	full := ParseDT("2026-07-15T00:00:00Z", "15:00", berlin)
	if full.UTC().Format(time.RFC3339) != "2026-07-15T13:00:00Z" {
		t.Errorf("ISO-timestamp date not normalised: got %s", full.UTC().Format(time.RFC3339))
	}

	midnight := ParseDT("2026-07-15", "", berlin)
	if h, m := midnight.Hour(), midnight.Minute(); h != 0 || m != 0 {
		t.Errorf("empty time should be 00:00 wall-clock, got %02d:%02d", h, m)
	}
}
