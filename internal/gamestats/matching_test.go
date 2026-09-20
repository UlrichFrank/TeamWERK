package gamestats

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// seedReport legt eine Begegnung mit Bericht an und liefert dessen ID.
func seedReport(t *testing.T, db *sql.DB, staffelID int, gameNo string) int {
	t.Helper()
	res, err := db.Exec(`
		INSERT INTO bwhv_games (staffel_id, game_no, sgid, date, time, home_team, guest_team)
		VALUES (?,?,?,?,'16:00','Verein A','Verein B')`,
		staffelID, gameNo, "sg"+gameNo, "2026-09-20")
	if err != nil {
		t.Fatal(err)
	}
	gid, _ := res.LastInsertId()
	res, err = db.Exec(
		`INSERT INTO bwhv_reports (bwhv_game_id, sgid, state) VALUES (?,?, 'parsed')`,
		gid, "sg"+gameNo)
	if err != nil {
		t.Fatal(err)
	}
	rid, _ := res.LastInsertId()
	return int(rid)
}

// resolveIn löst einen Spieler auf UND vermerkt seine Trikotnummer — genau in
// dieser Reihenfolge, wie SaveReport es tut. Ohne den zweiten Schritt wäre die
// Nummer für den nächsten Aufruf unbekannt statt widersprüchlich.
func resolveIn(t *testing.T, db *sql.DB, reportID, staffelID int, team, name string, year, number int) int {
	t.Helper()
	ctx := context.Background()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	id, err := resolvePlayer(ctx, tx, staffelID, team, name, year, number)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("resolvePlayer(%q, #%d): %v", name, number, err)
	}
	if _, err := tx.Exec(`
		INSERT INTO bwhv_player_games (report_id, player_id, side, jersey_number)
		VALUES (?,?,'home',?)
		ON CONFLICT (report_id, player_id) DO UPDATE SET jersey_number = excluded.jersey_number`,
		reportID, id, number); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return id
}

func playerRow(t *testing.T, db *sql.DB, id int) (name, conflict string) {
	t.Helper()
	if err := db.QueryRow(`SELECT name, conflict FROM bwhv_players WHERE id = ?`, id).
		Scan(&name, &conflict); err != nil {
		t.Fatal(err)
	}
	return name, conflict
}

// Eine Schreibvariante darf keine zweite Person erzeugen.
func TestMatching_SchreibvarianteIstDieselbePerson(t *testing.T) {
	db, _, _, staffelID := newStaffel(t)
	r1 := seedReport(t, db, staffelID, "900001")
	r2 := seedReport(t, db, staffelID, "900002")
	first := resolveIn(t, db, r1, staffelID, "Verein A", "Finn Haßlöcher", 2009, 29)
	second := resolveIn(t, db, r2, staffelID, "Verein A", "Finn Hasslocher", 2009, 29)
	if first != second {
		t.Errorf("zwei Datensätze (%d, %d) für dieselbe Person", first, second)
	}
	if n := countRows(t, db, "bwhv_players"); n != 1 {
		t.Errorf("bwhv_players = %d, erwartet 1", n)
	}
	if _, conflict := playerRow(t, db, first); !strings.Contains(conflict, "Schreibvariante") {
		t.Errorf("Zusammenführung nicht vermerkt: %q", conflict)
	}
}

// Ein geliehenes Trikot darf eine Person nicht zerreißen — genau deshalb ist
// die Nummer NICHT der Schlüssel.
func TestMatching_TrikotwechselZerreisstPersonNicht(t *testing.T) {
	db, _, _, staffelID := newStaffel(t)
	r1 := seedReport(t, db, staffelID, "900001")
	r2 := seedReport(t, db, staffelID, "900002")
	a := resolveIn(t, db, r1, staffelID, "Verein A", "Alexander Meier", 2009, 42)
	b := resolveIn(t, db, r2, staffelID, "Verein A", "Alexander Meier", 2009, 17)
	if a != b {
		t.Errorf("Trikotwechsel erzeugte zwei Personen (%d, %d)", a, b)
	}
}

// Zwei gleiche Namen in derselben Mannschaft bleiben zwei Personen; die
// Nummer ist das unterscheidende Merkmal.
func TestMatching_ZweiGleicheNamenWerdenPerNummerGetrennt(t *testing.T) {
	db, _, _, staffelID := newStaffel(t)
	r1 := seedReport(t, db, staffelID, "900001")
	a := resolveIn(t, db, r1, staffelID, "Verein A", "Lasse Ferber", 2009, 16)
	b := resolveIn(t, db, r1, staffelID, "Verein A", "Lasse Forber", 2010, 21)
	if a == b {
		t.Fatal("zwei ähnliche Namen wurden verschmolzen")
	}
	if n := countRows(t, db, "bwhv_players"); n != 2 {
		t.Errorf("bwhv_players = %d, erwartet 2", n)
	}
	if _, conflict := playerRow(t, db, b); !strings.Contains(conflict, "Trikotnummer abweichend") {
		t.Errorf("Widerspruch nicht vermerkt: %q", conflict)
	}
}

// Trikotnummern kollidieren zwischen Mannschaften — die Mannschaft ist Teil
// der Identität.
func TestMatching_GleicheNummerInAndererMannschaftIstAnderePerson(t *testing.T) {
	db, _, _, staffelID := newStaffel(t)
	r1 := seedReport(t, db, staffelID, "900001")
	a := resolveIn(t, db, r1, staffelID, "Verein A", "Noah Wolf", 2009, 46)
	b := resolveIn(t, db, r1, staffelID, "Verein B", "Enric Pena", 2009, 46)
	if a == b {
		t.Error("Nummer 46 zweier Mannschaften wurde zu einer Person")
	}
}

func TestMatching_KurzeNamenWerdenNichtUnscharfVerglichen(t *testing.T) {
	db, _, _, staffelID := newStaffel(t)
	r1 := seedReport(t, db, staffelID, "900001")
	a := resolveIn(t, db, r1, staffelID, "Verein A", "Ali", 2009, 3)
	b := resolveIn(t, db, r1, staffelID, "Verein A", "Ale", 2009, 4)
	if a == b {
		t.Error("zwei kurze Namen mit Distanz 1 wurden verschmolzen")
	}
}

func TestMatchMember_EindeutigerNameWirdZugeordnet(t *testing.T) {
	members := []kaderMember{{id: 7, name: "Samuel Birkle", year: 2009, number: 87}}
	m, ok := matchMember("Samuel Birkle", 2009, 87, members)
	if !ok || m.id != 7 {
		t.Fatalf("Zuordnung = %+v, ok = %v; erwartet Mitglied 7", m, ok)
	}
}

func TestMatchMember_UmgedrehteReihenfolgeTrifftTrotzdem(t *testing.T) {
	members := []kaderMember{{id: 7, name: "Samuel Birkle", year: 2009, number: 87}}
	if _, ok := matchMember("Birkle Samuel", 0, 0, members); !ok {
		t.Error("\"Nachname Vorname\" wurde nicht zugeordnet")
	}
}

func TestMatchMember_MehrdeutigBleibtOffen(t *testing.T) {
	members := []kaderMember{
		{id: 1, name: "Lukas Meier", year: 2009, number: 5},
		{id: 2, name: "Lukas Maier", year: 2009, number: 6},
	}
	if m, ok := matchMember("Lukas Meier", 0, 0, members); ok {
		t.Errorf("mehrdeutiger Name wurde zugeordnet: %+v", m)
	}
}

func TestMatchMember_JahrgangBrichtGleichstand(t *testing.T) {
	members := []kaderMember{
		{id: 1, name: "Lukas Meier", year: 2009, number: 5},
		{id: 2, name: "Lukas Maier", year: 2011, number: 6},
	}
	m, ok := matchMember("Lukas Meier", 2011, 0, members)
	if !ok || m.id != 2 {
		t.Errorf("Zuordnung = %+v, ok = %v; der Jahrgang sollte auf Mitglied 2 zeigen", m, ok)
	}
}

func TestMatchMember_OhneTrefferKeineZuordnung(t *testing.T) {
	members := []kaderMember{{id: 1, name: "Samuel Birkle", year: 2009, number: 87}}
	if _, ok := matchMember("Ganz Anders", 2009, 87, members); ok {
		t.Error("fremder Name wurde zugeordnet")
	}
}
