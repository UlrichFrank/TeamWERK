package matchreports

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ErrInvalidSeasonName wird von ParseSeasonName zurückgegeben, wenn der
// Saison-Name nicht dem erwarteten Format "YYYY/YY" entspricht oder das
// End-Jahr nicht auf Start-Jahr+1 folgt.
var ErrInvalidSeasonName = errors.New("invalid season name")

var seasonNameRe = regexp.MustCompile(`^(\d{4})/(\d{2})$`)

// ParseSeasonName wandelt einen Saison-Namen im Format "YYYY/YY" in das
// Slug-Segment "YYYY-YYYY" um. Beispiele:
//
//	"2026/27" -> "2026-2027"
//	"1999/00" -> "1999-2000" (Jahrhundert-Wechsel)
//
// Der zweite Teil muss Start-Jahr+1 modulo 100 entsprechen; ansonsten wird
// ErrInvalidSeasonName gewrappt zurückgegeben.
func ParseSeasonName(name string) (string, error) {
	m := seasonNameRe.FindStringSubmatch(name)
	if m == nil {
		return "", fmt.Errorf("%w: %q", ErrInvalidSeasonName, name)
	}
	startYear, err := strconv.Atoi(m[1])
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrInvalidSeasonName, name)
	}
	endTwo, err := strconv.Atoi(m[2])
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrInvalidSeasonName, name)
	}
	expectedEndTwo := (startYear + 1) % 100
	if endTwo != expectedEndTwo {
		return "", fmt.Errorf("%w: %q (end year must be start+1)", ErrInvalidSeasonName, name)
	}
	// Century-Handling: Wenn End-Jahres-Kurzform < Start-Jahres-Kurzform,
	// dann Jahrhundertwechsel (z. B. 1999/00 -> 2000).
	endYear := (startYear/100)*100 + endTwo
	if endTwo < startYear%100 {
		endYear += 100
	}
	return fmt.Sprintf("%d-%d", startYear, endYear), nil
}

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

// TitleSlug ist das letzte Pfad-Segment für die TYPO3-Seite, aus dem
// Bericht-Titel abgeleitet (Kleinbuchstaben, keine Umlaute, keine
// Sonderzeichen außer '-'). Den vollen Pfad
// /spielberichte/{YYYY-YYYY}/{title-slug} baut die Extension aus season+slug
// selbst; TeamWERK schickt nur beide Segmente einzeln.
func TitleSlug(title string) string {
	return slugify(title)
}

// BuildTitle erzeugt einen Default-Titel aus dem Gegnernamen, falls der Autor
// keinen expliziten Titel angegeben hat. Ohne Datums-Präfix — das Datum
// steht sowieso im Meta-Bereich des Berichts und würde im Titel doppelt und
// unnötig sperrig auftauchen.
func BuildTitle(opponent string) string {
	return opponent
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
