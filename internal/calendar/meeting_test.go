package calendar_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// meetingFeed liefert den Feed-Body eines Spielers mit einem Spiel (Anwurf
// 18:00), dessen Treffzeit-Spalten gesetzt sind (spiel-treffpunkt).
func meetingFeed(t *testing.T, date, start string, offset any, place, note string) (body string, gameID int) {
	t.Helper()
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "mB1")
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	db.Exec(`INSERT INTO kader_members (kader_id, member_id) VALUES (?, ?)`, kaderID, memberID)
	gameID = testutil.CreateGame(t, db, seasonID, teamID, date)
	if _, err := db.Exec(`UPDATE games SET time=?, meet_offset_minutes=?, meet_place=?, note=? WHERE id=?`,
		start, offset, place, note, gameID); err != nil {
		t.Fatal(err)
	}
	srv := prodserver.New(t, db)
	tok := postToken(t, srv, testutil.Token(t, userID, "standard", nil), allTogglesOn())
	res := testutil.Get(t, srv, "/api/calendar/feed/"+tok["token"].(string), "")
	defer res.Body.Close()
	return unfoldICS(readBody(t, res.Body)), gameID
}

func TestCalendar_DescriptionNenntTreffzeit(t *testing.T) {
	body, id := meetingFeed(t, "2026-08-15", "18:00", 90, "Parkplatz Vereinsheim", "Trikots mitbringen")
	block := vevent(t, body, fmt.Sprintf("game-%d@teamwerk", id))
	if l, _ := propLine(block, "DTSTART"); l != "DTSTART;TZID=Europe/Berlin:20260815T180000" {
		t.Errorf("DTSTART muss der Anwurf bleiben, got %q", l)
	}
	// Komma nach RFC 5545 escaped; der Aufstellungssatz folgt hinter der Notiz.
	want := "DESCRIPTION:Treffen: 16:30 Uhr\\, Parkplatz Vereinsheim\\n\\nTrikots mitbringen\\n\\n"
	if l, _ := propLine(block, "DESCRIPTION"); !strings.HasPrefix(l, want) {
		t.Errorf("DESCRIPTION = %q, want prefix %q", l, want)
	}
}

func TestCalendar_TreffzeitAmVortag(t *testing.T) {
	body, id := meetingFeed(t, "2026-08-15", "06:00", 420, "", "")
	block := vevent(t, body, fmt.Sprintf("game-%d@teamwerk", id))
	if l, _ := propLine(block, "DESCRIPTION"); !strings.HasPrefix(l, "DESCRIPTION:Treffen: 23:00 Uhr (Vortag)\\n\\n") {
		t.Errorf("DESCRIPTION = %q", l)
	}
}

func TestCalendar_OhneTreffzeitKeinAbsatz(t *testing.T) {
	body, id := meetingFeed(t, "2026-08-15", "18:00", nil, "", "Trikots mitbringen")
	block := vevent(t, body, fmt.Sprintf("game-%d@teamwerk", id))
	if l, _ := propLine(block, "DESCRIPTION"); !strings.HasPrefix(l, "DESCRIPTION:Trikots mitbringen\\n\\n") {
		t.Errorf("DESCRIPTION = %q", l)
	}
}
