package games

// Regen-Tests für die Dienst-Dauer (Change dienst-dauer): duty_types.hours_value
// ist ab hier die DAUER eines Dienstes und zugleich seine Gutschrift. Die Zahl
// wird über die Vorlagen-Zeile in den Slot materialisiert; bei einer reduzierten
// Variante gewinnt der Varianten-Typ (Decision 3).
//
// Geteilte Test-Helfer (insertDutyTypeI, insertTemplateI, insertGameI, regenDate,
// seedAgeClassRuleI, assignI, …) stammen aus regen_bulk_context_test.go bzw.
// regen_bulk_restore_test.go — siehe dort für die Begründung des internen
// Test-Packages.

import (
	"database/sql"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// setItemHours setzt die Dauer der (einzigen) Zeile einer Vorlage.
func setItemHours(t *testing.T, db *sql.DB, templateID int, hours float64) {
	t.Helper()
	if _, err := db.Exec(
		`UPDATE game_template_items SET hours_value=? WHERE template_id=?`, hours, templateID); err != nil {
		t.Fatalf("setItemHours: %v", err)
	}
}

// slotHours liest die materialisierte Dauer des Auto-Slots eines Spiels.
func slotHours(t *testing.T, db *sql.DB, gameID int) float64 {
	t.Helper()
	var h float64
	if err := db.QueryRow(
		`SELECT hours_value FROM duty_slots WHERE game_id=? AND is_custom=0`, gameID).Scan(&h); err != nil {
		t.Fatalf("slotHours(game=%d): %v", gameID, err)
	}
	return h
}

// TestRegen_SlotErbtHoursValueAusVorlage (tasks 2.5): der erzeugte Slot trägt die
// Dauer der VORLAGEN-ZEILE, nicht die des Diensttyps. Copy-on-pick heißt: die
// Zeile ist nach dem Anlegen eigenständig — eine spätere Änderung am Typ erreicht
// sie nicht mehr.
func TestRegen_SlotErbtHoursValueAusVorlage(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0) // hours_value = 2.0
	templateID := insertTemplateI(t, db, dutyID, 0, 1)
	setItemHours(t, db, templateID, 1.5)

	// Der Typ ändert sich NACH dem Anlegen der Vorlagen-Zeile.
	if _, err := db.Exec(`UPDATE duty_types SET hours_value=5.0 WHERE id=?`, dutyID); err != nil {
		t.Fatalf("update duty type hours: %v", err)
	}

	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	if got := slotHours(t, db, gameID); got != 1.5 {
		t.Errorf("Slot soll die Dauer der Vorlagen-Zeile tragen (1.5), nicht die des Typs (5.0): got %v", got)
	}
}

// TestRegen_ReducedVarianteBestimmtDauer (tasks 2.6): greift die Varianten-Logik,
// entsteht der Slot unter einem ANDEREN Diensttyp — dann gewinnt dessen Dauer
// (Decision 3: die Variante ist eine andere Arbeit, und die Dauer gehört zur
// Arbeit). Position und Personenzahl stammen weiterhin aus der Vorlagen-Zeile.
func TestRegen_ReducedVarianteBestimmtDauer(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	variantID := insertDutyTypeI(t, db, "Kuchen (Folgespiel)", "", 0, "", 0)
	if _, err := db.Exec(`UPDATE duty_types SET hours_value=0.5 WHERE id=?`, variantID); err != nil {
		t.Fatalf("update variant hours: %v", err)
	}
	dutyID := insertDutyTypeI(t, db, "Kuchen", "reduced", variantID, "", 0)

	templateID := insertTemplateI(t, db, dutyID, 60, 3)
	setItemHours(t, db, templateID, 4.0)

	// Offset +60min: der Slot von Spiel 1 landet um 11:00 ZWISCHEN beiden Spielen
	// (10:00 und 16:00) — nur dort greift same_day_behavior. Der Slot von Spiel 2
	// liegt um 17:00 nach allen Spielen und bleibt beim Original-Typ.
	// Zwei Heimspiele am selben Tag → same_day_behavior='reduced' greift für das
	// zweite. Das erste behält den Original-Typ.
	game1 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "10:00", templateID)
	game2 := insertGameI(t, db, seasonID, teamID, "2026-06-13", "16:00", templateID)
	regenDate(t, h, db, "2026-06-13", seasonID, nil)

	var reducedGame, normalGame int
	for _, gid := range []int{game1, game2} {
		var typeID int
		if err := db.QueryRow(
			`SELECT duty_type_id FROM duty_slots WHERE game_id=? AND is_custom=0`, gid).Scan(&typeID); err != nil {
			t.Fatalf("read slot duty_type (game=%d): %v", gid, err)
		}
		if typeID == variantID {
			reducedGame = gid
		} else {
			normalGame = gid
		}
	}
	if reducedGame == 0 || normalGame == 0 {
		t.Fatalf("erwartet je ein Slot mit Original- und Varianten-Typ, got reduced=%d normal=%d", reducedGame, normalGame)
	}

	if got := slotHours(t, db, reducedGame); got != 0.5 {
		t.Errorf("Varianten-Slot soll die Dauer des Varianten-Typs tragen (0.5), nicht die der Vorlage (4.0): got %v", got)
	}
	if got := slotHours(t, db, normalGame); got != 4.0 {
		t.Errorf("Nicht-reduzierter Slot soll die Vorlagen-Dauer tragen (4.0): got %v", got)
	}

	// Position und Personenzahl bleiben Sache der Vorlagen-Zeile.
	var slotsTotal int
	if err := db.QueryRow(
		`SELECT slots_total FROM duty_slots WHERE game_id=? AND is_custom=0`, reducedGame).Scan(&slotsTotal); err != nil {
		t.Fatalf("read slots_total: %v", err)
	}
	if slotsTotal != 3 {
		t.Errorf("slots_total soll aus der Vorlagen-Zeile stammen (3), got %d", slotsTotal)
	}
}

// TestRegen_DauerAenderungErhaeltZusagen (tasks 2.7): eine geänderte Dauer
// verschiebt event_time nicht und berührt damit den Restore-Schlüssel
// (duty_type_id, event_time, team_id) nicht — alle Zusagen überleben den Lauf,
// und es entsteht keine "entfernt"-Benachrichtigung.
func TestRegen_DauerAenderungErhaeltZusagen(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyID, 0, 1)
	setItemHours(t, db, templateID, 1.0)
	gameID := insertGameI(t, db, seasonID, teamID, "2026-06-13", "12:00", templateID)

	regenDate(t, h, db, "2026-06-13", seasonID, nil)
	slotID := autoSlotID(t, db, gameID)
	userID := testutil.CreateUser(t, db, "standard")
	assignI(t, db, slotID, userID, "assigned", nil, nil)

	setItemHours(t, db, templateID, 2.5)
	summary := regenDate(t, h, db, "2026-06-13", seasonID, nil)

	newSlotID := autoSlotID(t, db, gameID)
	if got := slotHours(t, db, gameID); got != 2.5 {
		t.Errorf("neuer Slot soll die geänderte Dauer tragen (2.5), got %v", got)
	}
	if _, _, _, found := assignmentByUser(t, db, newSlotID, userID); !found {
		t.Error("Zusage soll den Regen nach einer reinen Dauer-Änderung überleben")
	}
	for _, n := range summary.Notifications {
		if n.Kind != "variant_changed" {
			t.Errorf("keine Entfernt-Benachrichtigung erwartet, got %+v", n)
		}
	}

	var filled int
	if err := db.QueryRow(`SELECT slots_filled FROM duty_slots WHERE id=?`, newSlotID).Scan(&filled); err != nil {
		t.Fatalf("read slots_filled: %v", err)
	}
	if filled != 1 {
		t.Errorf("slots_filled soll nach dem Restore 1 sein, got %d", filled)
	}
}
