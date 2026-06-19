// Package beitragslauf berechnet und exportiert SEPA-Beitragsläufe.
//
// Bewusste Vereinfachungen: voller Jahresbeitrag (keine Pro-rata-Berechnung),
// Fälligkeit immer 01.07., alle Spieler als Kinder eingestuft (keine
// Volljährigkeits-/Ausbildungsprüfung), alle Einzüge RCUR.
package beitragslauf

import "strings"

// BeitragsGruppe bildet den Member-Status auf die Beitragsgruppe ab.
// "" bedeutet: nicht einzuziehen (ausgetreten/honorar/anwaerter o. Ä.).
func BeitragsGruppe(status string) string {
	switch status {
	case "aktiv", "verletzt":
		return "aktiv"
	case "pausiert", "passiv":
		return "passiv"
	default:
		return ""
	}
}

// AktivKategorie wählt innerhalb der Aktiv-Gruppe anhand der
// Stammverein-Zugehörigkeit. Volljährigkeit/Ausbildung spielen keine Rolle.
func AktivKategorie(mitStammverein bool) string {
	if mitStammverein {
		return "aktiv_mit"
	}
	return "aktiv_ohne"
}

// Mitgliedsvereine ist die hardcodierte Whitelist der 8 Vereine, deren
// Mitgliedschaft den ermäßigten Stammverein-Beitrag auslöst.
var Mitgliedsvereine = []string{
	"SKG Gablenberg 1884",
	"SKG Stuttgart Max-Eyth-See 1898",
	"SportKultur Stuttgart",
	"Spvgg 1897 Cannstatt",
	"TB Gaisburg 1886",
	"TB Untertürkheim 1888",
	"TSV Stuttgart-Münster 1875/99",
	"TV Cannstatt 1846",
}

// ClubMatch ist das Ergebnis von MatchHomeClub.
type ClubMatch struct {
	Matched   bool
	Canonical string
	Warning   string
}

// NormalizeClubName: lowercase, Whitespace zusammenfassen, Punkte/Bindestriche/
// Schrägstriche entfernen.
func NormalizeClubName(s string) string {
	s = strings.ToLower(s)
	repl := strings.NewReplacer(".", "", "-", "", "/", "")
	s = repl.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// MatchHomeClub ordnet einen home_club-Freitext einem Mitgliedsverein zu.
//   - leer → Matched=false, kein Warning (= ohne Stammverein)
//   - exakter (normalisierter) Treffer → Matched=true, kein Warning
//   - Teilstring- oder Fuzzy-Treffer (Levenshtein ≤ 3) → Matched=true, mit Warning
//   - sonst → Matched=false, mit Warning
func MatchHomeClub(homeClub string) ClubMatch {
	if strings.TrimSpace(homeClub) == "" {
		return ClubMatch{Matched: false}
	}
	norm := NormalizeClubName(homeClub)
	// exakter Treffer
	for _, v := range Mitgliedsvereine {
		if NormalizeClubName(v) == norm {
			return ClubMatch{Matched: true, Canonical: v}
		}
	}
	// Teilstring- oder Fuzzy-Treffer
	for _, v := range Mitgliedsvereine {
		nv := NormalizeClubName(v)
		if strings.Contains(nv, norm) || strings.Contains(norm, nv) || levenshtein(nv, norm) <= 3 {
			return ClubMatch{
				Matched:   true,
				Canonical: v,
				Warning:   "home_club='" + homeClub + "' unsicher zugeordnet zu '" + v + "'",
			}
		}
	}
	return ClubMatch{
		Matched: false,
		Warning: "home_club='" + homeClub + "' konnte keinem Mitgliedsverein zugeordnet werden",
	}
}

// levenshtein berechnet die Editierdistanz (begrenzt auf 50 Zeichen je Eingabe).
func levenshtein(a, b string) int {
	if len(a) > 50 {
		a = a[:50]
	}
	if len(b) > 50 {
		b = b[:50]
	}
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}
