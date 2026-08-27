package duties_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// notifiedUsers reads the recipients of the duties notification from user_events —
// notify.Send schreibt den Event-Log als ersten Schritt, vor jedem Präferenzfilter,
// und ist damit die verlässliche Beobachtungsstelle für die Empfängermenge.
func notifiedUsers(t *testing.T, db *sql.DB) map[int]bool {
	t.Helper()
	rows, err := db.Query(`SELECT user_id FROM user_events WHERE category='duties'`)
	if err != nil {
		t.Fatalf("read user_events: %v", err)
	}
	defer rows.Close()
	got := map[int]bool{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan user_events: %v", err)
		}
		got[id] = true
	}
	return got
}

// addPlayer legt einen User mit Vereinsfunktion 'spieler' im Kader des Teams an.
func addPlayer(t *testing.T, db *sql.DB, teamID, seasonID int) int {
	t.Helper()
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	if _, err := db.Exec(`INSERT INTO member_club_functions (member_id, function) VALUES (?, 'spieler')`, memberID); err != nil {
		t.Fatalf("insert club function: %v", err)
	}
	addPlayerMembership(t, db, memberID, teamID, seasonID)
	return userID
}

// Ein Slot ohne team_id, aber mit game_id, adressiert die Teams des Termins — nicht
// den ganzen Verein. Das ist die Kehrseite des Sichtbarkeits-Fallbacks: die Dienstbörse
// zeigt den Slot allen beteiligten Teams, also muss die Benachrichtigung dieselbe
// Menge treffen und nicht mehr.
func TestCreateSlot_OhneTeamMitSpiel_BenachrichtigtNurDieTeamsDesTermins(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")
	teamB := testutil.CreateTeam(t, db, "B-Jugend")
	teamC := testutil.CreateTeam(t, db, "Unbeteiligte C-Jugend")

	playerA := addPlayer(t, db, teamA, seasonID)
	playerB := addPlayer(t, db, teamB, seasonID)
	playerC := addPlayer(t, db, teamC, seasonID)

	gameID := testutil.CreateGame(t, db, seasonID, teamA, "2026-06-14")
	if _, err := db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamB); err != nil {
		t.Fatalf("link second team: %v", err)
	}

	dtID := createDutyType(t, db, "Aufbau", 2.0)
	adminID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, duties.NewHandler(db, testutil.TestConfig(), hub.NewHub()))

	res := testutil.Post(t, srv, "/api/duty-slots", testutil.Token(t, adminID, "admin", nil), map[string]any{
		"event_name":   "Turnier A-/B-Jugend",
		"event_date":   "2026-06-14",
		"duty_type_id": dtID,
		"slots_total":  9,
		"team_id":      nil,
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
		t.Errorf("Slot soll team-los bleiben, hat aber team_id=%d", teamID.Int64)
	}

	got := notifiedUsers(t, db)
	if !got[playerA] || !got[playerB] {
		t.Errorf("Spieler beider beteiligten Teams erwartet (A=%d, B=%d), benachrichtigt: %v", playerA, playerB, got)
	}
	if got[playerC] {
		t.Errorf("Spieler des unbeteiligten Teams C (%d) darf nicht benachrichtigt werden: %v", playerC, got)
	}
}

// Gegenstück: mit gesetztem team_id bleibt die Benachrichtigung auf dieses Team beschränkt.
func TestCreateSlot_MitTeam_BenachrichtigtNurDiesesTeam(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "A-Jugend")
	teamB := testutil.CreateTeam(t, db, "B-Jugend")

	playerA := addPlayer(t, db, teamA, seasonID)
	playerB := addPlayer(t, db, teamB, seasonID)

	dtID := createDutyType(t, db, "Aufbau", 2.0)
	adminID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, duties.NewHandler(db, testutil.TestConfig(), hub.NewHub()))

	res := testutil.Post(t, srv, "/api/duty-slots", testutil.Token(t, adminID, "admin", nil), map[string]any{
		"event_name":   "Heimspiel Aufbau",
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

	got := notifiedUsers(t, db)
	if !got[playerA] {
		t.Errorf("Spieler des Teams (%d) fehlt in der Benachrichtigung: %v", playerA, got)
	}
	if got[playerB] {
		t.Errorf("Spieler eines fremden Teams (%d) darf nicht benachrichtigt werden: %v", playerB, got)
	}
}
