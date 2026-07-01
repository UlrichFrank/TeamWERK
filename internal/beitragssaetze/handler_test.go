package beitragssaetze_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/beitragssaetze"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newSrv(t *testing.T) *httptest.Server {
	t.Helper()
	database := testutil.NewDB(t)
	h := beitragssaetze.NewHandler(database, hub.NewHub())
	return testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "kassierer"))
			r.Get("/api/fee-rates", h.List)
			r.Post("/api/fee-rates", h.Create)
			r.Delete("/api/fee-rates/{id}", h.Delete)
		})
	})
}

type listResp struct {
	Items []beitragssaetze.Satz `json:"items"`
}

func vorstandTok(t *testing.T) string { return testutil.Token(t, 1, "standard", []string{"vorstand"}) }

func TestSaetze_HistorieErhalten(t *testing.T) {
	srv := newSrv(t)
	tok := vorstandTok(t)
	// aktiv_mit ist bereits geseedet (9600/2026-07-01); zweiter Satz für 2027.
	if res := testutil.Post(t, srv, "/api/fee-rates", tok,
		map[string]any{"kategorie": "aktiv_mit", "betrag_cent": 10000, "valid_from": "2027-07-01"}); res.StatusCode != http.StatusCreated {
		t.Fatalf("POST: status %d", res.StatusCode)
	}
	res := testutil.Get(t, srv, "/api/fee-rates", tok)
	var lr listResp
	json.NewDecoder(res.Body).Decode(&lr)
	var mit []beitragssaetze.Satz
	for _, s := range lr.Items {
		if s.Kategorie == "aktiv_mit" {
			mit = append(mit, s)
		}
	}
	if len(mit) != 2 {
		t.Fatalf("aktiv_mit Sätze = %d, want 2", len(mit))
	}
	if mit[0].ValidFrom != "2027-07-01" {
		t.Errorf("erster Satz valid_from = %q, want 2027-07-01 (DESC)", mit[0].ValidFrom)
	}
}

func TestSaetze_NeueValidFromAnlegen(t *testing.T) {
	srv := newSrv(t)
	tok := vorstandTok(t)
	body := map[string]any{"kategorie": "passiv", "betrag_cent": 7000, "valid_from": "2028-01-01"}
	if res := testutil.Post(t, srv, "/api/fee-rates", tok, body); res.StatusCode != http.StatusCreated {
		t.Fatalf("erster POST: status %d", res.StatusCode)
	}
	// identische kategorie + valid_from erneut → erlaubt, kein 409
	if res := testutil.Post(t, srv, "/api/fee-rates", tok, body); res.StatusCode != http.StatusCreated {
		t.Errorf("zweiter POST: status %d, want 201", res.StatusCode)
	}
}

func TestSaetze_InvalidKategorie(t *testing.T) {
	srv := newSrv(t)
	tok := vorstandTok(t)
	res := testutil.Post(t, srv, "/api/fee-rates", tok,
		map[string]any{"kategorie": "quatsch", "betrag_cent": 100, "valid_from": "2026-07-01"})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, want 400", res.StatusCode)
	}
}

func TestSaetze_Forbidden(t *testing.T) {
	srv := newSrv(t)
	tok := testutil.Token(t, 2, "standard", []string{"spieler"})
	if res := testutil.Get(t, srv, "/api/fee-rates", tok); res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.StatusCode)
	}
}

func TestSaetze_Delete(t *testing.T) {
	srv := newSrv(t)
	tok := vorstandTok(t)
	// Neuen Satz anlegen und wieder löschen.
	res := testutil.Post(t, srv, "/api/fee-rates", tok,
		map[string]any{"kategorie": "passiv", "betrag_cent": 5000, "valid_from": "2030-07-01"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST: status %d", res.StatusCode)
	}
	var created beitragssaetze.Satz
	json.NewDecoder(res.Body).Decode(&created)
	del := testutil.Do(t, srv, http.MethodDelete, "/api/fee-rates/"+strconv.Itoa(created.ID), tok, nil)
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE: status %d, want 204", del.StatusCode)
	}
	// Nicht mehr in Liste.
	lr := testutil.Get(t, srv, "/api/fee-rates", tok)
	var list listResp
	json.NewDecoder(lr.Body).Decode(&list)
	for _, s := range list.Items {
		if s.ID == created.ID {
			t.Errorf("Satz %d nach DELETE noch in Liste", created.ID)
		}
	}
}

func TestSaetze_DeleteNotFound(t *testing.T) {
	srv := newSrv(t)
	tok := vorstandTok(t)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/fee-rates/99999", tok, nil)
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status %d, want 404", res.StatusCode)
	}
}

func TestSaetze_DeleteForbidden(t *testing.T) {
	srv := newSrv(t)
	tok := testutil.Token(t, 2, "standard", []string{"spieler"})
	res := testutil.Do(t, srv, http.MethodDelete, "/api/fee-rates/1", tok, nil)
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.StatusCode)
	}
}
