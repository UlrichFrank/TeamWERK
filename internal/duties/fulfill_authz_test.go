package duties_test

import (
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestFulfill_Spieler403: das Abhaken eines Dienstes ist eine Trainer-/
// Leitungs-Aktion (policy.CanFulfillAssignment). Der Handler prüft das selbst,
// nicht nur das Router-Tier — ohne die Prüfung hakte jeder Eingeloggte fremde
// Dienste ab, sobald die Route anders verdrahtet wird.
func TestFulfill_Spieler403(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")
	ownerID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, ownerID, "assigned")
	var assignmentID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=?`, slotID).Scan(&assignmentID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	playerID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, playerID, "standard", []string{"spieler"})

	r1 := testutil.Post(t, srv, "/api/duty-assignments/"+itoa(assignmentID)+"/fulfill", token, nil)
	r1.Body.Close()
	if r1.StatusCode != http.StatusForbidden {
		t.Fatalf("fulfill: expected 403 for spieler, got %d", r1.StatusCode)
	}

	r2 := testutil.Post(t, srv, "/api/duty-assignments/"+itoa(assignmentID)+"/cash-substitute",
		token, map[string]float64{"amount": 15})
	r2.Body.Close()
	if r2.StatusCode != http.StatusForbidden {
		t.Fatalf("cash-substitute: expected 403 for spieler, got %d", r2.StatusCode)
	}

	var status string
	db.QueryRow(`SELECT status FROM duty_assignments WHERE id=?`, assignmentID).Scan(&status)
	if status != "assigned" {
		t.Errorf("expected status unchanged ('assigned'), got %q", status)
	}
}

// TestFulfill_UnbekannteId404: eine nicht existierende Zuweisung ist kein
// stiller Erfolg mehr. Vorher antwortete die Route 204 und broadcastete,
// obwohl kein UPDATE eine Zeile traf.
func TestFulfill_UnbekannteId404(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)

	r1 := testutil.Post(t, srv, "/api/duty-assignments/99999/fulfill", token, nil)
	r1.Body.Close()
	if r1.StatusCode != http.StatusNotFound {
		t.Fatalf("fulfill: expected 404 for unknown id, got %d", r1.StatusCode)
	}

	r2 := testutil.Post(t, srv, "/api/duty-assignments/99999/cash-substitute", token,
		map[string]float64{"amount": 15})
	r2.Body.Close()
	if r2.StatusCode != http.StatusNotFound {
		t.Fatalf("cash-substitute: expected 404 for unknown id, got %d", r2.StatusCode)
	}
}

// TestCashSubstitute_UngueltigerBetrag400: ein Ablösebetrag <= 0 ist keine
// Ablösung — vorher wurde er klaglos gebucht und stand im Kassen-Export.
func TestCashSubstitute_UngueltigerBetrag400(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")
	ownerID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, ownerID, "assigned")
	var assignmentID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=?`, slotID).Scan(&assignmentID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	adminID := testutil.CreateUser(t, db, "admin")
	token := testutil.Token(t, adminID, "admin", nil)
	path := "/api/duty-assignments/" + itoa(assignmentID) + "/cash-substitute"

	for _, amount := range []float64{0, -5} {
		r := testutil.Post(t, srv, path, token, map[string]float64{"amount": amount})
		r.Body.Close()
		if r.StatusCode != http.StatusBadRequest {
			t.Fatalf("amount=%v: expected 400, got %d", amount, r.StatusCode)
		}
	}

	var status string
	db.QueryRow(`SELECT status FROM duty_assignments WHERE id=?`, assignmentID).Scan(&status)
	if status != "assigned" {
		t.Errorf("expected status unchanged ('assigned'), got %q", status)
	}
}

// TestFulfill_TrainerOK: die bisherigen Berechtigten (admin, trainer,
// sportliche_leitung — das Router-Tier) kommen unverändert durch.
func TestFulfill_TrainerOK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dutyTypeID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dutyTypeID, seasonID, teamID, 0, "2026-07-01")
	ownerID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, ownerID, "assigned")
	var assignmentID int
	db.QueryRow(`SELECT id FROM duty_assignments WHERE duty_slot_id=?`, slotID).Scan(&assignmentID)

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	trainerID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, trainerID, "standard", []string{"trainer"})

	r := testutil.Post(t, srv, "/api/duty-assignments/"+itoa(assignmentID)+"/fulfill", token, nil)
	r.Body.Close()
	if r.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for trainer, got %d", r.StatusCode)
	}

	var status string
	db.QueryRow(`SELECT status FROM duty_assignments WHERE id=?`, assignmentID).Scan(&status)
	if status != "fulfilled" {
		t.Errorf("expected status='fulfilled', got %q", status)
	}
}
