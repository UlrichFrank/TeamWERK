package h4aimport

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	// Zeilen-Anker: jede Spielzeile beginnt mit <tr id="game<internalId>">. Die
	// H4A-Zeilen sind NICHT sauber mit </tr> geschlossen und enthalten eine
	// verschachtelte <table> im Buttons-td — deshalb ist die game-ID der einzige
	// robuste Anker (nicht generisches <tr>/<td>-Matching). Siehe design.md §9.
	reGameRow = regexp.MustCompile(`<tr id="game(\d+)">`)
	// td-Zellen einer Zeile (non-greedy, DOTALL für mehrzeilige Zellen).
	reCell = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	// Alle HTML-Tags (zum Strippen von Zellinhalten).
	reTag = regexp.MustCompile(`<[^>]*>`)
	// <b>-Markierung des EIGENEN Teams.
	reBold = regexp.MustCompile(`(?i)<b>`)
	// Wochentag-Präfix im Datum, z.B. "Sa, ".
	reWeekday = regexp.MustCompile(`^(?:Mo|Di|Mi|Do|Fr|Sa|So),\s*`)
	// DD.MM.YYYY.
	reDate = regexp.MustCompile(`^(\d{2})\.(\d{2})\.(\d{4})$`)
	// Beginn des Buttons-td (td[13]) — ab hier wird die Zeile fürs td-Matching abgeschnitten,
	// damit die verschachtelte <table> das Zell-Matching nicht durcheinanderbringt.
	buttonsMarker = `<table class="ge_buttons_container"`
)

// Spaltenindizes gemäß design.md §1.2.
const (
	colStaffel = 3
	colGameNo  = 4
	colHalle   = 7
	colDate    = 8
	colTime    = 9
	colHome    = 10
	colGuest   = 11
	colComment = 12
	minCells   = 13 // td[0]..td[12] müssen vorhanden sein
)

// ParseGames parst die H4A-Spieltabelle (HTML aus dem gametable_container) in RawGames.
// Defensiv: findet sich keine Spielzeile, ist das ein Format-Bruch (Fehler statt leerer
// Liste) — der Import soll dann sichtbar scheitern statt stiller Teilergebnisse.
func ParseGames(htmlStr string) ([]RawGame, error) {
	anchors := reGameRow.FindAllStringSubmatchIndex(htmlStr, -1)
	if len(anchors) == 0 {
		return nil, errors.New("H4A-Spieltabelle nicht erkannt — Format geändert?")
	}

	games := make([]RawGame, 0, len(anchors))
	for i, a := range anchors {
		// Zeilen-Chunk = von diesem Anker bis zum nächsten Anker (bzw. Dokumentende).
		start := a[0]
		end := len(htmlStr)
		if i+1 < len(anchors) {
			end = anchors[i+1][0]
		}
		chunk := htmlStr[start:end]
		internalID := htmlStr[a[2]:a[3]]

		// Buttons-td (verschachtelte Tabelle) abschneiden.
		if idx := strings.Index(chunk, buttonsMarker); idx >= 0 {
			chunk = chunk[:idx]
		}

		cellMatches := reCell.FindAllStringSubmatch(chunk, -1)
		if len(cellMatches) < minCells {
			return nil, fmt.Errorf("H4A-Spielzeile game%s: unerwartete Spaltenzahl (%d) — Format geändert?", internalID, len(cellMatches))
		}
		cells := make([]string, len(cellMatches))
		for j, m := range cellMatches {
			cells[j] = m[1]
		}

		date, err := normalizeDate(cleanText(cells[colDate]))
		if err != nil {
			return nil, fmt.Errorf("H4A-Spielzeile game%s: %w", internalID, err)
		}

		games = append(games, RawGame{
			InternalID: internalID,
			GameNo:     cleanText(cells[colGameNo]),
			Staffel:    cleanText(cells[colStaffel]),
			HallNumber: cleanText(cells[colHalle]),
			Date:       date,
			Time:       cleanText(cells[colTime]),
			Home:       cleanText(cells[colHome]),
			Guest:      cleanText(cells[colGuest]),
			IsHome:     reBold.MatchString(cells[colHome]),
			Comment:    cleanText(cells[colComment]),
		})
	}
	return games, nil
}

// cleanText entfernt HTML-Tags, dekodiert Entities (&#160;, &Auml; …) und trimmt.
func cleanText(s string) string {
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

// normalizeDate wandelt "Sa, 19.09.2026" → "2026-09-19".
func normalizeDate(s string) (string, error) {
	s = reWeekday.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	m := reDate.FindStringSubmatch(s)
	if m == nil {
		return "", fmt.Errorf("datum %q nicht im Format DD.MM.YYYY", s)
	}
	return fmt.Sprintf("%s-%s-%s", m[3], m[2], m[1]), nil
}
