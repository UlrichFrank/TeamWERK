package bwhv

import (
	"fmt"
	"sort"
	"strings"
)

// column benennt eine Spalte der Mannschaftsliste.
type column int

const (
	colUnknown column = iota
	colNumber
	colName
	colBirthYear
	colGoals
	colSevenM
	colWarning
	colPenalty1
	colPenalty2
	colPenalty3
	colDisq
	colBer
	colTotalPenalty
	colIgnored // M / R — Markerspalten ohne Auswertung
)

// anchor ist eine erkannte Spalte mit ihrer X-Position im Dokument.
type anchor struct {
	X   float64
	Col column
	Raw string
}

// columnModel ordnet einen X-Wert der Spalte zu, deren Anker am nächsten liegt.
//
// Die Anker stammen aus der Kopfzeile des Dokuments, NICHT aus festen
// Koordinaten: der Dienst richtet Werte mal links, mal rechts ihres Kopfes aus
// (beobachtet: Tore +7,5 rechts vom Anker, erste Hinausstellung 5,5 links), und
// ein fest verdrahtetes x==296 lieferte beim ersten Layout-Update still falsche
// Zahlen (design.md §5.1).
type columnModel struct {
	anchors []anchor
}

// anchorMergeDistance fasst Kopfzeilen-Läufe zusammen, die zur selben Spalte
// gehören. Der Kopf steht auf drei Zeilen: "Tore" über "(ges)" (289,9 / 288,7),
// "7m/" über "Tore" (322,4 / 321,1), "Hinausstellungen" über "1." (384,4 /
// 387,3), "zus." über "Strafe" (527,7 / 523,5). Die nächstgelegenen echten
// Nachbarspalten liegen über 14 Einheiten auseinander, 8 trennt also sicher.
const anchorMergeDistance = 8.0

// headerWindow ist der Y-Abstand, in dem Zeilen noch zum selben Kopfblock
// zählen. Der Block umfasst drei Zeilen mit ~5,7 Einheiten Abstand.
const headerWindow = 14.0

// classifyHeader bildet einen zusammengesetzten Kopftext auf eine Spalte ab.
//
// Die Reihenfolge der Prüfungen trägt die Bedeutung: "7m/Tore" muss VOR "Tore"
// greifen, sonst landete die Siebenmeter-Spalte bei den Feldtoren. Ebenso muss
// "Hinausstellungen1." als erste Hinausstellung gelten, nicht als Sammelspalte.
func classifyHeader(raw string) column {
	s := strings.ToLower(strings.Join(strings.Fields(raw), ""))
	switch {
	case s == "":
		return colUnknown
	case strings.HasPrefix(s, "nr"):
		return colNumber
	case strings.HasPrefix(s, "name"):
		return colName
	case strings.Contains(s, "jahrgang"):
		return colBirthYear
	case strings.Contains(s, "7m"):
		return colSevenM
	case strings.Contains(s, "verw"):
		return colWarning
	case strings.Contains(s, "disq"):
		return colDisq
	case strings.Contains(s, "strafe"):
		return colTotalPenalty
	case strings.Contains(s, "1."):
		return colPenalty1
	case strings.Contains(s, "2."):
		return colPenalty2
	case strings.Contains(s, "3."):
		return colPenalty3
	case strings.HasPrefix(s, "ber"):
		return colBer
	case strings.Contains(s, "tore"):
		return colGoals
	case s == "m" || s == "r":
		return colIgnored
	}
	return colUnknown
}

// buildColumnModel liest den Kopfblock der Mannschaftsliste. Anker sind die
// Kopfzeile mit "Nr." und "Name" sowie die Zeilen unmittelbar darüber und
// darunter.
//
// Eine Kopfzeile, die sich nicht auf Nummer, Name und Tore abbilden lässt,
// führt zu einem Fehler mit ihrem Wortlaut — geraten wird hier nichts
// (design.md §5.1, Requirement "Unbekannte Kopfzeile bricht ab").
func buildColumnModel(d *document, headerIdx int) (*columnModel, error) {
	base := d.Lines[headerIdx]
	raw := map[float64]string{}
	var order []float64

	add := func(g textGroup) {
		for _, x := range order {
			if diff(x, g.X) <= anchorMergeDistance {
				raw[x] += g.S
				return
			}
		}
		order = append(order, g.X)
		raw[g.X] = g.S
	}
	for _, g := range base.Groups {
		add(g)
	}
	for _, l := range d.Lines {
		if l.Page != base.Page || diff(l.Y, base.Y) > headerWindow || l.Y == base.Y {
			continue
		}
		for _, g := range l.Groups {
			add(g)
		}
	}

	m := &columnModel{}
	for _, x := range order {
		col := classifyHeader(raw[x])
		if col == colUnknown {
			continue
		}
		m.anchors = append(m.anchors, anchor{X: x, Col: col, Raw: raw[x]})
	}
	sort.SliceStable(m.anchors, func(i, j int) bool { return m.anchors[i].X < m.anchors[j].X })

	for _, need := range []struct {
		col   column
		label string
	}{{colNumber, "Nr."}, {colName, "Name"}, {colGoals, "Tore"}} {
		if !m.has(need.col) {
			return nil, fmt.Errorf("kopfzeile der Mannschaftsliste nicht interpretierbar, Spalte %q fehlt: %q",
				need.label, base.Text())
		}
	}
	return m, nil
}

func (m *columnModel) has(c column) bool {
	for _, a := range m.anchors {
		if a.Col == c {
			return true
		}
	}
	return false
}

// columnAt liefert die Spalte, deren Anker dem X-Wert am nächsten liegt.
func (m *columnModel) columnAt(x float64) column {
	best, bestDist := colUnknown, 1e9
	for _, a := range m.anchors {
		if d := diff(a.X, x); d < bestDist {
			best, bestDist = a.Col, d
		}
	}
	return best
}

// cells ordnet die Läufe einer Zeile ihren Spalten zu. Mehrere Läufe in
// derselben Spalte werden mit Leerzeichen verbunden.
func (m *columnModel) cells(l textLine) map[column]string {
	out := map[column]string{}
	for _, g := range l.Groups {
		c := m.columnAt(g.X)
		if c == colUnknown || c == colIgnored {
			continue
		}
		if out[c] == "" {
			out[c] = g.S
		} else {
			out[c] += " " + g.S
		}
	}
	return out
}
