package bwhv

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Kopfzeile jeder Seite: "mB-RL-BW, Spiel Nr. 905272 am 20.09.26".
var reTitle = regexp.MustCompile(`^(\S+),\s*Spiel Nr\.\s*(\d+)\s+am\s+(\d{2}\.\d{2}\.\d{2})`)

// "Spiel/Datum 905272 , am 20.09.26 um 16:00h"
var reGameDate = regexp.MustCompile(`(\d+)\s*,\s*am\s+(\d{2}\.\d{2}\.\d{2})\s+um\s+(\d{1,2}:\d{2})`)

// "Spielort Erich-Bamberger Stadthalle in Östringen (21005)"
var reVenue = regexp.MustCompile(`^(.*?)\s+in\s+(.*?)\s*\((\d+)\)\s*$`)

// "Endstand 29:25 (14:13) , Sieger ..." — der Halbzeitstand steht in Klammern.
var reScore = regexp.MustCompile(`(\d+):(\d+)\s*\((\d+):(\d+)\)`)

// parseHeader liest die Kopfdaten von Seite 1.
func parseHeader(d *document) (Header, error) {
	var h Header

	for _, l := range d.Lines {
		if m := reTitle.FindStringSubmatch(strings.TrimSpace(l.Text())); m != nil {
			h.Staffel = m[1]
			h.GameNo = m[2]
			h.Date = normalizeDate(m[3])
			break
		}
	}
	if h.GameNo == "" {
		return h, fmt.Errorf("spielbericht ohne erkennbare Kopfzeile (Staffel, Spiel Nr., Datum)")
	}

	for _, l := range d.Lines {
		if l.Page != 1 {
			continue
		}
		txt := strings.TrimSpace(l.Text())
		switch {
		case strings.HasPrefix(txt, "Spiel/Datum"):
			if m := reGameDate.FindStringSubmatch(txt); m != nil {
				h.GameNo = m[1]
				h.Date = normalizeDate(m[2])
				h.Time = m[3]
			}
		case strings.HasPrefix(txt, "Spielort"):
			rest := strings.TrimSpace(strings.TrimPrefix(txt, "Spielort"))
			if m := reVenue.FindStringSubmatch(rest); m != nil {
				h.VenueName = strings.TrimSpace(m[1])
				h.VenueTown = strings.TrimSpace(m[2])
				h.HallNumber = m[3]
			}
		case strings.HasPrefix(txt, "Heim - Gast"):
			rest := strings.TrimSpace(strings.TrimPrefix(txt, "Heim - Gast"))
			if home, guest, ok := splitTeams(rest); ok {
				h.HomeTeam, h.GuestTeam = home, guest
			}
		case strings.HasPrefix(txt, "Endstand"):
			if m := reScore.FindStringSubmatch(txt); m != nil {
				h.HomeGoals, _ = strconv.Atoi(m[1])
				h.GuestGoals, _ = strconv.Atoi(m[2])
				h.HomeGoalsHT, _ = strconv.Atoi(m[3])
				h.GuestGoalsHT, _ = strconv.Atoi(m[4])
			}
			if i := strings.Index(txt, "Zuschauer:"); i >= 0 {
				h.Spectators = strings.TrimSpace(txt[i+len("Zuschauer:"):])
			}
		}
	}
	if h.HomeTeam == "" || h.GuestTeam == "" {
		return h, fmt.Errorf("spielbericht ohne erkennbare Mannschaften in der Zeile \"Heim - Gast\"")
	}
	h.Referees = parseReferees(d)
	return h, nil
}

// splitTeams trennt "Heim - Gast". Der Trenner ist " - " mit Leerzeichen —
// Vereinsnamen tragen selbst Bindestriche ("Rhein-Neckar Löwen 2"), ein
// Split auf "-" zerschnitte sie.
func splitTeams(s string) (home, guest string, ok bool) {
	parts := strings.SplitN(s, " - ", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	home = strings.TrimSpace(parts[0])
	guest = strings.TrimSpace(parts[1])
	return home, guest, home != "" && guest != ""
}

// parseReferees liest die Namenszeile unter "Schiedsrichter". Fehlen die
// Namen, steht dort "N.N. N.N." — das wird als leer behandelt.
func parseReferees(d *document) string {
	idx, ok := d.findLine("Schiedsrichter")
	if !ok {
		return ""
	}
	for i := idx; i < len(d.Lines) && i < idx+6; i++ {
		txt := strings.TrimSpace(d.Lines[i].Text())
		if !strings.HasPrefix(txt, "Name") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(txt, "Name"))
		if isPlaceholderName(rest) {
			return ""
		}
		return rest
	}
	return ""
}

// isPlaceholderName erkennt die Platzhalter des Dienstes für nicht gemeldete
// Personen. Sie dürfen nie als Spieler oder Schiedsrichter gelten.
func isPlaceholderName(s string) bool {
	t := strings.TrimSpace(s)
	if t == "" {
		return true
	}
	return strings.ReplaceAll(strings.ReplaceAll(t, "N.N.", ""), " ", "") == ""
}
