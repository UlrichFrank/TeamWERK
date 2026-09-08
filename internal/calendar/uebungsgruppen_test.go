package calendar_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// TestIcalFeed_OhneUebungsgruppe hält den strukturellen Ausschluss fest:
// `fetchTrainings` joint `teams` über `ts.team_id` — bei einer Übungsgruppe ist
// die Spalte NULL, der JOIN matcht nicht, der Termin fehlt im Feed. Das ist
// kein Versäumnis, sondern der Mechanismus (design.md — Entscheidung 2). Ein
// späterer „Reparatur"-Backfill von `team_id` würde diesen Test kippen — genau
// dafür steht er hier.
//
// Der Nutzer ist zugleich Mitglied einer echten Mannschaft: der Feed liefert
// also nachweislich Termine, nur eben nicht den der Übungsgruppe. Ohne diesen
// Gegenbeleg wäre ein leerer Feed von einem korrekt gefilterten nicht zu
// unterscheiden.
func TestIcalFeed_OhneUebungsgruppe(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")

	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)

	// Mannschaftstermin — muss im Feed erscheinen.
	teamID := testutil.CreateTeam(t, db, "Team A")
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderMember(t, db, kaderID, memberID)
	teamSessionID := testutil.CreateTrainingSession(t, db, teamID, seasonID, "2026-05-04")

	// Übungsgruppentermin desselben Nutzers — darf NICHT im Feed erscheinen.
	groupID := testutil.CreatePracticeGroup(t, db, seasonID, "Torwarttraining")
	testutil.AddKaderMember(t, db, groupID, memberID)
	res, err := db.Exec(
		`INSERT INTO training_sessions (kader_id, season_id, date, start_time, end_time, title)
		 VALUES (?, ?, '2026-05-05', '18:00', '20:00', 'Torwarttraining')`, groupID, seasonID)
	if err != nil {
		t.Fatalf("Übungsgruppen-Termin anlegen: %v", err)
	}
	groupSessionRaw, _ := res.LastInsertId()
	groupSessionID := int(groupSessionRaw)

	srv := prodserver.New(t, db)
	userToken := testutil.Token(t, userID, "standard", nil)
	tok := postToken(t, srv, userToken, allTogglesOn())

	feed := testutil.Get(t, srv, "/api/calendar/feed/"+tok["token"].(string), "")
	defer feed.Body.Close()
	if feed.StatusCode != http.StatusOK {
		t.Fatalf("erwartet 200, bekommen %d", feed.StatusCode)
	}
	body := unfoldICS(readBody(t, feed.Body))

	teamUID := "training-" + strconv.Itoa(teamSessionID)
	groupUID := "training-" + strconv.Itoa(groupSessionID)
	if !strings.Contains(body, teamUID) {
		t.Fatalf("Mannschaftstermin (%s) fehlt im Feed — der Gegenbeleg trägt nicht", teamUID)
	}
	if strings.Contains(body, groupUID) {
		t.Errorf("Übungsgruppen-Termin (%s) steht im iCal-Feed", groupUID)
	}
	if strings.Contains(body, "Torwarttraining") {
		t.Errorf("Titel der Übungsgruppe steht im iCal-Feed")
	}
}
