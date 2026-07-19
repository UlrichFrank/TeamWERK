package games_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestGameRespond_ForeignMember_Forbidden deckt die IDOR-Absicherung von
// RespondToGame ab: ein Standard-Nutzer, der das Spiel zwar sehen darf (eigenes
// Kader-Member im Team), aber weder Eigentümer noch Elternteil des Ziel-Members
// ist und keine Manage-Berechtigung hat, wird mit HTTP 403 abgewiesen — es wird
// keine game_responses-Zeile für das fremde Member angelegt.
func TestGameRespond_ForeignMember_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	// Zukünftiges Datum, damit der RSVP-Cutoff die Anfrage nicht mit 422 abfängt.
	gameDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, gameDate)

	// Aufrufer: Standard-Spieler mit eigenem Member im Kader → darf das Spiel sehen.
	callerUser := testutil.CreateUser(t, db, "standard")
	callerMember := testutil.CreateMember(t, db, callerUser)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, callerMember)

	// Fremdes Member (anderes Kind, keine Verknüpfung zum Aufrufer).
	foreignMember := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, foreignMember)

	srv := testServer(t, db)
	token := testutil.Token(t, callerUser, "standard", nil)

	res := testutil.Post(t, srv,
		fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "declined", "member_id": foreignMember})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for foreign member, got %d", res.StatusCode)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM game_responses WHERE game_id=? AND member_id=?`,
		gameID, foreignMember).Scan(&n)
	if n != 0 {
		t.Errorf("no game_responses row must be written for foreign member, got %d", n)
	}
}

// TestGameRespond_OwnMember_OK verankert den Happy-Path: der Aufrufer meldet
// sich für sein EIGENES Member-Record (ohne member_id) zurück → 204.
func TestGameRespond_OwnMember_OK(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	gameDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	gameID := testutil.CreateGame(t, db, seasonID, teamID, gameDate)

	callerUser := testutil.CreateUser(t, db, "standard")
	callerMember := testutil.CreateMember(t, db, callerUser)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, callerMember)

	srv := testServer(t, db)
	token := testutil.Token(t, callerUser, "standard", nil)

	res := testutil.Post(t, srv,
		fmt.Sprintf("/api/games/%d/respond", gameID), token,
		map[string]any{"status": "confirmed"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for own member, got %d", res.StatusCode)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM game_responses WHERE game_id=? AND member_id=?`,
		gameID, callerMember).Scan(&status); err != nil {
		t.Fatalf("own response not persisted: %v", err)
	}
	if status != "confirmed" {
		t.Errorf("expected confirmed, got %q", status)
	}
}
