package gamestats

import (
	"context"

	"os"
	"path/filepath"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/bwhv"
)

func fixturePDF(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "bwhv", "testdata", name))
	if err != nil {
		t.Fatalf("Fixture %s: %v", name, err)
	}
	return b
}

// seedPending legt eine Begegnung mit sGID an und liefert den PendingReport.
func seedPending(t *testing.T, s *Store, staffelID int, gameNo, sgid string) PendingReport {
	t.Helper()
	res, err := s.db.Exec(`
		INSERT INTO bwhv_games (staffel_id, game_no, sgid, date, time, home_team, guest_team)
		VALUES (?,?,?, '2026-09-20','16:00','Rhein-Neckar Löwen 2','Team Stuttgart')`,
		staffelID, gameNo, sgid)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return PendingReport{BwhvGameID: int(id), StaffelID: staffelID, GameNo: gameNo, SGID: sgid}
}

func TestSaveReport_SchreibtSpielerUndVerlauf(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	ctx := context.Background()
	p := seedPending(t, s, staffelID, "905272", "3504061")

	rep, err := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if err := s.SaveReport(ctx, p, rep, "26/905272.pdf"); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	if n := countRows(t, db, "bwhv_players"); n != 26 {
		t.Errorf("bwhv_players = %d, erwartet 26", n)
	}
	if n := countRows(t, db, "bwhv_player_games"); n != 26 {
		t.Errorf("bwhv_player_games = %d, erwartet 26", n)
	}
	if n := countRows(t, db, "bwhv_events"); n != 66 {
		t.Errorf("bwhv_events = %d, erwartet 66", n)
	}
	var state, warnings string
	if err := db.QueryRow(`SELECT state, warnings_json FROM bwhv_reports WHERE bwhv_game_id = ?`,
		p.BwhvGameID).Scan(&state, &warnings); err != nil {
		t.Fatal(err)
	}
	if state != "parsed" {
		t.Errorf("state = %q, erwartet parsed", state)
	}
}

// Tor-Ereignisse müssen mit ihrem Schützen verknüpft sein, sonst trägt die
// Saisonbilanz später nichts.
func TestSaveReport_VerknuepftEreignisseMitSpielern(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	ctx := context.Background()
	p := seedPending(t, s, staffelID, "905272", "3504061")
	rep, _ := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err := s.SaveReport(ctx, p, rep, "26/905272.pdf"); err != nil {
		t.Fatal(err)
	}

	var ohne int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM bwhv_events
		 WHERE kind IN ('goal','seven_m_goal') AND player_id IS NULL`).Scan(&ohne); err != nil {
		t.Fatal(err)
	}
	if ohne != 0 {
		t.Errorf("%d Tor-Ereignisse ohne Spieler-Verknüpfung", ohne)
	}
}

// Der harte Fehlschlag darf KEINE Teilergebnisse hinterlassen.
func TestMarkFailed_HinterlaesstKeineTeilergebnisse(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	ctx := context.Background()
	p := seedPending(t, s, staffelID, "905272", "3504061")

	// Erst einen gültigen Bericht schreiben ...
	rep, _ := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err := s.SaveReport(ctx, p, rep, "26/905272.pdf"); err != nil {
		t.Fatal(err)
	}
	if countRows(t, db, "bwhv_events") == 0 {
		t.Fatal("Vorbedingung: Ereignisse sollten vorhanden sein")
	}

	// ... dann scheitert ein erneuter Lauf.
	if err := s.MarkFailed(ctx, p.BwhvGameID, p.SGID, "26/905272.pdf", "Endstand weicht ab"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}
	if n := countRows(t, db, "bwhv_player_games"); n != 0 {
		t.Errorf("bwhv_player_games = %d, erwartet 0 nach Fehlschlag", n)
	}
	if n := countRows(t, db, "bwhv_events"); n != 0 {
		t.Errorf("bwhv_events = %d, erwartet 0 nach Fehlschlag", n)
	}
	var state, reason, pdf string
	if err := db.QueryRow(`SELECT state, failure_reason, pdf_path FROM bwhv_reports WHERE bwhv_game_id = ?`,
		p.BwhvGameID).Scan(&state, &reason, &pdf); err != nil {
		t.Fatal(err)
	}
	if state != "parse_failed" || reason == "" {
		t.Errorf("state = %q, Grund = %q", state, reason)
	}
	if pdf == "" {
		t.Error("das PDF muss liegen bleiben, damit ein Parser-Fix es ohne Fremdabruf nachverarbeiten kann")
	}
}

// Ein fehlgeschlagener Bericht bleibt offen, damit ein Parser-Fix ihn
// nachverarbeiten kann; ein ausgewerteter nicht.
func TestPendingReports_FehlgeschlageneBleibenOffen(t *testing.T) {
	_, s, _, staffelID := newStaffel(t)
	ctx := context.Background()
	p := seedPending(t, s, staffelID, "905272", "3504061")

	if err := s.MarkFailed(ctx, p.BwhvGameID, p.SGID, "x.pdf", "Grund"); err != nil {
		t.Fatal(err)
	}
	pend, err := s.PendingReports(ctx, staffelID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 {
		t.Fatalf("offene Berichte = %d, erwartet 1", len(pend))
	}

	rep, _ := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err := s.SaveReport(ctx, p, rep, "x.pdf"); err != nil {
		t.Fatal(err)
	}
	pend, err = s.PendingReports(ctx, staffelID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 0 {
		t.Errorf("offene Berichte = %d, erwartet 0 nach erfolgreicher Auswertung", len(pend))
	}
}

// Begegnungen ohne sGID dürfen nie zum Abruf anstehen.
func TestPendingReports_OhneSGIDNichtEnthalten(t *testing.T) {
	_, s, _, staffelID := newStaffel(t)
	seedGame(t, s, staffelID, "900001", "2026-09-20", "16:00", "")

	pend, err := s.PendingReports(context.Background(), staffelID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 0 {
		t.Errorf("offene Berichte = %d, erwartet 0 — ohne sGID gibt es keinen Bericht", len(pend))
	}
}

// Ein Transportfehler zählt den Versuch hoch und lässt den Bericht offen.
func TestMarkAttempt_BleibtWiederholbar(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	ctx := context.Background()
	p := seedPending(t, s, staffelID, "905272", "3504061")

	for i := 0; i < 3; i++ {
		if err := s.MarkAttempt(ctx, p.BwhvGameID, p.SGID); err != nil {
			t.Fatal(err)
		}
	}
	var state string
	var attempts int
	if err := db.QueryRow(`SELECT state, attempts FROM bwhv_reports WHERE bwhv_game_id = ?`,
		p.BwhvGameID).Scan(&state, &attempts); err != nil {
		t.Fatal(err)
	}
	if state != "pending" || attempts != 3 {
		t.Errorf("state = %q, Versuche = %d; erwartet pending/3", state, attempts)
	}
	pend, _ := s.PendingReports(ctx, staffelID)
	if len(pend) != 1 || pend[0].Attempts != 3 {
		t.Errorf("Bericht nicht mehr offen oder Versuchszähler falsch: %+v", pend)
	}
}

func TestReportStore_LegtAbUndOeffnet(t *testing.T) {
	dir := t.TempDir()
	rs := NewReportStore(dir)
	rel, err := rs.Save(7, "905272", []byte("%PDF-1.4 test"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := rs.Open(rel)
	if err != nil {
		t.Fatalf("Open(%q): %v", rel, err)
	}
	defer f.Close()
	if _, err := rs.Open("nicht-da.pdf"); err == nil {
		t.Error("erwartet: Fehler bei fehlender Datei")
	}
}

// Die getrennten Namen und der Vermerk der unsicheren Trennung müssen den Weg
// durch SaveReport und ReportForGame überstehen — die Rohzeile bleibt daneben
// stehen, sie ist der Beleg.
func TestSaveReport_SchiedsrichterWerdenGetrenntGespeichert(t *testing.T) {
	db, s, _, staffelID := newStaffel(t)
	ctx := context.Background()
	p := seedPending(t, s, staffelID, "905272", "3504061")

	rep, err := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	rep.Header.Referees = "Max Mustermann Peter Müller"
	rep.Header.RefereeNames = []string{"Max Mustermann", "Peter Müller"}
	rep.Header.RefereesUncertain = true
	if err := s.SaveReport(ctx, p, rep, "26/905272.pdf"); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	var raw, refJSON string
	var uncertain int
	if err := db.QueryRow(`SELECT referees, referees_json, referees_uncertain
		FROM bwhv_reports WHERE bwhv_game_id = ?`, p.BwhvGameID).Scan(&raw, &refJSON, &uncertain); err != nil {
		t.Fatal(err)
	}
	if raw != "Max Mustermann Peter Müller" {
		t.Errorf("referees = %q, erwartet die unveränderte Rohzeile", raw)
	}
	if uncertain != 1 {
		t.Errorf("referees_uncertain = %d, erwartet 1", uncertain)
	}

	detail, err := s.ReportForGame(ctx, p.BwhvGameID)
	if err != nil {
		t.Fatalf("ReportForGame: %v", err)
	}
	if len(detail.RefereeNames) != 2 || detail.RefereeNames[1] != "Peter Müller" {
		t.Errorf("RefereeNames = %#v, erwartet zwei Namen", detail.RefereeNames)
	}
	if !detail.RefereesUncertain {
		t.Error("RefereesUncertain = false, erwartet true")
	}
	if detail.Referees != raw {
		t.Errorf("Referees = %q, erwartet die Rohzeile %q", detail.Referees, raw)
	}
}
