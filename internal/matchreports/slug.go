package matchreports

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// SeasonRange liefert das Saisonfenster für ein Spiel.
// Bei fehlender Saison-Referenz wird per Fallback aus match_date das
// Sommer-Sommer-Fenster gebildet (heuristisch; siehe design.md D-8).
type SeasonRange struct {
	StartYear int
	EndYear   int
	Fallback  bool // true = kein DB-Match, aus match_date geraten
}

// LoadSeasonRange ermittelt den Season-Range für ein Spiel. Wenn games.season_id
// eine gültige Saison verweist, wird deren start_date/end_date verwendet.
// Sonst Fallback aus matchDateUnix.
func LoadSeasonRange(seasonStart, seasonEnd string, matchDateUnix int64) SeasonRange {
	if seasonStart != "" && seasonEnd != "" {
		if s, err := time.Parse("2006-01-02", seasonStart); err == nil {
			if e, err := time.Parse("2006-01-02", seasonEnd); err == nil {
				return SeasonRange{StartYear: s.Year(), EndYear: e.Year()}
			}
		}
	}
	m := time.Unix(matchDateUnix, 0).UTC()
	if m.Month() >= time.July {
		return SeasonRange{StartYear: m.Year(), EndYear: m.Year() + 1, Fallback: true}
	}
	return SeasonRange{StartYear: m.Year() - 1, EndYear: m.Year(), Fallback: true}
}

// SeasonSegment formatiert den Season-Anteil des Slugs: „2025-2026".
func (r SeasonRange) SeasonSegment() string {
	return fmt.Sprintf("%d-%d", r.StartYear, r.EndYear)
}

// BuildSlug baut den vollständigen TYPO3-slug für einen Bericht:
//
//	/spielberichte/{YYYY}-{YYYY}/{title-slug}
//
// title-slug wird aus dem Bericht-Titel abgeleitet (Kleinbuchstaben, keine
// Umlaute, keine Sonderzeichen außer '-').
func BuildSlug(season SeasonRange, title string) string {
	return fmt.Sprintf("/spielberichte/%s/%s", season.SeasonSegment(), slugify(title))
}

// BuildTitle erzeugt einen Default-Titel aus Datum + Gegner, falls der Autor
// keinen expliziten Titel angegeben hat.
func BuildTitle(matchDateUnix int64, opponent string) string {
	d := time.Unix(matchDateUnix, 0).UTC().Format("02.01.2006")
	return fmt.Sprintf("%s — %s", d, opponent)
}

// slugify normalisiert einen String zu einem TYPO3-kompatiblen slug-Segment.
// - Umlaute (äöüÄÖÜß) werden expandiert
// - Kleinbuchstaben
// - Nicht-Alphanumerik → '-'
// - Führende/anhängende und doppelte '-' entfernt
func slugify(s string) string {
	s = strings.ToLower(s)
	s = replaceUmlauts(s)

	var b strings.Builder
	prevDash := true
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "bericht"
	}
	return out
}

var umlautReplacer = strings.NewReplacer(
	"ä", "ae", "ö", "oe", "ü", "ue",
	"Ä", "ae", "Ö", "oe", "Ü", "ue",
	"ß", "ss",
	"é", "e", "è", "e", "ê", "e",
	"à", "a", "á", "a", "â", "a",
)

func replaceUmlauts(s string) string {
	return umlautReplacer.Replace(s)
}

// FormatMatchScore baut den TYPO3-`match_score`-String aus den strukturierten
// Feldern. Beispiel: home=24, away=22, ht_home=12, ht_away=9 → "24:22 (12:9)".
// Bei fehlenden Zahlen oder Turnier: leer.
func FormatMatchScore(home, away, htHome, htAway *int, tournament bool) string {
	if tournament {
		return ""
	}
	if home == nil || away == nil {
		return ""
	}
	base := fmt.Sprintf("%d:%d", *home, *away)
	if htHome != nil && htAway != nil {
		return fmt.Sprintf("%s (%d:%d)", base, *htHome, *htAway)
	}
	return base
}
