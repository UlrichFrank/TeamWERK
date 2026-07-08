package videos

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestVideoReady_BypassesPushPref nagelt fest: notifyReady sammelt die Empfänger
// (Hochladender + Team) OHNE FilterByPushPref → der Hochladende erhält die
// „Video ist bereit"-Push auch bei push_enabled=0.
//
// OFFENE DESIGN-FRAGE (nicht hier entschieden): gewollter Bypass oder Bug?
func TestVideoReady_BypassesPushPref(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team")
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, teamID, seasonID, uploader, "ready")
	testutil.CreateNotificationPreference(t, db, uploader, "games", false, false)

	wk, _, cfg := newTestWorker(t, db, fakeHLSTranscode)
	wk.notifyReady(vid)

	// pushSend läuft als Goroutine — auf Erfassung des Hochladenden warten.
	waitFor(t, func() bool {
		uids, _, ok := cfg.lastPush()
		if !ok {
			return false
		}
		for _, u := range uids {
			if u == uploader {
				return true
			}
		}
		return false
	})
}
