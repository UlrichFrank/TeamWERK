package duties_test

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// exportTestServer wires the export route under the same Vorstand+Trainer+sL
// gate it has in router.go (internal/app/router.go).
func exportTestServer(t *testing.T, h *duties.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Get("/api/duty-slots/export", h.ExportSlots)
		})
	})
}

// TestExportSlots_GameBoundRow prüft, dass ein spielgebundener Slot mit allen
// Zeiten (Start aus event_time, Ende aus event_time+hours_value) sowie
// Ausrichter und Tageskonstellation (Mehrfachspieltag, Heimspiel am Folgetag)
// korrekt als CSV-Zeile herauskommt.
func TestExportSlots_GameBoundRow(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Slots-Team")
	dutyTypeID := createDutyType(t, db, "Kuchen", 1.5)

	// Ausrichter: expliziter Eintrag für den Spieltag statt des Defaults.
	ausrichterRes, err := db.Exec(`INSERT INTO ausrichter (name, aktiv, is_default) VALUES ('Gastverein', 1, 0)`)
	if err != nil {
		t.Fatalf("insert ausrichter: %v", err)
	}
	ausrichterID, _ := ausrichterRes.LastInsertId()
	if _, err := db.Exec(`INSERT INTO spieltag_ausrichter (date, season_id, ausrichter_id) VALUES (?, ?, ?)`,
		"2026-03-08", seasonID, ausrichterID); err != nil {
		t.Fatalf("insert spieltag_ausrichter: %v", err)
	}

	// Heimspiel am Haupttag, an dem der Slot hängt.
	gameRes, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type) VALUES (?, 'Gegner A', ?, '10:00', 1, 'heim')`,
		seasonID, "2026-03-08")
	if err != nil {
		t.Fatalf("insert game: %v", err)
	}
	gameID, _ := gameRes.LastInsertId()
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamID); err != nil {
		t.Fatalf("insert game_teams: %v", err)
	}

	// Zweites Heimspiel am selben Tag → Mehrfachspieltag.
	if _, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type) VALUES (?, 'Gegner B', ?, '12:00', 1, 'heim')`,
		seasonID, "2026-03-08"); err != nil {
		t.Fatalf("insert second game: %v", err)
	}
	// Heimspiel am Folgetag.
	if _, err := db.Exec(
		`INSERT INTO games (season_id, opponent, date, time, is_home, event_type) VALUES (?, 'Gegner C', '2026-03-09', '10:00', 1, 'heim')`,
		seasonID); err != nil {
		t.Fatalf("insert next-day game: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, event_time, duty_type_id, slots_total, slots_filled, season_id, game_id, hours_value)
		 VALUES ('Kuchendienst', ?, '08:00', ?, 2, 1, ?, ?, 1.5)`,
		"2026-03-08", dutyTypeID, seasonID, gameID); err != nil {
		t.Fatalf("insert duty_slots: %v", err)
	}

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := exportTestServer(t, h)

	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Get(t, srv, "/api/duty-slots/export?date_from=2026-03-01&date_to=2026-03-31", token)
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/csv" {
		t.Errorf("expected Content-Type text/csv, got %q", ct)
	}

	records, err := csv.NewReader(res.Body).ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected header + 1 row, got %d rows", len(records))
	}
	header := records[0]
	row := records[1]
	col := func(name string) string {
		for i, h := range header {
			if h == name {
				return row[i]
			}
		}
		t.Fatalf("column %q not found in header %v", name, header)
		return ""
	}

	cases := map[string]string{
		"Datum":                      "2026-03-08",
		"Start":                      "08:00",
		"Ende":                       "09:30",
		"Diensttyp":                  "Kuchen",
		"Team":                       "Slots-Team",
		"Termin":                     "Kuchendienst",
		"Gegner":                     "Gegner A",
		"Terminart":                  "Heimspiel",
		"Plätze besetzt":             "1",
		"Plätze gesamt":              "2",
		"Ausrichter":                 "Gastverein",
		"Spiele am Tag":              "2",
		"Heimspiel Vortag":           "Nein",
		"Heimspiel Folgetag":         "Ja",
		"Verhalten Mehrfachspieltag": "normal",
		"Verhalten Nachbartag":       "normal",
	}
	for name, want := range cases {
		if got := col(name); got != want {
			t.Errorf("column %q: expected %q, got %q", name, want, got)
		}
	}
}

// TestExportSlots_Forbidden prüft, dass ein Spieler (kein Vorstand/Trainer/sL)
// den Export nicht aufrufen darf.
func TestExportSlots_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := exportTestServer(t, h)

	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, "/api/duty-slots/export", token)
	res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", res.StatusCode)
	}
}
