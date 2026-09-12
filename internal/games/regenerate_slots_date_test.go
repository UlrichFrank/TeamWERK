package games

// Regressionstest für die SQLite-DATE-Gotcha (docs/agent/06-gotchas.md) in
// RegenerateSlots (Betriebshärtung Welle 2, design.md Entscheidung 5): der
// Handler liest games.date über ein generisches SELECT und reicht den
// gescannten String unverändert an runAutoRegen weiter. Liefert der Scan einen
// ISO-Timestamp statt der reinen "2006-01-02"-Form, matcht loadDayGames'
// eigenes WHERE date=? nie gegen die Bestandszeile — der Lauf tut still gar
// nichts (kein Fehler, keine Slots).

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestRegenerateSlots_ErzeugtSlotsAuchBeiISODatum ruft den echten HTTP-Handler
// (nicht runAutoRegen direkt) auf, weil der Bug im Handler selbst sitzt, nicht
// in der Regen-Engine.
func TestRegenerateSlots_ErzeugtSlotsAuchBeiISODatum(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	seedAgeClassRuleI(t, db, teamID)
	h := newRegenTestHandler(t, db)

	dutyID := insertDutyTypeI(t, db, "Kuchen", "", 0, "", 0)
	templateID := insertTemplateI(t, db, dutyID, 0, 1)
	gameID := insertGameI(t, db, seasonID, teamID, "2026-10-03", "12:00", templateID)

	req := httptest.NewRequest(http.MethodPost, "/api/games/"+strconv.Itoa(gameID)+"/regenerate", nil)
	req.SetPathValue("id", strconv.Itoa(gameID))
	w := httptest.NewRecorder()

	h.RegenerateSlots(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("RegenerateSlots: erwartet 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM duty_slots WHERE game_id=?`, gameID).Scan(&count); err != nil {
		t.Fatalf("count duty_slots: %v", err)
	}
	if count == 0 {
		t.Error("erwartet mindestens einen Duty-Slot nach RegenerateSlots — " +
			"games.date muss vor der Weitergabe an runAutoRegen auf die reine Datumsform " +
			"getruncatet werden (SQLite-DATE-Gotcha, docs/agent/06-gotchas.md)")
	}
}
