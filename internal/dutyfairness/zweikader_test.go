package dutyfairness_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/dutyfairness"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Zählung je (Kind, Stammkader) — Change dienst-bilanz-je-kader. Ein Kind in
// zwei Stammkadern hat je Mannschaft eigene Zahlen; nicht eindeutig
// zuordenbare Dienste werden erst zwischen Geschwistern, dann zwischen den
// Kadern des Kindes gleichmäßig geteilt.

// jointGameSlot legt ein Spiel an, das über game_teams zu beiden Teams gehört.
func (f *fixture) jointGameSlot(teamA, teamB int, date string) int {
	f.t.Helper()
	gameID := testutil.CreateGame(f.t, f.db, f.seasonID, teamA, date)
	f.exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, gameID, teamB)
	id := testutil.CreateDutySlot(f.t, f.db, f.dutyType, f.seasonID, teamA, gameID, date)
	f.exec(`UPDATE duty_slots SET team_id=NULL WHERE id=?`, id)
	return id
}

// zweiKader legt Team A und B an und ein Kind in beiden Stammkadern, verknüpft
// mit einem Elternteil.
func (f *fixture) zweiKader() (teamA, kaderA, teamB, kaderB, parent, kid int) {
	f.t.Helper()
	teamA, kaderA = f.team("A")
	teamB, kaderB = f.team("B")
	parent = testutil.CreateUser(f.t, f.db, "standard")
	kid = f.player(kaderA, 0)
	testutil.AddKaderMember(f.t, f.db, kaderB, kid)
	testutil.AddFamilyLink(f.t, f.db, parent, kid)
	return
}

func wantPos(t *testing.T, p *dutyfairness.Position, geleistet, vorhersage float64) {
	t.Helper()
	if math.Abs(p.Geleistet-geleistet) > 1e-9 || math.Abs(p.Vorhersage-vorhersage) > 1e-9 {
		t.Errorf("Position (%d, %s) = %v/%v, want %v/%v",
			p.Member.MemberID, p.Team.Label, p.Geleistet, p.Vorhersage, geleistet, vorhersage)
	}
}

func TestCompute_ZweiKader_DienstZaehltNurFuerEigeneMannschaft(t *testing.T) {
	f := newFixture(t)
	teamA, _, teamB, _, parent, kid := f.zweiKader()

	f.assign(f.gameSlot(teamA, day(-2), 1), parent, "assigned")

	snap := f.compute()
	wantPos(t, member(t, snap.Teams[teamA], kid), 1, 0)
	wantPos(t, member(t, snap.Teams[teamB], kid), 0, 0)
}

func TestCompute_ZweiKader_GenerischWirdGeteilt(t *testing.T) {
	f := newFixture(t)
	teamA, _, teamB, _, parent, kid := f.zweiKader()

	f.assign(f.genericSlot(day(-2), 1), parent, "assigned")

	snap := f.compute()
	wantPos(t, member(t, snap.Teams[teamA], kid), 0.5, 0)
	wantPos(t, member(t, snap.Teams[teamB], kid), 0.5, 0)
}

func TestCompute_ZweiKader_GemeinsamesSpielWirdGeteilt(t *testing.T) {
	f := newFixture(t)
	teamA, _, teamB, _, parent, kid := f.zweiKader()

	f.assign(f.jointGameSlot(teamA, teamB, day(4)), parent, "assigned")

	snap := f.compute()
	wantPos(t, member(t, snap.Teams[teamA], kid), 0, 0.5)
	wantPos(t, member(t, snap.Teams[teamB], kid), 0, 0.5)
}

// Erst zwischen Geschwistern (je 0,5), dann zwischen den Kadern des Kindes —
// ein Kind mit zwei Kadern bekommt nicht mehr Gewicht als sein Geschwister.
func TestCompute_ZweiKader_GeschwisterVorKaderTeilung(t *testing.T) {
	f := newFixture(t)
	teamA, kaderA, teamB, _, parent, kid1 := f.zweiKader()
	kid2 := f.player(kaderA, 0)
	testutil.AddFamilyLink(t, f.db, parent, kid2)

	f.assign(f.genericSlot(day(-2), 1), parent, "assigned")

	snap := f.compute()
	wantPos(t, member(t, snap.Teams[teamA], kid2), 0.5, 0)
	wantPos(t, member(t, snap.Teams[teamA], kid1), 0.25, 0)
	wantPos(t, member(t, snap.Teams[teamB], kid1), 0.25, 0)
}

// Stufe 5: eigener Dienst bei einer Mannschaft, in der der Spieler weder im
// Stamm- noch im erweiterten Kader steht — geteilt auf seine Stammkader.
func TestCompute_ZweiKader_Stufe5WirdGeteilt(t *testing.T) {
	f := newFixture(t)
	teamA, kaderA := f.team("A")
	teamB, kaderB := f.team("B")
	teamC, kaderC := f.team("C")
	f.player(kaderC, 0)
	userID := testutil.CreateUser(t, f.db, "standard")
	m := f.player(kaderA, userID)
	testutil.AddKaderMember(t, f.db, kaderB, m)

	f.assign(f.teamSlot(teamC, day(-1), 1), userID, "assigned")

	snap := f.compute()
	wantPos(t, member(t, snap.Teams[teamA], m), 0.5, 0)
	wantPos(t, member(t, snap.Teams[teamB], m), 0.5, 0)
}

// Invariante: die Summe der Positionen eines Kindes ist sein Anteil, die Summe
// einer Familie die Zahl ihrer Zuweisungen — egal wie die Kader-Teilung fällt.
// Ein-Kader-Kinder tragen ihre Zuweisungen unverändert voll.
func TestCompute_SummeDerPositionenGleichKindZaehlung(t *testing.T) {
	f := newFixture(t)
	teamA, kaderA, teamB, _, parent, kid1 := f.zweiKader()
	teamC, kaderC := f.team("C")
	kid2 := f.player(kaderA, 0)
	kid3 := f.player(kaderC, 0)
	testutil.AddFamilyLink(t, f.db, parent, kid2)
	testutil.AddFamilyLink(t, f.db, parent, kid3)

	f.assign(f.gameSlot(teamA, day(-3), 1), parent, "assigned")                                   // kid1 A, kid2 A
	f.assign(f.gameSlot(teamB, day(-3), 1), parent, "assigned")                                   // kid1 B
	f.assign(f.jointGameSlot(teamA, teamB, day(-2)), parent, "assigned")                          // kid1 A/B, kid2 A
	f.assign(f.genericSlot(day(-1), 1), parent, "assigned")                                       // alle drei
	f.assign(f.gameSlot(teamC, day(-1), 1), parent, "assigned")                                   // kid3 C
	f.assign(f.teamSlot(teamC, day(-1), 1), testutil.CreateUser(t, f.db, "standard"), "assigned") // fremd

	snap := f.compute()
	sum := func(memberID int) float64 {
		var total float64
		for _, tid := range snap.TeamOrder {
			for _, p := range snap.Teams[tid].Members {
				if p.Member.MemberID == memberID {
					total += p.Geleistet + p.Vorhersage
				}
			}
		}
		return total
	}
	// kid1: 0,5 + 1 + 0,5 + 1/3; kid2: 0,5 + 0,5 + 1/3; kid3: 1/3 + 1.
	for _, c := range []struct {
		id   int
		want float64
	}{{kid1, 2 + 1.0/3}, {kid2, 1 + 1.0/3}, {kid3, 1 + 1.0/3}} {
		if got := sum(c.id); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Summe der Positionen von %d = %v, want %v", c.id, got, c.want)
		}
	}
	if family := sum(kid1) + sum(kid2) + sum(kid3); math.Abs(family-5) > 1e-9 {
		t.Errorf("Familiensumme = %v, want 5 (Zahl der Eltern-Zuweisungen)", family)
	}
	wantPos(t, member(t, snap.Teams[teamC], kid3), 1+1.0/3, 0)
}

// Rangliste: ein Kind in zwei Kadern steht in jedem Block mit den Zahlen der
// jeweiligen Position — in Team B hinter einem Kind mit einem Dienst, obwohl
// es in Team A drei geleistet hat.
func TestRangliste_ZweiKader_EigeneZahlenJeBlock(t *testing.T) {
	f := newFixture(t)
	teamA, _, teamB, kaderB, parent, kid := f.zweiKader()
	otherUser := testutil.CreateUser(t, f.db, "standard")
	f.player(kaderB, otherUser)
	for i := 1; i <= 3; i++ {
		f.assign(f.gameSlot(teamA, day(-i), 1), parent, "assigned")
	}
	f.assign(f.gameSlot(teamB, day(-1), 1), otherUser, "assigned")

	srv := ranglisteServer(t, f.db)
	token := testutil.TokenWithIsParent(t, parent, "standard", nil, true)
	status, body := getRangliste(t, srv, "", token)
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	blocks := map[int][]ranglisteRow{}
	for _, b := range body.Blocks {
		blocks[b.TeamID] = b.Rows
	}
	a, b := blocks[teamA], blocks[teamB]
	if len(a) != 1 || a[0].MemberID == nil || *a[0].MemberID != kid || a[0].Geleistet != 3 {
		t.Errorf("Block A = %+v, want Kind mit geleistet 3", a)
	}
	if len(b) != 2 || b[0].Geleistet != 1 || b[0].IsOwn {
		t.Fatalf("Block B = %+v, want Platz 1 fremdes Kind mit 1", b)
	}
	if b[1].MemberID == nil || *b[1].MemberID != kid || b[1].Rank != 2 || b[1].Geleistet != 0 {
		t.Errorf("Block B Platz 2 = %+v, want eigenes Kind mit 0 (nicht die 3 aus Team A)", b[1])
	}
}

// Fehlerfall bleibt: ein Team ohne eigene Verbindung ist auch mit einem
// Zwei-Kader-Kind 403.
func TestRangliste_ZweiKader_FremdesTeam403(t *testing.T) {
	f := newFixture(t)
	_, _, _, _, parent, _ := f.zweiKader()
	teamC, kaderC := f.team("C")
	f.player(kaderC, 0)

	srv := ranglisteServer(t, f.db)
	token := testutil.TokenWithIsParent(t, parent, "standard", nil, true)
	if status, _ := getRangliste(t, srv, "?team="+strconv.Itoa(teamC), token); status != 403 {
		t.Errorf("status = %d, want 403", status)
	}
}
