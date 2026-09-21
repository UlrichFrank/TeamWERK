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
