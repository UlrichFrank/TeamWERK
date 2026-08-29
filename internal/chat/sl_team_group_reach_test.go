package chat_test

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Die sportliche Leitung sieht alle Team-Standardgruppen der aktiven Saison
// (hasGlobalTeamGroupAccess). Damit diese Sicht auch benutzbar ist, muss der
// Kontaktcheck beim Gruppenaufbau dieselbe Reichweite haben — sonst löst sich
// eine fremde Kader-Gruppe zwar auf, das Anlegen des Gesprächs scheitert aber
// an einem 403 für jedes teamfremde Mitglied.
func TestCreateGroup_SLDarfFremdeKaderGruppeAlsGruppeAnlegen(t *testing.T) {
	f, tgMux := setupTwoTeams(t)
	h := chat.NewHandler(f.db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations", h.CreateConversation)
		r.Mount("/", tgMux)
	})

	sl := testutil.CreateUser(t, f.db, "standard")
	slMember := testutil.CreateMember(t, f.db, sl)
	testutil.AddClubFunction(t, f.db, slMember, "sportliche_leitung")
	token := testutil.Token(t, sl, "standard", []string{"sportliche_leitung"})

	// Die Spieler-Gruppe von T2 auflösen (sL ist in T2 nicht eingetragen).
	res := testutil.Get(t, srv, "/api/chat/team-groups/"+strconv.Itoa(f.team2)+"/spieler/members", token)
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		t.Fatalf("resolve T2/spieler: expected 200, got %d", res.StatusCode)
	}
	members := decodeJSON[[]chat.TeamGroupMember](t, res)
	if len(members) == 0 {
		t.Fatalf("T2/spieler should resolve to at least one member")
	}

	ids := make([]int, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.ID)
	}

	created := testutil.Post(t, srv, "/api/chat/conversations", token,
		map[string]any{"type": "group", "name": "T2 Spieler", "memberIds": ids})
	defer created.Body.Close()
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("sportliche_leitung should be able to create a group from a foreign team group, got %d",
			created.StatusCode)
	}
}

// Gegenprobe: Ein reiner Spieler bleibt auf sein Team beschränkt.
func TestCreateGroup_SpielerDarfTeamfremdeNichtGruppieren(t *testing.T) {
	f, tgMux := setupTwoTeams(t)
	h := chat.NewHandler(f.db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Post("/api/chat/conversations", h.CreateConversation)
		r.Mount("/", tgMux)
	})

	token := testutil.Token(t, f.playerU1, "standard", nil)
	res := testutil.Post(t, srv, "/api/chat/conversations", token,
		map[string]any{"type": "group", "name": "T2 Spieler", "memberIds": []int{f.playerU2}})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("player should not reach a foreign team's player, got %d", res.StatusCode)
	}
}
