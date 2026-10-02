package chat_test

import (
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func countFxRows(t *testing.T, f *tgFixture, query string, args ...any) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestCreateGroup_SpeichertSources(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))
	res := testutil.Post(t, srv, "/api/chat/conversations", testutil.Token(t, f.trainerU1, "standard", nil), map[string]any{
		"type": "group", "name": "mC1", "memberIds": []int{f.playerU1, f.parentU1},
		"sources": []map[string]any{
			{"groupType": "team", "refId": f.team1, "kind": "spieler"},
			{"groupType": "team", "refId": f.team1, "kind": "eltern"},
			{"groupType": "team", "refId": f.team1, "kind": "eltern"}, // Dublette
		},
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, erwartet 201", res.StatusCode)
	}
	conv := decodeJSON[struct{ ID int }](t, res)
	if n := countFxRows(t, f, `SELECT COUNT(*) FROM conversation_sources WHERE conversation_id = ?`, conv.ID); n != 2 {
		t.Errorf("%d Herkunftszeilen, erwartet 2 (Dublette zusammengefasst)", n)
	}
}

func TestCreateGroup_FremdeSource403(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))
	res := testutil.Post(t, srv, "/api/chat/conversations", testutil.Token(t, f.playerU1, "standard", nil), map[string]any{
		"type": "group", "name": "fremd", "memberIds": []int{f.trainerU1},
		"sources": []map[string]any{{"groupType": "team", "refId": f.team2, "kind": "spieler"}},
	})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d, erwartet 403", res.StatusCode)
	}
	if n := countFxRows(t, f, `SELECT COUNT(*) FROM conversations`); n != 0 {
		t.Errorf("%d Konversationen entstanden, erwartet 0", n)
	}
}

func TestCreateGroup_UngueltigerKind400(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))
	tok := testutil.Token(t, f.trainerU1, "standard", nil)
	for _, src := range []map[string]any{
		{"groupType": "team", "refId": f.team1, "kind": "foobar"},
		{"groupType": "verein", "refId": f.team1, "kind": "spieler"},
		{"groupType": "team", "refId": 999999, "kind": "spieler"}, // existiert nicht
	} {
		res := testutil.Post(t, srv, "/api/chat/conversations", tok, map[string]any{
			"type": "group", "name": "x", "memberIds": []int{f.playerU1},
			"sources": []map[string]any{src},
		})
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("%v: status %d, erwartet 400", src, res.StatusCode)
		}
	}
	if n := countFxRows(t, f, `SELECT COUNT(*) FROM conversations`); n != 0 {
		t.Errorf("%d Konversationen entstanden, erwartet 0", n)
	}
}

func TestCreateGroup_OhneSourcesWieBisher(t *testing.T) {
	f, _ := setupTwoTeams(t)
	srv := testutil.NewServer(t, newContactServer(t, f.db))
	res := testutil.Post(t, srv, "/api/chat/conversations", testutil.Token(t, f.trainerU1, "standard", nil), map[string]any{
		"type": "group", "name": "ohne", "memberIds": []int{f.playerU1},
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, erwartet 201", res.StatusCode)
	}
	if n := countFxRows(t, f, `SELECT COUNT(*) FROM conversation_sources`); n != 0 {
		t.Errorf("%d Herkunftszeilen, erwartet 0", n)
	}
	if n := countFxRows(t, f, `SELECT COUNT(*) FROM conversation_members`); n != 2 {
		t.Errorf("%d Mitglieder, erwartet 2", n)
	}
}
