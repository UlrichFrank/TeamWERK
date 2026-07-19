package trainings

import (
	"context"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Testet die Ableitung „greift eine Serien-Abmeldung für Session×Member?" als
// reinen Lookup (serien-abmeldung-Spec): Fensterränder, offene NULL-Grenzen,
// Einzeltermine ohne Serie und harmlose Überlappungen.
func TestSessionUnavailabilityDerivation(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminUser := testutil.CreateUser(t, db, "admin")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminUser)
	member := testutil.CreateMember(t, db, 0)

	// Sessions der Serie an verschiedenen Daten.
	sBefore := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, "2026-01-01")
	sInside := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, "2026-03-15")
	sAfter := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, "2026-06-01")
	// Einzeltermin ohne Serie (series_id NULL) — darf nie betroffen sein.
	sStandalone := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-03-15")

	// Befristete Abmeldung [2026-02-01, 2026-04-30].
	testutil.CreateSeriesUnavailability(t, db, member, seriesID, "2026-02-01", "2026-04-30", "A-Jugend", adminUser)

	ctx := context.Background()
	applies := func(sessionID int) bool {
		ok, _, err := sessionUnavailabilityForMember(ctx, db, sessionID, member)
		if err != nil {
			t.Fatalf("sessionUnavailabilityForMember: %v", err)
		}
		return ok
	}

	if applies(sBefore) {
		t.Error("session before start_date must not be affected")
	}
	if !applies(sInside) {
		t.Error("session inside window must be affected")
	}
	if applies(sAfter) {
		t.Error("session after end_date must not be affected")
	}
	if applies(sStandalone) {
		t.Error("standalone session (series_id NULL) must never be affected")
	}

	// Batch-Variante liefert Reason + permanent=false (end_date gesetzt).
	m, err := unavailableMembersForSession(ctx, db, sInside)
	if err != nil {
		t.Fatalf("unavailableMembersForSession: %v", err)
	}
	info, ok := m[member]
	if !ok {
		t.Fatal("member expected in batch result for inside session")
	}
	if info.Reason != "A-Jugend" {
		t.Errorf("reason = %q, want A-Jugend", info.Reason)
	}
	if info.Permanent {
		t.Error("permanent must be false when end_date is set")
	}
}

// NULL-Grenzen: start NULL = ab Serien-Beginn, end NULL = permanent.
func TestSessionUnavailabilityOpenBounds(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminUser := testutil.CreateUser(t, db, "admin")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminUser)

	early := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, "2026-01-01")
	late := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, "2026-12-31")
	ctx := context.Background()

	// Permanent (beide NULL) → jede Session betroffen, permanent=true.
	permMember := testutil.CreateMember(t, db, 0)
	testutil.CreateSeriesUnavailability(t, db, permMember, seriesID, "", "", "permanent", adminUser)
	m, _ := unavailableMembersForSession(ctx, db, early)
	if info, ok := m[permMember]; !ok || !info.Permanent {
		t.Errorf("permanent member expected with permanent=true, got %+v ok=%v", info, ok)
	}

	// start NULL, end gesetzt (bis 2026-06-30): early betroffen, late nicht.
	untilMember := testutil.CreateMember(t, db, 0)
	testutil.CreateSeriesUnavailability(t, db, untilMember, seriesID, "", "2026-06-30", "", adminUser)
	if ok, _, _ := sessionUnavailabilityForMember(ctx, db, early, untilMember); !ok {
		t.Error("start-NULL: early session must be affected")
	}
	if ok, _, _ := sessionUnavailabilityForMember(ctx, db, late, untilMember); ok {
		t.Error("start-NULL with end 2026-06-30: late session must not be affected")
	}
}

// Überlappende Abmeldungen (gleiche Serie/Member, verschiedene start_date, beide
// decken das Datum) sind harmlos: genau ein Treffer, kein Fehler.
func TestSessionUnavailabilityOverlapHarmless(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	adminUser := testutil.CreateUser(t, db, "admin")
	seriesID := testutil.CreateTrainingSeries(t, db, teamID, seasonID, adminUser)
	member := testutil.CreateMember(t, db, 0)
	session := testutil.CreateTrainingSessionForSeries(t, db, teamID, seasonID, seriesID, "2026-03-15")

	testutil.CreateSeriesUnavailability(t, db, member, seriesID, "2026-01-01", "2026-06-30", "eins", adminUser)
	testutil.CreateSeriesUnavailability(t, db, member, seriesID, "2026-02-01", "", "zwei", adminUser)

	ctx := context.Background()
	m, err := unavailableMembersForSession(ctx, db, session)
	if err != nil {
		t.Fatalf("overlap: %v", err)
	}
	if _, ok := m[member]; !ok {
		t.Fatal("overlapping unavailabilities must still mark the member")
	}
	if len(m) != 1 {
		t.Errorf("expected exactly one member entry, got %d", len(m))
	}
	// Permanent-bevorzugte Zeile gewinnt (die zweite hat end_date NULL).
	if !m[member].Permanent {
		t.Error("permanent-preferred row should win when one overlapping row is permanent")
	}
}
