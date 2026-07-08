package videos

import (
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestVideoReady_RespectsSonstigesOptOut — der Hochladende mit 'sonstiges'=aus
// erhält die „Video ist bereit"-Push NICHT mehr (zuvor Bypass).
func TestVideoReady_RespectsSonstigesOptOut(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team")
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, teamID, seasonID, uploader, "ready")
	testutil.CreateNotificationPreference(t, db, uploader, "sonstiges", false, false)

	wk, _, cfg := newTestWorker(t, db, fakeHLSTranscode)
	wk.notifyReady(vid)

	// Kurz warten; es darf KEIN Push mit dem Hochladenden erscheinen.
	waitForNoPush(t, cfg, uploader)
}

// TestVideoReady_DefaultSends — ohne Opt-out (Default) erhält der Hochladende
// die Push.
func TestVideoReady_DefaultSends(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team")
	uploader := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, teamID, seasonID, uploader, "ready")

	wk, _, cfg := newTestWorker(t, db, fakeHLSTranscode)
	wk.notifyReady(vid)

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

// waitForNoPush stellt sicher, dass innerhalb eines kurzen Fensters kein Push
// mit uid erfasst wird (notifyReady kehrt bei leerer Empfängerliste ohne
// pushSend zurück).
func waitForNoPush(t *testing.T, cfg *fakeConfig, uid int) {
	t.Helper()
	deadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(deadline) {
		if uids, _, ok := cfg.lastPush(); ok {
			for _, u := range uids {
				if u == uid {
					t.Fatalf("unerwarteter Push an user %d (Opt-out sollte greifen)", uid)
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}
