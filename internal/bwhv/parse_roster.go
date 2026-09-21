package bwhv

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// "Heim: Rhein-Neckar Löwen 2" bzw. "Gast: Team Stuttgart"
var reTeamSection = regexp.MustCompile(`^(Heim|Gast):\s*(.+)$`)

// Die Siebenmeter-Spalte trägt "Versuche/Tore", z.B. "3/2" oder "1/0".
var reSevenM = regexp.MustCompile(`^(\d+)\s*/\s*(\d+)$`)

// Eine Zeitangabe in einer Strafspalte ("30:12"). Ihre Anwesenheit zählt, der
// Wert selbst ist der Zeitpunkt und wird hier nicht ausgewertet.
var rePenaltyTime = regexp.MustCompile(`\d{1,3}:\d{2}`)

// parseRosters liest die Mannschaftslisten beider Teams.
func parseRosters(d *document, h Header) (home, guest Roster, err error) {
	headerIdx, ok := d.findLine("Nr.", "Name")
	if !ok {
		return home, guest, fmt.Errorf("spielbericht ohne Kopfzeile der Mannschaftsliste (erwartet \"Nr.\" und \"Name\")")
	}
	model, err := buildColumnModel(d, headerIdx)
	if err != nil {
		return home, guest, err
	}

	home = Roster{Side: "home", TeamName: h.HomeTeam}
	guest = Roster{Side: "guest", TeamName: h.GuestTeam}

	var cur *Roster
	for _, l := range d.Lines {
		txt := strings.TrimSpace(l.Text())
		if m := reTeamSection.FindStringSubmatch(txt); m != nil {
			if m[1] == "Heim" {
				cur = &home
			} else {
				cur = &guest
			}
			if name := strings.TrimSpace(m[2]); name != "" {
				cur.TeamName = name
			}
			continue
		}
		if cur == nil {
			continue
		}
		if p, ok := parseRosterLine(model, l); ok {
			cur.Players = append(cur.Players, p)
		}
	}
	if len(home.Players) == 0 && len(guest.Players) == 0 {
		return home, guest, fmt.Errorf("spielbericht ohne auswertbare Mannschaftslisten")
	}
	return home, guest, nil
}

// parseRosterLine wertet eine Zeile als Spielerzeile aus.
//
// Eine Zeile zählt nur, wenn die Nummernspalte eine Zahl trägt: die
// Betreuerzeilen darunter tragen dort die Buchstaben A bis E und fallen so von
// selbst heraus. Platzhalternamen erzeugen keinen Spieler.
func parseRosterLine(m *columnModel, l textLine) (RosterPlayer, bool) {
	cells := m.cells(l)
	numRaw := strings.TrimSpace(cells[colNumber])
	name := strings.TrimSpace(cells[colName])
	if numRaw == "" || name == "" {
		return RosterPlayer{}, false
	}
	num, err := strconv.Atoi(numRaw)
	if err != nil {
		return RosterPlayer{}, false
	}
	if isPlaceholderName(name) {
		return RosterPlayer{}, false
	}

	p := RosterPlayer{Number: num, Name: name}
	if y, err := strconv.Atoi(strings.TrimSpace(cells[colBirthYear])); err == nil {
		p.BirthYear = y
	}
	if g, err := strconv.Atoi(strings.TrimSpace(cells[colGoals])); err == nil {
		p.Goals = g
	}
	if sm := reSevenM.FindStringSubmatch(strings.TrimSpace(cells[colSevenM])); sm != nil {
		p.SevenMAtt, _ = strconv.Atoi(sm[1])
		p.SevenMGoals, _ = strconv.Atoi(sm[2])
	}
	for _, c := range []column{colPenalty1, colPenalty2, colPenalty3} {
		if rePenaltyTime.MatchString(cells[c]) {
			p.TwoMin++
		}
	}
	if rePenaltyTime.MatchString(cells[colWarning]) {
		p.Warnings = 1
	}
	if rePenaltyTime.MatchString(cells[colDisq]) {
		p.Disq = 1
	}
	return p, true
}
