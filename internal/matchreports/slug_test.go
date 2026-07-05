package matchreports

import (
	"testing"
	"time"
)

func TestTitleSlug_MatchesNachbarContract(t *testing.T) {
	// Fixture aus spike-match-report-import/fixture-payload.json:
	// slug ist NUR das title-Segment, kein /spielberichte/YYYY-YYYY/-Präfix.
	got := TitleSlug("Spike-Test — TWS mA vs. VfL Kirchheim")
	if got != "spike-test-tws-ma-vs-vfl-kirchheim" {
		t.Errorf("TitleSlug = %q, want spike-test-tws-ma-vs-vfl-kirchheim", got)
	}
}

func TestSlugify_UmlautsAndSpaces(t *testing.T) {
	cases := map[string]string{
		"Über die Alb":             "ueber-die-alb",
		"TWS mA vs. VfL Kirchheim": "tws-ma-vs-vfl-kirchheim",
		"Straße & Sonne":           "strasse-sonne",
		"":                         "bericht", // Fallback
		"   ---   ":                "bericht",
	}
	for in, want := range cases {
		got := slugify(in)
		if got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadSeasonRange_UsesSeasonDates(t *testing.T) {
	r := LoadSeasonRange("2025-08-01", "2026-06-30", 0)
	if r.StartYear != 2025 || r.EndYear != 2026 || r.Fallback {
		t.Errorf("expected 2025-2026 non-fallback, got %+v", r)
	}
	if got := r.SeasonSegment(); got != "2025-2026" {
		t.Errorf("SeasonSegment = %s", got)
	}
}

func TestLoadSeasonRange_FallbackAfterJuly(t *testing.T) {
	ts := time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC).Unix()
	r := LoadSeasonRange("", "", ts)
	if r.StartYear != 2026 || r.EndYear != 2027 || !r.Fallback {
		t.Errorf("expected 2026-2027 fallback, got %+v", r)
	}
}

func TestLoadSeasonRange_FallbackBeforeJuly(t *testing.T) {
	ts := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC).Unix()
	r := LoadSeasonRange("", "", ts)
	if r.StartYear != 2025 || r.EndYear != 2026 || !r.Fallback {
		t.Errorf("expected 2025-2026 fallback, got %+v", r)
	}
}

func TestFormatMatchScore(t *testing.T) {
	i := func(n int) *int { return &n }
	if got := FormatMatchScore(i(24), i(22), i(12), i(9), false); got != "24:22 (12:9)" {
		t.Errorf("full score: %q", got)
	}
	if got := FormatMatchScore(i(24), i(22), nil, nil, false); got != "24:22" {
		t.Errorf("no HT: %q", got)
	}
	if got := FormatMatchScore(i(24), i(22), i(12), i(9), true); got != "" {
		t.Errorf("tournament: %q", got)
	}
	if got := FormatMatchScore(nil, i(22), nil, nil, false); got != "" {
		t.Errorf("incomplete: %q", got)
	}
}
