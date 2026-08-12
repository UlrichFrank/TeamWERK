package games_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Tests für den Rotations-Schalter eines Vorlagen-Items
// (game_template_items.rotation_enabled, kuchendienst-rotation +
// bewirtung-cap-global). 0 (Default) lässt das bestehende Verhalten unverändert;
// 1 aktiviert den Rotations-Modus und setzt same_day_behavior='normal' UND
// adjacent_day_behavior='normal' auf dem referenzierten Duty-Type voraus. Die
// Obergrenze pro Mannschaft steckt nicht mehr hier, sondern vereinsweit in
// system_settings (siehe internal/settings/bewirtung_test.go).

// insertDutyTypeWithBehavior legt einen duty_types-Eintrag mit explizitem
// same_day_behavior/adjacent_day_behavior an — Geschwister von insertDutyType
// (handler_test.go), das immer bei den 'normal'-Defaults bleibt.
func insertDutyTypeWithBehavior(t *testing.T, db *sql.DB, name, sameDayBehavior, adjacentDayBehavior string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO duty_types (name, hours_value, same_day_behavior, adjacent_day_behavior) VALUES (?, 2.0, ?, ?)`,
		name, sameDayBehavior, adjacentDayBehavior)
	if err != nil {
		t.Fatalf("insertDutyTypeWithBehavior: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// TestUpdateTemplate_RotationEnabled_GespeichertBeiNormalBehavior: ein
// Vorstand kann die Rotation aktivieren, wenn der Duty-Type same_day_behavior
// UND adjacent_day_behavior auf 'normal' stehen (Bestandswert von 'Kuchen').
func TestUpdateTemplate_RotationEnabled_GespeichertBeiNormalBehavior(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyTypeWithBehavior(t, db, "Kuchen", "normal", "normal")
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"rotation_enabled": true,
		}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var stored bool
	if err := db.QueryRow(
		`SELECT rotation_enabled FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored); err != nil {
		t.Fatalf("read rotation_enabled: %v", err)
	}
	if !stored {
		t.Error("expected rotation_enabled=1")
	}

	items := templateItemsFromAPI(t, srv, templateID, token)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if got, ok := items[0]["rotation_enabled"].(bool); !ok || !got {
		t.Errorf("expected rotation_enabled=true in response, got %#v", items[0]["rotation_enabled"])
	}
}

// TestUpdateTemplate_RotationEnabled_LehntAbweichendesSameDayBehaviorAb:
// same_day_behavior != 'normal' + aktivierte Rotation → 400 rotation_requires_normal_behavior,
// die gesamte Vorlage bleibt unverändert (kein Teil-Write).
func TestUpdateTemplate_RotationEnabled_LehntAbweichendesSameDayBehaviorAb(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyTypeWithBehavior(t, db, "Kuchen", "skip", "normal")
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Umbenannt", "template_type": "heim", "duration_minutes": 90,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"rotation_enabled": true,
		}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&errBody)
	if errBody.Error != "rotation_requires_normal_behavior" {
		t.Errorf("expected error rotation_requires_normal_behavior, got %q", errBody.Error)
	}

	// Nichts wurde persistiert — weder Name noch der Rotations-Schalter.
	var name string
	db.QueryRow(`SELECT name FROM game_templates WHERE id=?`, templateID).Scan(&name)
	if name != "Heim" {
		t.Errorf("expected template name unchanged (Heim), got %q", name)
	}
	var stored bool
	db.QueryRow(`SELECT rotation_enabled FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored)
	if stored {
		t.Error("expected rotation_enabled still 0")
	}
}

// TestUpdateTemplate_RotationEnabled_LehntAbweichendesAdjacentDayBehaviorAb:
// dieselbe Ablehnung gilt für adjacent_day_behavior != 'normal'.
func TestUpdateTemplate_RotationEnabled_LehntAbweichendesAdjacentDayBehaviorAb(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyTypeWithBehavior(t, db, "Kuchen", "normal", "reduced")
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"rotation_enabled": true,
		}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&errBody)
	if errBody.Error != "rotation_requires_normal_behavior" {
		t.Errorf("expected error rotation_requires_normal_behavior, got %q", errBody.Error)
	}
}

// TestUpdateTemplate_RotationEnabled_StandardNutzerVerboten: ohne
// Vereinsfunktion vorstand bleibt PUT gesperrt (403), auch wenn der Request
// ein rotation_enabled-Feld trägt.
func TestUpdateTemplate_RotationEnabled_StandardNutzerVerboten(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyTypeWithBehavior(t, db, "Kuchen", "normal", "normal")
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	userID := testutil.CreateUser(t, db, "standard")
	srv := testServer(t, db)
	token := testutil.Token(t, userID, "standard", []string{"spieler"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Gekapert", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"rotation_enabled": true,
		}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	var stored bool
	db.QueryRow(`SELECT rotation_enabled FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored)
	if stored {
		t.Error("expected rotation_enabled still 0")
	}
}
