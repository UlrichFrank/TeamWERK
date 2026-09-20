package bwhv

import (
	"fmt"
	"sort"
	"strings"
)

// ParseError meldet einen Bericht, der nicht verwertbar ist. Der Aufrufer
// speichert daraufhin state='parse_failed' mit Reason und legt KEINE
// Spieler- oder Verlaufszeilen an.
type ParseError struct{ Reason string }

func (e *ParseError) Error() string { return e.Reason }

// ParseReport wertet ein Spielbericht-PDF vollständig aus.
//
// Die Kreuzprobe ist gestuft (design.md §5.3):
//
//	Kopf-Endstand  ≠  Summe der Tor-Ereignisse   →  harter Fehlschlag
//	Mannschaftsliste ≠ Zählung aus dem Verlauf   →  Warnung, Bericht gilt
//
// Der Unterschied ist keine Feinheit: weicht der Kopf vom Verlauf ab, hat der
// Parser ein Ereignis verloren und das Ergebnis wäre in jedem Fall falsch.
// Weicht die Liste vom Verlauf ab, kann das im Quelldokument selbst liegen —
// diesen Fall als Fehlschlag zu werten hieße, ein fremdes
// Dokumentationsproblem als eigenen Parser-Fehler auszugeben.
func ParseReport(raw []byte) (*Report, error) {
	doc, err := extract(raw)
	if err != nil {
		return nil, &ParseError{Reason: err.Error()}
	}
	header, err := parseHeader(doc)
	if err != nil {
		return nil, &ParseError{Reason: err.Error()}
	}
	home, guest, err := parseRosters(doc, header)
	if err != nil {
		return nil, &ParseError{Reason: err.Error()}
	}
	rep := &Report{
		Header: header,
		Home:   home,
		Guest:  guest,
		Events: parseTimeline(doc, header),
	}
	if err := checkScore(rep); err != nil {
		return nil, err
	}
	rep.Warnings = checkRosters(rep)
	return rep, nil
}

// checkScore ist die harte Stufe: die Summe der Tor-Ereignisse je Seite muss
// dem Endstand des Berichtskopfs entsprechen.
func checkScore(r *Report) error {
	var home, guest int
	for _, e := range r.Events {
		if e.Kind != EventGoal && e.Kind != EventSevenMGoal {
			continue
		}
		switch e.Side {
		case "home":
			home++
		case "guest":
			guest++
		default:
			return &ParseError{Reason: fmt.Sprintf(
				"Tor-Ereignis ohne zuordenbare Mannschaft: %q (Heim %q, Gast %q)",
				e.RawText, r.Header.HomeTeam, r.Header.GuestTeam)}
		}
	}
	if home != r.Header.HomeGoals || guest != r.Header.GuestGoals {
		return &ParseError{Reason: fmt.Sprintf(
			"Endstand im Kopf %d:%d, Summe des Spielverlaufs %d:%d — der Parser hat Ereignisse verloren",
			r.Header.HomeGoals, r.Header.GuestGoals, home, guest)}
	}
	return nil
}

// checkRosters ist die weiche Stufe: Abweichungen zwischen den Summenspalten
// der Mannschaftsliste und der Zählung aus dem Spielverlauf werden benannt,
// der Bericht bleibt gültig.
func checkRosters(r *Report) []string {
	type counts struct{ goals, sevenAtt, sevenGoals, twoMin, warnings, disq int }
	tally := map[string]*counts{}
	key := func(side string, num int) string { return fmt.Sprintf("%s#%d", side, num) }
	at := func(side string, num int) *counts {
		k := key(side, num)
		if tally[k] == nil {
			tally[k] = &counts{}
		}
		return tally[k]
	}
	for _, e := range r.Events {
		if e.Side == "" || e.Number == 0 {
			continue
		}
		c := at(e.Side, e.Number)
		switch e.Kind {
		case EventGoal:
			c.goals++
		case EventSevenMGoal:
			c.goals++
			c.sevenAtt++
			c.sevenGoals++
		case EventSevenMMiss:
			c.sevenAtt++
		case EventTwoMin:
			c.twoMin++
		case EventWarning:
			c.warnings++
		case EventDisq:
			c.disq++
		}
	}

	var out []string
	for _, ros := range []Roster{r.Home, r.Guest} {
		for _, p := range ros.Players {
			c := tally[key(ros.Side, p.Number)]
			if c == nil {
				c = &counts{}
			}
			for _, cmp := range []struct {
				label      string
				list, flow int
			}{
				{"Tore", p.Goals, c.goals},
				{"Siebenmeter-Versuche", p.SevenMAtt, c.sevenAtt},
				{"Siebenmeter-Treffer", p.SevenMGoals, c.sevenGoals},
				{"Zeitstrafen", p.TwoMin, c.twoMin},
				{"Verwarnungen", p.Warnings, c.warnings},
			} {
				if cmp.list != cmp.flow {
					out = append(out, fmt.Sprintf("%s #%d %s: Mannschaftsliste %d, Spielverlauf %d",
						ros.TeamName, p.Number, cmp.label, cmp.list, cmp.flow))
				}
			}
		}
	}
	sort.Strings(out)
	return out
}

// WarningsText fasst die Warnungen zu einer Zeile zusammen (für Logausgaben).
func WarningsText(w []string) string { return strings.Join(w, "; ") }
