package dutyfairness_test

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/dutyfairness"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Aushilfe (Change dienste-erweiterter-kader): Zurechnungsstufen 3/4, getrennte
// Zählung je (Mitglied, Team), kein Einfluss auf Soll und Rangliste.

func (f *fixture) extended(kaderID, memberID int) {
	f.t.Helper()
	testutil.AddExtendedKaderMember(f.t, f.db, kaderID, memberID)
}

func aushilfeOf(team *dutyfairness.Team, memberID int) *dutyfairness.AushilfePosition {
	for _, p := range team.Aushilfe {
		if p.Member.MemberID == memberID {
			return p
		}
	}
	return nil
}

func TestCompute_AushilfeZaehltNichtAufsStammteam(t *testing.T) {
	f := newFixture(t)
	teamA, kaderA := f.team("mC1")
	teamB, kaderB := f.team("mB1")
	userID := testutil.CreateUser(t, f.db, "standard")
	m := f.player(kaderA, userID)
	f.extended(kaderB, m)

	f.assign(f.gameSlot(teamB, day(-3), 2), userID, "assigned")

	snap := f.compute()
	if got := member(t, snap.Teams[teamA], m); got.Geleistet != 0 || got.Vorhersage != 0 {
		t.Errorf("Stammteam-Zählung = %v/%v, want 0/0 — Aushilfe darf nicht ins Stammteam rutschen", got.Geleistet, got.Vorhersage)
	}
	pos := aushilfeOf(snap.Teams[teamB], m)
	if pos == nil || pos.Geleistet != 1 || pos.Vorhersage != 0 {
		t.Fatalf("Aushilfe für Team B = %+v, want geleistet 1", pos)
	}
	if got := snap.AushilfeFor(userID); len(got) != 1 || got[0].Team.TeamID != teamB {
		t.Errorf("AushilfeFor = %+v, want eine Position für Team B", got)
	}
}

func TestCompute_AushilfeAendertSollNicht(t *testing.T) {
	f := newFixture(t)
	teamB, kaderB := f.team("mB1")
	f.player(kaderB, 0)
	f.player(kaderB, 0)
	f.gameSlot(teamB, day(5), 4)
	before := f.compute().Teams[teamB]
	total, soll, count := before.Total, before.Soll, before.PlayerCount

	helper := testutil.CreateUser(t, f.db, "standard")
	hm := testutil.CreateMember(t, f.db, helper)
	f.extended(kaderB, hm)
	// Slot mit slots_total=0: die Gesamtsumme bleibt gleich, geprüft wird nur
	// der Einfluss der Aushilfe-Zuweisung selbst.
	f.assign(f.gameSlot(teamB, day(6), 0), helper, "assigned")

	after := f.compute().Teams[teamB]
	if after.Total != total || after.Soll != soll || after.PlayerCount != count {
		t.Errorf("Total/Soll/PlayerCount = %v/%v/%d, want %v/%v/%d", after.Total, after.Soll, after.PlayerCount, total, soll, count)
	}
	for _, m := range after.Members {
		if m.MemberID == hm {
			t.Error("Aushilfe darf nicht in Team.Members stehen (Rangliste)")
		}
	}
	if aushilfeOf(after, hm) == nil {
		t.Error("Aushilfe-Position fehlt")
	}
}

func TestCompute_StammkindSchlaegtAushilfekind(t *testing.T) {
	f := newFixture(t)
	teamB, kaderB := f.team("mB1")
	parent := testutil.CreateUser(t, f.db, "standard")
	stamm := f.player(kaderB, 0)
	ext := testutil.CreateMember(t, f.db, 0)
	f.extended(kaderB, ext)
	testutil.AddFamilyLink(t, f.db, parent, stamm)
	testutil.AddFamilyLink(t, f.db, parent, ext)

	f.assign(f.gameSlot(teamB, day(-1), 1), parent, "assigned")

	snap := f.compute()
	if got := member(t, snap.Teams[teamB], stamm); got.Geleistet != 1 {
		t.Errorf("Stammkader-Kind geleistet = %v, want 1", got.Geleistet)
	}
	if pos := aushilfeOf(snap.Teams[teamB], ext); pos != nil {
		t.Errorf("Aushilfe-Kind darf nichts bekommen, bekam %+v", pos)
	}
}

func TestCompute_ElternZuweisungFuerKindImErweitertenKader(t *testing.T) {
	f := newFixture(t)
	teamA, kaderA := f.team("mC1")
	teamB, kaderB := f.team("mB1")
	parent := testutil.CreateUser(t, f.db, "standard")
	child := f.player(kaderA, 0)
	f.extended(kaderB, child)
	testutil.AddFamilyLink(t, f.db, parent, child)

	f.assign(f.gameSlot(teamB, day(4), 1), parent, "assigned")

	snap := f.compute()
	if got := member(t, snap.Teams[teamA], child); got.Vorhersage != 0 {
		t.Errorf("Stammteam-Vorhersage = %v, want 0", got.Vorhersage)
	}
	if pos := aushilfeOf(snap.Teams[teamB], child); pos == nil || pos.Vorhersage != 1 {
		t.Errorf("Aushilfe-Vorhersage für Team B = %+v, want 1", pos)
	}
	if got := snap.AushilfeFor(parent); len(got) != 1 {
		t.Errorf("AushilfeFor(Elternteil) = %d Positionen, want 1", len(got))
	}
}

// Ein Trainer eines Slot-Teams hilft dort nie aus — auch nicht über ein Kind im
// erweiterten Kader (dieselbe Antwort wie die Dienstbörse).
func TestCompute_TrainerIstKeineAushilfe(t *testing.T) {
	f := newFixture(t)
	teamB, kaderB := f.team("mB1")
	trainer := testutil.CreateUser(t, f.db, "standard")
	testutil.AddKaderTrainer(t, f.db, kaderB, testutil.CreateMember(t, f.db, trainer))
	child := testutil.CreateMember(t, f.db, 0)
	f.extended(kaderB, child)
	testutil.AddFamilyLink(t, f.db, trainer, child)

	f.assign(f.gameSlot(teamB, day(-1), 1), trainer, "assigned")

	if pos := aushilfeOf(f.compute().Teams[teamB], child); pos != nil {
		t.Errorf("Trainer-Dienst darf nicht als Aushilfe des Kindes zählen: %+v", pos)
	}
}

func TestCompute_AusgetretenImErweitertenKaderIstKeineAushilfe(t *testing.T) {
	f := newFixture(t)
	teamB, kaderB := f.team("mB1")
	userID := testutil.CreateUser(t, f.db, "standard")
	m := testutil.CreateMember(t, f.db, userID)
	f.exec(`UPDATE members SET status='ausgetreten' WHERE id=?`, m)
	f.extended(kaderB, m)
	f.assign(f.gameSlot(teamB, day(-1), 1), userID, "assigned")

	if pos := aushilfeOf(f.compute().Teams[teamB], m); pos != nil {
		t.Errorf("ausgetretenes Mitglied darf keine Aushilfe-Position haben: %+v", pos)
	}
}

func TestTeam_AushilfenSortiert(t *testing.T) {
	f := newFixture(t)
	teamB, kaderB := f.team("mB1")
	u1 := testutil.CreateUser(t, f.db, "standard")
	u2 := testutil.CreateUser(t, f.db, "standard")
	m1 := testutil.CreateMember(t, f.db, u1)
	m2 := testutil.CreateMember(t, f.db, u2)
	f.extended(kaderB, m1)
	f.extended(kaderB, m2)
	f.assign(f.gameSlot(teamB, day(-1), 1), u1, "assigned")
	f.assign(f.gameSlot(teamB, day(-2), 1), u2, "assigned")
	f.assign(f.gameSlot(teamB, day(-3), 1), u2, "assigned")

	got := f.compute().Teams[teamB].Aushilfen()
	if len(got) != 2 || got[0].Member.MemberID != m2 || got[1].Member.MemberID != m1 {
		t.Errorf("Reihenfolge falsch: %+v", got)
	}
}
