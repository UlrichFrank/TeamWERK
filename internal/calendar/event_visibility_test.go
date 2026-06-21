package calendar_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// TestCalendar_Filter: Der iCal-Feed des Users enthält nur Spiele der Teams,
// in denen er selbst Mitglied ist (kader_members) — Spiele anderer Teams
// kommen nicht hinein. Bestätigt die enge Personal-Scope-Filterung und
// damit die event-team-visibility-Invariante (Feed-Inhalt ⊆ visibility).
func TestCalendar_Filter(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	teamB := testutil.CreateTeam(t, db, "Team B")
	kaderA := testutil.CreateKader(t, db, teamA, seasonID)

	playerUID := testutil.CreateUser(t, db, "standard")
	playerMID := testutil.CreateMember(t, db, playerUID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderA, playerMID)

	testutil.CreateGame(t, db, seasonID, teamA, "2026-08-15") // gegen "Eigenspiel"-Default
	db.Exec(`UPDATE games SET opponent='OWNGAME' WHERE season_id=? AND date='2026-08-15'`, seasonID)
	testutil.CreateGame(t, db, seasonID, teamB, "2026-08-16")
	db.Exec(`UPDATE games SET opponent='OTHERGAME' WHERE season_id=? AND date='2026-08-16'`, seasonID)

	userToken := testutil.Token(t, playerUID, "standard", nil)
	srv := prodserver.New(t, db)

	// Token holen
	tokRes := testutil.Post(t, srv, "/api/calendar/token", userToken, map[string]any{
		"include_heim": true, "include_auswaerts": true, "include_training": true,
		"include_generisch": true, "include_duty": true,
	})
	defer tokRes.Body.Close()
	if tokRes.StatusCode != http.StatusOK {
		t.Fatalf("token create: got %d", tokRes.StatusCode)
	}
	var tokOut map[string]any
	json.NewDecoder(tokRes.Body).Decode(&tokOut)
	calToken := tokOut["token"].(string)

	feedRes := testutil.Get(t, srv, "/api/calendar/feed/"+calToken+".ics", "")
	defer feedRes.Body.Close()
	if feedRes.StatusCode != http.StatusOK {
		t.Fatalf("feed: got %d", feedRes.StatusCode)
	}
	bodyBytes, _ := io.ReadAll(feedRes.Body)
	body := string(bodyBytes)
	if !strings.Contains(body, "OWNGAME") {
		t.Errorf("Feed muss eigenes Team-Spiel enthalten")
	}
	if strings.Contains(body, "OTHERGAME") {
		t.Errorf("Feed darf Fremd-Team-Spiel NICHT enthalten — gefunden im Body")
	}
}
