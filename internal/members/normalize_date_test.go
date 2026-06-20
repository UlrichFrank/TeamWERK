package members

import "testing"

func TestNormalizeDate(t *testing.T) {
	// Fester Pivot-Jahr 2026 → deterministisch unabhängig von der Systemzeit.
	const currentYear = 2026
	tests := []struct {
		input string
		want  string
	}{
		// 2-stelliges Jahr, nicht in der Zukunft → 20xx
		{"01.03.25", "2025-03-01"},
		{"06.12.26", "2026-12-06"},
		// 2-stelliges Jahr, läge sonst in der Zukunft → 19xx
		{"10.05.30", "1930-05-10"},
		{"31.12.67", "1967-12-31"}, // Götz: 67 → 1967, nicht 2067
		{"06.12.67", "1967-12-06"},
		{"15.07.72", "1972-07-15"},
		{"01.01.68", "1968-01-01"},
		{"31.12.99", "1999-12-31"},
		// Vierstelliges Jahr unverändert (auch zukünftige Termine bleiben erhalten)
		{"10.05.2030", "2030-05-10"},
		{"01.01.1985", "1985-01-01"},
		// Bereits ISO — unverändert
		{"2024-06-15", "2024-06-15"},
		// Unbekannt — unverändert
		{"", ""},
		{"foobar", "foobar"},
	}
	for _, tt := range tests {
		got := normalizeDateAt(tt.input, currentYear)
		if got != tt.want {
			t.Errorf("normalizeDateAt(%q, %d) = %q, want %q", tt.input, currentYear, got, tt.want)
		}
	}
}
