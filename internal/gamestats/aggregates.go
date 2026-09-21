package gamestats

import (
	"context"
	"sort"
)

// Dieses File trägt die Auswertungen, die aus den bereits gespeicherten
// Ergebnissen und Spielberichten einer Staffel gerechnet werden: Kreuztabelle,
// Platzierungsverlauf, Mannschafts- und Schiedsrichter-Ranglisten.
//
// Entscheidung 1 aus design.md zieht sich durch alle: Tor-Werte kommen aus
// bwhv_games (den Ergebnissen), NICHT aus der Summe der Spielerzeilen. Eine
// Begegnung ohne freigegebenes PDF fehlte sonst in jeder Statistik, und ein
// Bericht mit abweichender Mannschaftsliste (ein gespeicherter Fall mit
// Warnung) lieferte eine andere Summe als der amtliche Endstand. Fair-Play und
// Torverteilung hängen zwangsläufig an den Berichten — deshalb stehen in
// derselben Ansicht zwei verschiedene Abdeckungen nebeneinander, und jede
// Mannschafts-Zeile weist beide Spielzahlen aus.

// trimDate schneidet einen gespeicherten DATE-Wert auf "2006-01-02".
//
// SQLite liefert DATE-Spalten je nach Schreibweg als reines Datum oder als
// ISO-Timestamp ("2026-09-20T00:00:00Z"). Beides landet hier als
// Gruppierungsschlüssel und als Anzeigewert; ohne den Schnitt bildeten
// dieselben Spieltage zwei Gruppen (docs/agent/06-gotchas.md).
func trimDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// CrossCell ist eine Zelle der Kreuztabelle.
//
// Played unterscheidet die beiden Fälle ausdrücklich, statt sie an einem
// Nullwert abzulesen: ein 0:0 ist ein gespieltes Spiel. Das Vorbild muss
// "beide Zahlen 0" als "nicht gespielt" behandeln und verliert damit jedes
// echte torlose Spiel — mit der NULL-Prüfung auf home_goals braucht es diese
// Krücke nicht.
type CrossCell struct {
	BwhvGameID int    `json:"bwhvGameId"`
	Played     bool   `json:"played"`
	HomeGoals  *int   `json:"homeGoals"`
	GuestGoals *int   `json:"guestGoals"`
	Date       string `json:"date"`
}

// CrossRow ist eine Zeile der Kreuztabelle: eine Heimmannschaft und ihre
// Zellen, parallel zu CrossTable.Teams. Ein nil-Eintrag ist die Diagonale oder
// eine Paarung ohne angesetzte Begegnung.
type CrossRow struct {
	Team  string       `json:"team"`
	Cells []*CrossCell `json:"cells"`
}

// CrossTable ist die Matrix aller Mannschaften einer Staffel: Heim in der
// Zeile, Gast in der Spalte.
type CrossTable struct {
	Teams []string   `json:"teams"`
	Rows  []CrossRow `json:"rows"`
}

// crossGame ist eine Begegnung, soweit die Kreuztabelle und der Verlauf sie
// brauchen.
type crossGame struct {
	id         int
	date       string
	home       string
	guest      string
	homeGoals  *int
	guestGoals *int
}

// played meldet, ob ein Ergebnis erfasst ist. Das ist die Definition von
// "gespielt" in diesem Paket — und der Grund, warum ein 0:0 nicht verschwindet.
func (g crossGame) played() bool { return g.homeGoals != nil && g.guestGoals != nil }

// loadGames liest die Begegnungen einer Staffel nach Datum sortiert.
func (s *Store) loadGames(ctx context.Context, staffelID int) ([]crossGame, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, date, home_team, guest_team, home_goals, guest_goals
		  FROM bwhv_games
		 WHERE staffel_id = ?
		 ORDER BY date, time, game_no`, staffelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []crossGame
	for rows.Next() {
		var g crossGame
		if err := rows.Scan(&g.id, &g.date, &g.home, &g.guest, &g.homeGoals, &g.guestGoals); err != nil {
			return nil, err
		}
		g.date = trimDate(g.date)
		out = append(out, g)
	}
	return out, rows.Err()
}

// teamsOf sammelt alle Mannschaften einer Staffel in stabiler Reihenfolge.
//
// Sortiert wird alphabetisch und nicht nach dem Tabellenstand: der Snapshot in
// bwhv_staffeln.table_json kann fehlen (vor dem ersten Abruf) oder älter sein
// als der Spielplan, und eine Achse, die je nach Abrufzustand die Reihenfolge
// wechselt, macht die Matrix unlesbar.
func teamsOf(games []crossGame) []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range games {
		for _, t := range []string{g.home, g.guest} {
			if t != "" && !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sort.Strings(out)
	return out
}

// CrossTable liefert die Kreuztabelle einer Staffel.
//
// Sie braucht KEINEN ausgewerteten Spielbericht: Ergebnis und Datum stehen
// beide in bwhv_games.
func (s *Store) CrossTable(ctx context.Context, staffelID int) (*CrossTable, error) {
	games, err := s.loadGames(ctx, staffelID)
	if err != nil {
		return nil, err
	}
	teams := teamsOf(games)
	idx := make(map[string]int, len(teams))
	for i, t := range teams {
		idx[t] = i
	}

	ct := &CrossTable{Teams: teams, Rows: make([]CrossRow, len(teams))}
	for i, t := range teams {
		ct.Rows[i] = CrossRow{Team: t, Cells: make([]*CrossCell, len(teams))}
	}
	for _, g := range games {
		row, ok := idx[g.home]
		if !ok {
			continue
		}
		col, ok := idx[g.guest]
		if !ok || row == col {
			// Die Diagonale bleibt leer: eine Mannschaft spielt nicht gegen
			// sich selbst.
			continue
		}
		c := &CrossCell{BwhvGameID: g.id, Played: g.played()}
		if c.Played {
			c.HomeGoals, c.GuestGoals = g.homeGoals, g.guestGoals
		} else {
			c.Date = g.date
		}
		ct.Rows[row].Cells[col] = c
	}
	return ct, nil
}

// pointsWin/pointsDraw ist die Zwei-Punkte-Wertung: Sieg zwei Punkte,
// Unentschieden ein Punkt, Niederlage keiner.
//
// Sie ist ANGENOMMEN, nicht abgeleitet — die Schnittstelle liefert keine
// Wertungsregel. In einer Staffel mit Drei-Punkte-Wertung zeigt der Verlauf
// deshalb plausible, aber falsche Ränge; erkennbar wäre das nur an einer
// Abweichung des letzten Spieltags von der amtlichen Tabelle
// (bwhv_staffeln.table_json). Die Annahme steht bewusst an EINER Stelle als
// benannte Konstante: die Nachrüstung ist dann eine Kalibrierung hier plus die
// Vergleichsprüfung, kein Suchlauf durch die Queries (design.md §2).
const (
	pointsWin  = 2
	pointsDraw = 1
)

// ProgressionEntry ist die Platzierung einer Mannschaft nach einem Spieltag.
type ProgressionEntry struct {
	Team     string `json:"team"`
	Rank     int    `json:"rank"`
	Points   int    `json:"points"`
	Games    int    `json:"games"`
	GoalsFor int    `json:"goalsFor"`
	GoalDiff int    `json:"goalDiff"`
}

// ProgressionDay ist ein Spieltag mit der Platzierung aller bis dahin
// beteiligten Mannschaften.
type ProgressionDay struct {
	Date    string             `json:"date"`
	Entries []ProgressionEntry `json:"entries"`
}

// standing ist der kumulierte Zwischenstand einer Mannschaft.
type standing struct {
	team     string
	points   int
	games    int
	goalsFor int
	goalsAg  int
}

func (s standing) diff() int { return s.goalsFor - s.goalsAg }

// StandingsProgression liefert die Platzierung jeder Mannschaft nach jedem
// Spieltag, an dem mindestens eine Begegnung gespielt wurde.
//
// Ein Spieltag IST ein Datum: eine Rundennummer liefert die Schnittstelle
// nicht, und der Verband spielt Staffeln mit unterschiedlich vielen
// Begegnungen je Termin — eine gezählte "Runde" wäre eine erfundene Ordnung
// (design.md §3).
//
// Gerechnet wird in Go und nicht in SQL: es ist eine Schleife über Spieltage
// mit kumulierendem Zustand und Neu-Sortierung nach jedem Schritt. Als Query
// wäre das ein Window-Function-Konstrukt, das niemand mehr liest; die
// Datenmenge (~810 Begegnungen je Staffel, eine Query) trägt die Schleife
// mühelos.
func (s *Store) StandingsProgression(ctx context.Context, staffelID int) ([]ProgressionDay, error) {
	games, err := s.loadGames(ctx, staffelID)
	if err != nil {
		return nil, err
	}
	table := map[string]*standing{}
	var out []ProgressionDay
	var day string

	flush := func() {
		if day == "" {
			return
		}
		out = append(out, ProgressionDay{Date: day, Entries: rankStandings(table)})
	}

	for _, g := range games {
		if !g.played() {
			// Begegnungen ohne Ergebnis bleiben draußen — auch ein künftiger
			// Termin mitten in der sortierten Liste.
			continue
		}
		if g.date != day {
			flush()
			day = g.date
		}
		applyResult(table, g)
	}
	flush()
	return out, nil
}

// applyResult schreibt ein Ergebnis in den Zwischenstand beider Mannschaften.
func applyResult(table map[string]*standing, g crossGame) {
	home, guest := standingOf(table, g.home), standingOf(table, g.guest)
	hg, gg := *g.homeGoals, *g.guestGoals
	home.games++
	guest.games++
	home.goalsFor += hg
	home.goalsAg += gg
	guest.goalsFor += gg
	guest.goalsAg += hg
	switch {
	case hg > gg:
		home.points += pointsWin
	case gg > hg:
		guest.points += pointsWin
	default:
		home.points += pointsDraw
		guest.points += pointsDraw
	}
}

func standingOf(table map[string]*standing, team string) *standing {
	if st, ok := table[team]; ok {
		return st
	}
	st := &standing{team: team}
	table[team] = st
	return st
}

// rankStandings sortiert den Zwischenstand: Punkte, dann Tordifferenz, dann
// geworfene Tore. Der Mannschaftsname bricht den Restgleichstand, damit
// dieselbe Lage nicht bei jedem Aufruf eine andere Reihenfolge ergibt.
func rankStandings(table map[string]*standing) []ProgressionEntry {
	list := make([]*standing, 0, len(table))
	for _, st := range table {
		list = append(list, st)
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		switch {
		case a.points != b.points:
			return a.points > b.points
		case a.diff() != b.diff():
			return a.diff() > b.diff()
		case a.goalsFor != b.goalsFor:
			return a.goalsFor > b.goalsFor
		}
		return a.team < b.team
	})
	out := make([]ProgressionEntry, len(list))
	for i, st := range list {
		out[i] = ProgressionEntry{
			Team: st.team, Rank: i + 1, Points: st.points, Games: st.games,
			GoalsFor: st.goalsFor, GoalDiff: st.diff(),
		}
	}
	return out
}

// FairPlayWeights ist die Gewichtung der Strafarten in der Mannschafts-Wertung.
//
// Sie wird MITGELIEFERT und nicht nur angewendet: eine Fair-Play-Zahl ohne
// ihre Gewichtung ist nicht nachvollziehbar, und die Ansicht soll sie
// ausweisen können, ohne sie ein zweites Mal abzutippen.
type FairPlayWeights struct {
	Yellow float64 `json:"yellow"`
	TwoMin float64 `json:"twoMin"`
	Red    float64 `json:"red"`
	Blue   float64 `json:"blue"`
}

// fairPlayWeights: Blau wiegt am schwersten (Bericht an den Verband), dann
// Rot, dann die Zeitstrafe, dann die Verwarnung. Kleiner ist besser.
//
// Bewusst eine andere Skala als fairPlayScore in stats.go: die dortige wertet
// eine Person über die Saison, diese eine Mannschaft über ihre Berichte. Sie
// zusammenzulegen hieße, zwei verschiedene Fragen an dieselbe Zahl zu stellen.
var fairPlayWeights = FairPlayWeights{Yellow: 1, TwoMin: 2, Red: 3, Blue: 4}

func (w FairPlayWeights) score(yellow, twoMin, red, blue int) float64 {
	return float64(yellow)*w.Yellow + float64(twoMin)*w.TwoMin +
		float64(red)*w.Red + float64(blue)*w.Blue
}

// GoalDistribution beschreibt, wie sich die Tore einer Mannschaft über ihre
// Spieler verteilen.
type GoalDistribution struct {
	Players int     `json:"players"`
	Average float64 `json:"average"`
	Median  float64 `json:"median"`
	Gini    float64 `json:"gini"`
}

// TeamStat ist die Mannschafts-Bilanz einer Staffel.
//
// Games und ReportGames stehen nebeneinander, weil sie verschieden groß sind:
// Games zählt die Begegnungen mit Ergebnis (Grundlage von Toren, Angriff,
// Verteidigung), ReportGames die ausgewerteten Spielberichte (Grundlage von
// Fair-Play und Torverteilung). Genau dieser Unterschied ist die Falle der
// Ansicht — deshalb weist jede Zeile beide Zahlen aus.
type TeamStat struct {
	Team         string `json:"team"`
	Games        int    `json:"games"`
	GoalsFor     int    `json:"goalsFor"`
	GoalsAgainst int    `json:"goalsAgainst"`
	GoalDiff     int    `json:"goalDiff"`

	ReportGames int `json:"reportGames"`
	TwoMin      int `json:"twoMin"`
	Yellow      int `json:"yellow"`
	Red         int `json:"red"`
	Blue        int `json:"blue"`
	// FairPlay und Distribution sind nil, solange kein Bericht vorliegt: eine
	// Mannschaft ohne Bericht ist nicht straffrei, sie ist unbekannt. Eine 0
	// führte sie in der aufsteigend sortierten Wertung als vorbildlich —
	// genau die stille Fehlinformation, die Entscheidung 1 vermeidet.
	FairPlay     *float64          `json:"fairPlayScore"`
	Distribution *GoalDistribution `json:"distribution"`
}

// TeamStats ist die Antwort der Mannschafts-Ranglisten: alle fünf Sichten
// (Torverhältnis, Angriff, Verteidigung, Fair-Play, Verteilung) entstehen aus
// derselben Aggregation über dieselben Zeilen.
type TeamStats struct {
	FairPlayWeights FairPlayWeights `json:"fairPlayWeights"`
	Teams           []TeamStat      `json:"teams"`
}

// TeamStats liefert die Mannschafts-Ranglisten einer Staffel.
func (s *Store) TeamStats(ctx context.Context, staffelID int) (*TeamStats, error) {
	games, err := s.loadGames(ctx, staffelID)
	if err != nil {
		return nil, err
	}
	stats := map[string]*TeamStat{}
	for _, t := range teamsOf(games) {
		stats[t] = &TeamStat{Team: t}
	}
	for _, g := range games {
		if !g.played() {
			continue
		}
		hg, gg := *g.homeGoals, *g.guestGoals
		addResult(stats[g.home], hg, gg)
		addResult(stats[g.guest], gg, hg)
	}
	if err := s.addReportStats(ctx, staffelID, stats); err != nil {
		return nil, err
	}
	if err := s.addDistribution(ctx, staffelID, stats); err != nil {
		return nil, err
	}

	out := &TeamStats{FairPlayWeights: fairPlayWeights, Teams: make([]TeamStat, 0, len(stats))}
	for _, st := range stats {
		out.Teams = append(out.Teams, *st)
	}
	sort.Slice(out.Teams, func(i, j int) bool { return out.Teams[i].Team < out.Teams[j].Team })
	return out, nil
}

func addResult(st *TeamStat, for_, against int) {
	if st == nil {
		return
	}
	st.Games++
	st.GoalsFor += for_
	st.GoalsAgainst += against
	st.GoalDiff = st.GoalsFor - st.GoalsAgainst
}

// teamNameExpr bildet die Mannschaft einer Spielerzeile auf die Schreibweise
// des SPIELPLANS ab, nicht auf die des Spielberichts.
//
// bwhv_players.team_name stammt aus der Mannschaftsliste des PDF,
// bwhv_games.home_team/guest_team aus der JSON-Schnittstelle — dieselbe
// Mannschaft, zwei mögliche Schreibweisen. Kreuztabelle, Verlauf, Tabelle und
// die Zugehörigkeit sprechen alle die Schreibweise des Spielplans; eine
// Mannschafts-Zeile, die sich über den Berichtsnamen bildet, hinge daneben und
// erschiene als elfte Mannschaft einer Zehnerstaffel. Die Seite (pg.side) ist
// die verlässliche Brücke.
const teamNameExpr = `CASE pg.side WHEN 'home' THEN g.home_team ELSE g.guest_team END`

// addReportStats ergänzt die aus den Spielberichten gebildeten Werte.
func (s *Store) addReportStats(ctx context.Context, staffelID int, stats map[string]*TeamStat) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+teamNameExpr+` AS team,
		       COUNT(DISTINCT pg.report_id),
		       COALESCE(SUM(pg.two_min), 0), COALESCE(SUM(pg.yellow), 0),
		       COALESCE(SUM(pg.red), 0), COALESCE(SUM(pg.blue), 0)
		  FROM bwhv_player_games pg
		  JOIN bwhv_reports r ON r.id = pg.report_id AND r.state = 'parsed'
		  JOIN bwhv_games g ON g.id = r.bwhv_game_id
		 WHERE g.staffel_id = ? AND pg.side IN ('home','guest')
		 GROUP BY team`, staffelID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var team string
		var reportGames, twoMin, yellow, red, blue int
		if err := rows.Scan(&team, &reportGames, &twoMin, &yellow, &red, &blue); err != nil {
			return err
		}
		st, ok := stats[team]
		if !ok {
			st = &TeamStat{Team: team}
			stats[team] = st
		}
		st.ReportGames, st.TwoMin, st.Yellow, st.Red, st.Blue = reportGames, twoMin, yellow, red, blue
		score := fairPlayWeights.score(yellow, twoMin, red, blue)
		st.FairPlay = &score
	}
	return rows.Err()
}

// addDistribution ergänzt die Torverteilung je Mannschaft.
//
// Bezugsgröße ist die SAISONSUMME je Spieler, nicht die Zeile je Spiel: die
// Frage lautet "hängt die Mannschaft an einzelnen Werfern", und das ist eine
// Frage an die Saisonbilanz. Zählte man Spieler-Spiele, erschiene ein Spieler
// mit vielen Einsätzen mehrfach und ein Ausfall zöge den Wert nach oben, ohne
// dass sich an der Rollenverteilung etwas geändert hätte (design.md §6).
func (s *Store) addDistribution(ctx context.Context, staffelID int, stats map[string]*TeamStat) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+teamNameExpr+` AS team, pg.player_id, COALESCE(SUM(pg.goals), 0)
		  FROM bwhv_player_games pg
		  JOIN bwhv_reports r ON r.id = pg.report_id AND r.state = 'parsed'
		  JOIN bwhv_games g ON g.id = r.bwhv_game_id
		 WHERE g.staffel_id = ? AND pg.side IN ('home','guest')
		 GROUP BY team, pg.player_id`, staffelID)
	if err != nil {
		return err
	}
	defer rows.Close()
	perTeam := map[string][]float64{}
	for rows.Next() {
		var team string
		var playerID, goals int
		if err := rows.Scan(&team, &playerID, &goals); err != nil {
			return err
		}
		perTeam[team] = append(perTeam[team], float64(goals))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for team, values := range perTeam {
		st, ok := stats[team]
		if !ok {
			st = &TeamStat{Team: team}
			stats[team] = st
		}
		d := distributionOf(values)
		st.Distribution = &d
	}
	return nil
}

// distributionOf bildet Durchschnitt, Median und Gini über die Saisonsummen.
func distributionOf(values []float64) GoalDistribution {
	d := GoalDistribution{Players: len(values)}
	if len(values) == 0 {
		return d
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	var sum float64
	for _, v := range sorted {
		sum += v
	}
	d.Average = sum / float64(len(sorted))
	d.Median = medianOf(sorted)
	d.Gini = gini(sorted)
	return d
}

func medianOf(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// gini ist der Gini-Koeffizient über aufsteigend sortierte Werte:
//
//	G = 2·Σ(i·xᵢ)/(n²·μ) − (n+1)/n
//
// Ein Wert nahe 0 heißt gleichmäßige Verteilung, ein hoher Wert Abhängigkeit
// von wenigen Werfern.
//
// Bei μ = 0 liefert die Funktion 0 statt einer Division durch null: eine
// Mannschaft ohne Tor hat keine Verteilung, und "0" liest sich hier richtig
// als "keine Ungleichheit feststellbar".
func gini(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	var sum, weighted float64
	for i, v := range sorted {
		sum += v
		weighted += float64(i+1) * v
	}
	mean := sum / float64(n)
	if mean == 0 {
		return 0
	}
	g := 2*weighted/(float64(n)*float64(n)*mean) - float64(n+1)/float64(n)
	if g < 0 {
		// Rundungsrest bei exakter Gleichverteilung.
		return 0
	}
	return g
}
