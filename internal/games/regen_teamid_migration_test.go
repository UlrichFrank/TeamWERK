package games

import (
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Die Übergangszusage der Migration (design.md Decision 2): ein Bestands-Slot trägt noch
// team_id, der an seiner Stelle neu entstehende Slot trägt NULL. Weil team_id nicht mehr
// Teil des Match-Schlüssels ist, gelten beide als derselbe Slot — die Zusage überlebt den
// Umstieg, ohne dass jemand eine "Dienst entfernt"-Meldung bekommt.
//
// Genau hier hingen die 38 Zusagen, die zum Zeitpunkt der Umstellung in der Zukunft lagen.
func TestRegen_ZusageUeberlebtWechselVonTeamIdAufNull(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamA := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamA)
	h := newRegenTestHandler(t, db)
	userID := testutil.CreateUser(t, db, "standard")

	kasse := insertDutyTypeI(t, db, "Kasse", "", 0, "", 0)
	tpl := insertTemplateI(t, db, kasse, 0, 1)
	gameID := insertGameI(t, db, seasonID, teamA, "2026-06-13", "10:00", tpl)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	var slotID int
	if err := db.QueryRow(
		`SELECT id FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).Scan(&slotID); err != nil {
		t.Fatalf("Slot nach erstem Lauf: %v", err)
	}
	// Zustand VOR der Migration nachstellen: die Zeile trägt noch ihr Team.
	if _, err := db.Exec(`UPDATE duty_slots SET team_id=? WHERE id=?`, teamA, slotID); err != nil {
		t.Fatalf("Bestandszustand herstellen: %v", err)
	}
	assignI(t, db, slotID, userID, "assigned", nil, nil)

	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	var neuerSlot int
	var teamIDNachher any
	if err := db.QueryRow(
		`SELECT id, team_id FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).
		Scan(&neuerSlot, &teamIDNachher); err != nil {
		t.Fatalf("Slot nach zweitem Lauf: %v", err)
	}
	if teamIDNachher != nil {
		t.Errorf("der neu erzeugte Slot soll team_id=NULL tragen, hat %v", teamIDNachher)
	}
	if _, _, _, ok := assignmentByUser(t, db, neuerSlot, userID); !ok {
		t.Fatalf("Zusage von Nutzer %d hat den Wechsel team_id=%d -> NULL nicht überlebt", userID, teamA)
	}
	for _, uid := range summary.NotifiedUsers {
		if uid == userID {
			t.Errorf("erwartet keine 'entfernt'-Meldung beim Übergang, bekam %v", summary.NotifiedUsers)
		}
	}
	var filled int
	db.QueryRow(`SELECT slots_filled FROM duty_slots WHERE id=?`, neuerSlot).Scan(&filled)
	if filled != 1 {
		t.Errorf("slots_filled soll die wiederhergestellte Zusage zählen, ist %d", filled)
	}
}
