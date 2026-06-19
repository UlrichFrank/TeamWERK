package sepa

import "testing"

func TestIsValidIBAN(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"DE gültig", "DE89370400440532013000", true},
		{"DE gültig mit Leerzeichen", "DE89 3704 0044 0532 0130 00", true},
		{"DE gültig kleingeschrieben", "de89370400440532013000", true},
		{"AT gültig", "AT611904300234573201", true},
		{"CH gültig", "CH9300762011623852957", true},
		{"DE falsche Prüfsumme", "DE88370400440532013000", false},
		{"DE zu kurz", "DE8937040044", false},
		{"Müll", "NICHTSGUELTIGES", false},
		{"leer", "", false},
		{"Ziffern statt Ländercode", "1289370400440532013000", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsValidIBAN(c.in); got != c.want {
				t.Errorf("IsValidIBAN(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestNormalizeIBAN(t *testing.T) {
	if got := NormalizeIBAN("  de89 3704 0044 "); got != "DE8937040044" {
		t.Errorf("NormalizeIBAN = %q", got)
	}
}
