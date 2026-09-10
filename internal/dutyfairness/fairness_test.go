package dutyfairness_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/dutyfairness"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// day liefert ein Datum relativ zu heute in Vereinszeit — dieselbe Uhr, gegen
// die die Zählung geleistet/vorhersage trennt.
func day(offset int) string {
	return time.Now().In(timez.Berlin()).AddDate(0, 0, offset).Format("2006-01-02")
}

type fixture struct {
	t        *testing.T
	db       *sql.DB
	seasonID int
	dutyType int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := testutil.NewDB(t)
	return &fixture{
		t:        t,
		db:       db,
		seasonID: testutil.CreateSeason(t, db, "2026/27"),
		dutyType: testutil.CreateDutyType(t, db, "Kasse", 1.0),
	}
}

// team legt eine Mannschaft mit Kader der aktiven Saison an.
func (f *fixture) team(name string) (teamID, kaderID int) {
	f.t.Helper()
	teamID = testutil.CreateTeam(f.t, f.db, name)
	return teamID, testutil.CreateKader(f.t, f.db, teamID, f.seasonID)
}

// player legt ein Kader-Mitglied an; userID=0 heißt ohne eigenen Account.
func (f *fixture) player(kaderID, userID int) int {
	f.t.Helper()
	memberID := testutil.CreateMember(f.t, f.db, userID)
	testutil.AddKaderMember(f.t, f.db, kaderID, memberID)
	return memberID
}

// teamSlot legt einen spiellosen Slot mit direkter team_id an.
func (f *fixture) teamSlot(teamID int, date string, total int) int {
	f.t.Helper()
	id := testutil.CreateDutySlot(f.t, f.db, f.dutyType, f.seasonID, teamID, 0, date)
	f.exec(`UPDATE duty_slots SET slots_total=? WHERE id=?`, total, id)
	return id
}

// gameSlot legt ein Spiel des Teams samt Slot an; das Team hängt nur über
// game_teams am Slot (team_id=NULL wie seit Migration 051).
func (f *fixture) gameSlot(teamID int, date string, total int) int {
	f.t.Helper()
	gameID := testutil.CreateGame(f.t, f.db, f.seasonID, teamID, date)
	id := testutil.CreateDutySlot(f.t, f.db, f.dutyType, f.seasonID, teamID, gameID, date)
	f.exec(`UPDATE duty_slots SET slots_total=?, team_id=NULL WHERE id=?`, total, id)
	return id
}

// genericSlot legt einen Slot ohne game_id und ohne team_id an (Vereinsfest).
func (f *fixture) genericSlot(date string, total int) int {
	f.t.Helper()
	res, err := f.db.Exec(
		`INSERT INTO duty_slots (event_name, event_date, duty_type_id, slots_total, season_id)
		 VALUES ('Vereinsfest', ?, ?, ?, ?)`, date, f.dutyType, total, f.seasonID)
	if err != nil {
		f.t.Fatalf("genericSlot: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func (f *fixture) assign(slotID, userID int, status string) {
	f.t.Helper()
	f.exec(`INSERT INTO duty_assignments (duty_slot_id, user_id, status) VALUES (?, ?, ?)`, slotID, userID, status)
}

func (f *fixture) exec(q string, args ...any) {
	f.t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		f.t.Fatalf("exec %q: %v", q, err)
	}
}

func (f *fixture) compute() *dutyfairness.Snapshot {
	f.t.Helper()
	snap, err := dutyfairness.Compute(context.Background(), f.db, f.seasonID)
	if err != nil {
		f.t.Fatalf("Compute: %v", err)
	}
	return snap
}

func member(t *testing.T, team *dutyfairness.Team, memberID int) *dutyfairness.Member {
	t.Helper()
	for _, m := range team.Members {
		if m.MemberID == memberID {
			return m
		}
	}
	t.Fatalf("member %d not in team %d", memberID, team.TeamID)
	return nil
}

// Zählung nach Datum: vor heute = geleistet, heute und später = Vorhersage.
// Der status ist ohne Bedeutung — auch „fulfilled"/„cash_substitute" zählen
// nur über ihr Datum.
func TestCompute_HeuteZaehltAlsVorhersage_StatusIstEgal(t *testing.T) {
	f := newFixture(t)
	teamID, kaderID := f.team("Damen 1")
	userID := testutil.CreateUser(t, f.db, "standard")
	memberID := f.player(kaderID, userID)

	f.assign(f.teamSlot(teamID, day(-2), 1), userID, "fulfilled")
	f.assign(f.teamSlot(teamID, day(-1), 1), userID, "cash_substitute")
	f.assign(f.teamSlot(teamID, day(0), 1), userID, "assigned")
	f.assign(f.teamSlot(teamID, day(3), 1), userID, "fulfilled")

	m := member(t, f.compute().Teams[teamID], memberID)
	if m.Geleistet != 2 || m.Vorhersage != 2 {
		t.Errorf("geleistet/vorhersage = %v/%v, want 2/2 (heute zählt als Vorhersage)", m.Geleistet, m.Vorhersage)
	}
}

// Gesamtsumme = team-gebundene Slots voll (über game_teams wie über direkte
// team_id) plus generische Slots anteilig nach Spieleranzahl.
func TestCompute_GesamtsummeTeamgebundenUndGenerischAnteilig(t *testing.T) {
	f := newFixture(t)
	teamA, kaderA := f.team("A")
	teamB, kaderB := f.team("B")
	for range 2 {
		f.player(kaderA, 0)
	}
	for range 8 {
		f.player(kaderB, 0)
	}

	f.gameSlot(teamA, day(7), 2) // A: +2 über game_teams
	f.teamSlot(teamB, day(7), 2) // B: +2 über team_id
	f.genericSlot(day(14), 10)   // 10 × 2/10 = 2 für A, 10 × 8/10 = 8 für B
	f.genericSlot(day(-14), 0)   // leere Slots verschieben nichts

	snap := f.compute()
	a, b := snap.Teams[teamA], snap.Teams[teamB]
	if a.Total != 4 || a.Soll != 2 {
		t.Errorf("A total/soll = %v/%v, want 4/2", a.Total, a.Soll)
	}
	if b.Total != 10 || b.Soll != 1.25 {
		t.Errorf("B total/soll = %v/%v, want 10/1.25", b.Total, b.Soll)
	}
}

// Geschwister werden nicht dedupliziert: jedes Kind trägt den vollen
// Fair-Anteil. Eine Eltern-Zuweisung am Team-Slot wird zwischen den passenden
// Geschwistern geteilt, damit die Familiensumme der tatsächlichen Arbeit
// entspricht.
func TestCompute_GeschwisterJeVollerAnteil_ElternZuweisungGeteilt(t *testing.T) {
	f := newFixture(t)
	teamID, kaderID := f.team("wCJ")
	parent := testutil.CreateUser(t, f.db, "standard")
	kid1, kid2 := f.player(kaderID, 0), f.player(kaderID, 0)
	testutil.AddFamilyLink(t, f.db, parent, kid1)
	testutil.AddFamilyLink(t, f.db, parent, kid2)
	f.player(kaderID, 0)
	f.player(kaderID, 0)

	f.teamSlot(teamID, day(10), 8)
	f.assign(f.teamSlot(teamID, day(-3), 0), parent, "assigned")

	team := f.compute().Teams[teamID]
	for _, id := range []int{kid1, kid2} {
		m := member(t, team, id)
		if team.Soll != 2 {
			t.Errorf("soll = %v, want 2 je Kind", team.Soll)
		}
		if m.Geleistet != 0.5 {
			t.Errorf("kind %d geleistet = %v, want 0.5 (Eltern-Zuweisung geteilt)", id, m.Geleistet)
		}
	}
}

// Zurechnung über Team-Match: eine Eltern-Zuweisung zählt nur für das Kind,
// dessen Team zum Slot passt; generisch passt zu allen; ein Slot ohne
// passendes Kind zählt für niemanden. Die eigene Zuweisung eines Kindes zählt
// immer voll, auch am Slot einer fremden Mannschaft.
func TestCompute_ZurechnungNachTeamMatch(t *testing.T) {
	f := newFixture(t)
	teamA, kaderA := f.team("A")
	teamB, kaderB := f.team("B")
	teamC, kaderC := f.team("C")
	f.player(kaderC, 0)

	parent := testutil.CreateUser(t, f.db, "standard")
	kidBUser := testutil.CreateUser(t, f.db, "standard") // Proxy-Account
	kidA, kidB := f.player(kaderA, 0), f.player(kaderB, kidBUser)
	testutil.AddFamilyLink(t, f.db, parent, kidA)
	testutil.AddFamilyLink(t, f.db, parent, kidB)

	f.assign(f.teamSlot(teamA, day(-1), 1), parent, "assigned")  // → kidA 1
	f.assign(f.genericSlot(day(-1), 1), parent, "assigned")      // → je 0.5
	f.assign(f.teamSlot(teamC, day(-1), 1), parent, "assigned")  // → niemand
	f.assign(f.teamSlot(teamA, day(2), 1), kidBUser, "assigned") // → kidB 1 (eigene)

	snap := f.compute()
	a := member(t, snap.Teams[teamA], kidA)
	b := member(t, snap.Teams[teamB], kidB)
	if a.Geleistet != 1.5 || a.Vorhersage != 0 {
		t.Errorf("kidA = %v/%v, want 1.5/0", a.Geleistet, a.Vorhersage)
	}
	if b.Geleistet != 0.5 || b.Vorhersage != 1 {
		t.Errorf("kidB = %v/%v, want 0.5/1", b.Geleistet, b.Vorhersage)
	}
}

// Ohne bekannte Slots ist die Gesamtsumme 0 und damit soll = 0 — kein Fehler,
// keine Division durch 0.
func TestCompute_GesamtsummeNullSollNull(t *testing.T) {
	f := newFixture(t)
	teamID, kaderID := f.team("Herren 1")
	f.player(kaderID, 0)
	f.player(kaderID, 0)

	team := f.compute().Teams[teamID]
	if team.Total != 0 || team.Soll != 0 {
		t.Errorf("total/soll = %v/%v, want 0/0", team.Total, team.Soll)
	}
}
