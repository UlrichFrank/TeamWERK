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
		r.Get("/api/staffeln/{id}/kreuztabelle", h.GetCrossTable)
		r.Get("/api/staffeln/{id}/tabellenverlauf", h.GetProgression)
		r.Get("/api/staffeln/{id}/teamstatistik", h.GetTeamStats)
		r.Get("/api/staffeln/{id}/schiedsrichter", h.GetRefereeStats)
		r.Get("/api/staffeln/{id}/affiliation", h.GetAffiliation)
		r.Get("/api/staffeln/{id}/player-games", h.GetPlayerGames)
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
	code, body := get(t, srv, "/api/staffeln", testutil.Token(t, 1, "standard", []string{"vorstand"}))
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

// Der Kurzname folgt der Kader-Zuordnung (Geschlecht + Altersklasse + Nummer),
// nicht dem Teamnamen, und fällt ohne Kader-Angaben auf den Namen zurück.
func TestListStaffeln_LiefertKurznamen(t *testing.T) {
	srv, s, seasonID, _ := newHandlerServer(t)
	teamID := testutil.CreateTeam(t, s.db, "B-Jugend männlich")
	kaderID := testutil.CreateKader(t, s.db, teamID, seasonID)
	if _, err := s.db.Exec(
		`UPDATE kader SET staffel = 'mB-RL-BW', age_class = 'B-Jugend', gender = 'm', team_number = 1 WHERE id = ?`,
		kaderID); err != nil {
		t.Fatal(err)
	}
	code, body := get(t, srv, "/api/staffeln", testutil.Token(t, 1, "standard", []string{"vorstand"}))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var list []staffelResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].TeamShort != "mB" {
		t.Errorf("TeamShort = %+v, erwartet mB (einziges m-B-Team)", list)
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

// newStatsServer legt zusätzlich eine kleine Staffel mit Ergebnissen an,
// damit die Statistik-Routen etwas zu antworten haben.
func newStatsServer(t *testing.T) (*httptest.Server, *Store, int, int) {
	t.Helper()
	srv, s, seasonID, staffelID := newHandlerServer(t)
	seedResult(t, s.db, staffelID, "1", "2026-09-20", "A", "B", intp(30), intp(20))
	seedResult(t, s.db, staffelID, "2", "2026-09-27", "B", "A", intp(25), intp(28))
	seedResult(t, s.db, staffelID, "3", "2026-10-04", "A", "B", nil, nil)
	return srv, s, seasonID, staffelID
}

// Die vier Statistik-Routen und die Zugehörigkeit teilen denselben
// Auth-Tier und dieselbe Staffel-Auflösung: ohne Token 401, bei unbekannter
// Staffel-ID 404. Eine Tabelle statt fünf Kopien.
func TestStatistikRouten_TierUndUnbekannteStaffel(t *testing.T) {
	srv, _, _, staffelID := newStatsServer(t)
	for _, suffix := range []string{"kreuztabelle", "tabellenverlauf", "teamstatistik", "schiedsrichter", "affiliation"} {
		if code, _ := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/"+suffix, ""); code != http.StatusUnauthorized {
			t.Errorf("%s ohne Token: Status = %d, erwartet 401", suffix, code)
		}
		if code, _ := get(t, srv, "/api/staffeln/9999/"+suffix, userToken(t)); code != http.StatusNotFound {
			t.Errorf("%s mit unbekannter Staffel: Status = %d, erwartet 404", suffix, code)
		}
	}
}

func TestGetCrossTable_LiefertErgebnisUndDatum(t *testing.T) {
	srv, _, _, staffelID := newStatsServer(t)
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/kreuztabelle", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var ct CrossTable
	if err := json.Unmarshal(body, &ct); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, body)
	}
	if len(ct.Teams) != 2 || len(ct.Rows) != 2 {
		t.Fatalf("Kreuztabelle = %+v, erwartet 2×2", ct)
	}
	for i, row := range ct.Rows {
		if row.Cells[i] != nil {
			t.Errorf("Diagonale von %q gefüllt", row.Team)
		}
	}
}

func TestGetProgression_RangJeSpieltag(t *testing.T) {
	srv, _, _, staffelID := newStatsServer(t)
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/tabellenverlauf", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var days []ProgressionDay
	if err := json.Unmarshal(body, &days); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, body)
	}
	if len(days) != 2 {
		t.Fatalf("Spieltage = %d, erwartet 2 — die dritte Begegnung ist ungespielt", len(days))
	}
	if len(days[1].Entries) != 2 || days[1].Entries[0].Rank != 1 {
		t.Errorf("Spieltag 2 = %+v, erwartet zwei Ränge beginnend bei 1", days[1])
	}
}

// Eine Staffel ohne gespielte Begegnung liefert einen leeren Verlauf, keinen
// Fehler — und ein leeres Array, kein null.
func TestGetProgression_OhneErgebnisLeer(t *testing.T) {
	srv, s, _, staffelID := newHandlerServer(t)
	seedResult(t, s.db, staffelID, "1", "2026-10-04", "A", "B", nil, nil)
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/tabellenverlauf", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	if string(body) != "[]\n" && string(body) != "[]" {
		t.Errorf("Antwort = %s, erwartet ein leeres Array", body)
	}
}

func TestGetTeamStats_ToreOhneBericht(t *testing.T) {
	srv, _, _, staffelID := newStatsServer(t)
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/teamstatistik", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var ts TeamStats
	if err := json.Unmarshal(body, &ts); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, body)
	}
	if len(ts.Teams) != 2 {
		t.Fatalf("Mannschaften = %d, erwartet 2", len(ts.Teams))
	}
	for _, st := range ts.Teams {
		if st.Games != 2 || st.GoalsFor == 0 {
			t.Errorf("%s = %+v, erwartet zwei Spiele mit Toren ohne jeden Bericht", st.Team, st)
		}
		if st.FairPlay != nil {
			t.Errorf("%s: Fair-Play = %v, erwartet leer ohne Bericht", st.Team, *st.FairPlay)
		}
	}
	if ts.FairPlayWeights.TwoMin == 0 {
		t.Error("Gewichtung fehlt in der Antwort — sie muss ausgewiesen werden")
	}
}

func TestGetRefereeStats_LeerOhneBerichte(t *testing.T) {
	srv, _, _, staffelID := newStatsServer(t)
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/schiedsrichter", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	if string(body) != "[]\n" && string(body) != "[]" {
		t.Errorf("Antwort = %s, erwartet ein leeres Array", body)
	}
}

// Ohne Zugehörigkeit stehen beide Mengen leer in der Antwort — als Arrays,
// nicht als null, damit das Frontend nicht dagegen prüfen muss.
func TestGetAffiliation_OhneZugehoerigkeitLeer(t *testing.T) {
	srv, _, _, staffelID := newStatsServer(t)
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/affiliation", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var aff Affiliation
	if err := json.Unmarshal(body, &aff); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, body)
	}
	if aff.TeamNames == nil || aff.PlayerIDs == nil {
		t.Errorf("Affiliation = %+v, erwartet leere Arrays statt null", aff)
	}
	if len(aff.TeamNames) != 0 || len(aff.PlayerIDs) != 0 {
		t.Errorf("Affiliation = %+v, erwartet beide Mengen leer", aff)
	}
}

// Der Nutzer des Tokens bekommt seine Mannschaft über die verknüpfte
// Begegnung — der Weg vom Token bis zur Verbandsschreibweise in einem Stück.
func TestGetAffiliation_SpielerSiehtMannschaft(t *testing.T) {
	srv, s, seasonID, staffelID := newHandlerServer(t)
	userID := testutil.CreateUser(t, s.db, "standard")
	memberID := testutil.CreateMember(t, s.db, userID)
	teamID := testutil.CreateTeam(t, s.db, "B-Jugend")
	addToKader(t, s.db, testutil.CreateKader(t, s.db, teamID, seasonID), memberID)
	bwhvGameID := seedResult(t, s.db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(10), intp(8))
	linkOwnGame(t, s.db, bwhvGameID, seasonID, teamID, true)

	token := testutil.Token(t, userID, "standard", []string{"spieler"})
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/affiliation", token)
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var aff Affiliation
	if err := json.Unmarshal(body, &aff); err != nil {
		t.Fatal(err)
	}
	if len(aff.TeamNames) != 1 || aff.TeamNames[0] != "Team Stuttgart 2" {
		t.Errorf("TeamNames = %#v, erwartet [Team Stuttgart 2]", aff.TeamNames)
	}
}

// Die Hallennummer der Begegnung wird über venues.hall_number zur Halle
// aufgelöst; ohne passende Halle bleibt Venue leer und die Nummer stehen.
func TestStaffelSpielplan_LoestHalleAuf(t *testing.T) {
	srv, s, _, staffelID := newHandlerServer(t)
	sch := sampleSchedule(2)
	sch.Games[0].HallNumber = "21005"
	sch.Games[1].HallNumber = "99999"
	if _, err := s.SaveSchedule(context.Background(), staffelID, sch); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO venues (name, street, city, postal_code, hall_number)
		VALUES ('Sporthalle Nord', 'Hallenweg 1', 'Stuttgart', '70000', '21005')`); err != nil {
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
	if len(games) != 2 {
		t.Fatalf("Spiele = %d, erwartet 2", len(games))
	}
	var bekannt, unbekannt *ScheduleGame
	for i := range games {
		if games[i].HallNumber == "21005" {
			bekannt = &games[i]
		} else {
			unbekannt = &games[i]
		}
	}
	if bekannt == nil || bekannt.Venue == nil || bekannt.Venue.Name != "Sporthalle Nord" || bekannt.Venue.City != "Stuttgart" {
		t.Errorf("bekannte Halle = %+v, erwartet aufgelöste Sporthalle Nord", bekannt)
	}
	if unbekannt == nil || unbekannt.Venue != nil {
		t.Errorf("unbekannte Halle = %+v, erwartet Venue == nil", unbekannt)
	}
}

// Sichtbarkeit wie der Teamfilter der Dienstbörse: Spieler und Eltern sehen nur
// die Staffeln ihres Stammkader-Teams, der erweiterte Kader zählt nicht,
// Vorstand und sportliche Leitung sehen alle.
func TestListStaffeln_Sichtbarkeit(t *testing.T) {
	srv, s, seasonID, _ := newHandlerServer(t)
	teamA := testutil.CreateTeam(t, s.db, "Team A")
	teamB := testutil.CreateTeam(t, s.db, "Team B")
	teamC := testutil.CreateTeam(t, s.db, "Team C")
	kA := testutil.CreateKader(t, s.db, teamA, seasonID)
	kB := testutil.CreateKader(t, s.db, teamB, seasonID)
	kC := testutil.CreateKader(t, s.db, teamC, seasonID)
	for k, code := range map[int]string{kA: "A-Staffel", kB: "B-Staffel", kC: "C-Staffel"} {
		if _, err := s.db.Exec(`UPDATE kader SET staffel = ? WHERE id = ?`, code, k); err != nil {
			t.Fatal(err)
		}
	}
	playerUser := testutil.CreateUser(t, s.db, "standard")
	playerMember := testutil.CreateMember(t, s.db, playerUser)
	testutil.AddKaderMember(t, s.db, kA, playerMember)
	testutil.AddExtendedKaderMember(t, s.db, kC, playerMember)

	parentUser := testutil.CreateUser(t, s.db, "standard")
	child := testutil.CreateMember(t, s.db, 0)
	testutil.AddKaderMember(t, s.db, kB, child)
	testutil.AddFamilyLink(t, s.db, parentUser, child)

	codes := func(token string) []string {
		code, body := get(t, srv, "/api/staffeln", token)
		if code != http.StatusOK {
			t.Fatalf("Status = %d: %s", code, body)
		}
		var list []staffelResponse
		if err := json.Unmarshal(body, &list); err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, r := range list {
			out = append(out, r.Code)
		}
		return out
	}
	eq := func(name string, got []string, want ...string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: %v, erwartet %v", name, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: %v, erwartet %v", name, got, want)
			}
		}
	}
	eq("Spieler", codes(testutil.Token(t, playerUser, "standard", []string{"spieler"})), "A-Staffel")
	eq("Elternteil", codes(testutil.Token(t, parentUser, "standard", nil)), "B-Staffel")
	eq("Vorstand", codes(testutil.Token(t, 999, "standard", []string{"vorstand"})), "A-Staffel", "B-Staffel", "C-Staffel")
	eq("sportliche Leitung", codes(testutil.Token(t, 999, "standard", []string{"sportliche_leitung"})), "A-Staffel", "B-Staffel", "C-Staffel")
	eq("Admin", codes(testutil.Token(t, 999, "admin", nil)), "A-Staffel", "B-Staffel", "C-Staffel")
}
