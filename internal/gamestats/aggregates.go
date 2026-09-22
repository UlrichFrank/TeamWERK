package gamestats

import (
	"context"
	"database/sql"
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
		       COALESCE(SUM(`+twoMinCounted+`), 0), COALESCE(SUM(pg.yellow), 0),
		       COALESCE(SUM(`+redCounted+`), 0), COALESCE(SUM(pg.blue), 0)
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

// RefereeStat ist die Bilanz eines Schiedsrichters über die Staffel.
//
// Die Zahlen sind die Strafen DES SPIELS, nicht eine Bewertung der Person —
// die Ansicht sagt das ausdrücklich. Technisch verhindern lässt sich die
// Fehllesart nicht.
type RefereeStat struct {
	Name   string `json:"name"`
	Games  int    `json:"games"`
	TwoMin int    `json:"twoMin"`
	Yellow int    `json:"yellow"`
	Red    int    `json:"red"`
	Blue   int    `json:"blue"`
	// Uncertain heißt: in mindestens einem der Spiele musste der Name von dem
	// des Gespannpartners geraten werden (parse_header.go).
	Uncertain bool `json:"uncertain"`
}

// RefereeStats liefert je Schiedsrichter die Zahl seiner Spiele und die in
// diesen Spielen verhängten Strafen — die BEIDER Mannschaften.
//
// Berichte ohne benannte Schiedsrichter erzeugen keine Zeile. Das schließt den
// gesamten Bestand vor Migration 069 ein: dort steht nur die ungetrennte
// Rohzeile, und ein Backfill über die Namensregel machte für alle Altberichte
// den unsicheren Pfad zum Hauptpfad (design.md §7).
func (s *Store) RefereeStats(ctx context.Context, staffelID int) ([]RefereeStat, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.referees_json, r.referees_uncertain,
		       COALESCE(SUM(`+twoMinCounted+`), 0), COALESCE(SUM(pg.yellow), 0),
		       COALESCE(SUM(`+redCounted+`), 0), COALESCE(SUM(pg.blue), 0)
		  FROM bwhv_reports r
		  JOIN bwhv_games g ON g.id = r.bwhv_game_id
		  LEFT JOIN bwhv_player_games pg ON pg.report_id = r.id
		 WHERE g.staffel_id = ? AND r.state = 'parsed' AND TRIM(r.referees_json) <> ''
		 GROUP BY r.id`, staffelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byName := map[string]*RefereeStat{}
	var order []string
	for rows.Next() {
		var refJSON string
		var uncertain bool
		var twoMin, yellow, red, blue int
		if err := rows.Scan(&refJSON, &uncertain, &twoMin, &yellow, &red, &blue); err != nil {
			return nil, err
		}
		for _, name := range decodeStrings(refJSON) {
			st, ok := byName[name]
			if !ok {
				st = &RefereeStat{Name: name}
				byName[name] = st
				order = append(order, name)
			}
			st.Games++
			st.TwoMin += twoMin
			st.Yellow += yellow
			st.Red += red
			st.Blue += blue
			// Ein einziger geratener Schnitt genügt, um den Namen als unsicher
			// zu kennzeichnen: er kann in diesem Spiel falsch geschnitten
			// worden sein, und die Zeile trägt dann fremde Spiele mit.
			st.Uncertain = st.Uncertain || uncertain
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]RefereeStat, 0, len(order))
	for _, name := range order {
		out = append(out, *byName[name])
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Games != out[j].Games {
			return out[i].Games > out[j].Games
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Affiliation ist die Zugehörigkeit eines Nutzers zu einer Staffel: welche
// Mannschaften der Staffel seine sind und welche Spielerzeilen ihm gehören.
//
// Eigene Route und eigener Typ, nicht in jede Statistik-Antwort gemischt: die
// vier Statistik-Routen bleiben damit nutzerunabhängig. Sonst hinge jede
// Antwort am Token, wäre pro Nutzer verschieden und jeder Statistik-Test
// müsste die Zugehörigkeit mit auswerten (design.md §10).
type Affiliation struct {
	TeamNames []string `json:"teamNames"`
	PlayerIDs []int    `json:"playerIds"`
}

// Affiliation löst die eigene Zugehörigkeit zu einer Staffel auf.
//
// Die Mannschaft wird ABGELEITET, nicht über einen Namensvergleich geraten:
// bwhv_games.home_team trägt die Schreibweise des Verbands ("Team Stuttgart
// 2"), teams.name die des Vereins. Ein Namensabgleich müsste Vereinsnamen,
// Mannschaftsnummer und Suffixe aufeinander abbilden — internal/h4aimport tut
// das für den Spielimport und braucht dafür eigene Regeln. Die Maschinerie
// hier zu wiederholen hieße, dieselbe Ratearbeit an zweiter Stelle zu pflegen,
// und ein Fehlschluss wäre unsichtbar: hervorgehoben wäre die falsche Zeile,
// und nichts würde widersprechen.
//
// Stattdessen liefert die Verknüpfung die Antwort umsonst: eine Begegnung mit
// bwhv_games.game_id IST ein eigenes Spiel, games.is_home sagt die Seite,
// game_teams die Mannschaft.
//
// PREIS: ohne eine einzige verknüpfte Begegnung gibt es keine Hervorhebung
// (Saisonbeginn, bevor ein eigener Termin importiert ist). Das ist gewollt —
// eine fehlende Markierung fällt auf, eine falsche nicht.
func (s *Store) Affiliation(ctx context.Context, staffelID, userID int) (*Affiliation, error) {
	out := &Affiliation{TeamNames: []string{}, PlayerIDs: []int{}}

	// Die Herleitung selbst steht in ownTeams — eine Kopie, zwei Nutzer
	// (diese Route und die Spielmatrix). user_accessible_teams fasst dort
	// Stammkader, erweiterten Kader, Trainer und Eltern schon zusammen, und die
	// Saison kommt aus der Staffel, nicht aus der aktiven: beide sind hier
	// dieselbe (staffelOfRequest lässt nur Staffeln der aktiven Saison durch),
	// und der Bezug auf die Staffel macht das unabhängig davon richtig.
	teams, err := s.ownTeams(ctx, staffelID, userID)
	if err != nil {
		return nil, err
	}
	for _, t := range teams {
		out.TeamNames = append(out.TeamNames, t.Name)
	}
	sort.Strings(out.TeamNames)

	// Die eigenen Spielerzeilen sind der einfache Teil: bwhv_players.member_id
	// ist für eigene Spieler schon gesetzt. Abgeglichen wird gegen die
	// Mitglieder des Accounts plus die Kinder über family_links — dieselbe
	// Menge, die dutyfairness und attendance.canSeeMemberStats heranziehen.
	prows, err := s.db.QueryContext(ctx, `
		SELECT p.id
		  FROM bwhv_players p
		 WHERE p.staffel_id = ?
		   AND p.member_id IS NOT NULL
		   AND (p.member_id IN (SELECT id FROM members WHERE user_id = ?)
		     OR p.member_id IN (SELECT member_id FROM family_links WHERE parent_user_id = ?))
		 ORDER BY p.id`, staffelID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	for prows.Next() {
		var id int
		if err := prows.Scan(&id); err != nil {
			return nil, err
		}
		out.PlayerIDs = append(out.PlayerIDs, id)
	}
	return out, prows.Err()
}

// --- Spielmatrix einer Mannschaft -----------------------------------------

// MatrixCell sind die Werte eines Spielers in einer Begegnung — und, in der
// Summenzeile, die einer ganzen Mannschaft.
//
// Ohne Spalte "Blau": bwhv_player_games.blue wird von SaveReport konstant als 0
// geschrieben, der Parser liest keine blauen Karten. Eine Spalte, die immer
// leer ist, behauptet eine Messung, die nicht stattfindet (design.md §10).
type MatrixCell struct {
	Goals       int `json:"goals"`
	SevenMAtt   int `json:"sevenMAttempts"`
	SevenMGoals int `json:"sevenMGoals"`
	TwoMin      int `json:"twoMin"`
	Warnings    int `json:"warnings"`
	Disq        int `json:"disq"`
}

func (c *MatrixCell) add(o MatrixCell) {
	c.Goals += o.Goals
	c.SevenMAtt += o.SevenMAtt
	c.SevenMGoals += o.SevenMGoals
	c.TwoMin += o.TwoMin
	c.Warnings += o.Warnings
	c.Disq += o.Disq
}

// MatrixGame ist eine Spalte der Matrix: eine gespielte Begegnung.
//
// HasReport ist kein Detail, sondern die zweite Abdeckung: eine Begegnung ohne
// ausgewerteten Bericht bleibt eine Spalte mit Endstand, aber ohne Zellen.
// Sie wegzulassen hieße, die Saisonsumme über eine unsichtbare Teilmenge zu
// bilden (design.md §3).
type MatrixGame struct {
	BwhvGameID int    `json:"bwhvGameId"`
	Date       string `json:"date"`
	HomeTeam   string `json:"homeTeam"`
	GuestTeam  string `json:"guestTeam"`
	IsHome     bool   `json:"isHome"`
	HomeGoals  *int   `json:"homeGoals"`
	GuestGoals *int   `json:"guestGoals"`
	HasReport  bool   `json:"hasReport"`
}

// MatrixPlayer ist eine Zeile der Matrix.
//
// Cells läuft parallel zu TeamMatrix.Games. Ein NIL-Eintrag heißt "stand in der
// Mannschaftsliste dieser Begegnung nicht" und ist ausdrücklich etwas anderes
// als eine Zelle mit lauter Nullen (design.md §4).
type MatrixPlayer struct {
	PlayerID int           `json:"playerId"`
	MemberID *int          `json:"memberId"`
	Name     string        `json:"name"`
	Cells    []*MatrixCell `json:"cells"`
	Total    MatrixCell    `json:"total"`
	Games    int           `json:"games"`
}

// TeamMatrix ist die Spielmatrix einer Mannschaft.
type TeamMatrix struct {
	Team string `json:"team"`
	// HalfDurationMinutes ist die für die Altersklasse gepflegte Halbzeitdauer
	// (age_class_game_rules). NIL heißt "keine Regel gepflegt" — bewusst kein
	// Standardwert, denn eine erfundene Zahl sähe aus wie eine gemessene
	// (design.md §7).
	HalfDurationMinutes *int           `json:"halfDurationMinutes"`
	Games               []MatrixGame   `json:"games"`
	Players             []MatrixPlayer `json:"players"`
	GameTotals          []MatrixCell   `json:"gameTotals"`
	Total               MatrixCell     `json:"total"`
	ReportGames         int            `json:"reportGames"`
}

// ownTeam ist eine dem Nutzer zuzurechnende Mannschaft einer Staffel, benannt
// in der Schreibweise des SPIELPLANS, mit der Halbzeitdauer ihrer Altersklasse.
type ownTeam struct {
	Name         string
	HalfDuration *int
}

// ownTeams löst die eigenen Mannschaften einer Staffel auf.
//
// Die Herleitung steht ausführlich an Affiliation: die Mannschaft wird über die
// Verknüpfung bwhv_games.game_id -> games -> game_teams gewonnen, NIE über
// einen Namensvergleich. Diese Funktion ist die einzige Kopie davon; Affiliation
// und PlayerGameMatrix greifen beide hierauf zu, damit die Regel nicht an zwei
// Stellen gepflegt werden muss.
//
// Die Halbzeitdauer kommt über teams.age_class aus age_class_game_rules —
// derselbe Weg, den die Dienst-Regeneration für die Spieldauer geht
// (internal/games/regen.go). LEFT JOIN, weil die Tabelle nur A- bis D-Jugend
// kennt und nicht vorbefüllt ist.
func (s *Store) ownTeams(ctx context.Context, staffelID, userID int) ([]ownTeam, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT CASE WHEN own.is_home = 1 THEN bg.home_team ELSE bg.guest_team END AS bwhv_name,
		       acr.half_duration_minutes
		  FROM bwhv_games bg
		  JOIN bwhv_staffeln st ON st.id = bg.staffel_id
		  JOIN games own ON own.id = bg.game_id
		  JOIN game_teams gt ON gt.game_id = own.id
		  JOIN user_accessible_teams uat
		    ON uat.team_id = gt.team_id AND uat.season_id = st.season_id
		  JOIN teams tm ON tm.id = gt.team_id
		  LEFT JOIN age_class_game_rules acr ON acr.age_class = tm.age_class
		 WHERE bg.staffel_id = ? AND uat.user_id = ?
		 ORDER BY bwhv_name, gt.team_id`, staffelID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ownTeam
	seen := map[string]int{}
	for rows.Next() {
		var name string
		var half sql.NullInt64
		if err := rows.Scan(&name, &half); err != nil {
			return nil, err
		}
		if name == "" {
			continue
		}
		// Dieselbe Mannschaft kann über mehrere verknüpfte Begegnungen kommen.
		// Die erste Zeile gewinnt; eine später auftauchende Halbzeitdauer füllt
		// eine noch offene Angabe nach.
		if i, ok := seen[name]; ok {
			if out[i].HalfDuration == nil && half.Valid {
				v := int(half.Int64)
				out[i].HalfDuration = &v
			}
			continue
		}
		t := ownTeam{Name: name}
		if half.Valid {
			v := int(half.Int64)
			t.HalfDuration = &v
		}
		seen[name] = len(out)
		out = append(out, t)
	}
	return out, rows.Err()
}

// PlayerGameMatrix liefert je eigener Mannschaft der Staffel die Werte jedes
// Spielers in jeder ihrer gespielten Begegnungen.
//
// Ohne zuzurechnende Mannschaft ist das Ergebnis leer und KEIN Fehler: zu
// Saisonbeginn, bevor ein eigener Termin mit external_id importiert ist, gibt
// es sie schlicht nicht (design.md §2).
func (s *Store) PlayerGameMatrix(ctx context.Context, staffelID, userID int) ([]TeamMatrix, error) {
	teams, err := s.ownTeams(ctx, staffelID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]TeamMatrix, 0, len(teams))
	for _, t := range teams {
		m, err := s.matrixForTeam(ctx, staffelID, t)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, nil
}

// matrixGames liefert die gespielten Begegnungen einer Mannschaft als Spalten.
//
// "Gespielt" heißt home_goals IS NOT NULL — dieselbe Regel wie in der
// Kreuztabelle, damit ein torloses 0:0 nicht als "nicht gespielt" verschwindet.
func (s *Store) matrixGames(ctx context.Context, staffelID int, team string) ([]MatrixGame, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id, g.date, g.home_team, g.guest_team, g.home_goals, g.guest_goals,
		       EXISTS (SELECT 1 FROM bwhv_reports r
		                WHERE r.bwhv_game_id = g.id AND r.state = 'parsed')
		  FROM bwhv_games g
		 WHERE g.staffel_id = ?
		   AND g.home_goals IS NOT NULL
		   AND (g.home_team = ? OR g.guest_team = ?)
		 ORDER BY g.date, g.game_no`, staffelID, team, team)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MatrixGame
	for rows.Next() {
		var g MatrixGame
		if err := rows.Scan(&g.BwhvGameID, &g.Date, &g.HomeTeam, &g.GuestTeam,
			&g.HomeGoals, &g.GuestGoals, &g.HasReport); err != nil {
			return nil, err
		}
		g.Date = trimDate(g.Date)
		g.IsHome = g.HomeTeam == team
		out = append(out, g)
	}
	return out, rows.Err()
}

// matrixForTeam setzt Spalten, Zeilen und Summen einer Mannschaft zusammen.
func (s *Store) matrixForTeam(ctx context.Context, staffelID int, t ownTeam) (*TeamMatrix, error) {
	games, err := s.matrixGames(ctx, staffelID, t.Name)
	if err != nil {
		return nil, err
	}
	m := &TeamMatrix{
		Team:                t.Name,
		HalfDurationMinutes: t.HalfDuration,
		Games:               games,
		Players:             []MatrixPlayer{},
		GameTotals:          make([]MatrixCell, len(games)),
	}
	column := make(map[int]int, len(games))
	for i, g := range games {
		column[g.BwhvGameID] = i
		if g.HasReport {
			m.ReportGames++
		}
	}
	if len(games) == 0 {
		return m, nil
	}
	if err := s.fillMatrixCells(ctx, staffelID, t.Name, column, m); err != nil {
		return nil, err
	}
	return m, nil
}

// fillMatrixCells trägt die Spielerzeilen ausgewerteter Berichte ein.
//
// Die Mannschaft wird über teamNameExpr an die Schreibweise des SPIELPLANS
// gebunden, nicht über bwhv_players.team_name: die stammt aus der
// Mannschaftsliste des PDF und kann abweichen. Ein Vergleich gegen den
// Berichtsnamen lieferte eine leere Matrix — ohne Fehler, ohne Hinweis
// (design.md §5).
//
// twoMinCounted/redCounted rechnen die dritte Zeitstrafe in eine Rote Karte um,
// wie in jeder anderen Auswertung dieser Staffel.
func (s *Store) fillMatrixCells(ctx context.Context, staffelID int, team string,
	column map[int]int, m *TeamMatrix) error {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id, p.id, p.member_id, p.name,
		       pg.goals, pg.seven_m_attempts, pg.seven_m_goals,
		       `+twoMinCounted+`, pg.yellow, `+redCounted+`
		  FROM bwhv_player_games pg
		  JOIN bwhv_reports r ON r.id = pg.report_id AND r.state = 'parsed'
		  JOIN bwhv_games g ON g.id = r.bwhv_game_id
		  JOIN bwhv_players p ON p.id = pg.player_id
		 WHERE g.staffel_id = ? AND `+teamNameExpr+` = ?
		 ORDER BY p.name, p.id`, staffelID, team)
	if err != nil {
		return err
	}
	defer rows.Close()

	byPlayer := map[int]int{}
	for rows.Next() {
		var bwhvGameID, playerID int
		var member sql.NullInt64
		var name string
		var c MatrixCell
		if err := rows.Scan(&bwhvGameID, &playerID, &member, &name,
			&c.Goals, &c.SevenMAtt, &c.SevenMGoals, &c.TwoMin, &c.Warnings, &c.Disq); err != nil {
			return err
		}
		col, ok := column[bwhvGameID]
		if !ok {
			// Bericht zu einer Begegnung ohne erfasstes Ergebnis — keine Spalte.
			continue
		}
		idx, ok := byPlayer[playerID]
		if !ok {
			p := MatrixPlayer{PlayerID: playerID, Name: name, Cells: make([]*MatrixCell, len(m.Games))}
			if member.Valid {
				v := int(member.Int64)
				p.MemberID = &v
			}
			idx = len(m.Players)
			byPlayer[playerID] = idx
			m.Players = append(m.Players, p)
		}
		p := &m.Players[idx]
		cell := c
		p.Cells[col] = &cell
		p.Total.add(c)
		p.Games++
		m.GameTotals[col].add(c)
		m.Total.add(c)
	}
	return rows.Err()
}
