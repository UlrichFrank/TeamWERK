package trainings_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

// createPracticeSession legt einen Termin an, dessen Besitzer eine Übungsgruppe
// ist: kader_id gesetzt, team_id NULL (proposal.md — Invariante 1).
func createPracticeSession(t *testing.T, db *sql.DB, groupID, seasonID int, date string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO training_sessions (kader_id, season_id, date, start_time, end_time, title)
		 VALUES (?, ?, ?, '18:00', '20:00', 'Torwarttraining')`, groupID, seasonID, date)
	if err != nil {
		t.Fatalf("createPracticeSession: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// listSessionIDs liefert die IDs, die ListSessions für den Token ausgibt.
func listSessionIDs(t *testing.T, srv interface{ Close() }, token string, do func(string) *http.Response) []int {
	t.Helper()
	res := do(token)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekommen %d", res.StatusCode)
	}
	var body struct {
		Items []struct {
			ID     int `json:"id"`
			TeamID int `json:"team_id"`
		} `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	ids := make([]int, 0, len(body.Items))
	for _, it := range body.Items {
		ids = append(ids, it.ID)
	}
	return ids
}

func contains(ids []int, id int) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// ── Anlage ────────────────────────────────────────────────────────────────────

// TestCreateTraining_FuerUebungsgruppe: kader_id ist der Besitzer, team_id die
// Projektion — bei einer Übungsgruppe NULL. Diese NULL trägt den gesamten
// Ausschluss und darf nie „repariert" werden.
func TestCreateTraining_FuerUebungsgruppe(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	uid := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, uid, "standard", []string{"sportliche_leitung"})

	res := testutil.Post(t, srv, "/api/training-sessions", token, map[string]any{
		"kader_id":   groupID,
		"title":      "Torwarttraining",
		"date":       "2026-03-01",
		"start_time": "18:00",
		"end_time":   "19:30",
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekommen %d", res.StatusCode)
	}
	var body struct {
		ID int `json:"id"`
	}
	json.NewDecoder(res.Body).Decode(&body)

	var kaderID, sessionSeason int
	var teamID sql.NullInt64
	err := db.QueryRow(`SELECT kader_id, team_id, season_id FROM training_sessions WHERE id=?`, body.ID).
		Scan(&kaderID, &teamID, &sessionSeason)
	if err != nil {
		t.Fatalf("Termin lesen: %v", err)
	}
	if kaderID != groupID {
		t.Errorf("kader_id = %d, erwartet %d", kaderID, groupID)
	}
	if teamID.Valid {
		t.Errorf("team_id = %d, erwartet NULL", teamID.Int64)
	}
	if sessionSeason != seasonID {
		t.Errorf("season_id = %d, erwartet %d (aus dem Kader abgeleitet)", sessionSeason, seasonID)
	}
}

// ── Sichtbarkeit ──────────────────────────────────────────────────────────────

// TestListTrainings_MitgliedSiehtUebungsgruppenTermin: die Sichtbarkeit hängt am
// Kader des Termins, nicht am Team — sonst sähe ein Mitglied einer
// Übungsgruppe seinen eigenen Termin nicht.
func TestListTrainings_MitgliedSiehtUebungsgruppenTermin(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	sessionID := createPracticeSession(t, db, groupID, seasonID, "2026-06-15")

	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	addKaderMember(t, db, groupID, mid)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, uid, "standard", []string{"spieler"})

	ids := listSessionIDs(t, srv, token, func(tok string) *http.Response {
		return testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-12-31", tok)
	})
	if !contains(ids, sessionID) {
		t.Fatalf("Mitglied sieht seinen Übungsgruppen-Termin nicht (bekommen: %v)", ids)
	}
}

// TestListTrainings_FremderSiehtUebungsgruppenTerminNicht: wer nicht in der
// Gruppe ist, sieht den Termin nicht — die Umstellung auf kader_id darf die
// Sichtbarkeit nicht aufweiten.
func TestListTrainings_FremderSiehtUebungsgruppenTerminNicht(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	sessionID := createPracticeSession(t, db, groupID, seasonID, "2026-06-15")

	// Der Fremde ist Spieler einer regulären Mannschaft, nur nicht in der Gruppe.
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	addKaderMember(t, db, kaderID, mid)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, uid, "standard", []string{"spieler"})

	ids := listSessionIDs(t, srv, token, func(tok string) *http.Response {
		return testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-12-31", tok)
	})
	if contains(ids, sessionID) {
		t.Fatalf("Nicht-Mitglied sieht den Übungsgruppen-Termin (bekommen: %v)", ids)
	}
}

// TestListTrainings_ElternSehenUebungsgruppenTermin: Eltern erreichen den Termin
// über family_links — derselbe Pfad wie bei der Mannschaftsvariante.
func TestListTrainings_ElternSehenUebungsgruppenTermin(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	sessionID := createPracticeSession(t, db, groupID, seasonID, "2026-06-15")

	childUID := testutil.CreateUser(t, db, "standard")
	childMID := testutil.CreateMember(t, db, childUID)
	addKaderMember(t, db, groupID, childMID)

	parentUID := testutil.CreateUser(t, db, "standard")
	testutil.AddFamilyLink(t, db, parentUID, childMID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.TokenWithIsParent(t, parentUID, "standard", nil, true)

	ids := listSessionIDs(t, srv, token, func(tok string) *http.Response {
		return testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-12-31", tok)
	})
	if !contains(ids, sessionID) {
		t.Fatalf("Elternteil sieht den Übungsgruppen-Termin des Kindes nicht (bekommen: %v)", ids)
	}
}

// TestListTrainings_TrainerSiehtUebungsgruppenTermin: Trainer der Gruppe stehen
// in kader_trainers — dieselbe Tabelle wie bei einer Mannschaft.
func TestListTrainings_TrainerSiehtUebungsgruppenTermin(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	sessionID := createPracticeSession(t, db, groupID, seasonID, "2026-06-15")

	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	testutil.AddKaderTrainer(t, db, groupID, mid)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, uid, "standard", []string{"trainer"})

	ids := listSessionIDs(t, srv, token, func(tok string) *http.Response {
		return testutil.Get(t, srv, "/api/training-sessions?from=2026-01-01&to=2026-12-31", tok)
	})
	if !contains(ids, sessionID) {
		t.Fatalf("Trainer sieht den Termin seiner Übungsgruppe nicht (bekommen: %v)", ids)
	}
}

// ── RSVP ──────────────────────────────────────────────────────────────────────

// TestRsvpUebungsgruppe_Erfolg: Zu-/Absagen funktionieren für beide Varianten
// über denselben Pfad.
func TestRsvpUebungsgruppe_Erfolg(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	sessionID := createPracticeSession(t, db, groupID, seasonID, "2026-06-15")

	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	addKaderMember(t, db, groupID, mid)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	// Vor dem RSVP-Cutoff: sonst antwortet Respond mit 422, unabhängig von der
	// Kader-Variante.
	h.SetNow(fixedNow(berlinTime(t, "2006-01-02 15:04", "2026-06-14 12:00")))
	srv := testServer(t, h)
	token := testutil.Token(t, uid, "standard", []string{"spieler"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/respond", sessionID), token,
		map[string]any{"status": "confirmed"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekommen %d", res.StatusCode)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM training_responses WHERE training_id=? AND member_id=?`,
		sessionID, mid).Scan(&status); err != nil {
		t.Fatalf("keine RSVP-Zeile: %v", err)
	}
	if status != "confirmed" {
		t.Errorf("status = %q, erwartet confirmed", status)
	}
}

// ── Anwesenheit erfassen ──────────────────────────────────────────────────────

// TestAttendanceUebungsgruppe_TrainerDarf: Anwesenheit ERFASSEN liegt in
// internal/trainings und folgt kader_id — anders als das AUSWERTEN in
// internal/attendance, das über team_id läuft und Übungsgruppen deshalb nie
// sieht (proposal.md — „Die Trennung fällt exakt auf die Paketgrenze").
func TestAttendanceUebungsgruppe_TrainerDarf(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	// In der Vergangenheit: SaveAttendances lehnt künftige Termine mit 422 ab.
	sessionID := createPracticeSession(t, db, groupID, seasonID, "2025-10-01")

	playerUID := testutil.CreateUser(t, db, "standard")
	playerMID := testutil.CreateMember(t, db, playerUID)
	addKaderMember(t, db, groupID, playerMID)

	trainerUID := testutil.CreateUser(t, db, "standard")
	trainerMID := testutil.CreateMember(t, db, trainerUID)
	testutil.AddKaderTrainer(t, db, groupID, trainerMID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	token := testutil.Token(t, trainerUID, "standard", []string{"trainer"})

	res := testutil.Post(t, srv, fmt.Sprintf("/api/training-sessions/%d/attendances", sessionID), token,
		[]map[string]any{{"member_id": playerMID, "present": true}})
	res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 200/204, bekommen %d", res.StatusCode)
	}

	var present int
	if err := db.QueryRow(`SELECT present FROM training_attendances WHERE training_id=? AND member_id=?`,
		sessionID, playerMID).Scan(&present); err != nil {
		t.Fatalf("keine Anwesenheits-Zeile: %v", err)
	}
	if present != 1 {
		t.Errorf("present = %d, erwartet 1", present)
	}
}
