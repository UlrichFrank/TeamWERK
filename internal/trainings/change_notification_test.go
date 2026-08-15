package trainings_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/trainings"
)

// ── Änderungs-Benachrichtigungen (openspec/changes/aenderungs-benachrichtigung) ──
//
// Invarianten:
//   - keine Änderungsmeldung ohne Titel + Zeitpunkt (Datum UND Uhrzeit) + Aktor,
//   - der alte Zeitpunkt steht genau dann im Text, wenn verschoben wurde,
//   - die Serie meldet ihren Zeitraum und führt auf /termine (kein einzelner Termin),
//   - kein Erfolg, keine Meldung.

// ── PUT /api/training-sessions/{id} ──

// Der Kernfall: eine Änderung ohne Verschiebung nennt Titel, Datum, Startzeit
// und Aktor — und nichts über einen früheren Zeitpunkt.
func TestUpdateSession_AenderungNenntTitelZeitpunktUndAktor(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	adminID := testutil.CreateUser(t, db, "admin")
	setUserName(t, db, adminID, "Tim", "Meier")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodPut, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, adminID, "admin", nil), updateSessionBody("active", ""))
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d: %+v", len(got), got)
	}
	if got[0].title != "Training geändert" {
		t.Errorf("title = %q, want %q", got[0].title, "Training geändert")
	}
	mustContainAll(t, got[0].body, "Krafteinheit", "15.03.2026", "18:00", "Tim Meier")
	if strings.Contains(got[0].body, "vorher") {
		t.Errorf("ohne Verschiebung darf keine Vorher-Angabe im Body stehen: %q", got[0].body)
	}
	if want := "/termine?focus=training-" + strconv.Itoa(sessionID); got[0].url != want {
		t.Errorf("url = %q, want %q", got[0].url, want)
	}
}

// Verschiebung: der Empfänger kennt die Einheit unter ihrem alten Zeitpunkt.
func TestUpdateSession_VerschiebungNenntAltenUndNeuenZeitpunkt(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	sessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")
	adminID := testutil.CreateUser(t, db, "admin")
	setUserName(t, db, adminID, "Tim", "Meier")

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	body := updateSessionBody("active", "")
	body["date"] = "2026-03-22"
	body["start_time"] = "19:30"
	res := testutil.Do(t, srv, http.MethodPut, "/api/training-sessions/"+strconv.Itoa(sessionID),
		testutil.Token(t, adminID, "admin", nil), body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d, want 204", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d: %+v", len(got), got)
	}
	mustContainAll(t, got[0].body, "22.03.2026", "19:30", "vorher 15.03.2026, 18:00 Uhr")
}

// ── PUT /api/training-series/{id} ──

// updateSeriesBody baut den Standard-Request-Body für UpdateSeries. Die Fixture
// legt die Serie mittwochs (day_of_week=2) um 18:00 an.
func updateSeriesBody(name string, dayOfWeek int, startTime string) map[string]any {
	return map[string]any{
		"name":        name,
		"day_of_week": dayOfWeek,
		"start_time":  startTime,
		"end_time":    "20:00",
		"valid_from":  "2025-10-01",
		"valid_until": "2026-06-30",
		"scope":       "all",
	}
}

// Ein PUT auf die Serie löscht und erzeugt alle Einheiten neu — bisher erfuhr
// das Team davon nur, wenn es die App gerade offen hatte.
func TestUpdateSeries_MeldetZeitraumUndAktor(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminID := testutil.CreateUser(t, db, "admin")
	setUserName(t, db, adminID, "Tim", "Meier")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	// Wochentag und Startzeit unverändert (mittwochs 18:00) — nur der Name wandert.
	res := testutil.Do(t, srv, http.MethodPut, "/api/training-series/"+strconv.Itoa(seriesID),
		testutil.Token(t, adminID, "admin", nil), updateSeriesBody("Athletik Mittwoch", 2, "18:00"))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d: %+v", len(got), got)
	}
	if got[0].title != "Trainingsserie geändert" {
		t.Errorf("title = %q, want %q", got[0].title, "Trainingsserie geändert")
	}
	if got[0].category != "trainings" {
		t.Errorf("category = %q, want trainings", got[0].category)
	}
	if got[0].url != "/termine" {
		t.Errorf("url = %q, want /termine (eine Serie ist kein einzelner Termin)", got[0].url)
	}
	mustContainAll(t, got[0].body, "Athletik Mittwoch", "01.10.2025", "30.06.2026", "Tim Meier")
	if strings.Contains(got[0].body, "vorher") {
		t.Errorf("unveränderter Rhythmus darf keine Vorher-Angabe erzeugen: %q", got[0].body)
	}
}

// Wandert der Rhythmus, muss der alte im Text stehen — sonst sucht das Team
// weiter am Mittwoch.
func TestUpdateSeries_RhythmusVerschiebungNenntAltenRhythmus(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminID := testutil.CreateUser(t, db, "admin")
	setUserName(t, db, adminID, "Tim", "Meier")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	// mittwochs 18:00 → dienstags 17:00
	res := testutil.Do(t, srv, http.MethodPut, "/api/training-series/"+strconv.Itoa(seriesID),
		testutil.Token(t, adminID, "admin", nil), updateSeriesBody("Test Series", 1, "17:00"))
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}

	got := drainTraining(sent)
	if len(got) != 1 {
		t.Fatalf("erwartete genau 1 Benachrichtigung, bekam %d: %+v", len(got), got)
	}
	mustContainAll(t, got[0].body, "vorher mittwochs 18:00 Uhr")
}

// Ohne erfolgreiche Änderung darf keine Meldung entstehen.
func TestUpdateSeries_FremdesTeamKeineBenachrichtigung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	otherTeam := testutil.CreateTeam(t, db, "Team B")
	ownerID := testutil.CreateUser(t, db, "admin")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, ownerID)
	trainerU := makeTrainer(t, db, otherTeam, seasonID)

	h := trainings.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)
	sent := captureTrainingNotifications(t)

	res := testutil.Do(t, srv, http.MethodPut, "/api/training-series/"+strconv.Itoa(seriesID),
		testutil.Token(t, trainerU, "standard", []string{"trainer"}), updateSeriesBody("Geklaut", 1, "17:00"))
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d, want 403", res.StatusCode)
	}
	if got := drainTraining(sent); len(got) != 0 {
		t.Fatalf("erwartete keine Benachrichtigung, bekam %+v", got)
	}
}
