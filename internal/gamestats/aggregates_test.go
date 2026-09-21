package gamestats

import (
	"context"
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// seedGame legt eine Begegnung an. Ein nil-Ergebnis heißt "noch nicht
// gespielt"; 0:0 ist ausdrücklich ein Ergebnis.
func seedResult(t *testing.T, db *sql.DB, staffelID int, gameNo, date, home, guest string, hg, gg *int) int {
	t.Helper()
	res, err := db.Exec(`
		INSERT INTO bwhv_games (staffel_id, game_no, date, time, home_team, guest_team,
			home_goals, guest_goals)
		VALUES (?,?,?, '16:00', ?,?,?,?)`, staffelID, gameNo, date, home, guest, hg, gg)
	if err != nil {
		t.Fatalf("Begegnung anlegen: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func cellOf(t *testing.T, ct *CrossTable, home, guest string) *CrossCell {
	t.Helper()
	for _, r := range ct.Rows {
		if r.Team != home {
			continue
		}
		for i, name := range ct.Teams {
			if name == guest {
				return r.Cells[i]
			}
		}
	}
	t.Fatalf("Paarung %q – %q nicht in der Kreuztabelle", home, guest)
	return nil
}

// Die drei Zellenarten in einem Durchgang: Endstand, torloses Spiel, Datum.
func TestCrossTable_ErgebnisTorlosUndDatum(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(29), intp(25))
	seedResult(t, db, staffelID, "2", "2026-09-27", "B", "C", intp(0), intp(0))
	seedResult(t, db, staffelID, "3", "2026-10-04", "C", "A", nil, nil)

	ct, err := s.CrossTable(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("CrossTable: %v", err)
	}
	if len(ct.Teams) != 3 {
		t.Fatalf("Mannschaften = %v, erwartet drei", ct.Teams)
	}

	ab := cellOf(t, ct, "A", "B")
	if !ab.Played || *ab.HomeGoals != 29 || *ab.GuestGoals != 25 {
		t.Errorf("A–B = %+v, erwartet gespielt 29:25", ab)
	}
	bc := cellOf(t, ct, "B", "C")
	if !bc.Played || *bc.HomeGoals != 0 || *bc.GuestGoals != 0 {
		t.Errorf("B–C = %+v, erwartet gespielt 0:0 — ein torloses Spiel ist gespielt", bc)
	}
	if bc.Date != "" {
		t.Errorf("B–C trägt das Datum %q, erwartet keines: die Begegnung ist gespielt", bc.Date)
	}
	ca := cellOf(t, ct, "C", "A")
	if ca.Played || ca.Date != "2026-10-04" {
		t.Errorf("C–A = %+v, erwartet ungespielt mit Datum", ca)
	}
}

// Die Diagonale ist leer, und eine nicht angesetzte Paarung ebenso.
func TestCrossTable_DiagonaleLeer(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(1), intp(2))

	ct, err := s.CrossTable(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("CrossTable: %v", err)
	}
	for i, r := range ct.Rows {
		if r.Cells[i] != nil {
			t.Errorf("Diagonale von %q ist gefüllt: %+v", r.Team, r.Cells[i])
		}
	}
	if c := cellOf(t, ct, "B", "A"); c != nil {
		t.Errorf("B–A = %+v, erwartet leer — das Rückspiel ist nicht angesetzt", c)
	}
}

// Ohne einen einzigen ausgewerteten Spielbericht ist die Kreuztabelle
// vollständig: sie hängt an den Ergebnissen, nicht an den PDFs.
func TestCrossTable_OhneBerichtVollstaendig(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(29), intp(25))
	seedResult(t, db, staffelID, "2", "2026-09-20", "B", "A", intp(20), intp(21))

	if n := countRows(t, db, "bwhv_reports"); n != 0 {
		t.Fatalf("Vorbedingung: %d Berichte, erwartet keinen", n)
	}
	ct, err := s.CrossTable(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("CrossTable: %v", err)
	}
	if !cellOf(t, ct, "A", "B").Played || !cellOf(t, ct, "B", "A").Played {
		t.Error("erwartet: beide Paarungen gefüllt ohne jeden Spielbericht")
	}
}

// Die DATE-Falle: ein als ISO-Timestamp gespeichertes Datum muss als reines
// Datum herauskommen, sonst zeigte die Zelle "2026-10-04T00:00:00Z".
func TestCrossTable_IsoTimestampWirdGetrimmt(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-10-04T00:00:00Z", "A", "B", nil, nil)

	ct, err := s.CrossTable(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("CrossTable: %v", err)
	}
	if got := cellOf(t, ct, "A", "B").Date; got != "2026-10-04" {
		t.Errorf("Datum = %q, erwartet %q", got, "2026-10-04")
	}
}

func rankOf(t *testing.T, d ProgressionDay, team string) int {
	t.Helper()
	for _, e := range d.Entries {
		if e.Team == team {
			return e.Rank
		}
	}
	t.Fatalf("Mannschaft %q fehlt im Spieltag %s", team, d.Date)
	return 0
}

// Drei Spieltage, drei Punkte im Verlauf — und die Rangfolge ändert sich
// unterwegs, weil der Verlauf kumuliert rechnet.
func TestStandingsProgression_RangJeSpieltag(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(30), intp(20))
	seedResult(t, db, staffelID, "2", "2026-09-27", "B", "C", intp(25), intp(20))
	seedResult(t, db, staffelID, "3", "2026-10-04", "C", "A", intp(40), intp(20))

	days, err := s.StandingsProgression(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("StandingsProgression: %v", err)
	}
	if len(days) != 3 {
		t.Fatalf("Spieltage = %d, erwartet 3", len(days))
	}
	if days[0].Date != "2026-09-20" || days[2].Date != "2026-10-04" {
		t.Errorf("Daten = %q…%q, erwartet chronologisch", days[0].Date, days[2].Date)
	}
	if r := rankOf(t, days[0], "A"); r != 1 {
		t.Errorf("A nach Spieltag 1 = Rang %d, erwartet 1", r)
	}
	// Nach Spieltag 3 haben A, B und C je zwei Punkte; die Tordifferenz
	// entscheidet (C +20, A -10... ): C führt.
	if r := rankOf(t, days[2], "C"); r != 1 {
		t.Errorf("C nach Spieltag 3 = Rang %d, erwartet 1 (beste Tordifferenz)", r)
	}
}

// Bei Punktgleichstand entscheidet die Tordifferenz, danach die geworfenen
// Tore.
func TestStandingsProgression_DifferenzBrichtGleichstand(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "A", "X", intp(30), intp(10))
	seedResult(t, db, staffelID, "2", "2026-09-20", "B", "Y", intp(21), intp(20))

	days, err := s.StandingsProgression(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("StandingsProgression: %v", err)
	}
	if len(days) != 1 {
		t.Fatalf("Spieltage = %d, erwartet 1 (beide Begegnungen am selben Datum)", len(days))
	}
	if rankOf(t, days[0], "A") >= rankOf(t, days[0], "B") {
		t.Error("erwartet: A vor B — gleich viele Punkte, aber bessere Tordifferenz")
	}
}

// Die DATE-Falle: derselbe Spieltag darf nicht zweimal erscheinen, nur weil
// eine Zeile als ISO-Timestamp gespeichert ist.
func TestStandingsProgression_IsoTimestampWirdGetrimmt(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(30), intp(20))
	seedResult(t, db, staffelID, "2", "2026-09-20T00:00:00Z", "C", "D", intp(25), intp(20))

	days, err := s.StandingsProgression(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("StandingsProgression: %v", err)
	}
	if len(days) != 1 {
		t.Fatalf("Spieltage = %d, erwartet 1 — %q und %q sind derselbe Tag",
			len(days), "2026-09-20", "2026-09-20T00:00:00Z")
	}
	if days[0].Date != "2026-09-20" || len(days[0].Entries) != 4 {
		t.Errorf("Spieltag = %q mit %d Mannschaften, erwartet 2026-09-20 mit 4",
			days[0].Date, len(days[0].Entries))
	}
}

// Ein noch nicht gespieltes Spiel erzeugt keinen Punkt im Verlauf — auch dann
// nicht, wenn es chronologisch zwischen gespielten liegt.
func TestStandingsProgression_OhneErgebnisLeer(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", nil, nil)

	days, err := s.StandingsProgression(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("StandingsProgression: %v", err)
	}
	if len(days) != 0 {
		t.Fatalf("Spieltage = %d, erwartet 0 — es wurde noch nichts gespielt", len(days))
	}
}

// Die Zwei-Punkte-Wertung steht als Konstante da; der Test hält sie fest,
// damit eine Änderung sichtbar wird statt sich still durch alle Ränge zu
// ziehen.
func TestStandingsProgression_ZweiPunkteWertung(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(30), intp(20))
	seedResult(t, db, staffelID, "2", "2026-09-27", "A", "C", intp(20), intp(20))

	days, err := s.StandingsProgression(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("StandingsProgression: %v", err)
	}
	for _, e := range days[1].Entries {
		if e.Team != "A" {
			continue
		}
		if e.Points != 3 {
			t.Errorf("Punkte von A = %d, erwartet 3 (Sieg 2 + Unentschieden 1)", e.Points)
		}
		if e.Games != 2 {
			t.Errorf("Spiele von A = %d, erwartet 2", e.Games)
		}
	}
}

// rosterLine ist eine Spielerzeile, wie sie eine Mannschaftsliste liefert.
type rosterLine struct {
	name                          string
	side                          string
	goals, sevenMAtt, sevenMGoals int
	twoMin, yellow, red           int
}

// seedParsedReport legt zu einer Begegnung einen Bericht im angegebenen Zustand an
// und schreibt die Spielerzeilen. team ordnet die Seite der Mannschaft in der
// Schreibweise der MANNSCHAFTSLISTE zu — bewusst abweichend von der des
// Spielplans, damit die Brücke über pg.side geprüft wird.
func seedParsedReport(t *testing.T, db *sql.DB, staffelID, bwhvGameID int, state string,
	rosterTeam map[string]string, lines []rosterLine) int {
	t.Helper()
	res, err := db.Exec(`
		INSERT INTO bwhv_reports (bwhv_game_id, sgid, state, parsed_at)
		VALUES (?, '1', ?, '2026-09-20T18:00:00Z')`, bwhvGameID, state)
	if err != nil {
		t.Fatalf("Bericht anlegen: %v", err)
	}
	id, _ := res.LastInsertId()
	reportID := int(id)
	if state != "parsed" {
		return reportID
	}
	for _, l := range lines {
		var playerID int
		err := db.QueryRow(`
			INSERT INTO bwhv_players (staffel_id, team_name, name)
			VALUES (?,?,?)
			ON CONFLICT (staffel_id, team_name, name) DO UPDATE SET name = excluded.name
			RETURNING id`, staffelID, rosterTeam[l.side], l.name).Scan(&playerID)
		if err != nil {
			t.Fatalf("Spieler anlegen: %v", err)
		}
		if _, err := db.Exec(`
			INSERT INTO bwhv_player_games (report_id, player_id, side, goals,
				seven_m_attempts, seven_m_goals, two_min, yellow, red, blue)
			VALUES (?,?,?,?,?,?,?,?,?,0)`,
			reportID, playerID, l.side, l.goals, l.sevenMAtt, l.sevenMGoals,
			l.twoMin, l.yellow, l.red); err != nil {
			t.Fatalf("Spielerzeile anlegen: %v", err)
		}
	}
	return reportID
}

func teamStatOf(t *testing.T, ts *TeamStats, team string) TeamStat {
	t.Helper()
	for _, s := range ts.Teams {
		if s.Team == team {
			return s
		}
	}
	t.Fatalf("Mannschaft %q fehlt in den Mannschafts-Ranglisten", team)
	return TeamStat{}
}

// Torverhältnis, Angriff und Verteidigung stehen ohne ein einziges PDF; die
// Fair-Play-Wertung bleibt dafür LEER statt null.
func TestTeamStats_ToreOhneBerichtFairPlayLeer(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(30), intp(20))
	seedResult(t, db, staffelID, "2", "2026-09-27", "B", "A", intp(25), intp(28))

	ts, err := s.TeamStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("TeamStats: %v", err)
	}
	a := teamStatOf(t, ts, "A")
	if a.Games != 2 || a.GoalsFor != 58 || a.GoalsAgainst != 45 || a.GoalDiff != 13 {
		t.Errorf("A = %+v, erwartet 2 Spiele, 58:45, Differenz 13", a)
	}
	if a.FairPlay != nil {
		t.Errorf("FairPlay von A = %v, erwartet leer — ohne Bericht ist die Mannschaft nicht straffrei, sondern unbekannt", *a.FairPlay)
	}
	if a.Distribution != nil {
		t.Errorf("Distribution von A = %+v, erwartet leer", *a.Distribution)
	}
	if a.ReportGames != 0 {
		t.Errorf("ReportGames von A = %d, erwartet 0", a.ReportGames)
	}
}

// Die Fair-Play-Wertung und die Torverteilung kommen aus den Berichten — und
// die Mannschaft wird über pg.side der Schreibweise des SPIELPLANS zugeordnet,
// nicht der des Berichts.
func TestTeamStats_FairPlayAusBerichten(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	gameID := seedResult(t, db, staffelID, "1", "2026-09-20", "Verein A", "Verein B", intp(10), intp(8))
	seedParsedReport(t, db, staffelID, gameID, "parsed",
		map[string]string{"home": "Verein A e.V.", "guest": "Verein B e.V."},
		[]rosterLine{
			{name: "Anna", side: "home", goals: 6, twoMin: 1, yellow: 2},
			{name: "Bea", side: "home", goals: 4},
			{name: "Cem", side: "guest", goals: 8, red: 1},
		})

	ts, err := s.TeamStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("TeamStats: %v", err)
	}
	a := teamStatOf(t, ts, "Verein A")
	if a.ReportGames != 1 || a.TwoMin != 1 || a.Yellow != 2 {
		t.Errorf("A = %+v, erwartet 1 Bericht, 1× 2 min, 2 Verwarnungen", a)
	}
	if a.FairPlay == nil || *a.FairPlay != 2*1+1*2 {
		t.Errorf("FairPlay von A = %v, erwartet 4 (2 Gelb à 1 + 1× 2 min à 2)", a.FairPlay)
	}
	if a.Distribution == nil || a.Distribution.Players != 2 || a.Distribution.Average != 5 {
		t.Errorf("Distribution von A = %+v, erwartet 2 Spieler mit Ø 5", a.Distribution)
	}
	if ts.FairPlayWeights.Blue != 4 || ts.FairPlayWeights.Yellow != 1 {
		t.Errorf("Gewichtung = %+v, erwartet Blau 4 / Gelb 1 — sie muss mitgeliefert werden", ts.FairPlayWeights)
	}
	// Die Mannschaft darf nicht zusätzlich unter dem Berichtsnamen erscheinen.
	for _, st := range ts.Teams {
		if st.Team == "Verein A e.V." {
			t.Error("Mannschaft unter der Schreibweise des Berichts geführt — erwartet die des Spielplans")
		}
	}
}

// Ein gescheiterter Bericht trägt keine Spielerzeilen und darf deshalb auch
// keine Fair-Play-Wertung erzeugen.
func TestTeamStats_ParseFailedZaehltNicht(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	gameID := seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(10), intp(8))
	seedParsedReport(t, db, staffelID, gameID, "parse_failed", nil, nil)

	ts, err := s.TeamStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("TeamStats: %v", err)
	}
	if a := teamStatOf(t, ts, "A"); a.FairPlay != nil || a.ReportGames != 0 {
		t.Errorf("A = %+v, erwartet keine Wertung aus einem parse_failed-Bericht", a)
	}
}

// Fremde Mannschaften sind gleichberechtigt enthalten — die Staffel ist der
// Bezugsraum, nicht der eigene Verein.
func TestTeamStats_FremdeMannschaftenEnthalten(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "Fremdverein X", "Fremdverein Y", intp(10), intp(8))

	ts, err := s.TeamStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("TeamStats: %v", err)
	}
	if len(ts.Teams) != 2 {
		t.Fatalf("Mannschaften = %d, erwartet 2", len(ts.Teams))
	}
}

// Der Gini über die Saisonsummen: gleichmäßig ≈ 0, ein Alleinwerfer nahe 1,
// eine torlose Mannschaft genau 0.
func TestGini_GleichverteilungMedianAlleinwerfer(t *testing.T) {
	for _, c := range []struct {
		name   string
		values []float64
		lo, hi float64
	}{
		{"gleichverteilt", []float64{5, 5, 5, 5}, 0, 0.001},
		{"alleinwerfer", []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 40}, 0.85, 1},
		{"torlos", []float64{0, 0, 0}, 0, 0},
	} {
		d := distributionOf(c.values)
		if d.Gini < c.lo || d.Gini > c.hi {
			t.Errorf("%s: Gini = %v, erwartet zwischen %v und %v", c.name, d.Gini, c.lo, c.hi)
		}
	}

	d := distributionOf([]float64{1, 2, 6})
	if d.Median != 2 {
		t.Errorf("Median = %v, erwartet 2", d.Median)
	}
	if d.Average != 3 {
		t.Errorf("Ø = %v, erwartet 3", d.Average)
	}
}

// Gerechnet wird über die Saisonsumme je Spieler, nicht über die Zeile je
// Spiel: zwei Spieler mit je zwei Einsätzen sind zwei Werte, nicht vier.
func TestGini_UeberSaisonsummeNichtProSpiel(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	roster := map[string]string{"home": "A", "guest": "B"}
	for i, no := range []string{"1", "2"} {
		gameID := seedResult(t, db, staffelID, no, "2026-09-2"+no, "A", "B", intp(10), intp(8))
		seedParsedReport(t, db, staffelID, gameID, "parsed", roster, []rosterLine{
			{name: "Anna", side: "home", goals: 5 + i},
			{name: "Bea", side: "home", goals: 5 - i},
		})
	}

	ts, err := s.TeamStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("TeamStats: %v", err)
	}
	a := teamStatOf(t, ts, "A")
	if a.Distribution == nil || a.Distribution.Players != 2 {
		t.Fatalf("Distribution = %+v, erwartet 2 Spieler (Saisonsummen), nicht 4 Spieler-Spiele", a.Distribution)
	}
	// Anna 5+6 = 11, Bea 5+4 = 9 → Ø 10.
	if a.Distribution.Average != 10 {
		t.Errorf("Ø = %v, erwartet 10 (Saisonsummen 11 und 9)", a.Distribution.Average)
	}
	if a.ReportGames != 2 {
		t.Errorf("ReportGames = %d, erwartet 2", a.ReportGames)
	}
}

// setReferees schreibt die getrennten Namen an einen Bericht.
func setReferees(t *testing.T, db *sql.DB, reportID int, uncertain bool, names ...string) {
	t.Helper()
	u := 0
	if uncertain {
		u = 1
	}
	if _, err := db.Exec(`UPDATE bwhv_reports SET referees_json = ?, referees_uncertain = ?
		WHERE id = ?`, encodeStrings(names), u, reportID); err != nil {
		t.Fatalf("Schiedsrichter setzen: %v", err)
	}
}

// Dasselbe Gespann in zwei Spielen: zwei Spiele und die Summe der Strafen
// BEIDER Mannschaften.
func TestRefereeStats_SpieleUndStrafen(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	roster := map[string]string{"home": "A", "guest": "B"}
	for _, no := range []string{"1", "2"} {
		gameID := seedResult(t, db, staffelID, no, "2026-09-2"+no, "A", "B", intp(10), intp(8))
		reportID := seedParsedReport(t, db, staffelID, gameID, "parsed", roster, []rosterLine{
			{name: "Anna" + no, side: "home", twoMin: 1},
			{name: "Cem" + no, side: "guest", twoMin: 2, yellow: 1},
		})
		setReferees(t, db, reportID, false, "Max Mustermann", "Peter Müller")
	}

	list, err := s.RefereeStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("RefereeStats: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Schiedsrichter = %d, erwartet 2", len(list))
	}
	for _, st := range list {
		if st.Games != 2 {
			t.Errorf("%s: Spiele = %d, erwartet 2", st.Name, st.Games)
		}
		if st.TwoMin != 6 {
			t.Errorf("%s: Zeitstrafen = %d, erwartet 6 — die Strafen BEIDER Mannschaften zählen", st.Name, st.TwoMin)
		}
		if st.Yellow != 2 {
			t.Errorf("%s: Verwarnungen = %d, erwartet 2", st.Name, st.Yellow)
		}
		if st.Uncertain {
			t.Errorf("%s: als unsicher gekennzeichnet, erwartet gesichert", st.Name)
		}
	}
}

// Ein Bericht ohne benannte Schiedsrichter erzeugt keine Zeile — das gilt auch
// für den Bestand vor Migration 069.
func TestRefereeStats_OhneNamenKeineZeile(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	gameID := seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(10), intp(8))
	seedParsedReport(t, db, staffelID, gameID, "parsed",
		map[string]string{"home": "A", "guest": "B"},
		[]rosterLine{{name: "Anna", side: "home", twoMin: 1}})

	list, err := s.RefereeStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("RefereeStats: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("Schiedsrichter = %#v, erwartet keine Zeile", list)
	}
}

// Ein geratener Schnitt wird durchgereicht, statt als gesichert zu erscheinen.
func TestRefereeStats_UnsichereTrennungGekennzeichnet(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	gameID := seedResult(t, db, staffelID, "1", "2026-09-20", "A", "B", intp(10), intp(8))
	reportID := seedParsedReport(t, db, staffelID, gameID, "parsed",
		map[string]string{"home": "A", "guest": "B"},
		[]rosterLine{{name: "Anna", side: "home"}})
	setReferees(t, db, reportID, true, "Max Mustermann", "Peter Müller")

	list, err := s.RefereeStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("RefereeStats: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Schiedsrichter = %d, erwartet 2", len(list))
	}
	for _, st := range list {
		if !st.Uncertain {
			t.Errorf("%s: nicht als unsicher gekennzeichnet", st.Name)
		}
	}
}

// linkOwnGame verknüpft eine BWHV-Begegnung mit einem eigenen Spieltermin —
// genau die Verknüpfung, aus der die Zugehörigkeit abgeleitet wird.
func linkOwnGame(t *testing.T, db *sql.DB, bwhvGameID, seasonID, teamID int, isHome bool) {
	t.Helper()
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-09-20")
	home := 0
	if isHome {
		home = 1
	}
	if _, err := db.Exec(`UPDATE games SET is_home = ? WHERE id = ?`, home, gameID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE bwhv_games SET game_id = ? WHERE id = ?`, gameID, bwhvGameID); err != nil {
		t.Fatal(err)
	}
}

// addToKader hängt ein Mitglied als Spieler an den Kader.
func addToKader(t *testing.T, db *sql.DB, kaderID, memberID int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?,?)`,
		kaderID, memberID); err != nil {
		t.Fatalf("kader_members: %v", err)
	}
}

// Ein Spieler sieht seine Mannschaft und sich selbst. Die Mannschaft kommt aus
// der Verknüpfung, nicht aus einem Namensvergleich: der Vereinsname
// ("B-Jugend") und der Verbandsname ("Team Stuttgart 2") haben nichts
// gemeinsam.
func TestAffiliation_SpielerSiehtMannschaftUndSichSelbst(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	teamID := testutil.CreateTeam(t, db, "B-Jugend")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	addToKader(t, db, kaderID, memberID)

	bwhvGameID := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(10), intp(8))
	linkOwnGame(t, db, bwhvGameID, seasonID, teamID, true)

	var playerID int
	if err := db.QueryRow(`INSERT INTO bwhv_players (staffel_id, team_name, name, member_id)
		VALUES (?,?,?,?) RETURNING id`, staffelID, "Team Stuttgart 2", "Anna", memberID).Scan(&playerID); err != nil {
		t.Fatal(err)
	}

	aff, err := s.Affiliation(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("Affiliation: %v", err)
	}
	if len(aff.TeamNames) != 1 || aff.TeamNames[0] != "Team Stuttgart 2" {
		t.Errorf("TeamNames = %#v, erwartet [Team Stuttgart 2]", aff.TeamNames)
	}
	if len(aff.PlayerIDs) != 1 || aff.PlayerIDs[0] != playerID {
		t.Errorf("PlayerIDs = %#v, erwartet [%d]", aff.PlayerIDs, playerID)
	}
}

// Auswärts steht die eigene Mannschaft auf der Gastseite — games.is_home sagt,
// welche.
func TestAffiliation_AuswaertsspielLiefertGastseite(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	teamID := testutil.CreateTeam(t, db, "B-Jugend")
	addToKader(t, db, testutil.CreateKader(t, db, teamID, seasonID), memberID)

	bwhvGameID := seedResult(t, db, staffelID, "1", "2026-09-20", "Fremd", "Team Stuttgart 2", intp(10), intp(8))
	linkOwnGame(t, db, bwhvGameID, seasonID, teamID, false)

	aff, err := s.Affiliation(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("Affiliation: %v", err)
	}
	if len(aff.TeamNames) != 1 || aff.TeamNames[0] != "Team Stuttgart 2" {
		t.Errorf("TeamNames = %#v, erwartet [Team Stuttgart 2]", aff.TeamNames)
	}
}

// Eltern bekommen Mannschaft UND Spielerzeilen ihres Kindes.
func TestAffiliation_ElternteilSiehtKind(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	parentID := testutil.CreateUser(t, db, "standard")
	childMemberID := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`,
		parentID, childMemberID); err != nil {
		t.Fatal(err)
	}
	teamID := testutil.CreateTeam(t, db, "B-Jugend")
	addToKader(t, db, testutil.CreateKader(t, db, teamID, seasonID), childMemberID)

	bwhvGameID := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(10), intp(8))
	linkOwnGame(t, db, bwhvGameID, seasonID, teamID, true)
	var playerID int
	if err := db.QueryRow(`INSERT INTO bwhv_players (staffel_id, team_name, name, member_id)
		VALUES (?,?,?,?) RETURNING id`, staffelID, "Team Stuttgart 2", "Kind", childMemberID).Scan(&playerID); err != nil {
		t.Fatal(err)
	}

	aff, err := s.Affiliation(context.Background(), staffelID, parentID)
	if err != nil {
		t.Fatalf("Affiliation: %v", err)
	}
	if len(aff.TeamNames) != 1 {
		t.Errorf("TeamNames = %#v, erwartet die Mannschaft des Kindes", aff.TeamNames)
	}
	if len(aff.PlayerIDs) != 1 || aff.PlayerIDs[0] != playerID {
		t.Errorf("PlayerIDs = %#v, erwartet die Zeile des Kindes", aff.PlayerIDs)
	}
}

// Trainer und erweiterter Kader sind über user_accessible_teams mit abgedeckt.
func TestAffiliation_TrainerUndErweiterterKader(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	teamID := testutil.CreateTeam(t, db, "B-Jugend")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)

	trainerUser := testutil.CreateUser(t, db, "standard")
	trainerMember := testutil.CreateMember(t, db, trainerUser)
	if _, err := db.Exec(`INSERT INTO kader_trainers (kader_id, member_id) VALUES (?,?)`,
		kaderID, trainerMember); err != nil {
		t.Fatal(err)
	}
	extUser := testutil.CreateUser(t, db, "standard")
	extMember := testutil.CreateMember(t, db, extUser)
	if _, err := db.Exec(`INSERT INTO kader_extended_members (kader_id, member_id) VALUES (?,?)`,
		kaderID, extMember); err != nil {
		t.Fatal(err)
	}

	bwhvGameID := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(10), intp(8))
	linkOwnGame(t, db, bwhvGameID, seasonID, teamID, true)

	for name, userID := range map[string]int{"Trainer": trainerUser, "erweiterter Kader": extUser} {
		aff, err := s.Affiliation(context.Background(), staffelID, userID)
		if err != nil {
			t.Fatalf("Affiliation (%s): %v", name, err)
		}
		if len(aff.TeamNames) != 1 || aff.TeamNames[0] != "Team Stuttgart 2" {
			t.Errorf("%s: TeamNames = %#v, erwartet [Team Stuttgart 2]", name, aff.TeamNames)
		}
	}
}

// Ohne verknüpfte Begegnung bleibt die Menge LEER — auch wenn der
// Mannschaftsname dem eigenen Verein zum Verwechseln ähnlich sieht. Geraten
// wird nicht.
func TestAffiliation_KeinNamensvergleich(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	teamID := testutil.CreateTeam(t, db, "Team Stuttgart 2")
	addToKader(t, db, testutil.CreateKader(t, db, teamID, seasonID), memberID)

	// Gleicher Name, aber keine Verknüpfung über bwhv_games.game_id.
	seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(10), intp(8))

	aff, err := s.Affiliation(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("Affiliation: %v", err)
	}
	if len(aff.TeamNames) != 0 {
		t.Errorf("TeamNames = %#v, erwartet leer — ohne Verknüpfung wird nichts geraten", aff.TeamNames)
	}
}

// Ein Nutzer ohne Kaderzugehörigkeit bekommt leere Mengen, keinen Fehler.
func TestAffiliation_OhneZugehoerigkeitLeer(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	userID := testutil.CreateUser(t, db, "standard")
	teamID := testutil.CreateTeam(t, db, "B-Jugend")
	testutil.CreateKader(t, db, teamID, seasonID)
	bwhvGameID := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(10), intp(8))
	linkOwnGame(t, db, bwhvGameID, seasonID, teamID, true)

	aff, err := s.Affiliation(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("Affiliation: %v", err)
	}
	if len(aff.TeamNames) != 0 || len(aff.PlayerIDs) != 0 {
		t.Errorf("Affiliation = %+v, erwartet beide Mengen leer", aff)
	}
}

// "Spiele" ist die Zahl der Berichte, in deren Mannschaftsliste der Spieler
// steht — nicht die Zahl der Begegnungen seiner Mannschaft. Die Fehlversuche
// bildet der Server, damit die Subtraktion nicht an zwei Stellen lebt.
func TestStaffelStats_SpieleUndSiebenmeterFehlversuche(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	roster := map[string]string{"home": "A", "guest": "B"}
	for i, no := range []string{"1", "2", "3"} {
		gameID := seedResult(t, db, staffelID, no, "2026-09-2"+no, "A", "B", intp(10), intp(8))
		lines := []rosterLine{{name: "Anna", side: "home", goals: 3, sevenMAtt: 2, sevenMGoals: 1}}
		if i == 0 {
			// Bea steht nur in einem Bericht.
			lines = append(lines, rosterLine{name: "Bea", side: "home", goals: 1})
		}
		seedParsedReport(t, db, staffelID, gameID, "parsed", roster, lines)
	}

	stats, err := s.StaffelStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("StaffelStats: %v", err)
	}
	for _, st := range stats {
		switch st.Name {
		case "Anna":
			if st.Games != 3 {
				t.Errorf("Anna: Spiele = %d, erwartet 3", st.Games)
			}
			if st.SevenMAtt != 6 || st.SevenMGoals != 3 || st.SevenMMissed != 3 {
				t.Errorf("Anna: 7m = %d/%d, Fehlversuche %d — erwartet 3/6 und 3",
					st.SevenMGoals, st.SevenMAtt, st.SevenMMissed)
			}
		case "Bea":
			if st.Games != 1 {
				t.Errorf("Bea: Spiele = %d, erwartet 1", st.Games)
			}
			if st.SevenMMissed != 0 {
				t.Errorf("Bea: Fehlversuche = %d, erwartet 0", st.SevenMMissed)
			}
		}
	}
}

// Die dritte Zeitstrafe IST die Disqualifikation (Rote Karte): der Bericht
// trägt dafür beide Einträge, gezählt wird sie nur einmal — als Rot. Eine
// Rote Karte ohne vorherige Zeitstrafen (0 bis 2) bleibt unberührt.
func TestTeamStats_DritteZeitstrafeIstDieRoteKarte(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	gameID := seedResult(t, db, staffelID, "1", "2026-09-20", "Verein A", "Verein B", intp(10), intp(8))
	seedParsedReport(t, db, staffelID, gameID, "parsed",
		map[string]string{"home": "Verein A", "guest": "Verein B"},
		[]rosterLine{
			// 3× 2 min mit eingetragener Disqualifikation → 2× 2 min + 1× Rot.
			{name: "Anna", side: "home", twoMin: 3, red: 1},
			// 3× 2 min, Rot nicht eigens eingetragen → dasselbe Ergebnis.
			{name: "Bea", side: "home", twoMin: 3},
			// Direkte Rote Karte nach einer Zeitstrafe bleibt, wie sie ist.
			{name: "Cem", side: "guest", twoMin: 1, red: 1},
			// Zwei Zeitstrafen sind noch kein Rot.
			{name: "Dan", side: "guest", twoMin: 2},
		})

	ts, err := s.TeamStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("TeamStats: %v", err)
	}
	a := teamStatOf(t, ts, "Verein A")
	if a.TwoMin != 4 || a.Red != 2 {
		t.Errorf("A = %d× 2 min, %d× Rot, erwartet 4× 2 min und 2× Rot (je Spieler 2 + 1)", a.TwoMin, a.Red)
	}
	b := teamStatOf(t, ts, "Verein B")
	if b.TwoMin != 3 || b.Red != 1 {
		t.Errorf("B = %d× 2 min, %d× Rot, erwartet 3× 2 min und 1× Rot", b.TwoMin, b.Red)
	}

	stats, err := s.StaffelStats(context.Background(), staffelID)
	if err != nil {
		t.Fatalf("StaffelStats: %v", err)
	}
	for _, p := range stats {
		if p.Name == "Anna" && (p.TwoMin != 2 || p.Disq != 1) {
			t.Errorf("Anna = %d× 2 min, %d× Disq, erwartet 2 und 1", p.TwoMin, p.Disq)
		}
	}
}
