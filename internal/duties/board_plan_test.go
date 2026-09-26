package duties_test

import (
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// assigneesPlanScans liefert die Planzeilen der Assignee-Query, die game_teams
// vollständig durchsuchen. Leer = das Team-Prädikat greift je Slot über den
// Primärschlüssel (game_id, team_id) zu.
func assigneesPlanScans(t *testing.T, f *aushilfeFixture, slotID int) []string {
	t.Helper()
	rows, err := f.db.Query(`EXPLAIN QUERY PLAN `+duties.BoardAssigneesSQL(1), slotID)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()
	var scans []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if strings.HasPrefix(detail, "SCAN gt_s") || strings.HasPrefix(detail, "SCAN game_teams") {
			scans = append(scans, detail)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return scans
}

// Change dienstboerse-ladezeit: das Aushilfe-Kennzeichen je Zusage ist über
// da.user_id korreliert. Ein Mengenvergleich `ds.game_id IN (SELECT game_id
// FROM game_teams WHERE team_id IN (…))` scannt dafür je Zusage alle
// game_teams-Zeilen (gemessen 2,9 s für 178 Zusagen). Der Plan darf das nicht —
// auch nicht nach ANALYZE, das einen Index auf team_id wieder aushebeln würde.
func TestBoardAssigneesPlan_KeinScanAufGameTeams(t *testing.T) {
	f, _ := newAushilfeFixture(t)
	user := testutil.CreateUser(t, f.db, "standard")
	var slotID int
	if err := f.db.QueryRow(`SELECT id FROM duty_slots WHERE game_id = ?`, f.gameA).Scan(&slotID); err != nil {
		t.Fatalf("slot: %v", err)
	}
	if _, err := f.db.Exec(`INSERT INTO duty_assignments (duty_slot_id, user_id) VALUES (?, ?)`, slotID, user); err != nil {
		t.Fatalf("assignment: %v", err)
	}

	if scans := assigneesPlanScans(t, f, slotID); len(scans) > 0 {
		t.Errorf("Assignee-Query scannt game_teams: %v", scans)
	}

	if _, err := f.db.Exec(`ANALYZE`); err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if scans := assigneesPlanScans(t, f, slotID); len(scans) > 0 {
		t.Errorf("Assignee-Query scannt game_teams nach ANALYZE: %v", scans)
	}
}
