package notify

import (
	"testing"
	"time"
)

func TestKaderTrainers_DedupliziertUndOhneKonto(t *testing.T) {
	db := newTestDB(t)
	season := insertSeason(t, db, "2025/26", true)
	kaderA := insertKader(t, db, insertTeam(t, db, "mB1"), season)
	kaderB := insertKader(t, db, insertTeam(t, db, "mB2"), season)

	uShared := insertUser(t, db, "shared@test.local")
	uOnlyB := insertUser(t, db, "onlyb@test.local")
	mShared := insertMember(t, db, "Shared", uShared)
	mOnlyB := insertMember(t, db, "OnlyB", uOnlyB)
	mNoAccount := insertMember(t, db, "NoAccount", 0)
	addKaderTrainer(t, db, kaderA, mShared)
	addKaderTrainer(t, db, kaderB, mShared)
	addKaderTrainer(t, db, kaderB, mOnlyB)
	addKaderTrainer(t, db, kaderA, mNoAccount)

	// Spieler ist kein Trainer und darf nicht auftauchen.
	uPlayer := insertUser(t, db, "player@test.local")
	addKaderMember(t, db, kaderA, insertMember(t, db, "Player", uPlayer))

	got := KaderTrainers(db, kaderA, kaderB)
	if !equalIDs(got, []int{uShared, uOnlyB}) {
		t.Fatalf("KaderTrainers = %v, want %v", got, []int{uShared, uOnlyB})
	}
	if got := KaderTrainers(db); got != nil {
		t.Fatalf("leere Eingabe: %v, want nil", got)
	}
}

func TestRSVPChangeBody(t *testing.T) {
	cases := []struct {
		name                                string
		member, subject, when, from, to, rs string
		want                                string
	}{
		{"mit Grund", "Max Muster", "Heimspiel vs. TV Beispiel", "am 27.09.2026 um 15:00 Uhr", "confirmed", "declined", "  krank ",
			"Max Muster hat für Heimspiel vs. TV Beispiel am 27.09.2026 um 15:00 Uhr von Zusage auf Absage umgestellt. Grund: krank"},
		{"ohne Grund", "Max Muster", "Training", "am 25.09.2026 um 18:00 Uhr", "declined", "maybe", "",
			"Max Muster hat für Training am 25.09.2026 um 18:00 Uhr von Absage auf Vielleicht umgestellt."},
		{"ohne Namen und Zeit", "", "", "", "maybe", "confirmed", "",
			"Ein Mitglied hat für Ein Termin von Vielleicht auf Zusage umgestellt."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := RSVPChangeBody(c.member, c.subject, c.when, c.from, c.to, c.rs); got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

func TestRSVPChangeRecipients(t *testing.T) {
	db := newTestDB(t)
	season := insertSeason(t, db, "2025/26", true)
	kader := insertKader(t, db, insertTeam(t, db, "mB1"), season)
	uTrainer1 := insertUser(t, db, "t1@test.local")
	uTrainer2 := insertUser(t, db, "t2@test.local")
	mTrainer1 := insertMember(t, db, "Trainer1", uTrainer1)
	addKaderTrainer(t, db, kader, mTrainer1)
	addKaderTrainer(t, db, kader, insertMember(t, db, "Trainer2", uTrainer2))
	uPlayer := insertUser(t, db, "p@test.local")
	mPlayer := insertMember(t, db, "Player", uPlayer)
	addKaderMember(t, db, kader, mPlayer)

	now := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	base := RSVPChange{
		KaderIDs: []int{kader}, MemberID: mPlayer, ActorUserID: uPlayer,
		PrevStatus: "confirmed", NewStatus: "declined",
		Start: now.Add(3 * 24 * time.Hour), Now: now,
	}

	cases := []struct {
		name string
		mod  func(c *RSVPChange)
		want []int
	}{
		{"Umentscheidung im Fenster", func(*RSVPChange) {}, []int{uTrainer1, uTrainer2}},
		{"genau 7 Tage", func(c *RSVPChange) { c.Start = now.Add(RSVPChangeWindow) }, []int{uTrainer1, uTrainer2}},
		{"7 Tage plus 1 Minute", func(c *RSVPChange) { c.Start = now.Add(RSVPChangeWindow + time.Minute) }, nil},
		{"Termin begonnen", func(c *RSVPChange) { c.Start = now.Add(-time.Minute) }, nil},
		{"erste Antwort", func(c *RSVPChange) { c.PrevStatus = "" }, nil},
		{"gleicher Status", func(c *RSVPChange) { c.PrevStatus = "declined" }, nil},
		{"Trainer eigene Antwort", func(c *RSVPChange) { c.MemberID = mTrainer1; c.ActorUserID = uTrainer1 }, nil},
		{"Trainer sagt für Spieler um", func(c *RSVPChange) { c.ActorUserID = uTrainer1 }, []int{uTrainer2}},
		{"kein Kader", func(c *RSVPChange) { c.KaderIDs = nil }, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			tc.mod(&c)
			if got := RSVPChangeRecipients(db, c); !equalIDs(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
