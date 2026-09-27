package games_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

type lineupParticipants struct {
	Items []struct {
		MemberID  int    `json:"member_id"`
		IsTrainer bool   `json:"is_trainer"`
		Lineup    string `json:"lineup"`
	} `json:"items"`
	LineupCount int `json:"lineup_count"`
}

func getLineupParticipants(t *testing.T, res *http.Response) lineupParticipants {
	t.Helper()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var out lineupParticipants
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

// Ohne gespeicherte Aufstellung ist jede Spielerzeile „offen" — nie „nicht
// aufgestellt"; Trainerzeilen tragen keinen Status.
func TestGetParticipants_LineupOpen(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "mB1")
	kader := testutil.CreateKader(t, db, team, season)
	playerUser := testutil.CreateUser(t, db, "standard")
	player := testutil.CreateMember(t, db, playerUser)
	testutil.AddKaderMember(t, db, kader, player)
	trainer := testutil.CreateMember(t, db, testutil.CreateUser(t, db, "standard"))
	testutil.AddKaderTrainer(t, db, kader, trainer)
	game := testutil.CreateGame(t, db, season, team, "2026-10-03")

	srv := testServer(t, db)
	got := getLineupParticipants(t, testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", game),
		testutil.Token(t, playerUser, "standard", nil)))

	if got.LineupCount != 0 {
		t.Errorf("lineup_count = %d, want 0", got.LineupCount)
	}
	for _, it := range got.Items {
		switch {
		case it.IsTrainer && it.Lineup != "":
			t.Errorf("Trainerzeile trägt lineup=%q", it.Lineup)
		case !it.IsTrainer && it.Lineup != "open":
			t.Errorf("Spieler %d: lineup=%q, want open", it.MemberID, it.Lineup)
		}
	}

	// Nach dem ersten Eintrag: aufgestellt / nicht aufgestellt.
	other := testutil.CreateMember(t, db, 0)
	testutil.AddKaderMember(t, db, kader, other)
	testutil.AddLineup(t, db, game, other)
	got = getLineupParticipants(t, testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", game),
		testutil.Token(t, playerUser, "standard", nil)))
	want := map[int]string{player: "out", other: "in"}
	for _, it := range got.Items {
		if w, ok := want[it.MemberID]; ok && !it.IsTrainer && it.Lineup != w {
			t.Errorf("Mitglied %d: lineup=%q, want %q", it.MemberID, it.Lineup, w)
		}
	}
	if got.LineupCount != 1 {
		t.Errorf("lineup_count = %d, want 1", got.LineupCount)
	}
}

// Ein Spieler sieht bei einem Spiel mit zwei Mannschaften die Zeilen der
// fremden Mannschaft nicht. Steht dort jemand in der Aufstellung, ist die
// eigene Zeile trotzdem „nicht aufgestellt" — der Status wird über die ganze
// Aufstellung abgeleitet, nicht über die ausgelieferten Zeilen.
func TestGetParticipants_LineupOutTrotzVerborgenerZeilen(t *testing.T) {
	db := testutil.NewDB(t)
	fx := newCrossTeamFixture(t, db)
	db.Exec(`UPDATE games SET event_type='heim' WHERE id=?`, fx.gameID)
	testutil.AddLineup(t, db, fx.gameID, fx.memberTeamB)

	srv := testServer(t, db)
	got := getLineupParticipants(t, testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", fx.gameID),
		testutil.Token(t, fx.userOfMemberA, "standard", nil)))

	seenOwn := false
	for _, it := range got.Items {
		if it.MemberID == fx.memberTeamB {
			t.Fatalf("Mitglied der fremden Mannschaft darf nicht ausgeliefert werden")
		}
		if it.MemberID == fx.memberTeamA {
			seenOwn = true
			if it.Lineup != "out" {
				t.Errorf("eigene Zeile: lineup=%q, want out", it.Lineup)
			}
		}
	}
	if !seenOwn {
		t.Fatal("eigene Zeile fehlt")
	}
	if got.LineupCount != 1 {
		t.Errorf("lineup_count = %d, want 1", got.LineupCount)
	}
}

// Generische Termine haben keine Aufstellung.
func TestGetParticipants_LineupFehltBeiGenerischemEvent(t *testing.T) {
	db := testutil.NewDB(t)
	fx := newCrossTeamFixture(t, db) // Event ist generisch
	srv := testServer(t, db)
	got := getLineupParticipants(t, testutil.Get(t, srv, fmt.Sprintf("/api/games/%d/participants", fx.gameID),
		testutil.Token(t, fx.userOfMemberA, "standard", nil)))
	for _, it := range got.Items {
		if it.Lineup != "" {
			t.Errorf("Mitglied %d: lineup=%q bei generischem Event", it.MemberID, it.Lineup)
		}
	}
}

type myGameLineup struct {
	ID           int    `json:"id"`
	MyLineup     string `json:"my_lineup"`
	ChildrenRSVP []struct {
		MemberID int    `json:"member_id"`
		Lineup   string `json:"lineup"`
	} `json:"children_rsvp"`
}

func myGamesByID(t *testing.T, res *http.Response) map[int]myGameLineup {
	t.Helper()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var list []myGameLineup
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	out := map[int]myGameLineup{}
	for _, g := range list {
		out[g.ID] = g
	}
	return out
}

// Stammspieler und erweiterter Kader bekommen den Status in allen drei
// Zuständen; ein reiner Trainer und ein generisches Event bekommen keinen.
func TestListMyGames_MyLineup(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "mB1")
	kader := testutil.CreateKader(t, db, team, season)

	playerUser := testutil.CreateUser(t, db, "standard")
	player := testutil.CreateMember(t, db, playerUser)
	testutil.AddKaderMember(t, db, kader, player)
	extUser := testutil.CreateUser(t, db, "standard")
	ext := testutil.CreateMember(t, db, extUser)
	testutil.AddExtendedKaderMember(t, db, kader, ext)
	trainerUser := testutil.CreateUser(t, db, "standard")
	trainer := testutil.CreateMember(t, db, trainerUser)
	testutil.AddKaderTrainer(t, db, kader, trainer)

	open := testutil.CreateGame(t, db, season, team, "2026-10-01")
	set := testutil.CreateGame(t, db, season, team, "2026-10-02")
	testutil.AddLineup(t, db, set, player)
	generic := testutil.CreateGame(t, db, season, team, "2026-10-03")
	db.Exec(`UPDATE games SET event_type='generisch' WHERE id=?`, generic)

	srv := testServer(t, db)
	cases := []struct {
		name  string
		token string
		want  map[int]string
	}{
		{"Stammspieler", testutil.Token(t, playerUser, "standard", nil), map[int]string{open: "open", set: "in", generic: ""}},
		{"erweiterter Kader", testutil.Token(t, extUser, "standard", nil), map[int]string{open: "open", set: "out", generic: ""}},
		{"reiner Trainer ohne Feld", testutil.Token(t, trainerUser, "standard", []string{"trainer"}), map[int]string{open: "", set: "", generic: ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			games := myGamesByID(t, testutil.Get(t, srv, "/api/games/my", c.token))
			for id, want := range c.want {
				g, ok := games[id]
				if !ok {
					t.Fatalf("Spiel %d fehlt", id)
				}
				if g.MyLineup != want {
					t.Errorf("Spiel %d: my_lineup=%q, want %q", id, g.MyLineup, want)
				}
			}
		})
	}
}

func TestListMyGames_ChildLineup(t *testing.T) {
	db := testutil.NewDB(t)
	season := testutil.CreateSeason(t, db, "2025/26")
	team := testutil.CreateTeam(t, db, "mB1")
	kader := testutil.CreateKader(t, db, team, season)
	parent := testutil.CreateUser(t, db, "standard")
	emil := testutil.CreateMember(t, db, 0)
	mia := testutil.CreateMember(t, db, 0)
	testutil.AddKaderMember(t, db, kader, emil)
	testutil.AddExtendedKaderMember(t, db, kader, mia)
	testutil.AddFamilyLink(t, db, parent, emil)
	testutil.AddFamilyLink(t, db, parent, mia)
	game := testutil.CreateGame(t, db, season, team, "2026-10-11")
	testutil.AddLineup(t, db, game, emil)

	srv := testServer(t, db)
	games := myGamesByID(t, testutil.Get(t, srv, "/api/games/my", testutil.TokenWithIsParent(t, parent, "standard", nil, true)))
	want := map[int]string{emil: "in", mia: "out"}
	got := games[game].ChildrenRSVP
	if len(got) != 2 {
		t.Fatalf("children_rsvp: %+v", got)
	}
	for _, c := range got {
		if c.Lineup != want[c.MemberID] {
			t.Errorf("Kind %d: lineup=%q, want %q", c.MemberID, c.Lineup, want[c.MemberID])
		}
	}
}

func TestListMyGames_Unauthenticated(t *testing.T) {
	db := testutil.NewDB(t)
	srv := testServer(t, db)
	res := testutil.Get(t, srv, "/api/games/my", "")
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}
