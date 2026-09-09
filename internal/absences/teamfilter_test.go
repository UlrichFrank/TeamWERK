package absences_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/absences"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Teil von openspec/changes/team-mehrfachfilter-alle-listen: der
// Kalender-Mannschaftsfilter ist eine Mehrfachauswahl, und die Abwesenheiten
// sind seine einzige serverseitig gefilterte Datenquelle. `team_id` nimmt
// deshalb eine ID-Liste; die Einzel-ID bleibt gültig (Bestandsverhalten,
// abgedeckt von TestCalendar_ShowTeam_VorstandSeesTeam).

// calendarFixture legt drei Mannschaften mit je einer öffentlichen Abwesenheit an
// und gibt den Trainer-Token-User samt der drei Mitglieds-IDs zurück. Der
// Anfragende ist Trainer aller drei Kader — ohne diesen Zugang wäre ein leeres
// Ergebnis überbestimmt (user_accessible_teams schnitte ohnehin alles weg).
func calendarFixture(t *testing.T, db *sql.DB) (userID int, memberIDs [3]int, teamIDs [3]int) {
	t.Helper()
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	userID = testutil.CreateUser(t, db, "standard")
	trainerMemberID := testutil.CreateMember(t, db, userID)

	for i := range teamIDs {
		teamIDs[i] = testutil.CreateTeam(t, db, fmt.Sprintf("Team %d", i+1))
		kaderID := testutil.CreateKader(t, db, teamIDs[i], seasonID)
		testutil.AddKaderTrainer(t, db, kaderID, trainerMemberID)

		memberIDs[i] = testutil.CreateMember(t, db, 0)
		testutil.AddKaderMember(t, db, kaderID, memberIDs[i])
		db.Exec(`UPDATE members SET absences_public=1 WHERE id=?`, memberIDs[i])
		testutil.CreateAbsence(t, db, memberIDs[i], "vacation", "2026-03-01", "2026-03-05", userID)
	}
	return userID, memberIDs, teamIDs
}

func calendarServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()
	h := absences.NewHandler(db, hub.NewHub())
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/absences/calendar", h.Calendar)
	})
}

func calendarMemberIDs(t *testing.T, db *sql.DB, userID int, teamParam string) map[int]bool {
	t.Helper()
	srv := calendarServer(t, db)
	url := "/api/absences/calendar?from=2026-03-01&to=2026-03-31&show_team=true"
	if teamParam != "" {
		url += "&team_id=" + teamParam
	}
	res := testutil.Get(t, srv, url, testutil.Token(t, userID, "standard", []string{"trainer"}))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	found := map[int]bool{}
	for _, id := range decodeAbsenceMemberIDs(t, res) {
		found[id] = true
	}
	return found
}

// TestCalendar_TeamIDListe: `team_id=A,B` liefert genau die Abwesenheiten beider
// Mannschaften — vorher hätte der Handler die Liste per Sscanf auf die erste ID
// verkürzt und die zweite Mannschaft stillschweigend unterschlagen.
func TestCalendar_TeamIDListe(t *testing.T) {
	db := testutil.NewDB(t)
	userID, members, teams := calendarFixture(t, db)

	found := calendarMemberIDs(t, db, userID, fmt.Sprintf("%d,%d", teams[0], teams[1]))

	if !found[members[0]] || !found[members[1]] {
		t.Errorf("expected absences of both filtered teams, got %v", found)
	}
	if found[members[2]] {
		t.Errorf("team %d was not selected, its absence must not appear", teams[2])
	}
}

// TestCalendar_TeamIDEinzeln: Bestandslinks mit genau einer ID bleiben gültig.
func TestCalendar_TeamIDEinzeln(t *testing.T) {
	db := testutil.NewDB(t)
	userID, members, teams := calendarFixture(t, db)

	found := calendarMemberIDs(t, db, userID, fmt.Sprintf("%d", teams[1]))

	if !found[members[1]] {
		t.Errorf("expected absence of team %d, got %v", teams[1], found)
	}
	if found[members[0]] || found[members[2]] {
		t.Errorf("only team %d was selected, got %v", teams[1], found)
	}
}

// TestCalendar_TeamIDUnbrauchbar: ein unlesbarer Filter verhält sich wie kein
// Filter (keine Fehlermeldung, keine leere Liste) — dieselbe Regel wie im
// Frontend (web/src/lib/teamFilter.ts).
func TestCalendar_TeamIDUnbrauchbar(t *testing.T) {
	db := testutil.NewDB(t)
	userID, members, _ := calendarFixture(t, db)

	found := calendarMemberIDs(t, db, userID, "abc")

	for i, m := range members {
		if !found[m] {
			t.Errorf("unbrauchbarer team_id-Filter muss wie kein Filter wirken; Abwesenheit %d (Team %d) fehlt", m, i+1)
		}
	}
}
