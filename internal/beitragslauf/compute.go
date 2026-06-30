// Package beitragslauf berechnet und exportiert SEPA-Beitragsläufe.
//
// Bewusste Vereinfachungen: Fälligkeit immer 01.07., alle Spieler als Kinder
// eingestuft (keine Volljährigkeits-/Ausbildungsprüfung), alle Einzüge RCUR.
// Der Jahresbeitrag wird exakt halbiert (keine monatsgenaue Pro-rata-Berechnung),
// wenn das Mitglied unterjährig ein- oder austritt oder die Saison das erste
// Abrechnungsjahr des Vereins ist (siehe halfFee).
package beitragslauf

import (
	"strings"
	"time"
)

// SeasonInfo beschreibt das Abrechnungsjahr für Satz-Stichtag und Halbierung.
type SeasonInfo struct {
	Label     string
	Start     time.Time // start_date der Saison
	End       time.Time // end_date der Saison
	Stichtag  time.Time // 01.07. des Startjahres (Fälligkeit + Satz-Stichtag)
	Inaugural bool      // erstes Abrechnungsjahr des Vereins → alle zahlen halb
}

// halfFee bestimmt, ob der Jahresbeitrag halbiert wird, und den Grund.
// Priorität: erstjahr → eintritt → austritt. Die Ermäßigungen stapeln NICHT —
// es wird höchstens einmal halbiert.
func halfFee(m MemberRow, s SeasonInfo) (bool, string) {
	if s.Inaugural {
		return true, "erstjahr"
	}
	if inWindow(m.JoinDate, s.Start, s.End) {
		return true, "eintritt"
	}
	if m.Status == "ausgetreten" && inWindow(m.ExitDate, s.Start, s.End) {
		return true, "austritt"
	}
	return false, ""
}

// inWindow prüft, ob ein ISO-Datum (YYYY-MM-DD) inklusive beider Grenzen im
// Fenster [start, end] liegt. Leeres/ungültiges Datum → false.
func inWindow(date string, start, end time.Time) bool {
	if len(date) < 10 {
		return false
	}
	d, err := time.Parse("2006-01-02", date[:10])
	if err != nil {
		return false
	}
	return !d.Before(start) && !d.After(end)
}

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

// Mitgliedsvereine ist die hardcodierte Whitelist der 8 Vereine.
//
// Deprecated: Die Stammvereine werden seit Migration 047 in der Tabelle
// stammvereine verwaltet; der Beitragslauf leitet die Kategorie deterministisch
// aus members.home_club_id ab. Diese Liste und MatchHomeClub werden im Lauf
// nicht mehr aufgerufen und bleiben nur als einmaliges Migrations-Hilfsmittel
// (Freitext → home_club_id) erhalten.
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
//
// Deprecated: nur noch als einmaliges Migrations-Hilfsmittel. Der Beitragslauf
// nutzt members.home_club_id (siehe Mitgliedsvereine).
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
