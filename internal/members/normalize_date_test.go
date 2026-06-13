package members

import "testing"

func TestNormalizeDate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Two-digit year < 68 → 20xx
		{"01.03.25", "2025-03-01"},
		{"10.05.30", "2030-05-10"},
		{"31.12.67", "2067-12-31"},
		// Two-digit year >= 68 → 19xx
		{"15.07.72", "1972-07-15"},
		{"01.01.68", "1968-01-01"},
		{"31.12.99", "1999-12-31"},
		// Four-digit year unchanged
		{"10.05.2030", "2030-05-10"},
		{"01.01.1985", "1985-01-01"},
		// Already ISO — unchanged
		{"2024-06-15", "2024-06-15"},
		// Unrecognized — unchanged
		{"", ""},
		{"foobar", "foobar"},
	}
	for _, tt := range tests {
		got := normalizeDate(tt.input)
		if got != tt.want {
			t.Errorf("normalizeDate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
