package games_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Tests für die optionale Rotations-Cap-Spalte eines Vorlagen-Items
// (game_template_items.rotation_max_per_team, kuchendienst-rotation).
// NULL (Default) lässt das bestehende Verhalten unverändert; ein gesetzter
// Wert aktiviert den Rotations-Modus und setzt same_day_behavior='normal'
// UND adjacent_day_behavior='normal' auf dem referenzierten Duty-Type voraus.

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

// TestUpdateTemplate_RotationMaxPerTeam_GespeichertBeiNormalBehavior: ein
// Vorstand kann den Rotations-Cap setzen, wenn der Duty-Type same_day_behavior
// UND adjacent_day_behavior auf 'normal' stehen (Bestandswert von 'Kuchen').
func TestUpdateTemplate_RotationMaxPerTeam_GespeichertBeiNormalBehavior(t *testing.T) {
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
			"rotation_max_per_team": 2,
		}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var stored sql.NullInt64
	if err := db.QueryRow(
		`SELECT rotation_max_per_team FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored); err != nil {
		t.Fatalf("read rotation_max_per_team: %v", err)
	}
	if !stored.Valid || stored.Int64 != 2 {
		t.Errorf("expected rotation_max_per_team=2, got valid=%v value=%v", stored.Valid, stored.Int64)
	}

	items := templateItemsFromAPI(t, srv, templateID, token)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	got, ok := items[0]["rotation_max_per_team"].(float64)
	if !ok || int(got) != 2 {
		t.Errorf("expected rotation_max_per_team=2 in response, got %#v", items[0]["rotation_max_per_team"])
	}
}

// TestUpdateTemplate_RotationMaxPerTeam_LehntAbweichendesSameDayBehaviorAb:
// same_day_behavior != 'normal' + gesetzter Cap → 400 rotation_requires_normal_behavior,
// die gesamte Vorlage bleibt unverändert (kein Teil-Write).
func TestUpdateTemplate_RotationMaxPerTeam_LehntAbweichendesSameDayBehaviorAb(t *testing.T) {
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
			"rotation_max_per_team": 2,
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

	// Nichts wurde persistiert — weder Name noch die (fehlende) rotation_max_per_team-Spalte.
	var name string
	db.QueryRow(`SELECT name FROM game_templates WHERE id=?`, templateID).Scan(&name)
	if name != "Heim" {
		t.Errorf("expected template name unchanged (Heim), got %q", name)
	}
	var stored sql.NullInt64
	db.QueryRow(`SELECT rotation_max_per_team FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored)
	if stored.Valid {
		t.Errorf("expected rotation_max_per_team still NULL, got %v", stored.Int64)
	}
}

// TestUpdateTemplate_RotationMaxPerTeam_LehntAbweichendesAdjacentDayBehaviorAb:
// dieselbe Ablehnung gilt für adjacent_day_behavior != 'normal'.
func TestUpdateTemplate_RotationMaxPerTeam_LehntAbweichendesAdjacentDayBehaviorAb(t *testing.T) {
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
			"rotation_max_per_team": 1,
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

// TestUpdateTemplate_RotationMaxPerTeam_StandardNutzerVerboten: ohne
// Vereinsfunktion vorstand bleibt PUT gesperrt (403), auch wenn der Request
// ein rotation_max_per_team-Feld trägt.
func TestUpdateTemplate_RotationMaxPerTeam_StandardNutzerVerboten(t *testing.T) {
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
			"rotation_max_per_team": 3,
		}},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}

	var stored sql.NullInt64
	db.QueryRow(`SELECT rotation_max_per_team FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored)
	if stored.Valid {
		t.Errorf("expected rotation_max_per_team still NULL, got %v", stored.Int64)
	}
}
