package gamestats

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/bwhv"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// newHandlerServer baut einen Server mit den Lese-Routen dieses Pakets.
func newHandlerServer(t *testing.T) (*httptest.Server, *Store, int, int) {
	t.Helper()
	db, s, seasonID, staffelID := newStaffel(t)
	if _, err := db.Exec(`UPDATE seasons SET is_active = 1 WHERE id = ?`, seasonID); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(db, hub.NewHub(), bwhv.NewClient(), NewReportStore(t.TempDir()), 216)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/staffeln", h.ListStaffeln)
		r.Get("/api/staffeln/{id}/tabelle", h.GetTable)
		r.Get("/api/staffeln/{id}/spielplan", h.GetSchedule)
		r.Get("/api/staffeln/{id}/ranglisten", h.GetRanglisten)
		r.Get("/api/bwhv-games/{id}/report", h.GetReport)
		r.Get("/api/bwhv-reports/{id}/pdf", h.GetReportPDF)
		r.Get("/api/members/{id}/saisonstatistik", h.GetMemberStats)
	})
	return srv, s, seasonID, staffelID
}

// get nutzt testutil.Do — dessen token-Parameter enthält das Bearer-Präfix
// bereits, testutil.Token liefert es mit.
func get(t *testing.T, srv *httptest.Server, path, token string) (int, []byte) {
	t.Helper()
	resp := testutil.Do(t, srv, http.MethodGet, path, token, nil)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func userToken(t *testing.T) string {
	t.Helper()
	return testutil.Token(t, 1, "standard", []string{"spieler"})
}

// Die Liste folgt der Zuordnung am Kader, nicht dem Snapshot — deshalb braucht
// dieser Test einen Kader mit gesetzter Staffel.
func TestListStaffeln_HappyPath(t *testing.T) {
	srv, s, seasonID, staffelID := newHandlerServer(t)
	teamID := testutil.CreateTeam(t, s.db, "B-Jugend männlich")
	kaderID := testutil.CreateKader(t, s.db, teamID, seasonID)
	if _, err := s.db.Exec(`UPDATE kader SET staffel = 'mB-RL-BW' WHERE id = ?`, kaderID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveSchedule(context.Background(), staffelID, sampleSchedule(3)); err != nil {
		t.Fatal(err)
	}
	code, body := get(t, srv, "/api/staffeln", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d, erwartet 200: %s", code, body)
	}
	var list []staffelResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, body)
	}
	if len(list) != 1 || list[0].Code != "mB-RL-BW" {
		t.Errorf("Staffeln = %+v, erwartet eine Zeile mB-RL-BW", list)
	}
}

func TestListStaffeln_Unauthenticated(t *testing.T) {
	srv, _, _, _ := newHandlerServer(t)
	if code, _ := get(t, srv, "/api/staffeln", ""); code != http.StatusUnauthorized {
		t.Errorf("Status = %d, erwartet 401", code)
	}
}

func TestStaffelTabelle_HappyPath(t *testing.T) {
	srv, s, _, staffelID := newHandlerServer(t)
	if _, err := s.SaveSchedule(context.Background(), staffelID, sampleSchedule(2)); err != nil {
		t.Fatal(err)
	}
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/tabelle", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var table []bwhv.TableRow
	if err := json.Unmarshal(body, &table); err != nil {
		t.Fatal(err)
	}
	if len(table) != 1 {
		t.Errorf("Tabellenzeilen = %d, erwartet 1", len(table))
	}
}

func TestStaffelTabelle_UnbekannteStaffel(t *testing.T) {
	srv, _, _, _ := newHandlerServer(t)
	if code, _ := get(t, srv, "/api/staffeln/9999/tabelle", userToken(t)); code != http.StatusNotFound {
		t.Errorf("Status = %d, erwartet 404", code)
	}
}

func TestStaffelSpielplan_EnthaeltFremdeBegegnungen(t *testing.T) {
	srv, s, _, staffelID := newHandlerServer(t)
	if _, err := s.SaveSchedule(context.Background(), staffelID, sampleSchedule(5)); err != nil {
		t.Fatal(err)
	}
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/spielplan", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var games []ScheduleGame
	if err := json.Unmarshal(body, &games); err != nil {
		t.Fatal(err)
	}
	if len(games) != 5 {
		t.Fatalf("Begegnungen = %d, erwartet 5", len(games))
	}
	for _, g := range games {
		if g.GameID != nil {
			t.Errorf("Begegnung %s ist verknüpft, obwohl kein eigenes Spiel existiert", g.GameNo)
		}
	}
}

func TestStaffelRanglisten_HappyPath(t *testing.T) {
	srv, s, _, staffelID := newHandlerServer(t)
	ctx := context.Background()
	p := seedPending(t, s, staffelID, "905272", "3504061")
	rep, err := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveReport(ctx, p, rep, "x.pdf"); err != nil {
		t.Fatal(err)
	}

	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/ranglisten", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var stats []PlayerStat
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatal(err)
	}
	if len(stats) != 26 {
		t.Fatalf("Spieler = %d, erwartet 26", len(stats))
	}
	for i := 1; i < len(stats); i++ {
		if stats[i].Goals > stats[i-1].Goals {
			t.Errorf("Rangliste nicht absteigend sortiert an Position %d", i)
			break
		}
	}
}

func TestReport_HappyPath(t *testing.T) {
	srv, s, _, staffelID := newHandlerServer(t)
	ctx := context.Background()
	p := seedPending(t, s, staffelID, "905272", "3504061")
	rep, _ := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err := s.SaveReport(ctx, p, rep, "x.pdf"); err != nil {
		t.Fatal(err)
	}

	code, body := get(t, srv, "/api/bwhv-games/"+itoa(p.BwhvGameID)+"/report", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var d ReportDetail
	if err := json.Unmarshal(body, &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Players) != 26 || len(d.Events) != 66 {
		t.Errorf("Spieler = %d, Ereignisse = %d; erwartet 26 und 66", len(d.Players), len(d.Events))
	}
}

func TestReport_NochKeinBericht(t *testing.T) {
	srv, s, _, staffelID := newHandlerServer(t)
	seedGame(t, s, staffelID, "900001", "2026-09-20", "16:00", "")
	var id int
	if err := s.db.QueryRow(`SELECT id FROM bwhv_games WHERE game_no = '900001'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if code, _ := get(t, srv, "/api/bwhv-games/"+itoa(id)+"/report", userToken(t)); code != http.StatusNotFound {
		t.Errorf("Status = %d, erwartet 404 solange keine sGID vorliegt", code)
	}
}

func TestReportPDF_HappyPath(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	if _, err := db.Exec(`UPDATE seasons SET is_active = 1 WHERE id = ?`, seasonID); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	rs := NewReportStore(dir)
	rel, err := rs.Save(seasonID, "905272", []byte("%PDF-1.4 fixture"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	p := seedPending(t, s, staffelID, "905272", "3504061")
	rep, _ := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err := s.SaveReport(ctx, p, rep, rel); err != nil {
		t.Fatal(err)
	}
	var reportID int
	if err := db.QueryRow(`SELECT id FROM bwhv_reports WHERE bwhv_game_id = ?`, p.BwhvGameID).Scan(&reportID); err != nil {
		t.Fatal(err)
	}

	h := NewHandler(db, hub.NewHub(), bwhv.NewClient(), rs, 216)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/bwhv-reports/{id}/pdf", h.GetReportPDF)
	})
	resp := testutil.Do(t, srv, http.MethodGet,
		"/api/bwhv-reports/"+itoa(reportID)+"/pdf", userToken(t), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Status = %d, erwartet 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q, erwartet application/pdf (explizit gesetzt, nicht geraten)", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); cd == "" || cd[:10] != "attachment" {
		t.Errorf("Content-Disposition = %q, erwartet attachment", cd)
	}
}

func TestReportPDF_OhneDatei(t *testing.T) {
	srv, _, _, _ := newHandlerServer(t)
	if code, _ := get(t, srv, "/api/bwhv-reports/9999/pdf", userToken(t)); code != http.StatusNotFound {
		t.Errorf("Status = %d, erwartet 404", code)
	}
}

func TestSaisonstatistik_HappyPath(t *testing.T) {
	srv, s, seasonID, staffelID := newHandlerServer(t)
	ctx := context.Background()
	memberID := testutil.CreateMember(t, s.db, 0)
	p := seedPending(t, s, staffelID, "905272", "3504061")
	rep, _ := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err := s.SaveReport(ctx, p, rep, "x.pdf"); err != nil {
		t.Fatal(err)
	}
	// Einen beliebigen Spieler mit dem Mitglied verknüpfen.
	if _, err := s.db.Exec(
		`UPDATE bwhv_players SET member_id = ? WHERE id = (SELECT MIN(id) FROM bwhv_players)`, memberID); err != nil {
		t.Fatal(err)
	}
	_ = seasonID

	code, body := get(t, srv, "/api/members/"+itoa(memberID)+"/saisonstatistik", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var stats []PlayerStat
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatal(err)
	}
	if len(stats) != 1 {
		t.Fatalf("Zeilen = %d, erwartet 1", len(stats))
	}
	if stats[0].Games != 1 {
		t.Errorf("Spiele = %d, erwartet 1", stats[0].Games)
	}
}

func TestSaisonstatistik_UnbekanntesMitglied(t *testing.T) {
	srv, _, _, _ := newHandlerServer(t)
	if code, _ := get(t, srv, "/api/members/9999/saisonstatistik", userToken(t)); code != http.StatusNotFound {
		t.Errorf("Status = %d, erwartet 404", code)
	}
}

// Ein Mitglied ohne ausgewertete Spiele liefert eine leere Bilanz, keinen Fehler.
func TestSaisonstatistik_OhneSpiele(t *testing.T) {
	srv, s, _, _ := newHandlerServer(t)
	memberID := testutil.CreateMember(t, s.db, 0)
	code, body := get(t, srv, "/api/members/"+itoa(memberID)+"/saisonstatistik", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var stats []PlayerStat
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatal(err)
	}
	if len(stats) != 0 {
		t.Errorf("Zeilen = %d, erwartet 0", len(stats))
	}
}

// Ein gescheiterter Bericht darf nicht in die Bilanz einfließen.
func TestSaisonstatistik_FehlgeschlagenerBerichtZaehltNicht(t *testing.T) {
	srv, s, _, staffelID := newHandlerServer(t)
	ctx := context.Background()
	memberID := testutil.CreateMember(t, s.db, 0)
	p := seedPending(t, s, staffelID, "905272", "3504061")
	rep, _ := bwhv.ParseReport(fixturePDF(t, "spielbericht_905272.pdf"))
	if err := s.SaveReport(ctx, p, rep, "x.pdf"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(
		`UPDATE bwhv_players SET member_id = ? WHERE id = (SELECT MIN(id) FROM bwhv_players)`, memberID); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFailed(ctx, p.BwhvGameID, p.SGID, "x.pdf", "Endstand weicht ab"); err != nil {
		t.Fatal(err)
	}

	_, body := get(t, srv, "/api/members/"+itoa(memberID)+"/saisonstatistik", userToken(t))
	var stats []PlayerStat
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatal(err)
	}
	if len(stats) != 0 {
		t.Errorf("Zeilen = %d, erwartet 0 — parse_failed zählt nicht", len(stats))
	}
}
