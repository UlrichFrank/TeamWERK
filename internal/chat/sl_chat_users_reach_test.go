package chat_test

import (
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/chat"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Die Nutzersuche muss dieselbe Reichweite haben wie der Kontaktcheck — sonst
// erscheint ein teamfremdes Mitglied als Gruppen-Chip, ist über die Suche aber
// nicht auffindbar.
func TestChatUsers_SLFindetTeamfremdeNutzer(t *testing.T) {
	f, _ := setupTwoTeams(t)
	h := chat.NewHandler(f.db, hub.NewHub(), testutil.TestConfig())
	srv := testutil.NewServer(t, func(r chi.Router) { r.Get("/api/chat/users", h.Users) })

	sl := testutil.CreateUser(t, f.db, "standard")
	slMember := testutil.CreateMember(t, f.db, sl)
	testutil.AddClubFunction(t, f.db, slMember, "sportliche_leitung")

	res := testutil.Get(t, srv, "/api/chat/users",
		testutil.Token(t, sl, "standard", []string{"sportliche_leitung"}))
	users := decodeJSON[[]chat.TeamGroupMember](t, res)

	found := false
	for _, u := range users {
		if u.ID == f.playerU2 {
			found = true
		}
	}
	if !found {
		t.Errorf("sportliche_leitung should find a foreign team's player, got %+v", users)
	}
}
