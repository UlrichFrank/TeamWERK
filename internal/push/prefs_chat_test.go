package push_test

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/push"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestFilterByPushPref_ChatDisabledExcludes — Regression Defekt 1: eine
// gespeicherte 'chat'-Zeile mit push_enabled=0 schließt den Nutzer bei
// Chat-Pushes tatsächlich aus (früher unmöglich, da der CHECK 'chat' nicht
// zuließ und die Zeile nie existieren konnte).
func TestFilterByPushPref_ChatDisabledExcludes(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	testutil.CreateNotificationPreference(t, db, uid, "chat", false, false)

	got := push.FilterByPushPref(db, []int{uid}, "chat")

	if len(got) != 0 {
		t.Fatalf("FilterByPushPref = %v, want [] (chat push disabled)", got)
	}
}
