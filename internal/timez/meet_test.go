package timez

import (
	"database/sql"
	"testing"
)

func off(n int64) sql.NullInt64 { return sql.NullInt64{Int64: n, Valid: true} }

func TestMeetTime_GleicherTag(t *testing.T) {
	d, hm, ok := MeetTime("2026-10-11T00:00:00Z", "15:00", off(90))
	if !ok || d != "2026-10-11" || hm != "13:30" {
		t.Fatalf("got %q %q %v, want 2026-10-11 13:30", d, hm, ok)
	}
}

func TestMeetTime_Vortag(t *testing.T) {
	d, hm, ok := MeetTime("2026-10-11", "06:00", off(420))
	if !ok || d != "2026-10-10" || hm != "23:00" {
		t.Fatalf("got %q %q %v, want 2026-10-10 23:00", d, hm, ok)
	}
}

func TestMeetTime_KeinAbstand(t *testing.T) {
	if _, _, ok := MeetTime("2026-10-11", "15:00", sql.NullInt64{}); ok {
		t.Fatal("ohne Abstand darf keine Treffzeit entstehen")
	}
}

// Am Tag der Zeitumstellung darf die Treffzeit nicht um eine Stunde springen:
// sie ist eine Wandzeit-Angabe, kein Zeitpunkt.
func TestMeetTime_Zeitumstellung(t *testing.T) {
	_, hm, ok := MeetTime("2026-10-25", "15:00", off(90))
	if !ok || hm != "13:30" {
		t.Fatalf("got %q, want 13:30", hm)
	}
}

func TestParseHHMM(t *testing.T) {
	if m, ok := ParseHHMM("13:30"); !ok || m != 810 {
		t.Fatalf("13:30 → %d %v", m, ok)
	}
	for _, bad := range []string{"", "1:30", "25:00", "13:60", "abc", "13:30:00"} {
		if _, ok := ParseHHMM(bad); ok {
			t.Errorf("%q sollte ungültig sein", bad)
		}
	}
}
