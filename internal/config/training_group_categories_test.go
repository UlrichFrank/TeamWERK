package config_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// newTGCServer verdrahtet die drei Trainingsgruppen-Kategorie-Routen in denselben
// Auth-Tiers wie im Produktiv-Router: GET für jeden Authentifizierten, POST/DELETE
// nur für den Vorstand (System-Rolle admin fällt in RequireClubFunction ohnehin durch).
func newTGCServer(t *testing.T, h *config.Handler) *httptest.Server {
	t.Helper()
	return testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/training-group-categories", h.GetTrainingGroupCategoriesHandler)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand"))
			r.Post("/api/training-group-categories", h.CreateTrainingGroupCategory)
			r.Delete("/api/training-group-categories/{name}", h.DeleteTrainingGroupCategory)
		})
	})
}

// TestTrainingGroupCategories_ListSeed — GET liefert den Migration-034-Seed in
// sort_order-Reihenfolge (Perspektivkader vor Förderkader), nicht alphabetisch.
func TestTrainingGroupCategories_ListSeed(t *testing.T) {
	database := testutil.NewDB(t)
	h := config.NewHandler(database, hub.NewHub())
	srv := newTGCServer(t, h)
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"spieler"})

	res := testutil.Get(t, srv, "/api/training-group-categories", tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET: status %d, want 200", res.StatusCode)
	}
	var cats []config.TrainingGroupCategory
	if err := json.NewDecoder(res.Body).Decode(&cats); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("Seed-Länge = %d, want 2 (%v)", len(cats), cats)
	}
	if cats[0].Name != "Perspektivkader" || cats[0].SortOrder != 1 {
		t.Errorf("cats[0] = %+v, want Perspektivkader/1", cats[0])
	}
	if cats[1].Name != "Förderkader" || cats[1].SortOrder != 2 {
		t.Errorf("cats[1] = %+v, want Förderkader/2", cats[1])
	}
}

// TestTrainingGroupCategories_CreateAsVorstand — Vorstand legt eine Kategorie an
// (201) und sie erscheint danach in der Liste.
func TestTrainingGroupCategories_CreateAsVorstand(t *testing.T) {
	database := testutil.NewDB(t)
	h := config.NewHandler(database, hub.NewHub())
	srv := newTGCServer(t, h)
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	body := map[string]any{"name": "Leistungskader", "sort_order": 3}
	res := testutil.Do(t, srv, http.MethodPost, "/api/training-group-categories", tok, body)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST: status %d, want 201", res.StatusCode)
	}

	// Danach sichtbar.
	list := testutil.Get(t, srv, "/api/training-group-categories", tok)
	defer list.Body.Close()
	var cats []config.TrainingGroupCategory
	json.NewDecoder(list.Body).Decode(&cats)
	found := false
	for _, c := range cats {
		if c.Name == "Leistungskader" && c.SortOrder == 3 {
			found = true
		}
	}
	if !found {
		t.Errorf("neue Kategorie nicht in Liste: %v", cats)
	}
}

// TestTrainingGroupCategories_CreateEmptyName — leerer Name → 400, kein Insert.
func TestTrainingGroupCategories_CreateEmptyName(t *testing.T) {
	database := testutil.NewDB(t)
	h := config.NewHandler(database, hub.NewHub())
	srv := newTGCServer(t, h)
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPost, "/api/training-group-categories", tok,
		map[string]any{"name": "   "})
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST leerer Name: status %d, want 400", res.StatusCode)
	}
	var n int
	database.QueryRow(`SELECT COUNT(*) FROM training_group_categories`).Scan(&n)
	if n != 2 {
		t.Errorf("Anzahl Kategorien = %d, want unverändert 2", n)
	}
}

// TestTrainingGroupCategories_CreateDuplicate — Duplikat (PK-Verletzung) → 409.
func TestTrainingGroupCategories_CreateDuplicate(t *testing.T) {
	database := testutil.NewDB(t)
	h := config.NewHandler(database, hub.NewHub())
	srv := newTGCServer(t, h)
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Do(t, srv, http.MethodPost, "/api/training-group-categories", tok,
		map[string]any{"name": "Förderkader"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("POST Duplikat: status %d, want 409", res.StatusCode)
	}
}

// TestTrainingGroupCategories_CreateForbidden — Nicht-Vorstand ohne admin → 403,
// die Liste bleibt unverändert.
func TestTrainingGroupCategories_CreateForbidden(t *testing.T) {
	database := testutil.NewDB(t)
	h := config.NewHandler(database, hub.NewHub())
	srv := newTGCServer(t, h)
	userID := testutil.CreateUser(t, database, "standard")
	spieler := testutil.Token(t, userID, "standard", []string{"spieler"})

	res := testutil.Do(t, srv, http.MethodPost, "/api/training-group-categories", spieler,
		map[string]any{"name": "Heimlichkader"})
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("POST als Spieler: status %d, want 403", res.StatusCode)
	}
	var n int
	database.QueryRow(`SELECT COUNT(*) FROM training_group_categories`).Scan(&n)
	if n != 2 {
		t.Errorf("Anzahl Kategorien = %d, want unverändert 2", n)
	}
}

// TestTrainingGroupCategories_DeleteAsVorstand — Vorstand löscht eine Kategorie
// (204), danach ist sie weg.
func TestTrainingGroupCategories_DeleteAsVorstand(t *testing.T) {
	database := testutil.NewDB(t)
	h := config.NewHandler(database, hub.NewHub())
	srv := newTGCServer(t, h)
	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	res := testutil.Delete(t, srv, "/api/training-group-categories/"+url.PathEscape("Förderkader"), tok)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		t.Fatalf("DELETE: status %d, want 204/200", res.StatusCode)
	}
	var n int
	database.QueryRow(`SELECT COUNT(*) FROM training_group_categories WHERE name=?`, "Förderkader").Scan(&n)
	if n != 0 {
		t.Errorf("Kategorie noch vorhanden nach DELETE (n=%d)", n)
	}
}

// TestTrainingGroupCategories_DeleteForbidden — Nicht-Vorstand → 403, Kategorie bleibt.
func TestTrainingGroupCategories_DeleteForbidden(t *testing.T) {
	database := testutil.NewDB(t)
	h := config.NewHandler(database, hub.NewHub())
	srv := newTGCServer(t, h)
	userID := testutil.CreateUser(t, database, "standard")
	spieler := testutil.Token(t, userID, "standard", []string{"spieler"})

	res := testutil.Delete(t, srv, "/api/training-group-categories/"+url.PathEscape("Förderkader"), spieler)
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("DELETE als Spieler: status %d, want 403", res.StatusCode)
	}
	var n int
	database.QueryRow(`SELECT COUNT(*) FROM training_group_categories WHERE name=?`, "Förderkader").Scan(&n)
	if n != 1 {
		t.Errorf("Kategorie fälschlich entfernt (n=%d)", n)
	}
}

// TestTrainingGroupCategories_DeleteUsedByKader — das Löschen einer noch von einem
// Kader als age_class-Freitext genutzten Kategorie ist erlaubt (kein FK, kein
// Fehler); der Kader bleibt mit unverändertem age_class intakt und nutzbar.
func TestTrainingGroupCategories_DeleteUsedByKader(t *testing.T) {
	database := testutil.NewDB(t)
	h := config.NewHandler(database, hub.NewHub())
	srv := newTGCServer(t, h)

	seasonID := testutil.CreateSeason(t, database, "2025/26")
	teamID := testutil.CreateTeam(t, database, "Förderkader-Team")
	// Kader mit age_class = Kategoriename (Freitext, kein FK auf die Referenzliste).
	res, err := database.Exec(
		`INSERT INTO kader (season_id, age_class, gender, team_id, team_number) VALUES (?, ?, ?, ?, ?)`,
		seasonID, "Förderkader", "mixed", teamID, 1)
	if err != nil {
		t.Fatalf("insert kader: %v", err)
	}
	kaderID, _ := res.LastInsertId()

	userID := testutil.CreateUser(t, database, "standard")
	tok := testutil.Token(t, userID, "standard", []string{"vorstand"})

	del := testutil.Delete(t, srv, "/api/training-group-categories/"+url.PathEscape("Förderkader"), tok)
	defer del.Body.Close()
	if del.StatusCode != http.StatusNoContent && del.StatusCode != http.StatusOK {
		t.Fatalf("DELETE genutzte Kategorie: status %d, want 204/200", del.StatusCode)
	}

	// Kategorie weg …
	var catN int
	database.QueryRow(`SELECT COUNT(*) FROM training_group_categories WHERE name=?`, "Förderkader").Scan(&catN)
	if catN != 0 {
		t.Errorf("Kategorie nicht entfernt (n=%d)", catN)
	}
	// … Kader unverändert erhalten (gleicher age_class).
	var ac string
	if err := database.QueryRow(`SELECT age_class FROM kader WHERE id=?`, kaderID).Scan(&ac); err != nil {
		t.Fatalf("Kader nach Kategorie-Löschung nicht mehr vorhanden: %v", err)
	}
	if ac != "Förderkader" {
		t.Errorf("Kader age_class = %q, want unverändert Förderkader", ac)
	}
}
