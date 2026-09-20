package bwhv

import (
	"regexp"
	"strconv"
	"strings"
)

// Eine Verlaufszeile beginnt mit Uhrzeit und Spielzeit; der Spielstand fehlt
// bei Ereignissen ohne Tor. Danach folgt der Aktionstext.
var reTimelineRow = regexp.MustCompile(
	`^(\d{1,2}:\d{2}:\d{2})\s+(\d{1,3}):(\d{2})\s+(?:(\d+):(\d+)\s+)?(.+)$`)

// Die Aktionsformen, wörtlich aus dem Bericht. Der Spieler steht als
// "Name (Nummer, Mannschaft)".
var (
	reGoal      = regexp.MustCompile(`^Tor durch\s+(.+?)\s+\((\d+),\s*(.+?)\)\s*$`)
	reSevenGoal = regexp.MustCompile(`^7m-Tor durch\s+(.+?)\s+\((\d+),\s*(.+?)\)\s*$`)
	reSevenMiss = regexp.MustCompile(`^7m,\s*KEIN Tor durch\s+(.+?)\s+\((\d+),\s*(.+?)\)\s*$`)
	reWarn      = regexp.MustCompile(`^Verwarnung für\s+(.+?)\s+\((\d+),\s*(.+?)\)\s*$`)
	reTwoMin    = regexp.MustCompile(`^2-min Strafe für\s+(.+?)\s+\((\d+),\s*(.+?)\)\s*$`)
	reDisq      = regexp.MustCompile(`^Disqualifikation für\s+(.+?)\s+\((\d+),\s*(.+?)\)\s*$`)
	reTimeout   = regexp.MustCompile(`^Auszeit\s+(.+?)\s*$`)
)

// actionPattern verbindet eine Form mit ihrer Ereignisart.
var actionPatterns = []struct {
	re   *regexp.Regexp
	kind EventKind
}{
	// 7m-Formen VOR der allgemeinen Tor-Form: "7m-Tor durch ..." würde sonst
	// nie greifen, weil reGoal am Zeilenanfang ankert und "7m-Tor" nicht mit
	// "Tor durch" beginnt — die Reihenfolge ist trotzdem festgehalten, damit
	// eine spätere Lockerung der Muster diese Falle nicht aufreißt.
	{reSevenGoal, EventSevenMGoal},
	{reSevenMiss, EventSevenMMiss},
	{reGoal, EventGoal},
	{reTwoMin, EventTwoMin},
	{reWarn, EventWarning},
	{reDisq, EventDisq},
}

// parseTimeline liest den Spielverlauf der Seiten ab "Spielverlauf".
//
// Eine Zeile unbekannter Form geht NICHT verloren: sie wird als EventOther mit
// ihrem Rohtext gespeichert, damit keine Information verschwindet und ein neuer
// Ereignistyp beim Nachsehen auffällt (design.md §5.2).
func parseTimeline(d *document, h Header) []Event {
	start, ok := d.findLine("Spielverlauf")
	if !ok {
		return nil
	}
	var events []Event
	seq := 0
	for _, l := range d.Lines[start:] {
		m := reTimelineRow.FindStringSubmatch(strings.TrimSpace(l.Text()))
		if m == nil {
			continue
		}
		seq++
		min, _ := strconv.Atoi(m[2])
		sec, _ := strconv.Atoi(m[3])
		ev := Event{
			Seq:        seq,
			ClockTime:  m[1],
			GameSecond: min*60 + sec,
			Kind:       EventOther,
			RawText:    strings.TrimSpace(m[6]),
		}
		if m[4] != "" {
			hg, _ := strconv.Atoi(m[4])
			gg, _ := strconv.Atoi(m[5])
			ev.ScoreHome, ev.ScoreGuest = &hg, &gg
		}
		applyAction(&ev, h)
		events = append(events, ev)
	}
	return events
}

// applyAction bestimmt Art, Spieler und Seite eines Ereignisses.
func applyAction(ev *Event, h Header) {
	action := ev.RawText
	for _, p := range actionPatterns {
		m := p.re.FindStringSubmatch(action)
		if m == nil {
			continue
		}
		ev.Kind = p.kind
		ev.PlayerName = strings.TrimSpace(m[1])
		ev.Number, _ = strconv.Atoi(m[2])
		ev.TeamName = strings.TrimSpace(m[3])
		ev.Side = sideOf(ev.TeamName, h)
		return
	}
	if m := reTimeout.FindStringSubmatch(action); m != nil {
		ev.Kind = EventTimeout
		ev.TeamName = strings.TrimSpace(m[1])
		ev.Side = sideOf(ev.TeamName, h)
	}
}

// sideOf ordnet den Mannschaftsnamen einer Verlaufszeile einer Seite zu.
//
// Der Verlauf nutzt teils Kurzformen ("R-N Löwen 2" statt "Rhein-Neckar
// Löwen 2"), deshalb reicht kein Gleichheitsvergleich. Entschieden wird über
// die längere gemeinsame Übereinstimmung mit den beiden Kopfnamen; bleibt es
// unklar, bleibt die Seite leer statt geraten zu werden.
func sideOf(team string, h Header) string {
	hs := similarity(team, h.HomeTeam)
	gs := similarity(team, h.GuestTeam)
	if hs == gs {
		return ""
	}
	if hs > gs {
		return "home"
	}
	return "guest"
}

// similarity zählt die gemeinsamen normalisierten Wortanfänge zweier
// Mannschaftsnamen. "R-N Löwen 2" und "Rhein-Neckar Löwen 2" teilen sich
// "löwen" und "2" und gewinnen damit gegen "Team Stuttgart".
func similarity(a, b string) int {
	fa := nameTokens(a)
	fb := nameTokens(b)
	score := 0
	for _, x := range fa {
		for _, y := range fb {
			if x == y {
				score += 2
				break
			}
			if len(x) >= 3 && len(y) >= 3 && (strings.HasPrefix(y, x) || strings.HasPrefix(x, y)) {
				score++
				break
			}
		}
	}
	return score
}

func nameTokens(s string) []string {
	repl := strings.NewReplacer("-", " ", "/", " ", ".", " ")
	fields := strings.Fields(strings.ToLower(repl.Replace(s)))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}
