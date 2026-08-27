package duties_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Die Schreibseite wird direkt geprüft, nicht nur über die Sichtbarkeit: die Leseabfragen
// ignorieren ds.team_id bei gesetztem game_id inzwischen, ein versehentlich wieder
// geschriebenes Team fiele dort also gar nicht mehr auf (design.md, Risiken).
func TestCreateSlot_MitSpiel_SchreibtKeinTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")
	gameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-06-14")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	adminID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, duties.NewHandler(db, testutil.TestConfig(), hub.NewHub()))

	// Ein Client schickt trotzdem ein team_id mit — es muss verworfen werden, nicht 400.
	res := testutil.Post(t, srv, "/api/duty-slots", testutil.Token(t, adminID, "admin", nil), map[string]any{
		"event_name":   "Aufbau",
		"event_date":   "2026-06-14",
		"duty_type_id": dtID,
		"slots_total":  2,
		"team_id":      teamA,
		"season_id":    seasonID,
		"game_id":      gameID,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	var teamID sql.NullInt64
	if err := db.QueryRow(`SELECT team_id FROM duty_slots WHERE game_id=?`, gameID).Scan(&teamID); err != nil {
		t.Fatalf("read team_id: %v", err)
	}
	if teamID.Valid {
		t.Errorf("spielgebundener Slot soll team_id=NULL tragen, hat %d", teamID.Int64)
	}
}

// Gegenstück: ohne game_id bleibt team_id der Geltungsbereich und wird gespeichert.
func TestCreateSlot_OhneSpiel_BehaeltTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")
	dtID := createDutyType(t, db, "Vereinsfest", 2.0)
	adminID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, duties.NewHandler(db, testutil.TestConfig(), hub.NewHub()))

	res := testutil.Post(t, srv, "/api/duty-slots", testutil.Token(t, adminID, "admin", nil), map[string]any{
		"event_name":   "Vereinsfest",
		"event_date":   "2026-06-14",
		"duty_type_id": dtID,
		"slots_total":  2,
		"team_id":      teamA,
		"season_id":    seasonID,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.StatusCode)
	}

	var teamID sql.NullInt64
	if err := db.QueryRow(`SELECT team_id FROM duty_slots WHERE duty_type_id=?`, dtID).Scan(&teamID); err != nil {
		t.Fatalf("read team_id: %v", err)
	}
	if !teamID.Valid || int(teamID.Int64) != teamA {
		t.Errorf("Slot ohne Spiel soll team_id=%d behalten, hat valid=%v value=%d", teamA, teamID.Valid, teamID.Int64)
	}
}

// Bestandszeile aus der Zeit vor der Migration: game_id gesetzt UND team_id gesetzt.
// Die Sichtbarkeit muss trotzdem allen Teams des Termins folgen — das ist die Zusage,
// die die Migration von einer Voraussetzung zu reiner Hygiene macht (design.md Dec. 1).
func TestBoard_BestandsSlotMitTeamIdFolgtDenTeamsDesTermins(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")
	teamB := testutil.CreateTeam(t, db, "B-Jugend")
	gameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-06-14")
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamB); err != nil {
		t.Fatalf("link second team: %v", err)
	}

	// Spieler in Team B — im Bestands-Slot steht aber Team A.
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	addPlayerMembership(t, db, memberID, teamB, seasonID)

	dtID := createDutyType(t, db, "Aufbau", 2.0)
	createDutySlot(t, db, dtID, seasonID, teamA, gameID, "2026-06-14")

	srv := testServer(t, duties.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	res := testutil.Get(t, srv, "/api/duty-board", testutil.Token(t, userID, "standard", nil))
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var groups []struct {
		Slots []struct {
			ID int `json:"id"`
		} `json:"slots"`
	}
	json.NewDecoder(res.Body).Decode(&groups)
	total := 0
	for _, g := range groups {
		total += len(g.Slots)
	}
	if total != 1 {
		t.Errorf("Spieler aus Team B soll den Bestands-Slot (team_id=Team A) sehen, bekam %d Slots", total)
	}
}
