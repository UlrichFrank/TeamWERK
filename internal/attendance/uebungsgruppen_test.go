package attendance_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestAttendanceStats_UebungsgruppeTauchtNichtAuf hält die Paketgrenze fest:
// Anwesenheit **erfassen** liegt in `internal/trainings` und folgt `kader_id`
// (gilt also für Übungsgruppen), Anwesenheit **auswerten** liegt hier und folgt
// `team_id` — bei einer Übungsgruppe NULL, der JOIN `ts.team_id = k.team_id`
// matcht nie.
//
// Der Test misst deshalb einen Zähler, nicht eine Fehlermeldung: dasselbe
// Mitglied ist in beiden Gefäßen, war bei beiden Terminen anwesend, und die
// Team-Statistik darf trotzdem nur den Mannschaftstermin kennen. Ein
// „Reparatur"-Backfill von `training_sessions.team_id` würde hier auf 2
// springen — genau die Regression, gegen die dieser Test steht.
func TestAttendanceStats_UebungsgruppeTauchtNichtAuf(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")

	trainerUserID, kaderID := makeTrainer(t, db, teamID, seasonID)

	playerUserID := testutil.CreateUser(t, db, "standard")
	playerMemberID := testutil.CreateMember(t, db, playerUserID)
	testutil.AddKaderMember(t, db, kaderID, playerMemberID)

	// Mannschaftstermin — zählt.
	teamSessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, pastDate1)
	markTracked(t, db, teamSessionID)
	markPresent(t, db, teamSessionID, playerMemberID)

	// Übungsgruppentermin desselben Mitglieds, ebenfalls anwesend — zählt nicht.
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	testutil.AddKaderMember(t, db, groupID, playerMemberID)
	groupSessionID := createPracticeSessionForStats(t, db, groupID, seasonID, pastDate2)
	markTracked(t, db, groupSessionID)
	markPresent(t, db, groupSessionID, playerMemberID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{clubFnTrainr})

	res := testutil.Get(t, srv,
		fmt.Sprintf("/api/teams/%d/attendance-stats?season=%d", teamID, seasonID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekommen %d", res.StatusCode)
	}

	var body struct {
		RegularMembers []struct {
			MemberID        int `json:"member_id"`
			TrainingPresent int `json:"training_present"`
		} `json:"regular_members"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var found bool
	for _, m := range body.RegularMembers {
		if m.MemberID != playerMemberID {
			continue
		}
		found = true
		if m.TrainingPresent != 1 {
			t.Errorf("training_present = %d, erwartet 1 — der Übungsgruppen-Termin darf nicht mitzählen",
				m.TrainingPresent)
		}
	}
	if !found {
		t.Fatalf("Mitglied %d fehlt in der Team-Statistik: %+v", playerMemberID, body.RegularMembers)
	}
}

// createPracticeSessionForStats legt einen Termin einer Übungsgruppe an:
// kader_id gesetzt, team_id NULL.
func createPracticeSessionForStats(t *testing.T, db *sql.DB, groupID, seasonID int, date string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO training_sessions (kader_id, season_id, date, start_time, end_time, title)
		 VALUES (?, ?, ?, '18:00', '20:00', 'Torwarttraining')`, groupID, seasonID, date)
	if err != nil {
		t.Fatalf("createPracticeSessionForStats: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func markTracked(t *testing.T, db *sql.DB, sessionID int) {
	t.Helper()
	if _, err := db.Exec(`UPDATE training_sessions SET attendance_tracked = 1 WHERE id = ?`, sessionID); err != nil {
		t.Fatalf("markTracked: %v", err)
	}
}

func markPresent(t *testing.T, db *sql.DB, sessionID, memberID int) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO training_attendances (training_id, member_id, present) VALUES (?, ?, 1)`,
		sessionID, memberID); err != nil {
		t.Fatalf("markPresent: %v", err)
	}
}
