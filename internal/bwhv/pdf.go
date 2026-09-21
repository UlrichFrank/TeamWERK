package bwhv

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

// textGroup ist ein zusammenhängender Textlauf einer Zeile mit seiner
// Startposition. Die Startposition ist das, woran die Spaltenzuordnung hängt —
// die Breite interessiert nur, um benachbarte Läufe zu trennen.
type textGroup struct {
	X    float64
	End  float64
	S    string
	Bold bool
}

// textLine ist eine Zeile einer Seite, von links nach rechts sortiert.
type textLine struct {
	Page   int
	Y      float64
	Groups []textGroup
}

// Text liefert die Zeile als Fließtext mit einfachen Leerzeichen zwischen den
// Läufen — die Form, auf der die Spielverlauf-Muster arbeiten.
func (l textLine) Text() string {
	parts := make([]string, 0, len(l.Groups))
	for _, g := range l.Groups {
		parts = append(parts, g.S)
	}
	return strings.Join(parts, " ")
}

// document ist die aufbereitete Textebene eines Spielberichts.
type document struct {
	Lines []textLine
}

// groupGapFactor bestimmt, ab welchem Abstand (relativ zur Schriftgröße) zwei
// Textläufe als getrennt gelten. Der Wert stammt aus der Beobachtung, dass der
// Dienst innerhalb eines Wortes Läufe ohne Lücke setzt, zwischen Spalten aber
// deutlich abgesetzt.
const groupGapFactor = 0.45

// glyphWidthFactor schätzt die Breite eines Zeichens relativ zur Schriftgröße.
// Er dient nur der Lückenerkennung zwischen Läufen, nicht der Spaltenzuordnung
// — eine Ungenauigkeit hier verschiebt keine Spalte.
const glyphWidthFactor = 0.55

// yTolerance fasst Läufe zu einer Zeile zusammen, deren Grundlinien minimal
// differieren (Hoch-/Tiefstellung, Rundung).
const yTolerance = 1.0

// extract liest die Textebene eines PDF mit Koordinaten und verdichtet sie zu
// Zeilen aus Textläufen.
func extract(raw []byte) (*document, error) {
	r, err := pdf.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("PDF nicht lesbar: %w", err)
	}
	doc := &document{}
	for p := 1; p <= r.NumPage(); p++ {
		page := r.Page(p)
		if page.V.IsNull() {
			continue
		}
		doc.Lines = append(doc.Lines, linesOfPage(p, page.Content().Text)...)
	}
	if len(doc.Lines) == 0 {
		return nil, fmt.Errorf("PDF enthält keine lesbare Textebene")
	}
	return doc, nil
}

// linesOfPage gruppiert die Textläufe einer Seite zu Zeilen.
func linesOfPage(page int, texts []pdf.Text) []textLine {
	type bucket struct {
		y     float64
		items []pdf.Text
	}
	var buckets []*bucket
	sorted := make([]pdf.Text, len(texts))
	copy(sorted, texts)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Y > sorted[j].Y })

	for _, t := range sorted {
		placed := false
		for _, b := range buckets {
			if diff(b.y, t.Y) <= yTolerance {
				b.items = append(b.items, t)
				placed = true
				break
			}
		}
		if !placed {
			buckets = append(buckets, &bucket{y: t.Y, items: []pdf.Text{t}})
		}
	}

	out := make([]textLine, 0, len(buckets))
	for _, b := range buckets {
		sort.SliceStable(b.items, func(i, j int) bool { return b.items[i].X < b.items[j].X })
		line := textLine{Page: page, Y: b.y}
		var cur *textGroup
		for _, t := range b.items {
			w := t.FontSize * glyphWidthFactor * float64(maxInt(len([]rune(t.S)), 1))
			if cur != nil && t.X-cur.End < t.FontSize*groupGapFactor {
				cur.S += t.S
				cur.End = t.X + w
				continue
			}
			if cur != nil {
				line.Groups = append(line.Groups, *cur)
			}
			cur = &textGroup{X: t.X, End: t.X + w, S: t.S, Bold: strings.Contains(t.Font, "Bold")}
		}
		if cur != nil {
			line.Groups = append(line.Groups, *cur)
		}
		out = append(out, line)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Y > out[j].Y })
	return out
}

func diff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// findLine liefert die erste Zeile, deren Text alle Teilstrings enthält.
func (d *document) findLine(parts ...string) (int, bool) {
	for i, l := range d.Lines {
		txt := l.Text()
		all := true
		for _, p := range parts {
			if !strings.Contains(txt, p) {
				all = false
				break
			}
		}
		if all {
			return i, true
		}
	}
	return 0, false
}
