package matchreports

import (
	"testing"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// consentMissing listet Team-Mitglieder anhand von foto_veroeffentlichung=0 —
// unabhängig von photo_visible.
func TestConsentMissing_NutztFotoVeroeffentlichung(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameID := testutil.CreateGame(t, db, seasonID, teamID, "2026-03-01")

	// Ohne Einwilligung (foto=0), aber intern sichtbar (photo_visible=1) → MUSS gelistet werden.
	mNoConsent := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(
		`UPDATE members SET first_name='Ohne', last_name='Freigabe', foto_veroeffentlichung=0, photo_visible=1 WHERE id=?`,
		mNoConsent); err != nil {
		t.Fatalf("seed mNoConsent: %v", err)
	}
	// Mit Einwilligung (foto=1), aber intern unsichtbar (photo_visible=0) → darf NICHT gelistet werden.
	mConsent := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(
		`UPDATE members SET first_name='Mit', last_name='Freigabe', foto_veroeffentlichung=1, photo_visible=0 WHERE id=?`,
		mConsent); err != nil {
		t.Fatalf("seed mConsent: %v", err)
	}
	for _, mid := range []int{mNoConsent, mConsent} {
		if _, err := db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, mid); err != nil {
			t.Fatalf("kader_members: %v", err)
		}
	}

	h := NewHandler(db, hub.NewHub(), &appconfig.Config{})
	missing := h.consentMissing(gameID)

	if len(missing) != 1 {
		t.Fatalf("consentMissing: got %d Einträge, want 1 (%+v)", len(missing), missing)
	}
	if missing[0].LastName != "Freigabe" || missing[0].FirstName != "Ohne" {
		t.Errorf("consentMissing: got %+v, want {Ohne Freigabe}", missing[0])
	}
}
