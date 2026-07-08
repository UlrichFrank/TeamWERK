package matchreports

import (
	"database/sql"
	"testing"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/push"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// capturePushRecipients ersetzt den push.SendToUsers-Seam und liefert einen
// Getter der zuletzt übergebenen Empfänger.
func capturePushRecipients(t *testing.T) *[]int {
	t.Helper()
	var captured []int
	orig := push.SendToUsers
	push.SendToUsers = func(_ *sql.DB, _ *appconfig.Config, uids []int, _, _, _ string) {
		captured = append([]int(nil), uids...)
	}
	t.Cleanup(func() { push.SendToUsers = orig })
	return &captured
}

func contains(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// TestNotifyReviewers_RespectsOperativOptOut — Freigeber mit 'operativ'=aus
// erhält die Freigabe-Benachrichtigung NICHT mehr (zuvor Bypass).
func TestNotifyReviewers_RespectsOperativOptOut(t *testing.T) {
	db := testutil.NewDB(t)
	reviewerID := testutil.CreateMedienUser(t, db)
	testutil.CreateNotificationPreference(t, db, reviewerID, "operativ", false, false)

	got := capturePushRecipients(t)
	notifyReviewers(db, testutil.TestConfig(), "Titel", "Body", "/spielberichte/1")

	if contains(*got, reviewerID) {
		t.Fatalf("Freigeber %d trotz Opt-out in Empfängern %v", reviewerID, *got)
	}
}

// TestNotifyReviewers_DefaultSends — ohne Opt-out (Default) geht die Push raus.
func TestNotifyReviewers_DefaultSends(t *testing.T) {
	db := testutil.NewDB(t)
	reviewerID := testutil.CreateMedienUser(t, db)

	got := capturePushRecipients(t)
	notifyReviewers(db, testutil.TestConfig(), "Titel", "Body", "/spielberichte/1")

	if !contains(*got, reviewerID) {
		t.Fatalf("Freigeber %d nicht in Empfängern %v (Default sollte senden)", reviewerID, *got)
	}
}
