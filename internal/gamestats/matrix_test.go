package gamestats

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ownTeamSetup legt einen Spieler mit Kaderzugehörigkeit an und verknüpft eine
// BWHV-Begegnung mit einem eigenen Spieltermin — die einzige Quelle, aus der
// die Zugehörigkeit abgeleitet wird (design.md §2).
func ownTeamSetup(t *testing.T, db *sql.DB, seasonID, staffelID, bwhvGameID int, isHome bool) (userID, teamID int) {
	t.Helper()
	userID = testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	teamID = testutil.CreateTeam(t, db, "B-Jugend")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	addToKader(t, db, kaderID, memberID)
	linkOwnGame(t, db, bwhvGameID, seasonID, teamID, isHome)
	return userID, teamID
}

func setAgeClassRule(t *testing.T, db *sql.DB, teamID int, ageClass string, half int) {
	t.Helper()
	if _, err := db.Exec(`UPDATE teams SET age_class = ? WHERE id = ?`, ageClass, teamID); err != nil {
		t.Fatalf("teams.age_class: %v", err)
	}
	if half <= 0 {
		return
	}
	if _, err := db.Exec(`INSERT INTO age_class_game_rules (age_class, half_duration_minutes, break_minutes)
		VALUES (?,?,10) ON CONFLICT (age_class) DO UPDATE SET half_duration_minutes = excluded.half_duration_minutes`,
		ageClass, half); err != nil {
		t.Fatalf("age_class_game_rules: %v", err)
	}
}

func matrixOf(t *testing.T, list []TeamMatrix, team string) TeamMatrix {
	t.Helper()
	for _, m := range list {
		if m.Team == team {
			return m
		}
	}
	t.Fatalf("Mannschaft %q nicht in der Matrix (%d Einträge)", team, len(list))
	return TeamMatrix{}
}

func matrixPlayerRow(t *testing.T, m TeamMatrix, name string) MatrixPlayer {
	t.Helper()
	for _, p := range m.Players {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("Spieler %q nicht in der Matrix", name)
	return MatrixPlayer{}
}

// Der Normalfall: je Spieler eine Zeile, je gespielter Begegnung eine Spalte,
// und die Zelle trägt die Werte genau dieses Berichts.
func TestPlayerGameMatrix_MatrixDerEigenenMannschaft(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	g1 := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	g2 := seedResult(t, db, staffelID, "2", "2026-09-27", "Fremd", "Team Stuttgart 2", intp(20), intp(24))
	userID, _ := ownTeamSetup(t, db, seasonID, staffelID, g1, true)

	seedParsedReport(t, db, staffelID, g1, "parsed",
		map[string]string{"home": "TS 2", "guest": "Fremd"},
		[]rosterLine{{name: "Anna", side: "home", goals: 5, sevenMAtt: 2, sevenMGoals: 1}})
	seedParsedReport(t, db, staffelID, g2, "parsed",
		map[string]string{"home": "Fremd", "guest": "TS 2"},
		[]rosterLine{{name: "Anna", side: "guest", goals: 3, yellow: 1}})

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	m := matrixOf(t, list, "Team Stuttgart 2")
	if len(m.Games) != 2 {
		t.Fatalf("Spalten = %d, erwartet 2", len(m.Games))
	}
	if m.Games[0].Date != "2026-09-20" || !m.Games[0].IsHome || m.Games[1].IsHome {
		t.Errorf("Spalten falsch: %#v", m.Games)
	}
	p := matrixPlayerRow(t, m, "Anna")
	if p.Cells[0] == nil || p.Cells[0].Goals != 5 || p.Cells[0].SevenMAtt != 2 || p.Cells[0].SevenMGoals != 1 {
		t.Errorf("Zelle Spiel 1 = %#v, erwartet 5 Tore, 7m 2/1", p.Cells[0])
	}
	if p.Cells[1] == nil || p.Cells[1].Goals != 3 || p.Cells[1].Warnings != 1 {
		t.Errorf("Zelle Spiel 2 = %#v, erwartet 3 Tore und eine Verwarnung", p.Cells[1])
	}
	if p.Total.Goals != 8 || p.Games != 2 {
		t.Errorf("Summe = %d Tore in %d Spielen, erwartet 8 in 2", p.Total.Goals, p.Games)
	}
	if m.GameTotals[0].Goals != 5 || m.Total.Goals != 8 {
		t.Errorf("Mannschaftssummen = %d / %d, erwartet 5 / 8", m.GameTotals[0].Goals, m.Total.Goals)
	}
	if m.ReportGames != 2 {
		t.Errorf("ReportGames = %d, erwartet 2", m.ReportGames)
	}
}

// Ohne eine einzige verknüpfte Begegnung gibt es keine Mannschaft — auch nicht
// über einen ähnlichen Namen. Eine falsche Matrix sähe genauso plausibel aus
// wie die richtige (design.md §2).
func TestPlayerGameMatrix_OhneZugehoerigkeitLeer(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	teamID := testutil.CreateTeam(t, db, "Team Stuttgart 2")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	addToKader(t, db, kaderID, memberID)

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("Matrizen = %#v, erwartet keine — trotz namensgleicher Mannschaft", list)
	}
}

// Eine gespielte Begegnung ohne ausgewerteten Bericht bleibt eine Spalte mit
// Endstand. Sie wegzulassen hieße, die Saisonsumme über eine unsichtbare
// Teilmenge zu bilden (design.md §3).
func TestPlayerGameMatrix_SpielOhneBerichtIstSpalteOhneWerte(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	g1 := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	seedResult(t, db, staffelID, "2", "2026-09-27", "Team Stuttgart 2", "Fremd", intp(20), intp(24))
	seedResult(t, db, staffelID, "3", "2026-10-04", "Team Stuttgart 2", "Fremd", nil, nil)
	userID, _ := ownTeamSetup(t, db, seasonID, staffelID, g1, true)
	seedParsedReport(t, db, staffelID, g1, "parsed",
		map[string]string{"home": "TS 2", "guest": "Fremd"},
		[]rosterLine{{name: "Anna", side: "home", goals: 5}})

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	m := matrixOf(t, list, "Team Stuttgart 2")
	if len(m.Games) != 2 {
		t.Fatalf("Spalten = %d, erwartet 2 — die ungespielte Begegnung erzeugt keine", len(m.Games))
	}
	if !m.Games[0].HasReport || m.Games[1].HasReport {
		t.Errorf("HasReport = %v / %v, erwartet true / false", m.Games[0].HasReport, m.Games[1].HasReport)
	}
	if m.ReportGames != 1 {
		t.Errorf("ReportGames = %d, erwartet 1 von 2 Spielen", m.ReportGames)
	}
	if p := matrixPlayerRow(t, m, "Anna"); p.Cells[1] != nil {
		t.Errorf("Zelle der berichtslosen Spalte = %#v, erwartet keine", p.Cells[1])
	}
}

// "Nicht im Kader" und "kein Tor" sind zwei Aussagen. Ohne die Unterscheidung
// wäre die Spalte "Spiele" nicht erklärbar (design.md §4).
func TestPlayerGameMatrix_NichtImKaderIstLeerNichtNull(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	g1 := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	g2 := seedResult(t, db, staffelID, "2", "2026-09-27", "Team Stuttgart 2", "Fremd", intp(20), intp(24))
	userID, _ := ownTeamSetup(t, db, seasonID, staffelID, g1, true)
	seedParsedReport(t, db, staffelID, g1, "parsed",
		map[string]string{"home": "TS 2", "guest": "Fremd"},
		[]rosterLine{{name: "Anna", side: "home", goals: 5}, {name: "Bea", side: "home", goals: 0}})
	seedParsedReport(t, db, staffelID, g2, "parsed",
		map[string]string{"home": "TS 2", "guest": "Fremd"},
		[]rosterLine{{name: "Anna", side: "home", goals: 2}})

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	m := matrixOf(t, list, "Team Stuttgart 2")
	bea := matrixPlayerRow(t, m, "Bea")
	if bea.Cells[0] == nil || bea.Cells[0].Goals != 0 {
		t.Errorf("Bea Spiel 1 = %#v, erwartet eine Zelle mit 0 Toren (sie war dabei)", bea.Cells[0])
	}
	if bea.Cells[1] != nil {
		t.Errorf("Bea Spiel 2 = %#v, erwartet keine Zelle (nicht in der Mannschaftsliste)", bea.Cells[1])
	}
	if bea.Games != 1 {
		t.Errorf("Bea Spiele = %d, erwartet 1", bea.Games)
	}
}

// Die dritte Zeitstrafe ist die Rote Karte und zählt einmal — dieselbe
// Umrechnung wie in jeder anderen Auswertung (design.md §5).
func TestPlayerGameMatrix_DritteZeitstrafeZaehltAlsRot(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	g1 := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	userID, _ := ownTeamSetup(t, db, seasonID, staffelID, g1, true)
	seedParsedReport(t, db, staffelID, g1, "parsed",
		map[string]string{"home": "TS 2", "guest": "Fremd"},
		[]rosterLine{{name: "Anna", side: "home", twoMin: 3, red: 1}})

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	p := matrixPlayerRow(t, matrixOf(t, list, "Team Stuttgart 2"), "Anna")
	if p.Cells[0].TwoMin != 2 || p.Cells[0].Disq != 1 {
		t.Errorf("Strafen = %d× 2min, %d× Rot; erwartet 2 und 1",
			p.Cells[0].TwoMin, p.Cells[0].Disq)
	}
}

// Die Mannschaftsliste des PDF schreibt "TS 2", der Spielplan "Team Stuttgart
// 2". Ein Vergleich gegen den Berichtsnamen lieferte eine leere Matrix — ohne
// Fehler, ohne Hinweis (design.md §5).
func TestPlayerGameMatrix_SchreibweiseDesSpielplans(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	g1 := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	userID, _ := ownTeamSetup(t, db, seasonID, staffelID, g1, true)
	seedParsedReport(t, db, staffelID, g1, "parsed",
		map[string]string{"home": "TS 2", "guest": "Fremd"},
		[]rosterLine{{name: "Anna", side: "home", goals: 5}, {name: "Gegner", side: "guest", goals: 7}})

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	m := matrixOf(t, list, "Team Stuttgart 2")
	if len(m.Players) != 1 || m.Players[0].Name != "Anna" {
		t.Fatalf("Spieler = %#v, erwartet nur Anna — der Gegner gehört nicht in diese Matrix", m.Players)
	}
}

// Ein gescheiterter Bericht trägt keine Zeile bei.
func TestPlayerGameMatrix_ParseFailedLiefertKeineWerte(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	g1 := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	userID, _ := ownTeamSetup(t, db, seasonID, staffelID, g1, true)
	seedParsedReport(t, db, staffelID, g1, "parse_failed",
		map[string]string{"home": "TS 2", "guest": "Fremd"},
		[]rosterLine{{name: "Anna", side: "home", goals: 5}})

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	m := matrixOf(t, list, "Team Stuttgart 2")
	if len(m.Players) != 0 {
		t.Errorf("Spieler = %#v, erwartet keine", m.Players)
	}
	if m.Games[0].HasReport {
		t.Error("HasReport = true, erwartet false bei parse_failed")
	}
}

// Die gepflegte Halbzeitdauer der Altersklasse wird mitgeliefert.
func TestPlayerGameMatrix_SpieldauerAusDenEinstellungen(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	g1 := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	userID, teamID := ownTeamSetup(t, db, seasonID, staffelID, g1, true)
	setAgeClassRule(t, db, teamID, "B-Jugend", 25)

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	m := matrixOf(t, list, "Team Stuttgart 2")
	if m.HalfDurationMinutes == nil || *m.HalfDurationMinutes != 25 {
		t.Errorf("HalfDurationMinutes = %v, erwartet 25", m.HalfDurationMinutes)
	}
}

// Ohne gepflegte Regel bleibt das Feld leer — kein erfundener Standardwert,
// der aussähe wie eine gemessene Zahl (design.md §7).
func TestPlayerGameMatrix_OhneAltersklassenRegelKeineSpieldauer(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	g1 := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	userID, teamID := ownTeamSetup(t, db, seasonID, staffelID, g1, true)
	setAgeClassRule(t, db, teamID, "E-Jugend", 0)

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	if m := matrixOf(t, list, "Team Stuttgart 2"); m.HalfDurationMinutes != nil {
		t.Errorf("HalfDurationMinutes = %v, erwartet keine Angabe", *m.HalfDurationMinutes)
	}
}

// Zwei eigene Mannschaften derselben Staffel ergeben zwei Matrizen. "Die erste
// gewinnt" wäre ein stiller Verlust genau dort, wo er nicht auffällt
// (design.md §2).
func TestPlayerGameMatrix_ZweiEigeneMannschaftenInEinerStaffel(t *testing.T) {
	db, s, seasonID, staffelID := newStaffel(t)
	g1 := seedResult(t, db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	g2 := seedResult(t, db, staffelID, "2", "2026-09-27", "Team Stuttgart 3", "Fremd", intp(20), intp(24))

	userID, _ := ownTeamSetup(t, db, seasonID, staffelID, g1, true)
	memberID := testutil.CreateMember(t, db, userID)
	team3 := testutil.CreateTeam(t, db, "B-Jugend 2")
	kader3 := testutil.CreateKader(t, db, team3, seasonID)
	addToKader(t, db, kader3, memberID)
	linkOwnGame(t, db, g2, seasonID, team3, true)

	list, err := s.PlayerGameMatrix(context.Background(), staffelID, userID)
	if err != nil {
		t.Fatalf("PlayerGameMatrix: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("Matrizen = %d, erwartet 2", len(list))
	}
	m2 := matrixOf(t, list, "Team Stuttgart 2")
	m3 := matrixOf(t, list, "Team Stuttgart 3")
	if len(m2.Games) != 1 || m2.Games[0].BwhvGameID != g1 {
		t.Errorf("Mannschaft 2: Spalten = %#v, erwartet nur die eigene Begegnung", m2.Games)
	}
	if len(m3.Games) != 1 || m3.Games[0].BwhvGameID != g2 {
		t.Errorf("Mannschaft 3: Spalten = %#v, erwartet nur die eigene Begegnung", m3.Games)
	}
}

// --- Route ----------------------------------------------------------------

type playerGamesResponse struct {
	Teams []TeamMatrix `json:"teams"`
}

// Der Weg vom Token bis zur Matrix in einem Stück.
func TestGetPlayerGames_HappyPath(t *testing.T) {
	srv, s, seasonID, staffelID := newHandlerServer(t)
	g1 := seedResult(t, s.db, staffelID, "1", "2026-09-20", "Team Stuttgart 2", "Fremd", intp(29), intp(25))
	userID, teamID := ownTeamSetup(t, s.db, seasonID, staffelID, g1, true)
	setAgeClassRule(t, s.db, teamID, "B-Jugend", 25)
	seedParsedReport(t, s.db, staffelID, g1, "parsed",
		map[string]string{"home": "TS 2", "guest": "Fremd"},
		[]rosterLine{{name: "Anna", side: "home", goals: 5}})

	token := testutil.Token(t, userID, "standard", []string{"spieler"})
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/player-games", token)
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var resp playerGamesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, body)
	}
	if len(resp.Teams) != 1 {
		t.Fatalf("Mannschaften = %d, erwartet 1: %s", len(resp.Teams), body)
	}
	m := resp.Teams[0]
	if m.Team != "Team Stuttgart 2" || len(m.Games) != 1 || len(m.Players) != 1 {
		t.Errorf("Matrix = %+v, erwartet eine Mannschaft mit einem Spiel und einem Spieler", m)
	}
	if m.HalfDurationMinutes == nil || *m.HalfDurationMinutes != 25 {
		t.Errorf("HalfDurationMinutes = %v, erwartet 25", m.HalfDurationMinutes)
	}
}

// Ohne Zugehörigkeit steht die Menge leer in der Antwort — als Array, nicht
// als null, damit das Frontend nicht dagegen prüfen muss.
func TestGetPlayerGames_OhneZugehoerigkeitLeer(t *testing.T) {
	srv, _, _, staffelID := newStatsServer(t)
	code, body := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/player-games", userToken(t))
	if code != http.StatusOK {
		t.Fatalf("Status = %d: %s", code, body)
	}
	var resp playerGamesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Antwort nicht lesbar: %v (%s)", err, body)
	}
	if resp.Teams == nil {
		t.Errorf("teams = null, erwartet leeres Array: %s", body)
	}
	if len(resp.Teams) != 0 {
		t.Errorf("teams = %#v, erwartet leer", resp.Teams)
	}
}

// Eine unbekannte Staffel erfindet keine Matrix.
func TestGetPlayerGames_UnbekannteStaffel(t *testing.T) {
	srv, _, _, _ := newHandlerServer(t)
	code, _ := get(t, srv, "/api/staffeln/999999/player-games", userToken(t))
	if code != http.StatusNotFound {
		t.Errorf("Status = %d, erwartet 404", code)
	}
}

// Das Authenticated-Tier greift.
func TestGetPlayerGames_OhneToken(t *testing.T) {
	srv, _, _, staffelID := newHandlerServer(t)
	code, _ := get(t, srv, "/api/staffeln/"+itoa(staffelID)+"/player-games", "")
	if code != http.StatusUnauthorized {
		t.Errorf("Status = %d, erwartet 401", code)
	}
}
