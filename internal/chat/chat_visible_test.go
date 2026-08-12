package chat_test

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestChatUsers_ChatVisible_SpielerFindetTeamfremdesMitglied — ein Opt-in über
// members.chat_visible macht ein Mitglied für die Nutzersuche im "Neue
// Nachricht"-Dialog zusätzlich auffindbar, auch ohne gemeinsames Team.
// Gegenprobe zu TestChatUsers_SpielerFindetTeamfremdenTrainerNicht: ohne das
// Flag bleibt trainerU2 unsichtbar (siehe dieser Test), mit gesetztem Flag
// wird er gefunden.
func TestChatUsers_ChatVisible_SpielerFindetTeamfremdesMitglied(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))

	if _, err := f.db.Exec(`UPDATE members SET chat_visible=1 WHERE user_id=?`, f.trainerU2); err != nil {
		t.Fatalf("set chat_visible: %v", err)
	}

	res := testutil.Get(t, srv, "/api/chat/users",
		testutil.Token(t, f.playerU1, "standard", nil))
	type row struct {
		ID int `json:"id"`
	}
	rows := decodeJSON[[]row](t, res)
	found := false
	for _, r := range rows {
		if r.ID == f.trainerU2 {
			found = true
		}
	}
	if !found {
		t.Errorf("player should find chat_visible=1 member %d, got %+v", f.trainerU2, rows)
	}
}
