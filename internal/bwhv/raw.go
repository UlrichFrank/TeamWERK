package bwhv

import (
	"encoding/json"
	"strconv"
	"strings"
)

// flexString liest ein Feld, das der Dienst je nach Zustand als String ODER als
// Zahl liefert. Konkret betrifft das sGID: gespielte Begegnungen tragen
// "3504061", noch nicht gespielte die Zahl 0 — ein reines string-Feld bräche
// die Dekodierung der gesamten Antwort an der ersten künftigen Begegnung.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		*f = ""
		return nil
	}
	if strings.HasPrefix(s, `"`) {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*f = flexString(strings.TrimSpace(v))
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexString(n.String())
	return nil
}

// String liefert den Wert, wobei "0" als leer gilt — der Dienst kodiert damit
// "kein Spielbericht vorhanden".
func (f flexString) String() string {
	s := strings.TrimSpace(string(f))
	if s == "0" {
		return ""
	}
	return s
}

// flexInt liest eine Zahl, die als Zahl oder als String kommen kann. Die
// Tabellenzeilen nutzen echte Zahlen, die Spielzeilen Strings — und leere
// Felder sind dort nicht "" sondern " " (ein Leerzeichen).
type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		*f = 0
		return nil // fehlertolerant: eine unlesbare Zahl ist 0, kein Abbruch der Antwort
	}
	*f = flexInt(n)
	return nil
}

// rawGame ist eine Spielzeile, wie der Dienst sie liefert.
type rawGame struct {
	GNo           string     `json:"gNo"`
	SGID          flexString `json:"sGID"`
	GDate         string     `json:"gDate"`
	GTime         string     `json:"gTime"`
	GGymnasiumNo  flexString `json:"gGymnasiumNo"`
	GGymnasiumNm  string     `json:"gGymnasiumName"`
	GGymnasiumTwn string     `json:"gGymnasiumTown"`
	GHomeTeam     string     `json:"gHomeTeam"`
	GGuestTeam    string     `json:"gGuestTeam"`
	GHomeGoals    string     `json:"gHomeGoals"`
	GGuestGoals   string     `json:"gGuestGoals"`
	GHomeGoals1   string     `json:"gHomeGoals_1"`
	GGuestGoals1  string     `json:"gGuestGoals_1"`
}

// optInt liest ein Ergebnisfeld. Ein nicht gespieltes Spiel trägt dort " "
// (Leerzeichen), kein leeres Feld — ohne TrimSpace liefe jede künftige
// Begegnung als "0:0" durch und sähe aus wie ein torloses Unentschieden.
func optInt(s string) *int {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	n, err := strconv.Atoi(t)
	if err != nil {
		return nil
	}
	return &n
}

// normalizeDate wandelt "20.09.26" in "2026-09-20". Zweistellige Jahre liest
// der Dienst ohne Jahrhundert; 20xx ist für den Spielbetrieb die einzige
// sinnvolle Lesart.
func normalizeDate(s string) string {
	parts := strings.Split(strings.TrimSpace(s), ".")
	if len(parts) != 3 {
		return ""
	}
	d, errD := strconv.Atoi(parts[0])
	m, errM := strconv.Atoi(parts[1])
	y, errY := strconv.Atoi(parts[2])
	if errD != nil || errM != nil || errY != nil {
		return ""
	}
	if y < 100 {
		y += 2000
	}
	if d < 1 || d > 31 || m < 1 || m > 12 {
		return ""
	}
	return strconv.Itoa(y) + "-" + twoDigit(m) + "-" + twoDigit(d)
}

func twoDigit(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// toGame bildet die Rohzeile ab. Eine Zeile ohne Spielnummer oder ohne
// lesbares Datum wird verworfen (ok=false) — sie hätte keinen Anker und keine
// Einordnung, und geraten wird hier nichts.
func (r rawGame) toGame() (Game, bool) {
	no := strings.TrimSpace(r.GNo)
	date := normalizeDate(r.GDate)
	if no == "" || date == "" {
		return Game{}, false
	}
	return Game{
		GameNo:       no,
		SGID:         r.SGID.String(),
		Date:         date,
		Time:         strings.TrimSpace(r.GTime),
		HomeTeam:     strings.TrimSpace(r.GHomeTeam),
		GuestTeam:    strings.TrimSpace(r.GGuestTeam),
		HomeGoals:    optInt(r.GHomeGoals),
		GuestGoals:   optInt(r.GGuestGoals),
		HomeGoalsHT:  optInt(r.GHomeGoals1),
		GuestGoalsHT: optInt(r.GGuestGoals1),
		HallNumber:   r.GGymnasiumNo.String(),
		HallName:     strings.TrimSpace(r.GGymnasiumNm),
		HallTown:     strings.TrimSpace(r.GGymnasiumTwn),
	}, true
}

// rawScore ist eine Tabellenzeile, wie der Dienst sie liefert.
type rawScore struct {
	TabTeamname    string  `json:"tabTeamname"`
	NumPlayedGames flexInt `json:"numPlayedGames"`
	NumWonGames    flexInt `json:"numWonGames"`
	NumEqualGames  flexInt `json:"numEqualGames"`
	NumLostGames   flexInt `json:"numLostGames"`
	NumGoalsShot   flexInt `json:"numGoalsShot"`
	NumGoalsGot    flexInt `json:"numGoalsGot"`
	PointsPlus     flexInt `json:"pointsPlus"`
	PointsMinus    flexInt `json:"pointsMinus"`
}

func (r rawScore) toRow(position int) TableRow {
	return TableRow{
		Position:     position,
		TeamName:     strings.TrimSpace(r.TabTeamname),
		Games:        int(r.NumPlayedGames),
		Won:          int(r.NumWonGames),
		Drawn:        int(r.NumEqualGames),
		Lost:         int(r.NumLostGames),
		GoalsFor:     int(r.NumGoalsShot),
		GoalsAgainst: int(r.NumGoalsGot),
		PointsPlus:   int(r.PointsPlus),
		PointsMinus:  int(r.PointsMinus),
	}
}
