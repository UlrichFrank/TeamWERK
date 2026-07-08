package matchreports

import (
	"database/sql"
	"testing"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/push"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestNotifyReviewers_BypassesPushPref nagelt das aktuelle Verhalten fest:
// notifyReviewers ruft push.SendToUsers OHNE FilterByPushPref auf → ein
// Freigeber mit push_enabled=0 erhält die Push dennoch.
//
// OFFENE DESIGN-FRAGE (nicht hier entschieden): gewollter Bypass für
// operativ wichtige Freigabe-Benachrichtigungen oder Bug?
func TestNotifyReviewers_BypassesPushPref(t *testing.T) {
	db := testutil.NewDB(t)
	reviewerID := testutil.CreateMedienUser(t, db)
	testutil.CreateNotificationPreference(t, db, reviewerID, "games", false, false)

	var captured []int
	orig := push.SendToUsers
	push.SendToUsers = func(_ *sql.DB, _ *appconfig.Config, uids []int, _, _, _ string) {
		captured = uids
	}
	t.Cleanup(func() { push.SendToUsers = orig })

	notifyReviewers(db, testutil.TestConfig(), "Titel", "Body", "/spielberichte/1")

	found := false
	for _, u := range captured {
		if u == reviewerID {
			found = true
		}
	}
	if !found {
		t.Fatalf("Freigeber %d nicht in Push-Empfängern %v", reviewerID, captured)
	}
}
