package duties_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// aushilfeGroup ist die Sicht der Aushilfe-Tests auf eine Board-Gruppe
// (Change dienste-erweiterter-kader).
type aushilfeGroup struct {
	GameID   *int `json:"game_id"`
	Aushilfe bool `json:"aushilfe"`
	Slots    []struct {
		ID        int `json:"id"`
		Assignees []struct {
			UserID   int  `json:"user_id"`
			Aushilfe bool `json:"aushilfe"`
		} `json:"assignees"`
	} `json:"slots"`
}

type aushilfeFixture struct {
	db             *sql.DB
	season         int
	teamA, teamB   int
	kaderA, kaderB int
	gameA, gameB   int
	dt             int
}

// newAushilfeFixture: Team A und B der aktiven Saison mit je einem künftigen
// Spiel samt Dienst-Slot.
func newAushilfeFixture(t *testing.T) (*aushilfeFixture, func(token string) []aushilfeGroup) {
	t.Helper()
	db := testutil.NewDB(t)
	f := &aushilfeFixture{db: db}
	f.season = testutil.CreateSeason(t, db, "2025/26")
	f.teamA = testutil.CreateTeam(t, db, "mC1")
	f.teamB = testutil.CreateTeam(t, db, "mB1")
	f.kaderA = testutil.CreateKader(t, db, f.teamA, f.season)
	f.kaderB = testutil.CreateKader(t, db, f.teamB, f.season)
	f.dt = createDutyType(t, db, "Bewirtung", 2)
	f.gameA = testutil.CreateGame(t, db, f.season, f.teamA, "2099-03-07")
	f.gameB = testutil.CreateGame(t, db, f.season, f.teamB, "2099-03-08")
	createDutySlot(t, db, f.dt, f.season, f.teamA, f.gameA, "2099-03-07")
	createDutySlot(t, db, f.dt, f.season, f.teamB, f.gameB, "2099-03-08")

	srv := testServer(t, duties.NewHandler(db, testutil.TestConfig(), hub.NewHub()))
	board := func(token string) []aushilfeGroup {
		t.Helper()
		resp := testutil.Get(t, srv, "/api/duty-board", token)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var groups []aushilfeGroup
		if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return groups
	}
	return f, board
}

func groupFor(groups []aushilfeGroup, gameID int) *aushilfeGroup {
	for i := range groups {
		if groups[i].GameID != nil && *groups[i].GameID == gameID {
			return &groups[i]
		}
	}
	return nil
}

func TestBoard_ErweiterterKaderSiehtTeamDienste(t *testing.T) {
	f, board := newAushilfeFixture(t)
	player := testutil.CreateUser(t, f.db, "standard")
	m := testutil.CreateMember(t, f.db, player)
	testutil.AddKaderMember(t, f.db, f.kaderA, m)
	testutil.AddExtendedKaderMember(t, f.db, f.kaderB, m)

	groups := board(testutil.Token(t, player, "standard", []string{"spieler"}))
	a, b := groupFor(groups, f.gameA), groupFor(groups, f.gameB)
	if a == nil || b == nil {
		t.Fatalf("erwartet Gruppen beider Teams, bekam A=%v B=%v", a != nil, b != nil)
	}
	if a.Aushilfe {
		t.Error("Stammteam-Gruppe darf nicht als Aushilfe markiert sein")
	}
	if !b.Aushilfe {
		t.Error("Gruppe des erweiterten Teams muss aushilfe=true tragen")
	}
}

func TestBoard_ElternErweiterterKaderSehenElternSlot(t *testing.T) {
	f, board := newAushilfeFixture(t)
	eltern := createDutySlot(t, f.db, f.dt, f.season, f.teamB, f.gameB, "2099-03-08")
	f.db.Exec(`UPDATE duty_slots SET audiences='["eltern"]' WHERE id=?`, eltern)

	parent := testutil.CreateUser(t, f.db, "standard")
	child := testutil.CreateMember(t, f.db, 0)
	testutil.AddFamilyLink(t, f.db, parent, child)
	testutil.AddExtendedKaderMember(t, f.db, f.kaderB, child)

	b := groupFor(board(testutil.Token(t, parent, "standard", nil)), f.gameB)
	if b == nil {
		t.Fatal("Elternteil eines Kindes im erweiterten Kader muss die Gruppe von Team B sehen")
	}
	found := false
	for _, s := range b.Slots {
		found = found || s.ID == eltern
	}
	if !found {
		t.Error("Slot mit audiences=[eltern] fehlt — Eltern-Zielgruppe muss den erweiterten Kader umfassen")
	}
	if !b.Aushilfe {
		t.Error("aushilfe=true erwartet")
	}
}

func TestBoard_AusgetretenImErweitertenKaderSiehtNichts(t *testing.T) {
	f, board := newAushilfeFixture(t)
	player := testutil.CreateUser(t, f.db, "standard")
	m := testutil.CreateMember(t, f.db, player)
	f.db.Exec(`UPDATE members SET status='ausgetreten' WHERE id=?`, m)
	testutil.AddExtendedKaderMember(t, f.db, f.kaderB, m)

	if groups := board(testutil.Token(t, player, "standard", []string{"spieler"})); len(groups) != 0 {
		t.Errorf("Ausgetretene im erweiterten Kader dürfen nichts sehen, bekam %d Gruppen", len(groups))
	}
}

func TestBoard_FoerderkindImErweitertenKader(t *testing.T) {
	f, board := newAushilfeFixture(t)
	parent := testutil.CreateUser(t, f.db, "standard")
	child := testutil.CreateMember(t, f.db, 0)
	f.db.Exec(`UPDATE members SET status='foerderkind' WHERE id=?`, child)
	testutil.AddFamilyLink(t, f.db, parent, child)
	testutil.AddExtendedKaderMember(t, f.db, f.kaderB, child)

	if groupFor(board(testutil.Token(t, parent, "standard", nil)), f.gameB) == nil {
		t.Error("Förderkind im erweiterten Kader: Gruppe von Team B fehlt (Statusfilter darf nicht = 'aktiv' sein)")
	}
}

func TestBoard_StammSchlaegtErweitert(t *testing.T) {
	f, board := newAushilfeFixture(t)
	// Gemeinsames Spiel von A und B.
	f.db.Exec(`INSERT INTO game_teams (game_id, team_id) VALUES (?, ?)`, f.gameA, f.teamB)
	player := testutil.CreateUser(t, f.db, "standard")
	m := testutil.CreateMember(t, f.db, player)
	testutil.AddKaderMember(t, f.db, f.kaderA, m)
	testutil.AddExtendedKaderMember(t, f.db, f.kaderB, m)

	a := groupFor(board(testutil.Token(t, player, "standard", []string{"spieler"})), f.gameA)
	if a == nil {
		t.Fatal("Gruppe des gemeinsamen Spiels fehlt")
	}
	if a.Aushilfe {
		t.Error("gemeinsames Spiel mit dem Stammteam darf nicht als Aushilfe gelten")
	}
}

func TestBoard_AssigneeAushilfeKennzeichen(t *testing.T) {
	f, board := newAushilfeFixture(t)
	var slotB int
	f.db.QueryRow(`SELECT id FROM duty_slots WHERE game_id=?`, f.gameB).Scan(&slotB)

	helper := testutil.CreateUser(t, f.db, "standard")
	hm := testutil.CreateMember(t, f.db, helper)
	testutil.AddExtendedKaderMember(t, f.db, f.kaderB, hm)

	parent := testutil.CreateUser(t, f.db, "standard")
	child := testutil.CreateMember(t, f.db, 0)
	testutil.AddFamilyLink(t, f.db, parent, child)
	testutil.AddKaderMember(t, f.db, f.kaderB, child)

	f.db.Exec(`INSERT INTO duty_assignments (duty_slot_id, user_id) VALUES (?, ?), (?, ?)`, slotB, helper, slotB, parent)
	f.db.Exec(`UPDATE duty_slots SET slots_filled=2 WHERE id=?`, slotB)

	trainer := testutil.CreateUser(t, f.db, "standard")
	tm := testutil.CreateMember(t, f.db, trainer)
	testutil.AddKaderTrainer(t, f.db, f.kaderB, tm)

	check := func(viewer string, token string) {
		b := groupFor(board(token), f.gameB)
		if b == nil || len(b.Slots) == 0 {
			t.Fatalf("%s: Gruppe von Team B fehlt", viewer)
		}
		flags := map[int]bool{}
		for _, a := range b.Slots[0].Assignees {
			flags[a.UserID] = a.Aushilfe
		}
		if !flags[helper] {
			t.Errorf("%s: Aushilfe aus dem erweiterten Kader muss aushilfe=true tragen", viewer)
		}
		if v, ok := flags[parent]; !ok || v {
			t.Errorf("%s: Elternteil eines Stammkader-Kindes muss aushilfe=false tragen (ok=%v)", viewer, ok)
		}
	}
	check("Trainer", testutil.Token(t, trainer, "standard", []string{"trainer"}))
	check("Aushilfe selbst", testutil.Token(t, helper, "standard", []string{"spieler"}))
}

func TestBoard_FremdesTeamWeiterhinUnsichtbar(t *testing.T) {
	f, board := newAushilfeFixture(t)
	teamC := testutil.CreateTeam(t, f.db, "mA1")
	testutil.CreateKader(t, f.db, teamC, f.season)
	gameC := testutil.CreateGame(t, f.db, f.season, teamC, "2099-03-09")
	createDutySlot(t, f.db, f.dt, f.season, teamC, gameC, "2099-03-09")

	player := testutil.CreateUser(t, f.db, "standard")
	m := testutil.CreateMember(t, f.db, player)
	testutil.AddExtendedKaderMember(t, f.db, f.kaderB, m)

	groups := board(testutil.Token(t, player, "standard", []string{"spieler"}))
	if groupFor(groups, gameC) != nil {
		t.Error("Team ohne jede Kader-Verbindung darf nicht sichtbar werden")
	}
	if groupFor(groups, f.gameA) != nil {
		t.Error("Team A (keine Verbindung) darf nicht sichtbar werden")
	}
}
