package games_test

// HTTP-Tests für das Ablösungs-Kennzeichen an der Vorlagen-Zeile (Change
// dienst-abloesung). `game_template_items.end_at_next_duty` folgt derselben
// Copy-on-pick-Regel wie hours_value/duration_mode/end_anchor/end_offset_minutes:
// fehlt das Feld im Request, erbt die Zeile den Wert ihres Diensttyps; ist es gesetzt,
// führt die Zeile ihn eigenständig weiter.

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestUpdateTemplate_AbloesungJeItemWirdPersistiert: Happy-Path — die Zeile behält ihr
// eigenes Kennzeichen, auch wenn der Diensttyp das Gegenteil sagt.
func TestUpdateTemplate_AbloesungJeItemWirdPersistiert(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyTypeWithBehavior(t, db, "Kuchen", "normal", "normal")
	// Der Typ sagt „lässt sich nicht ablösen" — die Zeile darf trotzdem anders wollen.
	if _, err := db.Exec(`UPDATE duty_types SET end_at_next_duty=0 WHERE id=?`, dutyTypeID); err != nil {
		t.Fatalf("seed duty type: %v", err)
	}
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -30, "slots_count": 1,
			"duration_mode": "dynamisch", "end_anchor": "end", "end_offset_minutes": 30,
			"end_at_next_duty": true,
		}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, got %d", res.StatusCode)
	}

	var stored bool
	if err := db.QueryRow(
		`SELECT end_at_next_duty FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored); err != nil {
		t.Fatalf("read end_at_next_duty: %v", err)
	}
	if !stored {
		t.Error("die Vorlagen-Zeile soll ihr eigenes Kennzeichen tragen (true), nicht das des Typs (false)")
	}
}

// TestUpdateTemplate_AbloesungOhneFeldErbtVomTyp: ein Client, der das Feld nicht kennt,
// bekommt den Wert des Diensttyps eingesetzt — nicht stumm `false`.
func TestUpdateTemplate_AbloesungOhneFeldErbtVomTyp(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyTypeWithBehavior(t, db, "Kuchen", "normal", "normal")
	if _, err := db.Exec(`UPDATE duty_types SET end_at_next_duty=1 WHERE id=?`, dutyTypeID); err != nil {
		t.Fatalf("seed duty type: %v", err)
	}
	templateID := seedTeamScopeTemplate(t, db, "Heim", dutyTypeID, nil)

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -30, "slots_count": 1,
		}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, got %d", res.StatusCode)
	}

	var stored bool
	if err := db.QueryRow(
		`SELECT end_at_next_duty FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored); err != nil {
		t.Fatalf("read end_at_next_duty: %v", err)
	}
	if !stored {
		t.Error("fehlendes Feld soll per Copy-on-pick vom Diensttyp erben (true), nicht auf false fallen")
	}
}

// TestUpdateTemplate_AbloesungOhneVorstandsrecht: Fehlerfall — ein eingeloggter Nutzer
// ohne Vereinsfunktion kommt nicht an die Route, der Bestand bleibt unverändert.
func TestUpdateTemplate_AbloesungOhneVorstandsrecht(t *testing.T) {
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
			"offset_minutes": -30, "slots_count": 1, "end_at_next_duty": true,
		}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("erwartet 403, got %d", res.StatusCode)
	}

	var name string
	var stored bool
	if err := db.QueryRow(`
		SELECT gt.name, gti.end_at_next_duty
		FROM game_templates gt JOIN game_template_items gti ON gti.template_id = gt.id
		WHERE gt.id=?`, templateID).Scan(&name, &stored); err != nil {
		t.Fatalf("read template: %v", err)
	}
	if name != "Heim" || stored {
		t.Errorf("erwartet unveränderten Bestand (Heim/false), bekam %q/%v", name, stored)
	}
}
