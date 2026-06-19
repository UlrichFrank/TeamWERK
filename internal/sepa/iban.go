// Package sepa enthält Hilfsfunktionen für SEPA-Lastschriften
// (IBAN-Validierung, später XML-Erzeugung).
package sepa

import (
	"strings"
	"unicode"
)

// ibanLength bildet den ISO-Ländercode auf die erwartete IBAN-Gesamtlänge ab.
// Praktisch relevant für TeamWERK sind DE/AT/CH; weitere gängige SEPA-Länder
// sind der Vollständigkeit halber enthalten.
var ibanLength = map[string]int{
	"DE": 22, "AT": 20, "CH": 21, "LI": 21,
	"FR": 27, "IT": 27, "ES": 24, "NL": 18,
	"BE": 16, "LU": 20, "DK": 18, "PL": 28,
}

// NormalizeIBAN entfernt Leerzeichen und wandelt in Großbuchstaben um.
func NormalizeIBAN(s string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
}

// IsValidIBAN prüft Ländercode-spezifische Länge, erlaubte Zeichen und die
// Mod-97-Prüfsumme (ISO 13616 / ISO 7064).
func IsValidIBAN(iban string) bool {
	iban = NormalizeIBAN(iban)
	if len(iban) < 5 {
		return false
	}
	country := iban[:2]
	if !isUpperAlpha(country[0]) || !isUpperAlpha(country[1]) {
		return false
	}
	if want, ok := ibanLength[country]; ok && len(iban) != want {
		return false
	}
	// Nur Buchstaben/Ziffern zulässig.
	for _, r := range iban {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return mod97(iban) == 1
}

func isUpperAlpha(b byte) bool { return b >= 'A' && b <= 'Z' }

// mod97 setzt die ersten vier Zeichen ans Ende, ersetzt Buchstaben durch
// Zahlen (A=10 … Z=35) und berechnet den Rest modulo 97 stückweise.
func mod97(iban string) int {
	rearranged := iban[4:] + iban[:4]
	remainder := 0
	for _, r := range rearranged {
		var val int
		switch {
		case r >= '0' && r <= '9':
			val = int(r - '0')
		case r >= 'A' && r <= 'Z':
			val = int(r-'A') + 10
		default:
			return -1
		}
		if val >= 10 {
			remainder = (remainder*100 + val) % 97
		} else {
			remainder = (remainder*10 + val) % 97
		}
	}
	return remainder
}
