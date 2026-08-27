package games_test

// HTTP-Tests für die Dauer an der Vorlagen-Zeile (Change dienst-dauer):
// game_template_items.hours_value ist Copy-on-pick — der Wert der Zeile gilt,
// unabhängig davon, was der Diensttyp inzwischen sagt. Ein fehlendes Feld erbt
// einmalig vom Typ (Alt-Client), eine Null oder ein negativer Wert ist ungültig.

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestUpdateTemplate_SpeichertHoursValueJeItem (tasks 3.3): die Zeile behält ihre
// eigene Dauer, auch wenn der Diensttyp einen anderen Wert trägt.
func TestUpdateTemplate_SpeichertHoursValueJeItem(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyTypeWithBehavior(t, db, "Kuchen", "normal", "normal") // hours_value = 2.0
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"hours_value": 1.5,
		}},
	})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, got %d", res.StatusCode)
	}

	var stored float64
	if err := db.QueryRow(
		`SELECT hours_value FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored); err != nil {
		t.Fatalf("read hours_value: %v", err)
	}
	if stored != 1.5 {
		t.Errorf("Vorlagen-Zeile soll ihre eigene Dauer tragen (1.5), nicht die des Typs (2.0): got %v", stored)
	}
}

// TestUpdateTemplate_OhneHoursValueErbtVomTyp: ein Client, der das Feld nicht
// kennt, bekommt die Dauer des Diensttyps eingesetzt — nicht 0.
func TestUpdateTemplate_OhneHoursValueErbtVomTyp(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyTypeWithBehavior(t, db, "Kuchen", "normal", "normal") // hours_value = 2.0
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
		}},
	})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, got %d", res.StatusCode)
	}

	var stored float64
	if err := db.QueryRow(
		`SELECT hours_value FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored); err != nil {
		t.Fatalf("read hours_value: %v", err)
	}
	if stored != 2.0 {
		t.Errorf("fehlendes hours_value soll vom Typ erben (2.0), got %v", stored)
	}
}

// TestUpdateTemplate_HoursValueNullOderNegativ_400 (tasks 3.4): ungültige Dauer
// wird abgelehnt, BEVOR die Transaktion die Bestandszeilen löscht — sonst
// hinterließe das 400 eine leergeräumte Vorlage.
func TestUpdateTemplate_HoursValueNullOderNegativ_400(t *testing.T) {
	for _, tc := range []struct {
		name  string
		hours float64
	}{
		{"null", 0},
		{"negativ", -1.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			dutyTypeID := insertDutyTypeWithBehavior(t, db, "Kuchen", "normal", "normal")
			templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

			var before int
			if err := db.QueryRow(
				`SELECT COUNT(*) FROM game_template_items WHERE template_id=?`, templateID).Scan(&before); err != nil {
				t.Fatalf("count before: %v", err)
			}

			adminUserID := testutil.CreateUser(t, db, "admin")
			srv := testServer(t, db)
			token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

			res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
				"name": "Heim", "template_type": "heim", "duration_minutes": 75,
				"items": []map[string]any{{
					"duty_type_id": dutyTypeID, "anchor": "start",
					"offset_minutes": -60, "slots_count": 1,
					"hours_value": tc.hours,
				}},
			})
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("erwartet 400, got %d", res.StatusCode)
			}

			var after int
			if err := db.QueryRow(
				`SELECT COUNT(*) FROM game_template_items WHERE template_id=?`, templateID).Scan(&after); err != nil {
				t.Fatalf("count after: %v", err)
			}
			if after != before {
				t.Errorf("400 darf keine Teil-Persistenz hinterlassen: %d Zeilen vorher, %d nachher", before, after)
			}
		})
	}
}
