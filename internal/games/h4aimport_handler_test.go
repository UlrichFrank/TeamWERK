package games

// Tests für den H4A-Import (preview/apply). Interner Test-Package, weil der
// H4A-Client über den unexportierten Seam Handler.newH4A injiziert wird — im
// Test darf niemals ein echter Netzzugriff auf Handball4All passieren.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/h4aimport"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// --- Fake-Client ----------------------------------------------------------------

type fakeH4A struct {
	loginErr error
	fetchErr error
	parseBad bool
	html     string
	periods  []h4aimport.Period

	gotUser   string
	gotPw     string
	loggedOut bool
}

func (f *fakeH4A) Login(ctx context.Context, user, pw string) error {
	f.gotUser, f.gotPw = user, pw
	return f.loginErr
}

func (f *fakeH4A) FetchPeriods(ctx context.Context) ([]h4aimport.Period, error) {
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	return f.periods, nil
}

func (f *fakeH4A) FetchGamesHTML(ctx context.Context, periodID string) (string, error) {
	if f.fetchErr != nil {
		return "", f.fetchErr
	}
	if f.parseBad {
		return "<html>kein gametable</html>", nil
	}
	return f.html, nil
}

func (f *fakeH4A) Logout(ctx context.Context) error { f.loggedOut = true; return nil }

// --- Fixtures -------------------------------------------------------------------

// h4aRow baut eine Spielzeile im H4A-Format (13 td-Zellen + Buttons-Tabelle,
// Zeile bewusst ohne </tr> — genau wie im echten Response, siehe parse.go).
func h4aRow(internalID, staffel, gameNo, halle, datum, zeit, heim, gast string) string {
	return `<tr id="game` + internalID + `">` +
		`<td class="ge_ok">&#160;</td><td>&#160;</td><td>&#160;</td>` +
		`<td>` + staffel + `</td><td>` + gameNo + `</td><td>109</td><td>1</td>` +
		`<td>` + halle + `</td><td>` + datum + `</td><td>` + zeit + `</td>` +
		`<td>` + heim + `</td><td>` + gast + `</td><td></td>` +
		`<td><table class="ge_buttons_container"><tr><td>x</td></tr>`
}

func h4aTable(rows ...string) string {
	return `<table class="ge_gameday"><tbody>` + strings.Join(rows, "\n") + `</tbody></table>`
}

// zwei Zeilen: eine Heim-Partie mit bekannter Staffel + bekannter Halle,
// eine Auswärts-Partie mit unbekannter Staffel + unbekannter Halle.
func twoGameFixture() string {
	return h4aTable(
		h4aRow("9570086", "mC-OL-3-BW", "905996", "3059", "Sa, 26.09.2026", "14:45",
			"<b>Team Stuttgart</b>", "Bregenz Handb."),
		h4aRow("9715601", "xY-UNBEKANNT", "211004", "9999", "Sa, 19.09.2026", "12:30",
			"HSC Schm/Oeff", "<b>Team Stuttgart 2</b>"),
	)
}

// h4aTestServer mountet die beiden Import-Routen mit injiziertem Fake-Client.
func h4aTestServer(t *testing.T, db *sql.DB, fake *fakeH4A) (*httptest.Server, *hub.EventHub) {
	t.Helper()
	hb := hub.NewHub()
	h := NewHandler(db, testutil.TestConfig(), hb)
	h.newH4A = func() h4aFetcher { return fake }
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/games/import/h4a/preview", h.PreviewH4AImport)
		r.Post("/api/games/import/h4a/apply", h.ApplyH4AImport)
	})
	return srv, hb
}

// h4aBaseData legt aktive Saison, Mannschaft, Halle (hall_number 3059) und das
// gelernte Staffel-Mapping an.
func h4aBaseData(t *testing.T, db *sql.DB) (seasonID, teamID, venueID int) {
	t.Helper()
	seasonID = testutil.CreateSeason(t, db, "2026/27")
	if _, err := db.Exec(`UPDATE seasons SET is_active=1 WHERE id=?`, seasonID); err != nil {
		t.Fatalf("Saison aktivieren: %v", err)
	}
	teamID = testutil.CreateTeam(t, db, "Männliche C")
	res, err := db.Exec(
		`INSERT INTO venues (name, street, city, postal_code, hall_number)
		 VALUES ('Sporthalle Fixture', 'Hallenweg 1', 'Stuttgart', '70173', 3059)`)
	if err != nil {
		t.Fatalf("Venue anlegen: %v", err)
	}
	vid, _ := res.LastInsertId()
	venueID = int(vid)
	if _, err := db.Exec(
		`INSERT INTO h4a_staffel_team_map (staffel, club_alias, team_id) VALUES (?,?,?)`,
		"mC-OL-3-BW", "Team Stuttgart", teamID); err != nil {
		t.Fatalf("Staffel-Mapping anlegen: %v", err)
	}
	return seasonID, teamID, venueID
}

func vorstandToken(t *testing.T, db *sql.DB) string {
	t.Helper()
	uid := testutil.CreateUser(t, db, "standard")
	return testutil.Token(t, uid, "standard", []string{"vorstand"})
}

func decodePreview(t *testing.T, res *http.Response) h4aPreviewResponse {
	t.Helper()
	var got h4aPreviewResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Preview-Response dekodieren: %v", err)
	}
	return got
}

// --- 4.4 / 6.1 Preview ----------------------------------------------------------

// TestPreviewH4A_HappyPath deckt zugleich die Mapping-Logik ab: bekannte Staffel
// wird vorbelegt, unbekannte Staffel und unaufgelöste Halle werden als Warnung
// gemeldet, der Spieltyp kommt aus dem eigenen Vereinsnamen (nicht aus der Halle).
func TestPreviewH4A_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	_, teamID, venueID := h4aBaseData(t, db)
	fake := &fakeH4A{html: twoGameFixture()}
	srv, _ := h4aTestServer(t, db, fake)

	res := testutil.Post(t, srv, "/api/games/import/h4a/preview", vorstandToken(t, db),
		map[string]any{"user": "v_109", "pw": "geheim", "period_id": "142"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	got := decodePreview(t, res)
	if len(got.New) != 2 {
		t.Fatalf("erwartet 2 neue Spiele, got %d (changed=%d unchanged=%d)",
			len(got.New), len(got.Changed), len(got.Unchanged))
	}

	// Zeile 1: bekannte Staffel + bekannte Halle, wir sind Heim.
	home := got.New[0]
	if home.TeamID == nil || *home.TeamID != teamID {
		t.Errorf("bekannte Staffel: erwartet team_id=%d, got %v", teamID, home.TeamID)
	}
	if home.VenueID == nil || *home.VenueID != venueID {
		t.Errorf("Halle 3059: erwartet venue_id=%d, got %v", venueID, home.VenueID)
	}
	if !home.IsHome || home.EventType != "heim" {
		t.Errorf("erwartet Heimspiel, got is_home=%v event_type=%q", home.IsHome, home.EventType)
	}
	if home.Opponent != "Bregenz Handb." {
		t.Errorf("Gegner: erwartet 'Bregenz Handb.', got %q", home.Opponent)
	}
	if home.Date != "2026-09-26" || home.Time != "14:45" {
		t.Errorf("Datum/Zeit: got %q %q", home.Date, home.Time)
	}
	if len(home.Warnings) != 0 {
		t.Errorf("vollständig aufgelöste Zeile darf keine Warnung tragen, got %v", home.Warnings)
	}

	// Zeile 2: unbekannte Staffel + unbekannte Halle, wir sind Gast.
	away := got.New[1]
	if away.TeamID != nil {
		t.Errorf("unbekannte Staffel: erwartet team_id=nil, got %v", *away.TeamID)
	}
	if away.VenueID != nil {
		t.Errorf("unbekannte Halle: erwartet venue_id=nil, got %v", *away.VenueID)
	}
	if away.IsHome || away.EventType != "auswärts" {
		t.Errorf("erwartet Auswärtsspiel, got is_home=%v event_type=%q", away.IsHome, away.EventType)
	}
	if away.Opponent != "HSC Schm/Oeff" {
		t.Errorf("Gegner: erwartet 'HSC Schm/Oeff', got %q", away.Opponent)
	}
	joined := strings.Join(away.Warnings, "|")
	if !strings.Contains(joined, "Mannschaft nicht zugeordnet") {
		t.Errorf("erwartet Warnung 'Mannschaft nicht zugeordnet', got %v", away.Warnings)
	}
	if !strings.Contains(joined, "Halle 9999 unbekannt") {
		t.Errorf("erwartet Warnung zur unbekannten Halle, got %v", away.Warnings)
	}
	if !fake.loggedOut {
		t.Error("Preview muss die H4A-Session am Ende schließen (Logout)")
	}
}

func TestPreviewH4A_OhneVorstand_403(t *testing.T) {
	db := testutil.NewDB(t)
	h4aBaseData(t, db)
	srv, _ := h4aTestServer(t, db, &fakeH4A{html: twoGameFixture()})

	uid := testutil.CreateUser(t, db, "standard")
	res := testutil.Post(t, srv, "/api/games/import/h4a/preview",
		testutil.Token(t, uid, "standard", nil),
		map[string]any{"user": "v_109", "pw": "geheim", "period_id": "142"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("Standard-Nutzer: erwartet 403, got %d", res.StatusCode)
	}
}

func TestPreviewH4A_LoginFehler_502(t *testing.T) {
	db := testutil.NewDB(t)
	h4aBaseData(t, db)
	fake := &fakeH4A{loginErr: errors.New("401 unauthorized: user v_109 pw geheim")}
	srv, _ := h4aTestServer(t, db, fake)

	res := testutil.Post(t, srv, "/api/games/import/h4a/preview", vorstandToken(t, db),
		map[string]any{"user": "v_109", "pw": "geheim", "period_id": "142"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("Login-Fehler: erwartet 502, got %d", res.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(res.Body).Decode(&body)
	if body["error"] != "h4a_login_failed" {
		t.Errorf("erwartet generischen Fehlercode, got %v", body)
	}
	// Weder Passwort noch H4A-interne Fehlermeldung dürfen zurückgespiegelt werden.
	for k, v := range body {
		if strings.Contains(v, "geheim") || strings.Contains(v, "unauthorized") {
			t.Errorf("Antwort spiegelt Interna: %s=%q", k, v)
		}
	}
}

func TestPreviewH4A_FehlendeCredentials_400(t *testing.T) {
	db := testutil.NewDB(t)
	h4aBaseData(t, db)
	srv, _ := h4aTestServer(t, db, &fakeH4A{html: twoGameFixture()})
	token := vorstandToken(t, db)

	for _, body := range []map[string]any{
		{"period_id": "142"},
		{"user": "v_109", "period_id": "142"},
		{"pw": "geheim", "period_id": "142"},
	} {
		res := testutil.Post(t, srv, "/api/games/import/h4a/preview", token, body)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("Body %v: erwartet 400, got %d", body, res.StatusCode)
		}
		res.Body.Close()
	}
}

func TestPreviewH4A_OhnePeriode_LiefertPeriodenliste(t *testing.T) {
	db := testutil.NewDB(t)
	h4aBaseData(t, db)
	fake := &fakeH4A{periods: []h4aimport.Period{{ID: "142", Name: "Hallenrunde 26/27"}}}
	srv, _ := h4aTestServer(t, db, fake)

	res := testutil.Post(t, srv, "/api/games/import/h4a/preview", vorstandToken(t, db),
		map[string]any{"user": "v_109", "pw": "geheim"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	got := decodePreview(t, res)
	if !got.NeedsPeriod || len(got.Periods) != 1 || got.Periods[0].ID != "142" {
		t.Errorf("erwartet needs_period + 1 Periode, got %+v", got)
	}
}

func TestPreviewH4A_FormatBruch_502(t *testing.T) {
	db := testutil.NewDB(t)
	h4aBaseData(t, db)
	srv, _ := h4aTestServer(t, db, &fakeH4A{parseBad: true})

	res := testutil.Post(t, srv, "/api/games/import/h4a/preview", vorstandToken(t, db),
		map[string]any{"user": "v_109", "pw": "geheim", "period_id": "142"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("Parse-Fehler: erwartet 502, got %d", res.StatusCode)
	}
}

// --- 6.1 Diff -------------------------------------------------------------------

func TestPreviewH4A_DiffErkenntUnverändertUndGeändert(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, venueID := h4aBaseData(t, db)

	// Bestandsspiel identisch zur Fixture → unchanged.
	if _, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type, venue_id, external_id)
		 VALUES (?,?,?,?,?,?,?,?)`,
		seasonID, "Bregenz Handb.", "2026-09-26", "14:45", 1, "heim", venueID, "905996"); err != nil {
		t.Fatalf("Bestandsspiel anlegen: %v", err)
	}
	// Bestandsspiel mit abweichender Zeit → changed.
	if _, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type, external_id)
		 VALUES (?,?,?,?,?,?,?)`,
		seasonID, "HSC Schm/Oeff", "2026-09-19", "10:00", 0, "auswärts", "211004"); err != nil {
		t.Fatalf("Bestandsspiel anlegen: %v", err)
	}
	_ = teamID

	srv, _ := h4aTestServer(t, db, &fakeH4A{html: twoGameFixture()})
	res := testutil.Post(t, srv, "/api/games/import/h4a/preview", vorstandToken(t, db),
		map[string]any{"user": "v_109", "pw": "geheim", "period_id": "142"})
	defer res.Body.Close()
	got := decodePreview(t, res)

	if len(got.New) != 0 {
		t.Errorf("erwartet 0 neue Spiele, got %d", len(got.New))
	}
	if len(got.Unchanged) != 1 || got.Unchanged[0].GameNo != "905996" {
		t.Errorf("erwartet 1 unverändertes Spiel 905996, got %+v", got.Unchanged)
	}
	if len(got.Changed) != 1 {
		t.Fatalf("erwartet 1 geändertes Spiel, got %d", len(got.Changed))
	}
	ch := got.Changed[0]
	if ch.GameNo != "211004" {
		t.Errorf("erwartet Spielnummer 211004, got %q", ch.GameNo)
	}
	var timeChange *h4aFieldChange
	for i := range ch.Changes {
		if ch.Changes[i].Field == "time" {
			timeChange = &ch.Changes[i]
		}
	}
	if timeChange == nil {
		t.Fatalf("erwartet Feld-Änderung 'time', got %+v", ch.Changes)
	}
	if timeChange.Old != "10:00" || timeChange.New != "12:30" {
		t.Errorf("erwartet time 10:00→12:30, got %s→%s", timeChange.Old, timeChange.New)
	}
	if ch.ExistingGameID == nil {
		t.Error("geändertes Spiel muss die bestehende game-ID tragen")
	}
}

// Ein Bestandsspiel, das im Abruf fehlt, darf nicht als Löschung auftauchen.
func TestPreviewH4A_FehlendesBestandsspielBleibtUnangetastet(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, _, _ := h4aBaseData(t, db)
	if _, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type, external_id)
		 VALUES (?,?,?,?,?,?,?)`,
		seasonID, "Nicht im Abruf", "2026-10-10", "15:00", 1, "heim", "999999"); err != nil {
		t.Fatalf("Bestandsspiel anlegen: %v", err)
	}

	srv, _ := h4aTestServer(t, db, &fakeH4A{html: twoGameFixture()})
	res := testutil.Post(t, srv, "/api/games/import/h4a/preview", vorstandToken(t, db),
		map[string]any{"user": "v_109", "pw": "geheim", "period_id": "142"})
	defer res.Body.Close()
	got := decodePreview(t, res)

	for _, g := range append(append(append([]h4aPlanGame{}, got.New...), got.Changed...), got.Unchanged...) {
		if g.GameNo == "999999" {
			t.Fatalf("Spiel 999999 darf im Plan nicht auftauchen (keine Löschableitung)")
		}
	}
	var still int
	db.QueryRow(`SELECT COUNT(*) FROM games WHERE external_id='999999'`).Scan(&still)
	if still != 1 {
		t.Errorf("Bestandsspiel muss unangetastet bleiben, got %d Zeilen", still)
	}
}

// Vor dem ersten Import tragen alle Spiele keine external_id — ein handangelegtes
// Spiel am selben Datum gegen denselben Gegner muss als Dublettenkandidat auffallen.
func TestPreviewH4A_ErkenntMöglicheDublette(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, _ := h4aBaseData(t, db)

	res, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type)
		 VALUES (?,?,?,?,?,?)`,
		seasonID, "Bregenz Handb.", "2026-09-26", "14:45", 1, "heim")
	if err != nil {
		t.Fatalf("manuelles Spiel anlegen: %v", err)
	}
	manualID, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?,?)`,
		manualID, teamID); err != nil {
		t.Fatalf("game_teams: %v", err)
	}

	srv, _ := h4aTestServer(t, db, &fakeH4A{html: twoGameFixture()})
	resp := testutil.Post(t, srv, "/api/games/import/h4a/preview", vorstandToken(t, db),
		map[string]any{"user": "v_109", "pw": "geheim", "period_id": "142"})
	defer resp.Body.Close()
	got := decodePreview(t, resp)

	if len(got.New) != 2 {
		t.Fatalf("erwartet 2 neue Zeilen, got %d", len(got.New))
	}
	dup := got.New[0]
	if dup.DuplicateOfGameID == nil || *dup.DuplicateOfGameID != int(manualID) {
		t.Errorf("erwartet Dublettenhinweis auf game %d, got %v", manualID, dup.DuplicateOfGameID)
	}
	if !strings.Contains(strings.Join(dup.Warnings, "|"), "mögliche Dublette") {
		t.Errorf("erwartet Dubletten-Warnung, got %v", dup.Warnings)
	}
	// Die zweite Zeile (anderes Datum/Gegner) darf NICHT als Dublette gelten.
	if got.New[1].DuplicateOfGameID != nil {
		t.Errorf("Zeile ohne Bestandspendant darf keinen Dublettenhinweis tragen")
	}
}

// --- 6.4 Credential-Nichtpersistenz ---------------------------------------------

func TestPreviewH4A_CredentialsWerdenNichtPersistiertOderGeloggt(t *testing.T) {
	const password = "S3hrGeheimesPasswort!"
	db := testutil.NewDB(t)
	h4aBaseData(t, db)
	srv, _ := h4aTestServer(t, db, &fakeH4A{html: twoGameFixture()})

	var logBuf strings.Builder
	oldOut := log.Writer()
	oldFlags := log.Flags()
	log.SetOutput(&logBuf)
	t.Cleanup(func() { log.SetOutput(oldOut); log.SetFlags(oldFlags) })

	res := testutil.Post(t, srv, "/api/games/import/h4a/preview", vorstandToken(t, db),
		map[string]any{"user": "v_109", "pw": password, "period_id": "142"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	body := make([]byte, 1<<16)
	n, _ := res.Body.Read(body)

	if strings.Contains(logBuf.String(), password) {
		t.Error("Passwort landet im Log")
	}
	if strings.Contains(string(body[:n]), password) {
		t.Error("Passwort wird in der Response zurückgespiegelt")
	}
	if table, col := findStringInDB(t, db, password); table != "" {
		t.Errorf("Passwort in der DB persistiert: %s.%s", table, col)
	}

	// Sanity: der Scanner muss etwas finden können, sonst wäre die Assertion oben
	// wertlos (ein kaputter Scanner meldet immer „nicht gefunden").
	if table, _ := findStringInDB(t, db, "mC-OL-3-BW"); table != "h4a_staffel_team_map" {
		t.Fatalf("DB-Scanner findet bekannten Wert nicht — Assertion oben wäre wertlos (got %q)", table)
	}
}

// Der Fehlerpfad ist die gefährlichste Stelle: dort landen erfahrungsgemäß
// Request-Dumps im Log. Die Diagnose-Ausgabe muss den Grund nennen, aber
// niemals die Zugangsdaten.
func TestPreviewH4A_FehlerdiagnoseOhneZugangsdaten(t *testing.T) {
	const password = "S3hrGeheimesPasswort!"
	const user = "v_109"
	db := testutil.NewDB(t)
	h4aBaseData(t, db)
	fake := &fakeH4A{loginErr: errors.New("Post \"https://meinh4a.handball4all.de/index.php\": dial tcp: i/o timeout")}
	srv, _ := h4aTestServer(t, db, fake)

	var logBuf strings.Builder
	old := h4aLogOut
	h4aLogOut = &logBuf
	t.Cleanup(func() { h4aLogOut = old })

	res := testutil.Post(t, srv, "/api/games/import/h4a/preview", vorstandToken(t, db),
		map[string]any{"user": user, "pw": password, "period_id": "142"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadGateway {
		t.Fatalf("erwartet 502, got %d", res.StatusCode)
	}

	logged := logBuf.String()
	// Der Grund muss drinstehen — sonst ist ein Fehlschlag nicht diagnostizierbar.
	if !strings.Contains(logged, "login") || !strings.Contains(logged, "i/o timeout") {
		t.Errorf("Fehlergrund fehlt im Log: %q", logged)
	}
	// Die Zugangsdaten dürfen es nicht sein.
	if strings.Contains(logged, password) {
		t.Errorf("Passwort im Diagnose-Log: %q", logged)
	}
	if strings.Contains(logged, user) {
		t.Errorf("Benutzername im Diagnose-Log: %q", logged)
	}
}

// findStringInDB durchsucht alle Tabellen/Spalten nach needle und liefert den
// ersten Fundort (leer, wenn nirgends gefunden).
func findStringInDB(t *testing.T, db *sql.DB, needle string) (string, string) {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatalf("Tabellenliste: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}
	rows.Close()

	for _, tbl := range tables {
		r, err := db.Query(`SELECT * FROM "` + tbl + `"`)
		if err != nil {
			continue
		}
		cols, _ := r.Columns()
		for r.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := r.Scan(ptrs...); err != nil {
				continue
			}
			for i, v := range vals {
				var s string
				switch tv := v.(type) {
				case string:
					s = tv
				case []byte:
					s = string(tv)
				default:
					continue
				}
				if strings.Contains(s, needle) {
					r.Close()
					return tbl, cols[i]
				}
			}
		}
		r.Close()
	}
	return "", ""
}

// --- 6.2 Apply ------------------------------------------------------------------

func applyDecision(gameNo string, teamID int, venueID *int, date, tim, eventType string) map[string]any {
	d := map[string]any{
		"game_no":    gameNo,
		"staffel":    "mC-OL-3-BW",
		"club_alias": "Team Stuttgart",
		"team_id":    teamID,
		"opponent":   "Bregenz Handb.",
		"date":       date,
		"time":       tim,
		"event_type": eventType,
	}
	if venueID != nil {
		d["venue_id"] = *venueID
	}
	return d
}

func TestApplyH4A_SchreibtNeueUndGeänderteSpiele(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, venueID := h4aBaseData(t, db)
	// Bestandsspiel, das geändert wird.
	if _, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type, external_id)
		 VALUES (?,?,?,?,?,?,?)`,
		seasonID, "Bregenz Handb.", "2026-09-19", "10:00", 0, "auswärts", "211004"); err != nil {
		t.Fatalf("Bestandsspiel anlegen: %v", err)
	}
	srv, _ := h4aTestServer(t, db, &fakeH4A{})

	res := testutil.Post(t, srv, "/api/games/import/h4a/apply", vorstandToken(t, db),
		map[string]any{"decisions": []map[string]any{
			applyDecision("905996", teamID, &venueID, "2026-09-26", "14:45", "heim"),
			applyDecision("211004", teamID, nil, "2026-09-19", "12:30", "auswärts"),
		}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	var out h4aApplyResponse
	json.NewDecoder(res.Body).Decode(&out)
	if out.Imported != 1 || out.Updated != 1 || out.Skipped != 0 {
		t.Errorf("erwartet imported=1 updated=1 skipped=0, got %+v", out)
	}

	var gotVenue sql.NullInt64
	var gotDate, gotTime, gotType string
	if err := db.QueryRow(
		`SELECT date, time, event_type, venue_id FROM games WHERE external_id='905996'`).
		Scan(&gotDate, &gotTime, &gotType, &gotVenue); err != nil {
		t.Fatalf("neues Spiel nicht gefunden: %v", err)
	}
	if gotDate[:10] != "2026-09-26" || gotTime != "14:45" || gotType != "heim" {
		t.Errorf("neues Spiel falsch geschrieben: %s %s %s", gotDate, gotTime, gotType)
	}
	if !gotVenue.Valid || int(gotVenue.Int64) != venueID {
		t.Errorf("venue_id nicht gesetzt: %+v", gotVenue)
	}

	var updTime string
	db.QueryRow(`SELECT time FROM games WHERE external_id='211004'`).Scan(&updTime)
	if updTime != "12:30" {
		t.Errorf("geändertes Spiel: erwartet time=12:30, got %q", updTime)
	}

	// game_teams verknüpft, Staffel-Mapping gelernt.
	var teamLinks int
	db.QueryRow(`SELECT COUNT(*) FROM game_teams gt JOIN games g ON g.id=gt.game_id
	             WHERE g.external_id IN ('905996','211004') AND gt.team_id=?`, teamID).Scan(&teamLinks)
	if teamLinks != 2 {
		t.Errorf("erwartet 2 game_teams-Verknüpfungen, got %d", teamLinks)
	}
	var mapCount int
	db.QueryRow(`SELECT COUNT(*) FROM h4a_staffel_team_map WHERE staffel='mC-OL-3-BW'`).Scan(&mapCount)
	if mapCount != 1 {
		t.Errorf("Staffel-Mapping soll idempotent genau einmal existieren, got %d", mapCount)
	}
}

func TestApplyH4A_OhneAktiveSaison_400(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID, teamID, _ := h4aBaseData(t, db)
	if _, err := db.Exec(`UPDATE seasons SET is_active=0 WHERE id=?`, seasonID); err != nil {
		t.Fatalf("Saison deaktivieren: %v", err)
	}
	srv, _ := h4aTestServer(t, db, &fakeH4A{})

	res := testutil.Post(t, srv, "/api/games/import/h4a/apply", vorstandToken(t, db),
		map[string]any{"decisions": []map[string]any{
			applyDecision("905996", teamID, nil, "2026-09-26", "14:45", "heim"),
		}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("ohne aktive Saison: erwartet 400, got %d", res.StatusCode)
	}
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM games WHERE external_id='905996'`).Scan(&count)
	if count != 0 {
		t.Errorf("ohne aktive Saison darf kein Spiel geschrieben werden, got %d", count)
	}
}

func TestApplyH4A_OhneVorstand_403(t *testing.T) {
	db := testutil.NewDB(t)
	_, teamID, _ := h4aBaseData(t, db)
	srv, _ := h4aTestServer(t, db, &fakeH4A{})

	uid := testutil.CreateUser(t, db, "standard")
	res := testutil.Post(t, srv, "/api/games/import/h4a/apply",
		testutil.Token(t, uid, "standard", nil),
		map[string]any{"decisions": []map[string]any{
			applyDecision("905996", teamID, nil, "2026-09-26", "14:45", "heim"),
		}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("Standard-Nutzer: erwartet 403, got %d", res.StatusCode)
	}
}

func TestApplyH4A_ZeileOhneMannschaftWirdÜbersprungen(t *testing.T) {
	db := testutil.NewDB(t)
	_, teamID, _ := h4aBaseData(t, db)
	srv, _ := h4aTestServer(t, db, &fakeH4A{})

	res := testutil.Post(t, srv, "/api/games/import/h4a/apply", vorstandToken(t, db),
		map[string]any{"decisions": []map[string]any{
			applyDecision("905996", 0, nil, "2026-09-26", "14:45", "heim"),      // keine Mannschaft
			applyDecision("211004", 999999, nil, "2026-09-19", "12:30", "heim"), // Mannschaft existiert nicht
			applyDecision("905997", teamID, nil, "2026-09-27", "16:00", "heim"), // gültig
		}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	var out h4aApplyResponse
	json.NewDecoder(res.Body).Decode(&out)
	if out.Skipped != 2 || out.Imported != 1 {
		t.Errorf("erwartet skipped=2 imported=1, got %+v", out)
	}
	var written int
	db.QueryRow(`SELECT COUNT(*) FROM games WHERE external_id IN ('905996','211004')`).Scan(&written)
	if written != 0 {
		t.Errorf("übersprungene Zeilen dürfen nichts schreiben, got %d", written)
	}
}

func TestApplyH4A_UngültigesTemplateWirdÜbersprungen(t *testing.T) {
	db := testutil.NewDB(t)
	_, teamID, _ := h4aBaseData(t, db)
	srv, _ := h4aTestServer(t, db, &fakeH4A{})

	d := applyDecision("905996", teamID, nil, "2026-09-26", "14:45", "heim")
	d["template_id"] = 424242
	res := testutil.Post(t, srv, "/api/games/import/h4a/apply", vorstandToken(t, db),
		map[string]any{"decisions": []map[string]any{d}})
	defer res.Body.Close()

	var out h4aApplyResponse
	json.NewDecoder(res.Body).Decode(&out)
	if out.Skipped != 1 || out.Imported != 0 {
		t.Errorf("ungültiges Template: erwartet skipped=1 imported=0, got %+v", out)
	}
}

// --- 6.3 Idempotenz -------------------------------------------------------------

func TestApplyH4A_ZweimaligesApplyErzeugtKeineDubletten(t *testing.T) {
	db := testutil.NewDB(t)
	_, teamID, venueID := h4aBaseData(t, db)
	srv, _ := h4aTestServer(t, db, &fakeH4A{})
	token := vorstandToken(t, db)
	body := map[string]any{"decisions": []map[string]any{
		applyDecision("905996", teamID, &venueID, "2026-09-26", "14:45", "heim"),
	}}

	first := testutil.Post(t, srv, "/api/games/import/h4a/apply", token, body)
	var out1 h4aApplyResponse
	json.NewDecoder(first.Body).Decode(&out1)
	first.Body.Close()
	if out1.Imported != 1 {
		t.Fatalf("erster Lauf: erwartet imported=1, got %+v", out1)
	}

	second := testutil.Post(t, srv, "/api/games/import/h4a/apply", token, body)
	var out2 h4aApplyResponse
	json.NewDecoder(second.Body).Decode(&out2)
	second.Body.Close()
	if out2.Imported != 0 || out2.Updated != 1 {
		t.Errorf("zweiter Lauf: erwartet imported=0 updated=1, got %+v", out2)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM games WHERE external_id='905996'`).Scan(&count)
	if count != 1 {
		t.Errorf("external_id 905996: erwartet genau 1 Spiel, got %d", count)
	}
	var links int
	db.QueryRow(`SELECT COUNT(*) FROM game_teams gt JOIN games g ON g.id=gt.game_id
	             WHERE g.external_id='905996'`).Scan(&links)
	if links != 1 {
		t.Errorf("game_teams: erwartet genau 1 Verknüpfung, got %d", links)
	}
}

// --- 6.5 Batch: ein Broadcast über mehrere Tage ---------------------------------

func TestApplyH4A_MehrereTage_GenauEinBroadcast(t *testing.T) {
	db := testutil.NewDB(t)
	_, teamID, _ := h4aBaseData(t, db)
	srv, hb := h4aTestServer(t, db, &fakeH4A{})

	// Aktiver Leser, damit kein Event im 1er-Puffer verloren geht.
	ch := hb.Subscribe()
	defer hb.Unsubscribe(ch)
	var mu sync.Mutex
	var events []string
	done := make(chan struct{})
	go func() {
		for {
			select {
			case e := <-ch:
				mu.Lock()
				events = append(events, e)
				mu.Unlock()
			case <-done:
				return
			}
		}
	}()

	res := testutil.Post(t, srv, "/api/games/import/h4a/apply", vorstandToken(t, db),
		map[string]any{"decisions": []map[string]any{
			applyDecision("905996", teamID, nil, "2026-09-26", "14:45", "heim"),
			applyDecision("905997", teamID, nil, "2026-10-03", "16:00", "heim"),
			applyDecision("905998", teamID, nil, "2026-10-17", "18:00", "auswärts"),
		}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, got %d", res.StatusCode)
	}
	var out h4aApplyResponse
	json.NewDecoder(res.Body).Decode(&out)
	if out.Imported != 3 {
		t.Fatalf("erwartet imported=3, got %+v", out)
	}

	time.Sleep(50 * time.Millisecond) // Broadcast-Zustellung abwarten
	close(done)
	mu.Lock()
	defer mu.Unlock()
	var gamesEvents int
	for _, e := range events {
		if e == "games" {
			gamesEvents++
		}
	}
	if gamesEvents != 1 {
		t.Errorf("erwartet genau 1 'games'-Broadcast für den gesamten Lauf, got %d (%v)", gamesEvents, events)
	}
}

// --- 4.4 Alias-Erkennung --------------------------------------------------------

func TestOwnAlias(t *testing.T) {
	cases := []struct {
		name       string
		game       h4aimport.RawGame
		wantAlias  string
		wantIsHome bool
		wantOK     bool
	}{
		{"eigener Verein im Heim-Feld",
			h4aimport.RawGame{Home: "Team Stuttgart", Guest: "HSC Schm/Oeff"},
			"Team Stuttgart", true, true},
		{"eigener Verein im Gast-Feld",
			h4aimport.RawGame{Home: "HSC Schm/Oeff", Guest: "Team Stuttgart 2"},
			"Team Stuttgart 2", false, true},
		{"zweite Mannschaft wird nicht als erste erkannt",
			h4aimport.RawGame{Home: "Team Stuttgart 2", Guest: "HC Winnenden"},
			"Team Stuttgart 2", true, true},
		{"unbekannter Verein fällt auf das Parser-Signal zurück",
			h4aimport.RawGame{Home: "HandballTeam Heckengäu", Guest: "TSV Heiningen", IsHome: false},
			"TSV Heiningen", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			alias, isHome, ok := ownAlias(tc.game)
			if alias != tc.wantAlias || isHome != tc.wantIsHome || ok != tc.wantOK {
				t.Errorf("ownAlias = (%q, %v, %v), erwartet (%q, %v, %v)",
					alias, isHome, ok, tc.wantAlias, tc.wantIsHome, tc.wantOK)
			}
		})
	}
}
