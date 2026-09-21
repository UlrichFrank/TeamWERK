package gamestats

import (
	"context"
	"database/sql"
	"testing"
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
