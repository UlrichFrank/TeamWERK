package chat_test

import (
	"fmt"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Ausgetretene gehören zu keiner Standard-Gruppe (chat-gruppe-aktualisieren):
// sonst nähme der Anlege-Dialog sie auf und der erste Abgleich schlüge vor,
// sie gleich wieder zu entfernen. Der Filter ist `<> 'ausgetreten'`, nicht
// `= 'aktiv'` — verletzt/pausiert/foerderkind bleiben drin.

func setMemberStatusByUser(t *testing.T, f *tgFixture, userID int, status string) {
	t.Helper()
	extra := ""
	if status == "ausgetreten" {
		extra = ", exit_date = '2025-10-01'"
	}
	if _, err := f.db.Exec(`UPDATE members SET status = ?`+extra+` WHERE user_id = ?`, status, userID); err != nil {
		t.Fatalf("set status: %v", err)
	}
}

func resolveTeam(t *testing.T, f *tgFixture, mux *chi.Mux, teamID int, kind string, caller int) map[int]bool {
	t.Helper()
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/team-groups/%d/%s/members", teamID, kind),
		testutil.Token(t, caller, "standard", nil))
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	return memberIDs(decodeJSON[[]chat.TeamGroupMember](t, res))
}

func TestResolveSource_AusgetretenerFehlt(t *testing.T) {
	f, mux := setupTwoTeams(t)
	setMemberStatusByUser(t, f, f.playerU1, "ausgetreten")
	setMemberStatusByUser(t, f, f.extPlayerU1, "ausgetreten")
	got := resolveTeam(t, f, mux, f.team1, "spieler", f.trainerU1)
	if got[f.playerU1] || got[f.extPlayerU1] {
		t.Errorf("ausgetretene Spieler (Stamm- und erweiterter Kader) in der Gruppe: %v", got)
	}
}

func TestResolveSource_VerletztBleibt(t *testing.T) {
	f, mux := setupTwoTeams(t)
	setMemberStatusByUser(t, f, f.playerU1, "verletzt")
	if got := resolveTeam(t, f, mux, f.team1, "spieler", f.trainerU1); !got[f.playerU1] {
		t.Errorf("verletzter Spieler fehlt: %v", got)
	}
}

func TestResolveSource_ElternNurUeberAusgetretenesKindFehlt(t *testing.T) {
	f, mux := setupTwoTeams(t)
	setMemberStatusByUser(t, f, f.playerU1, "ausgetreten")
	got := resolveTeam(t, f, mux, f.team1, "eltern", f.trainerU1)
	if got[f.parentU1] {
		t.Errorf("Elternteil eines ausgetretenen Kindes in der Eltern-Gruppe: %v", got)
	}
	if !got[f.extParentU1] {
		t.Errorf("Elternteil eines nicht ausgetretenen Kindes fehlt: %v", got)
	}
}

func TestResolveSource_AusgetretenerTrainerFehlt(t *testing.T) {
	f, mux := setupTwoTeams(t)
	setMemberStatusByUser(t, f, f.trainerU2, "ausgetreten")
	if got := resolveTeam(t, f, mux, f.team2, "trainer", f.playerU2); got[f.trainerU2] {
		t.Errorf("ausgetretener Trainer in der Trainer-Gruppe: %v", got)
	}
	vorstand := testutil.CreateVorstandUser(t, f.db)
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	res := testutil.Get(t, srv, "/api/chat/team-groups/0/alle_trainer/members",
		testutil.Token(t, vorstand, "standard", []string{"vorstand"}))
	if got := memberIDs(decodeJSON[[]chat.TeamGroupMember](t, res)); got[f.trainerU2] || !got[f.trainerU1] {
		t.Errorf("Alle Trainer = %v, erwartet trainerU1 ohne den ausgetretenen trainerU2", got)
	}
}

func TestResolveSource_UebungsgruppeOhneAusgetretene(t *testing.T) {
	f, mux := setupPracticeGroup(t)
	if _, err := f.db.Exec(`UPDATE members SET status = 'ausgetreten', exit_date = '2025-10-01' WHERE user_id = ?`, f.playerU); err != nil {
		t.Fatal(err)
	}
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })
	tok := testutil.Token(t, f.trainerU, "standard", nil)
	for _, kind := range []string{"spieler", "eltern"} {
		res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/practice-groups/%d/%s/members", f.groupID, kind), tok)
		got := memberIDs(decodeJSON[[]chat.TeamGroupMember](t, res))
		if got[f.playerU] || got[f.parentU] {
			t.Errorf("%s: Ausgetretener bzw. dessen Elternteil enthalten: %v", kind, got)
		}
	}
}

// TestListTeamGroups_CountGleichMembers: die Zahl auf der Kachel und die
// aufgelöste Liste lesen dieselbe Menge — auch mit einem Ausgetretenen darin.
func TestListTeamGroups_CountGleichMembers(t *testing.T) {
	f, mux := setupTwoTeams(t)
	setMemberStatusByUser(t, f, f.playerU1, "ausgetreten")
	vorstand := testutil.CreateVorstandUser(t, f.db)
	tok := testutil.Token(t, vorstand, "standard", []string{"vorstand"})
	srv := testutil.NewServer(t, func(r chi.Router) { r.Mount("/", mux) })

	groups := decodeJSON[[]chat.TeamGroup](t, testutil.Get(t, srv, "/api/chat/team-groups", tok))
	if len(groups) == 0 {
		t.Fatal("keine Kacheln")
	}
	for _, g := range groups {
		if g.GroupType != "team" {
			continue
		}
		res := testutil.Get(t, srv, fmt.Sprintf("/api/chat/team-groups/%d/%s/members", g.TeamID, g.Kind), tok)
		members := decodeJSON[[]chat.TeamGroupMember](t, res)
		if len(members) != g.Count {
			t.Errorf("%d/%s: count=%d, members=%d", g.TeamID, g.Kind, g.Count, len(members))
		}
	}
}
