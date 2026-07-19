package attendance_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Eine serien-abgemeldete Session fällt aus allen drei Säulen; der Ausschluss
// dominiert eine parallele entschuldigte Absage und erscheint in der
// Detail-Liste als Kategorie "unavailable".
func TestGetMemberStats_UnavailableExcluded_DominatesExcused(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, kaderID := makeTrainer(t, db, teamID, seasonID)
	user := testutil.CreateUser(t, db, "standard")
	member := testutil.CreateMember(t, db, user)
	addKaderMember(t, db, kaderID, member)
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, trainerUserID)

	// Session A (2026-04-15): entschuldigt (member_absences 2026-04..) UND per
	// Serien-Abmeldung [2026-04-01,2026-04-30] abgemeldet → muss ausgeschlossen sein.
	sessA := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, pastDate1)
	recordExcusedTrainingResponse(t, db, sessA, member, trainerUserID)
	testutil.CreateSeriesUnavailability(t, db, member, seriesID, "2026-04-01", "2026-04-30", "A-Jugend", trainerUserID)

	// Session B (2026-05-20): anwesend, keine Abmeldung → zählt als present.
	sessB := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, pastDate2)
	recordTrainingAttendance(t, db, sessB, member, 1)

	srv := testServer(t, db)
	token := testutil.Token(t, user, "standard", []string{"spieler"})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/members/%d/attendance-stats", member), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)

	counts := body["counts"].(map[string]any)
	if int(counts["training_present"].(float64)) != 1 {
		t.Errorf("training_present=1 expected, got %v", counts["training_present"])
	}
	if int(counts["training_excused"].(float64)) != 0 {
		t.Errorf("training_excused must be 0 (unavailable dominates excused), got %v", counts["training_excused"])
	}
	if int(counts["training_missed"].(float64)) != 0 {
		t.Errorf("training_missed=0 expected, got %v", counts["training_missed"])
	}

	events := body["events"].([]any)
	var unavailReason string
	var sawUnavailable, sawPresent bool
	for _, e := range events {
		ev := e.(map[string]any)
		switch ev["category"].(string) {
		case "unavailable":
			sawUnavailable = true
			if r, ok := ev["reason"].(string); ok {
				unavailReason = r
			}
		case "present":
			sawPresent = true
		}
	}
	if !sawUnavailable {
		t.Error("session A must appear with category 'unavailable'")
	}
	if unavailReason != "A-Jugend" {
		t.Errorf("unavailable event reason = %q, want A-Jugend", unavailReason)
	}
	if !sawPresent {
		t.Error("session B must still appear as 'present'")
	}
}

// Abgemeldete Sessions liegen nicht im Team-Nenner: die Pro-Spieler-Counts
// klammern sie aus (Team-Quote = Ø der Pro-Spieler-Quoten).
func TestGetTeamStats_UnavailableNotInDenominator(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	trainerUserID, kaderID := makeTrainer(t, db, teamID, seasonID)
	player := testutil.CreateMember(t, db, 0)
	addKaderMember(t, db, kaderID, player)
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, trainerUserID)

	// Einzige Session ist abgemeldet — obwohl eine Anwesenheit erfasst wurde,
	// darf sie in keiner Säule zählen.
	sess := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, pastDate1)
	recordTrainingAttendance(t, db, sess, player, 1)
	testutil.CreateSeriesUnavailability(t, db, player, seriesID, "", "", "", trainerUserID)

	srv := testServer(t, db)
	token := testutil.Token(t, trainerUserID, "standard", []string{clubFnTrainr})
	res := testutil.Get(t, srv, fmt.Sprintf("/api/teams/%d/attendance-stats", teamID), token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	json.NewDecoder(res.Body).Decode(&body)
	regular := body["regular_members"].([]any)
	if len(regular) != 1 {
		t.Fatalf("expected 1 regular member, got %d", len(regular))
	}
	m := regular[0].(map[string]any)
	if int(m["training_present"].(float64)) != 0 {
		t.Errorf("unavailable session must not count as present, got %v", m["training_present"])
	}
	if int(m["training_missed"].(float64)) != 0 || int(m["training_excused"].(float64)) != 0 {
		t.Errorf("unavailable session must not count in any pillar, got missed=%v excused=%v", m["training_missed"], m["training_excused"])
	}
}
