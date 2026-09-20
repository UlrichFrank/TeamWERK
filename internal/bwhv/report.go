package bwhv

// EventKind benennt die Art eines Ereignisses im Spielverlauf.
type EventKind string

const (
	EventGoal       EventKind = "goal"
	EventSevenMGoal EventKind = "seven_m_goal"
	EventSevenMMiss EventKind = "seven_m_miss"
	EventTwoMin     EventKind = "two_min"
	EventWarning    EventKind = "warning"
	EventDisq       EventKind = "disqualification"
	EventTimeout    EventKind = "timeout"
	EventOther      EventKind = "other"
)

// Header sind die Kopfdaten eines Spielberichts (Seite 1).
type Header struct {
	Staffel      string
	GameNo       string
	Date         string // "2006-01-02"
	Time         string // "16:00"
	VenueName    string
	VenueTown    string
	HallNumber   string
	HomeTeam     string
	GuestTeam    string
	HomeGoals    int
	GuestGoals   int
	HomeGoalsHT  int
	GuestGoalsHT int
	Spectators   string
	Referees     string
}

// RosterPlayer ist eine Zeile der Mannschaftsliste.
//
// Die Zähler stammen aus den Summenspalten des Dokuments, nicht aus dem
// Spielverlauf — genau diese Doppelung macht die Kreuzprobe möglich.
type RosterPlayer struct {
	Number      int
	Name        string
	BirthYear   int
	Goals       int
	SevenMAtt   int
	SevenMGoals int
	TwoMin      int
	Warnings    int
	Disq        int
}

// Roster ist die Mannschaftsliste eines Teams.
type Roster struct {
	TeamName string
	Side     string // "home" | "guest"
	Players  []RosterPlayer
}

// Event ist eine Zeile des Spielverlaufs.
type Event struct {
	Seq        int
	ClockTime  string
	GameSecond int
	ScoreHome  *int
	ScoreGuest *int
	Kind       EventKind
	Side       string // "home" | "guest" | ""
	PlayerName string
	Number     int
	TeamName   string
	RawText    string
}

// Report ist ein ausgewerteter Spielbericht.
//
// Warnings trägt Abweichungen, die den Bericht NICHT verwerfen: Der Endstand
// im Kopf muss zur Summe des Spielverlaufs passen (sonst gibt es gar keinen
// Report), Unterschiede zwischen Mannschaftsliste und Verlauf werden dagegen
// vermerkt und mitgeliefert (design.md §5.3).
type Report struct {
	Header   Header
	Home     Roster
	Guest    Roster
	Events   []Event
	Warnings []string
}

// PlayerBySide liefert die Mannschaftsliste einer Seite.
func (r *Report) PlayerBySide(side string) *Roster {
	if side == "guest" {
		return &r.Guest
	}
	return &r.Home
}
