package duties_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Dauer-Invariante am Diensttyp (dienst-dauer-dynamisch, Aufgabe 9.3): die
// garantierte Zusage des Changes ist „ein Slot trägt nach jedem Regen-Lauf eine
// Dauer > 0". Slot- und Vorlagen-Routen prüfen dafür jeweils ihre eigene
// Eingabe — der Diensttyp war die letzte offene Stelle: seine Dauer wandert per
// Copy-on-pick in die Vorlagen-Zeile und von dort in den Slot, ohne dass sie
// dabei noch einmal explizit gesendet und geprüft würde.

// typeServer registriert die beiden Typ-Schreibrouten im Vorstand-Tier wie in
// BuildRouter; testServer aus handler_test.go führt sie bewusst nicht.
func typeServer(t *testing.T, h *duties.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand"))
			r.Post("/api/duty-types", h.CreateType)
			r.Put("/api/duty-types/{id}", h.UpdateType)
		})
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "trainer", "sportliche_leitung"))
			r.Get("/api/duty-types", h.ListTypes)
		})
	})
}

func typeHours(t *testing.T, srv *httptest.Server, token string, id int) float64 {
	t.Helper()
	res := testutil.Get(t, srv, "/api/duty-types", token)
	defer res.Body.Close()
	var list []struct {
		ID         int     `json:"id"`
		HoursValue float64 `json:"hours_value"`
	}
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatalf("decode duty-types: %v", err)
	}
	for _, dt := range list {
		if dt.ID == id {
			return dt.HoursValue
		}
	}
	t.Fatalf("Diensttyp %d nicht in der Liste", id)
	return 0
}

// TestCreateType_DauerNullWirdAbgewiesen: eine explizit gesendete Dauer ≤ 0
// endet mit 400, und es entsteht kein Diensttyp.
func TestCreateType_DauerNullWirdAbgewiesen(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	for _, hours := range []float64{0, -1.5} {
		res := testutil.Post(t, srv, "/api/duty-types", token, map[string]any{
			"name": "Kaputt", "hours_value": hours, "default_anchor": "start",
		})
		res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("hours_value=%v: erwartet 400, bekam %d", hours, res.StatusCode)
		}
	}
	if n := countRows(t, db, "duty_types", "name='Kaputt'"); n != 0 {
		t.Errorf("erwartet keinen angelegten Diensttyp, fand %d", n)
	}
}

// TestCreateType_OhneDauerNutztDBDefault: fehlt das Feld, gilt der Default der
// DB-Spalte (1.0) — dieselbe Regel wie bei default_anchor/duration_mode. Vorher
// landete hier eine stille 0, weil die Spalte explizit mitgeschrieben wird und
// der Zero-Value des Decoders 0 ist.
func TestCreateType_OhneDauerNutztDBDefault(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	res := testutil.Post(t, srv, "/api/duty-types", token, map[string]any{
		"name": "Ohne Dauer", "default_anchor": "start",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("erwartet 201, bekam %d", res.StatusCode)
	}
	var hours float64
	if err := db.QueryRow(`SELECT hours_value FROM duty_types WHERE name='Ohne Dauer'`).Scan(&hours); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if hours != 1.0 {
		t.Errorf("erwartet DB-Default 1.0, bekam %v", hours)
	}
}

// TestUpdateType_DauerNullWirdAbgewiesen: 400 vor dem Schreiben — der Bestand
// bleibt vollständig unangetastet (auch der Name, der im Request mitkommt).
func TestUpdateType_DauerNullWirdAbgewiesen(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	dtID := createDutyType(t, db, "Zeitnehmer", 2.5)
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	res := testutil.Put(t, srv, "/api/duty-types/"+itoa(dtID), token, map[string]any{
		"name": "Umbenannt", "hours_value": 0, "default_anchor": "start",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("erwartet 400, bekam %d", res.StatusCode)
	}
	var name string
	var hours float64
	if err := db.QueryRow(`SELECT name, hours_value FROM duty_types WHERE id=?`, dtID).Scan(&name, &hours); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if name != "Zeitnehmer" || hours != 2.5 {
		t.Errorf("erwartet unveränderten Bestand (Zeitnehmer/2.5), bekam %q/%v", name, hours)
	}
}

// TestUpdateType_DauerBleibtLesbar: der Happy-Path schreibt die Dauer und
// ListTypes liefert sie zurück (die Route ist ein voller Replace, kein Patch).
func TestUpdateType_DauerBleibtLesbar(t *testing.T) {
	db := testutil.NewDB(t)
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	dtID := createDutyType(t, db, "Zeitnehmer", 1.0)
	userID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, userID, "standard", []string{"vorstand"})
	srv := typeServer(t, h)

	res := testutil.Put(t, srv, "/api/duty-types/"+itoa(dtID), token, map[string]any{
		"name": "Zeitnehmer", "hours_value": 2.25, "default_anchor": "start",
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("erwartet 204, bekam %d", res.StatusCode)
	}
	if got := typeHours(t, srv, token, dtID); got != 2.25 {
		t.Errorf("erwartet 2.25, bekam %v", got)
	}
}
