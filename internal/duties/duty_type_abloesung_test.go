package duties_test

// HTTP-Tests für das Ablösungs-Kennzeichen am Diensttyp (Change dienst-abloesung).
//
// `end_at_next_duty` ist bewusst das erste Dauer-Feld OHNE eigene Validierung: ein bool
// kennt keinen ungültigen Wert, und das Kennzeichen kann eine Definition auch nicht
// unmöglich machen (es verkürzt nur und sieht nur Nachfolger nach dem eigenen Start).
// Zu prüfen bleibt deshalb genau die Persistenz — inklusive des Falls „im Modus absolut
// bedeutungslos, aber gespeichert", der einen Moduswechsel hin und zurück überleben soll.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func typeEndAtNextDuty(t *testing.T, srv *httptest.Server, token string, id int) bool {
	t.Helper()
	res := testutil.Get(t, srv, "/api/duty-types", token)
	defer res.Body.Close()
	var list []struct {
		ID            int  `json:"id"`
		EndAtNextDuty bool `json:"end_at_next_duty"`
	}
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatalf("decode duty-types: %v", err)
	}
	for _, dt := range list {
		if dt.ID == id {
			return dt.EndAtNextDuty
		}
	}
	t.Fatalf("Diensttyp %d nicht in der Liste", id)
	return false
}

// TestCreateType_AbloesungWirdPersistiert: Happy-Path — das Kennzeichen kommt im
// Request, landet in der Spalte und wird von ListTypes wieder ausgeliefert (die Maske
// braucht es, um das Häkchen zu setzen).
func TestCreateType_AbloesungWirdPersistiert(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	res := testutil.Post(t, srv, "/api/duty-types", token, map[string]any{
		"name": "Kuchenverkauf", "hours_value": 2.0, "default_anchor": "start",
		"default_offset_minutes": -30,
		"duration_mode":          "dynamisch", "end_anchor": "end", "end_offset_minutes": 30,
		"end_at_next_duty": true,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekam %d", res.StatusCode)
	}

	var id int
	var stored bool
	if err := db.QueryRow(
		`SELECT id, end_at_next_duty FROM duty_types WHERE name='Kuchenverkauf'`).Scan(&id, &stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !stored {
		t.Error("end_at_next_duty soll gespeichert sein")
	}
	if got := typeEndAtNextDuty(t, srv, token, id); !got {
		t.Error("ListTypes soll end_at_next_duty=true liefern")
	}
}

// TestCreateType_AbloesungOhneDynamischenModusWirdGespeichertAberIgnoriert: im Modus
// `absolut` ist das Kennzeichen bedeutungslos — abgewiesen wird es trotzdem nicht,
// sondern gespeichert, damit ein Moduswechsel hin und zurück den Wert nicht verliert
// (dieselbe Regel wie für end_anchor/end_offset_minutes).
func TestCreateType_AbloesungOhneDynamischenModusWirdGespeichertAberIgnoriert(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	res := testutil.Post(t, srv, "/api/duty-types", token, map[string]any{
		"name": "Absolut mit Haken", "hours_value": 2.0, "default_anchor": "start",
		"duration_mode": "absolut", "end_at_next_duty": true,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201 (kein 400 — das Kennzeichen ist im absoluten Modus nur wirkungslos), bekam %d", res.StatusCode)
	}
	var mode string
	var stored bool
	if err := db.QueryRow(
		`SELECT duration_mode, end_at_next_duty FROM duty_types WHERE name='Absolut mit Haken'`).
		Scan(&mode, &stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if mode != "absolut" || !stored {
		t.Errorf("erwartet absolut/true, bekam %q/%v", mode, stored)
	}
}

// TestUpdateType_AbloesungWirdPersistiert: die Route ist ein voller Replace — das
// Kennzeichen lässt sich in beide Richtungen umschalten.
func TestUpdateType_AbloesungWirdPersistiert(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	dtID := createDutyType(t, db, "Kuchenverkauf", 2.0)
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	body := map[string]any{
		"name": "Kuchenverkauf", "hours_value": 2.0, "default_anchor": "start",
		"duration_mode": "dynamisch", "end_anchor": "end", "end_offset_minutes": 30,
		"end_at_next_duty": true,
	}
	res := testutil.Put(t, srv, "/api/duty-types/"+itoa(dtID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekam %d", res.StatusCode)
	}
	if got := typeEndAtNextDuty(t, srv, token, dtID); !got {
		t.Fatal("erwartet end_at_next_duty=true nach dem Setzen")
	}

	body["end_at_next_duty"] = false
	res = testutil.Put(t, srv, "/api/duty-types/"+itoa(dtID), token, body)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204 beim Zurücksetzen, bekam %d", res.StatusCode)
	}
	if got := typeEndAtNextDuty(t, srv, token, dtID); got {
		t.Error("erwartet end_at_next_duty=false nach dem Zurücksetzen")
	}
}

// TestUpdateType_AbloesungUnauthentifiziert: Fehlerfall der Schreibroute — ohne Token
// kein Zugriff, und der Bestand bleibt unangetastet.
func TestUpdateType_AbloesungUnauthentifiziert(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	dtID := createDutyType(t, db, "Kuchenverkauf", 2.0)
	srv := typeServer(t, h)

	res := testutil.Put(t, srv, "/api/duty-types/"+itoa(dtID), "", map[string]any{
		"name": "Gekapert", "hours_value": 2.0, "default_anchor": "start",
		"end_at_next_duty": true,
	})
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("erwartet 401, bekam %d", res.StatusCode)
	}
	var name string
	var stored bool
	if err := db.QueryRow(
		`SELECT name, end_at_next_duty FROM duty_types WHERE id=?`, dtID).Scan(&name, &stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if name != "Kuchenverkauf" || stored {
		t.Errorf("erwartet unveränderten Bestand, bekam %q/%v", name, stored)
	}
}
