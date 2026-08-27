package duties_test

import (
	"database/sql"
	"encoding/csv"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── Helfer ───────────────────────────────────────────────────────────────────

// exportServer hängt die Export-Route in genau das Tier, in dem sie auch in
// BuildRouter steht (vorstand/trainer/sportliche_leitung) — sonst prüfte der
// 403-Test nichts.
func exportServer(t *testing.T, h *duties.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Get("/api/duty-slots/export", h.ExportSlots)
		})
	})
}

// createTimedDutySlot legt einen Slot mit Uhrzeit und Dauer an — der lokale
// createDutySlot aus handler_test.go lässt beides auf den Defaults.
func createTimedDutySlot(t *testing.T, db *sql.DB, dutyTypeID, seasonID, teamID, gameID int, date, eventTime string, hours float64) int {
	t.Helper()
	var gameArg any
	if gameID > 0 {
		gameArg = gameID
	}
	res, err := db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, role_desc,
		        slots_total, slots_filled, team_id, season_id, game_id, is_custom, hours_value)
		 VALUES (?, ?, ?, ?, ?, 3, 0, ?, ?, ?, 0, ?)`,
		"Heimspieltag", date, eventTime, dutyTypeID, "Kasse links", teamID, seasonID, gameArg, hours)
	if err != nil {
		t.Fatalf("createTimedDutySlot: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// readCSV liest die Antwort als CSV (';' getrennt) und prüft dabei, dass die
// Datei mit UTF-8-BOM beginnt — ohne ihn zerlegt Excel die Umlaute.
func readCSV(t *testing.T, resp *http.Response) [][]string {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekam %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("body lesen: %v", err)
	}
	const bom = "\ufeff"
	if !strings.HasPrefix(string(raw), bom) {
		t.Error("CSV ohne UTF-8-BOM — Excel zeigt Umlaute als Mojibake")
	}
	cr := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), bom)))
	cr.Comma = ';'
	records, err := cr.ReadAll()
	if err != nil {
		t.Fatalf("CSV parsen: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("CSV ohne Kopfzeile")
	}
	return records
}

// col liefert den Wert einer Zeile über den Spaltennamen der Kopfzeile — die
// Tests sollen beim Umsortieren/Einfügen von Spalten nicht brechen.
func col(t *testing.T, records [][]string, row int, name string) string {
	t.Helper()
	for i, h := range records[0] {
		if h == name {
			if i >= len(records[row]) {
				t.Fatalf("Zeile %d hat keine Spalte %d (%q)", row, i, name)
			}
			return records[row][i]
		}
	}
	t.Fatalf("Spalte %q nicht in der Kopfzeile %v", name, records[0])
	return ""
}

// ── Tests ────────────────────────────────────────────────────────────────────

// TestExportSlots_HappyPath: ein Dienst an einem Heimspiel liefert eine Zeile
// mit allen Zeiten (Beginn aus event_time, Ende aus Beginn + Dauer, Anwurf und
// Anwurfzeiten des Tages) sowie dem Tages-Ausrichter.
func TestExportSlots_HappyPath(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Männer 1")
	dtID := createDutyType(t, db, "Kassendienst", 1.5)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-03-14")
	createVenue(t, db, gameID, "Sporthalle Ost")
	createTimedDutySlot(t, db, dtID, seasonID, teamID, gameID, "2099-03-14", "17:15", 1.5)

	vorstandID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})
	srv := exportServer(t, h)

	records := readCSV(t, testutil.Get(t,
		srv, "/api/duty-slots/export?from=2099-03-01&to=2099-03-31", token))
	if len(records) != 2 {
		t.Fatalf("erwartet Kopfzeile + 1 Datenzeile, bekam %d Zeilen", len(records))
	}

	checks := map[string]string{
		"Datum":               "14.03.2099",
		"Wochentag":           "Sa",
		"Termin":              "Heimspieltag",
		"Termin-Art":          "Heimspiel",
		"Gegner":              "Test Opponent",
		"Mannschaften":        "Männer 1",
		"Halle":               "Sporthalle Ost",
		"Anwurf":              "18:00",
		"Anwurfzeiten am Tag": "18:00",
		"Spiele am Tag":       "1",
		"Dienst":              "Kassendienst",
		"Beschreibung":        "Kasse links",
		"Dienst-Beginn":       "17:15",
		"Dienst-Ende":         "18:45",
		"Dauer (Std.)":        "1,50",
		"Plätze":              "3",
		"Herkunft":            "Vorlage",
		"Zielgruppe":          "alle",
		"Heimspiel Vortag":    "nein",
		"Heimspiel Folgetag":  "nein",

		"Regel bei mehreren Spielen am Tag": "normal",
		"Regel bei Spiel am Nachbartag":     "normal",
	}
	for name, want := range checks {
		if got := col(t, records, 1, name); got != want {
			t.Errorf("Spalte %q: erwartet %q, bekam %q", name, want, got)
		}
	}
	// Der per Migration 048 geseedete Default-Ausrichter greift, solange für den
	// Tag nichts gesetzt ist — die Auflösung ist total, die Spalte nie leer.
	if got := col(t, records, 1, "Ausrichter"); got == "" {
		t.Error("Ausrichter-Spalte leer, erwartet den Default-Ausrichter")
	}
	if got := col(t, records, 1, "Ausrichter für Tag gesetzt"); got != "nein" {
		t.Errorf("ohne spieltag_ausrichter-Zeile erwartet \"nein\", bekam %q", got)
	}
}

// TestExportSlots_TageskonstellationUndAusrichter: der Export weist die
// Eingangsgrößen der Regen-Engine aus — mehrere Spiele am Tag, Heimspiel am
// Vor-/Folgetag — und den für den Tag explizit gesetzten Ausrichter.
func TestExportSlots_TageskonstellationUndAusrichter(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Männer 1")
	dtID := createDutyType(t, db, "Aufbau", 1)

	// Zwei Spiele am Zieltag (unterschiedliche Anwurfzeiten) + je ein Heimspiel
	// am Vor- und Folgetag.
	gameA := testutil.CreateGame(t, db, seasonID, teamID, "2099-04-11")
	gameB := testutil.CreateGame(t, db, seasonID, teamID, "2099-04-11")
	if _, err := db.Exec(`UPDATE games SET time='20:00' WHERE id=?`, gameB); err != nil {
		t.Fatalf("Anwurfzeit setzen: %v", err)
	}
	testutil.CreateGame(t, db, seasonID, teamID, "2099-04-10")
	testutil.CreateGame(t, db, seasonID, teamID, "2099-04-12")

	if _, err := db.Exec(
		`INSERT INTO ausrichter (name, aktiv, is_default) VALUES ('TSV Nachbarort', 1, 0)`); err != nil {
		t.Fatalf("Ausrichter anlegen: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO spieltag_ausrichter (date, season_id, ausrichter_id)
		 VALUES ('2099-04-11', ?, (SELECT id FROM ausrichter WHERE name='TSV Nachbarort'))`,
		seasonID); err != nil {
		t.Fatalf("spieltag_ausrichter setzen: %v", err)
	}

	createTimedDutySlot(t, db, dtID, seasonID, teamID, gameA, "2099-04-11", "16:00", 1)

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := exportServer(t, h)

	records := readCSV(t, testutil.Get(t,
		srv, "/api/duty-slots/export?from=2099-04-11&to=2099-04-11", token))
	if len(records) != 2 {
		t.Fatalf("erwartet Kopfzeile + 1 Datenzeile, bekam %d", len(records))
	}
	want := map[string]string{
		"Ausrichter":                 "TSV Nachbarort",
		"Ausrichter für Tag gesetzt": "ja",
		"Spiele am Tag":              "2",
		"Anwurfzeiten am Tag":        "18:00, 20:00",
		"Heimspiel Vortag":           "ja",
		"Heimspiel Folgetag":         "ja",
	}
	for name, w := range want {
		if got := col(t, records, 1, name); got != w {
			t.Errorf("Spalte %q: erwartet %q, bekam %q", name, w, got)
		}
	}
}

// TestExportSlots_DiensttypRegelnUndHerkunft: die am Diensttyp konfigurierten
// Regeln stehen als Text in der Zeile (skip/reduced inkl. Variantenname), ein
// handangelegter Slot als „manuell".
func TestExportSlots_DiensttypRegelnUndHerkunft(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Frauen 1")
	variantID := createDutyType(t, db, "Zeitnahme kurz", 1)
	dtID := createDutyType(t, db, "Zeitnahme", 2)
	if _, err := db.Exec(
		`UPDATE duty_types SET same_day_behavior='reduced', same_day_variant_id=?,
		        adjacent_day_behavior='skip' WHERE id=?`, variantID, dtID); err != nil {
		t.Fatalf("Regeln setzen: %v", err)
	}

	slotID := createTimedDutySlot(t, db, dtID, seasonID, teamID, 0, "2099-05-02", "09:00", 2)
	if _, err := db.Exec(`UPDATE duty_slots SET is_custom=1, audiences='["eltern","trainer"]' WHERE id=?`, slotID); err != nil {
		t.Fatalf("is_custom/audiences setzen: %v", err)
	}

	trainerID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, trainerID, "standard", []string{"trainer"})
	srv := exportServer(t, h)

	records := readCSV(t, testutil.Get(t,
		srv, "/api/duty-slots/export?from=2099-05-01&to=2099-05-31", token))
	if len(records) != 2 {
		t.Fatalf("erwartet Kopfzeile + 1 Datenzeile, bekam %d", len(records))
	}
	want := map[string]string{
		"Regel bei mehreren Spielen am Tag": "reduziert → Zeitnahme kurz",
		"Regel bei Spiel am Nachbartag":     "entfällt",
		"Herkunft":                          "manuell",
		"Zielgruppe":                        "Eltern, Trainer",
		// Slot ohne Spiel: Termin-Art benennt das statt eine Spielart zu erfinden,
		// die Mannschaft kommt dann aus duty_slots.team_id.
		"Termin-Art":    "ohne Termin",
		"Mannschaften":  "Frauen 1",
		"Dienst-Beginn": "09:00",
		"Dienst-Ende":   "11:00",
		"Spiele am Tag": "0",
	}
	for name, w := range want {
		if got := col(t, records, 1, name); got != w {
			t.Errorf("Spalte %q: erwartet %q, bekam %q", name, w, got)
		}
	}
}

// TestExportSlots_OhneBelegungUndNamen: der Export ist die Planungssicht — er
// enthält weder eine Belegungs-Spalte noch Namen von Zugewiesenen, auch wenn
// Zuweisungen existieren. Das ist die Zusage, unter der das Blatt weitergegeben
// werden darf.
func TestExportSlots_OhneBelegungUndNamen(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Männer 1")
	dtID := createDutyType(t, db, "Kasse", 1)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2099-06-06")
	slotID := createTimedDutySlot(t, db, dtID, seasonID, teamID, gameID, "2099-06-06", "17:00", 1)

	assigneeID := testutil.CreateUser(t, db, "standard")
	if _, err := db.Exec(
		`UPDATE users SET first_name='Wilhelmine', last_name='Schmalzhuber' WHERE id=?`, assigneeID); err != nil {
		t.Fatalf("Namen setzen: %v", err)
	}
	insertDutyAssignment(t, db, slotID, assigneeID, "assigned")
	if _, err := db.Exec(`UPDATE duty_slots SET slots_filled=1 WHERE id=?`, slotID); err != nil {
		t.Fatalf("slots_filled setzen: %v", err)
	}

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := exportServer(t, h)

	resp := testutil.Get(t, srv, "/api/duty-slots/export?from=2099-06-01&to=2099-06-30", token)
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	body := string(raw)
	for _, forbidden := range []string{"Wilhelmine", "Schmalzhuber", "Besetzt", "Zugewiesen", "Offen"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("CSV enthält %q — der Export soll ohne Belegung/Namen auskommen:\n%s", forbidden, body)
		}
	}
}

// TestExportSlots_ZeitraumUndISODatum: der Zeitraum grenzt inklusive ab, und ein
// als ISO-Timestamp gespeichertes event_date fällt trotzdem in den Bereich
// (SQLite-DATE-Gotcha — ein nacktes BETWEEN verlöre den Tag am Bereichsende).
func TestExportSlots_ZeitraumUndISODatum(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())

	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 1)

	createTimedDutySlot(t, db, dtID, seasonID, teamID, 0, "2099-07-31", "10:00", 1) // vor dem Bereich
	createTimedDutySlot(t, db, dtID, seasonID, teamID, 0, "2099-08-01", "11:00", 1) // untere Grenze
	isoSlot := createTimedDutySlot(t, db, dtID, seasonID, teamID, 0, "2099-08-31", "12:00", 1)
	if _, err := db.Exec(
		`UPDATE duty_slots SET event_date='2099-08-31T00:00:00Z' WHERE id=?`, isoSlot); err != nil {
		t.Fatalf("ISO-Timestamp setzen: %v", err)
	}
	createTimedDutySlot(t, db, dtID, seasonID, teamID, 0, "2099-09-01", "13:00", 1) // nach dem Bereich

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := exportServer(t, h)

	records := readCSV(t, testutil.Get(t,
		srv, "/api/duty-slots/export?from=2099-08-01&to=2099-08-31", token))
	if len(records) != 3 {
		t.Fatalf("erwartet Kopfzeile + 2 Datenzeilen (Grenzen inklusive), bekam %d", len(records))
	}
	if got := col(t, records, 1, "Datum"); got != "01.08.2099" {
		t.Errorf("erste Zeile: erwartet 01.08.2099, bekam %q", got)
	}
	if got := col(t, records, 2, "Datum"); got != "31.08.2099" {
		t.Errorf("ISO-Timestamp-Zeile: erwartet 31.08.2099, bekam %q", got)
	}
}

// TestExportSlots_UngueltigerZeitraum: beide Grenzen sind Pflicht und müssen in
// der reinen Datumsform kommen; ein verdrehter Bereich ist ebenfalls 400.
func TestExportSlots_UngueltigerZeitraum(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	srv := exportServer(t, h)

	for _, q := range []string{
		"",
		"?from=2099-01-01",
		"?to=2099-01-31",
		"?from=01.01.2099&to=31.01.2099",
		"?from=2099-02-01&to=2099-01-31",
	} {
		resp := testutil.Get(t, srv, "/api/duty-slots/export"+q, token)
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Query %q: erwartet 400, bekam %d", q, resp.StatusCode)
		}
	}
}

// TestExportSlots_403OhneVereinsfunktion: der Export liegt im Tier der
// Dienst-Slot-Pflege — ein Spieler kommt nicht daran.
func TestExportSlots_403OhneVereinsfunktion(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"spieler"})
	srv := exportServer(t, h)

	resp := testutil.Get(t, srv, "/api/duty-slots/export?from=2099-01-01&to=2099-01-31", token)
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("erwartet 403 für Spieler, bekam %d", resp.StatusCode)
	}
}
