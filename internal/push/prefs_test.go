package push_test

import (
	"sort"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/push"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestFilterByPushPref_DefaultIncluded — ohne Präferenz-Zeile gilt push=true,
// der Nutzer bleibt im Ergebnis.
func TestFilterByPushPref_DefaultIncluded(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")

	got := push.FilterByPushPref(db, []int{uid}, "games")

	if len(got) != 1 || got[0] != uid {
		t.Fatalf("FilterByPushPref = %v, want [%d] (Default push=true)", got, uid)
	}
}

// TestFilterByPushPref_DisabledExcluded — push_enabled=0 filtert aus.
func TestFilterByPushPref_DisabledExcluded(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	testutil.CreateNotificationPreference(t, db, uid, "games", false, false)

	got := push.FilterByPushPref(db, []int{uid}, "games")

	if len(got) != 0 {
		t.Fatalf("FilterByPushPref = %v, want []", got)
	}
}

// TestFilterByPushPref_CategoryIsolation — ein Opt-out für 'games' lässt
// 'trainings' unberührt.
func TestFilterByPushPref_CategoryIsolation(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	testutil.CreateNotificationPreference(t, db, uid, "games", false, false)

	got := push.FilterByPushPref(db, []int{uid}, "trainings")

	if len(got) != 1 || got[0] != uid {
		t.Fatalf("FilterByPushPref(trainings) = %v, want [%d]", got, uid)
	}
}

// TestFilterByPushPref_MixedSet — nur der deaktivierte Nutzer fällt raus.
func TestFilterByPushPref_MixedSet(t *testing.T) {
	db := testutil.NewDB(t)
	on := testutil.CreateUser(t, db, "standard")
	off := testutil.CreateUser(t, db, "standard")
	noRow := testutil.CreateUser(t, db, "standard")
	testutil.CreateNotificationPreference(t, db, on, "duties", true, false)
	testutil.CreateNotificationPreference(t, db, off, "duties", false, false)

	got := push.FilterByPushPref(db, []int{on, off, noRow}, "duties")
	sort.Ints(got)
	want := []int{on, noRow}
	sort.Ints(want)

	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("FilterByPushPref = %v, want %v", got, want)
	}
}

// TestHasEmailEnabled — true nur bei email_enabled=1; kein Row ⇒ false.
func TestHasEmailEnabled(t *testing.T) {
	db := testutil.NewDB(t)
	on := testutil.CreateUser(t, db, "standard")
	off := testutil.CreateUser(t, db, "standard")
	noRow := testutil.CreateUser(t, db, "standard")
	testutil.CreateNotificationPreference(t, db, on, "duty_reminders", true, true)
	testutil.CreateNotificationPreference(t, db, off, "duty_reminders", true, false)

	if !push.HasEmailEnabled(db, on, "duty_reminders") {
		t.Errorf("on: HasEmailEnabled = false, want true")
	}
	if push.HasEmailEnabled(db, off, "duty_reminders") {
		t.Errorf("off: HasEmailEnabled = true, want false")
	}
	if push.HasEmailEnabled(db, noRow, "duty_reminders") {
		t.Errorf("noRow: HasEmailEnabled = true, want false (Default)")
	}
	// Kategorie-Trennung: on hat nur duty_reminders aktiviert
	if push.HasEmailEnabled(db, on, "games") {
		t.Errorf("on/games: HasEmailEnabled = true, want false")
	}
}

// TestGetAllPreferences_DefaultsAndOverride — alle Kategorien inkl. chat mit
// Defaults; gespeicherte Zeilen überschreiben.
func TestGetAllPreferences_DefaultsAndOverride(t *testing.T) {
	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	testutil.CreateNotificationPreference(t, db, uid, "chat", false, false)

	prefs := push.GetAllPreferences(db, uid)

	for _, c := range push.ValidCategories {
		if _, ok := prefs[c]; !ok {
			t.Errorf("Kategorie %q fehlt in GetAllPreferences", c)
		}
	}
	// chat wurde überschrieben
	if prefs["chat"]["push"] {
		t.Errorf("chat.push = true, want false (überschrieben)")
	}
	// games unberührt ⇒ Default
	if !prefs["games"]["push"] || prefs["games"]["email"] {
		t.Errorf("games = %v, want push=true/email=false (Default)", prefs["games"])
	}
}
