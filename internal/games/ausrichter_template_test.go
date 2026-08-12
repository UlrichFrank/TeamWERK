package games_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Tests für die optionale Ausrichter-Bindung eines Vorlagen-Items
// (game_template_items.ausrichter_id, heimspieltag-ausrichter). NULL (Default)
// lässt das bestehende Verhalten unverändert (Item gilt an jedem Spieltag);
// ein gesetzter Wert schränkt die Erzeugung auf Spieltage mit passendem
// aufgelöstem Tages-Ausrichter ein (Auswertung des Gates selbst: regen.go,
// Gruppe 5). Hier wird nur die CRUD-/Validierungsseite in handler.go geprüft.

// insertAusrichter legt eine zusätzliche Ausrichter-Zeile an (neben dem
// Default, den Migration 048 seedet) und gibt ihre ID zurück.
func insertAusrichter(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO ausrichter (name) VALUES (?)`, name)
	if err != nil {
		t.Fatalf("insertAusrichter: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// seedTemplateWithType legt eine leere Vorlage (ohne Items) mit dem
// gewünschten template_type an — Geschwister von seedTeamScopeTemplate, das
// fest auf 'heim' steht und hier für auswärts/generisch nicht passt.
func seedTemplateWithType(t *testing.T, db *sql.DB, name, templateType string) int {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO game_templates (name, template_type, duration_minutes) VALUES (?, ?, 75)`,
		name, templateType)
	if err != nil {
		t.Fatalf("seedTemplateWithType: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// TestUpdateTemplate_AusrichterId_GespeichertUndGeladen: ein gesetztes
// ausrichter_id auf einer Heim-Vorlage wird persistiert und kommt über GET
// unverändert zurück.
func TestUpdateTemplate_AusrichterId_GespeichertUndGeladen(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyType(t, db, "Kuchen", 2.0)
	ausrichterID := insertAusrichter(t, db, "TV Cannstatt")
	templateID := seedTemplateWithType(t, db, "Heim", "heim")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"ausrichter_id": ausrichterID,
		}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var stored sql.NullInt64
	if err := db.QueryRow(
		`SELECT ausrichter_id FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored); err != nil {
		t.Fatalf("read ausrichter_id: %v", err)
	}
	if !stored.Valid || int(stored.Int64) != ausrichterID {
		t.Errorf("expected ausrichter_id=%d, got %#v", ausrichterID, stored)
	}

	items := templateItemsFromAPI(t, srv, templateID, token)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	gotAusrichter, ok := items[0]["ausrichter_id"].(float64)
	if !ok || int(gotAusrichter) != ausrichterID {
		t.Errorf("expected ausrichter_id=%d in response, got %#v", ausrichterID, items[0]["ausrichter_id"])
	}
}

// TestUpdateTemplate_AusrichterId_OhneWert_BleibtNull: ein Item ohne
// ausrichter_id bleibt NULL — das bisherige, ungebundene Verhalten ist
// unverändert erreichbar.
func TestUpdateTemplate_AusrichterId_OhneWert_BleibtNull(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyType(t, db, "Kuchen", 2.0)
	templateID := seedTemplateWithType(t, db, "Heim", "heim")

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
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	var stored sql.NullInt64
	if err := db.QueryRow(
		`SELECT ausrichter_id FROM game_template_items WHERE template_id=?`, templateID).Scan(&stored); err != nil {
		t.Fatalf("read ausrichter_id: %v", err)
	}
	if stored.Valid {
		t.Errorf("expected ausrichter_id NULL, got %v", stored.Int64)
	}

	items := templateItemsFromAPI(t, srv, templateID, token)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if v, present := items[0]["ausrichter_id"]; present && v != nil {
		t.Errorf("expected ausrichter_id absent/null in response, got %#v", v)
	}
}

// TestUpdateTemplate_AusrichterId_LehntAuswaertsVorlageAb: ausrichter_id auf
// einer auswärts-Vorlage → 400 ausrichter_requires_heim_template, nichts wird
// geschrieben.
func TestUpdateTemplate_AusrichterId_LehntAuswaertsVorlageAb(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyType(t, db, "Kuchen", 2.0)
	ausrichterID := insertAusrichter(t, db, "TV Cannstatt")
	templateID := seedTemplateWithType(t, db, "Auswärts", "auswärts")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Umbenannt", "template_type": "auswärts", "duration_minutes": 90,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"ausrichter_id": ausrichterID,
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
	if errBody.Error != "ausrichter_requires_heim_template" {
		t.Errorf("expected error ausrichter_requires_heim_template, got %q", errBody.Error)
	}

	var name string
	db.QueryRow(`SELECT name FROM game_templates WHERE id=?`, templateID).Scan(&name)
	if name != "Auswärts" {
		t.Errorf("expected template name unchanged (Auswärts), got %q", name)
	}
	var itemCount int
	db.QueryRow(`SELECT COUNT(*) FROM game_template_items WHERE template_id=?`, templateID).Scan(&itemCount)
	if itemCount != 0 {
		t.Errorf("expected no item written, got %d", itemCount)
	}
}

// TestUpdateTemplate_AusrichterId_LehntGenerischeVorlageAb: dieselbe
// Ablehnung gilt für template_type='generisch'.
func TestUpdateTemplate_AusrichterId_LehntGenerischeVorlageAb(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyType(t, db, "Kuchen", 2.0)
	ausrichterID := insertAusrichter(t, db, "TV Cannstatt")
	templateID := seedTemplateWithType(t, db, "Generisch", "generisch")

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Generisch", "template_type": "generisch", "duration_minutes": 90,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"ausrichter_id": ausrichterID,
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
	if errBody.Error != "ausrichter_requires_heim_template" {
		t.Errorf("expected error ausrichter_requires_heim_template, got %q", errBody.Error)
	}

	var itemCount int
	db.QueryRow(`SELECT COUNT(*) FROM game_template_items WHERE template_id=?`, templateID).Scan(&itemCount)
	if itemCount != 0 {
		t.Errorf("expected no item written, got %d", itemCount)
	}
}

// TestUpdateTemplate_AusrichterId_LehntUnbekannteIdAb: ein ausrichter_id, das
// in der Tabelle ausrichter nicht existiert, wird abgelehnt — auch auf einer
// Heim-Vorlage, wo das Feld grundsätzlich zulässig wäre.
func TestUpdateTemplate_AusrichterId_LehntUnbekannteIdAb(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyType(t, db, "Kuchen", 2.0)
	templateID := seedTemplateWithType(t, db, "Heim", "heim")

	// Höchste vergebene ID + 1 ist garantiert unbekannt, unabhängig vom
	// Default-Seed aus Migration 048.
	var maxID int
	db.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM ausrichter`).Scan(&maxID)
	unknownID := maxID + 1000

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{{
			"duty_type_id": dutyTypeID, "anchor": "start",
			"offset_minutes": -60, "slots_count": 1,
			"ausrichter_id": unknownID,
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
	if errBody.Error != "invalid_ausrichter" {
		t.Errorf("expected error invalid_ausrichter, got %q", errBody.Error)
	}

	var itemCount int
	db.QueryRow(`SELECT COUNT(*) FROM game_template_items WHERE template_id=?`, templateID).Scan(&itemCount)
	if itemCount != 0 {
		t.Errorf("expected no item written, got %d", itemCount)
	}
}

// TestUpdateTemplate_AusrichterId_ZweitesItemUngueltig_NichtsGeschrieben:
// Regressionsschutz für die Reihenfolge der Validierung — sie muss VOR der
// Insert-Schleife über alle Items laufen (nicht erst beim Scheitern des
// jeweiligen Inserts), sonst hinterließe ein ungültiges zweites Item das
// bereits geschriebene erste Item als Teil-Persistenz.
func TestUpdateTemplate_AusrichterId_ZweitesItemUngueltig_NichtsGeschrieben(t *testing.T) {
	db := testutil.NewDB(t)
	dutyTypeID := insertDutyType(t, db, "Kuchen", 2.0)
	ausrichterID := insertAusrichter(t, db, "TV Cannstatt")
	templateID := seedTemplateWithType(t, db, "Heim", "heim")

	var maxID int
	db.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM ausrichter`).Scan(&maxID)
	unknownID := maxID + 1000

	adminUserID := testutil.CreateUser(t, db, "admin")
	srv := testServer(t, db)
	token := testutil.Token(t, adminUserID, "admin", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/duty-templates/"+strconv.Itoa(templateID), token, map[string]any{
		"name": "Heim", "template_type": "heim", "duration_minutes": 75,
		"items": []map[string]any{
			{
				"duty_type_id": dutyTypeID, "anchor": "start",
				"offset_minutes": -60, "slots_count": 1,
				"ausrichter_id": ausrichterID, // gültig — würde als erstes Item geschrieben, gäbe es keine Vorab-Prüfung
			},
			{
				"duty_type_id": dutyTypeID, "anchor": "start",
				"offset_minutes": -30, "slots_count": 1,
				"ausrichter_id": unknownID, // ungültig
			},
		},
	})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	json.NewDecoder(res.Body).Decode(&errBody)
	if errBody.Error != "invalid_ausrichter" {
		t.Errorf("expected error invalid_ausrichter, got %q", errBody.Error)
	}

	var itemCount int
	db.QueryRow(`SELECT COUNT(*) FROM game_template_items WHERE template_id=?`, templateID).Scan(&itemCount)
	if itemCount != 0 {
		t.Errorf("expected no item written at all, got %d", itemCount)
	}
}
