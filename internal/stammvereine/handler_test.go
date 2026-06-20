package stammvereine_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/stammvereine"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func newSrv(t *testing.T) *httptest.Server {
	t.Helper()
	database := testutil.NewDB(t)
	h := stammvereine.NewHandler(database, hub.NewHub())
	return testutil.NewServer(t, func(r chi.Router) {
		// GET: jeder Eingeloggte (auth.Middleware in NewServer).
		r.Get("/api/stammvereine", h.List)
		// Mutationen: nur Vorstand.
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand"))
			r.Post("/api/stammvereine", h.Create)
			r.Put("/api/stammvereine/{id}", h.Update)
			r.Delete("/api/stammvereine/{id}", h.Delete)
		})
	})
}

type listResp struct {
	Items []stammvereine.Verein `json:"items"`
}

func vorstandTok(t *testing.T) string { return testutil.Token(t, 1, "standard", []string{"vorstand"}) }
func spielerTok(t *testing.T) string  { return testutil.Token(t, 2, "standard", []string{"spieler"}) }

func decodeList(t *testing.T, res *http.Response) listResp {
	t.Helper()
	var lr listResp
	if err := json.NewDecoder(res.Body).Decode(&lr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return lr
}

func TestStammvereine_ListActive(t *testing.T) {
	srv := newSrv(t)
	res := testutil.Get(t, srv, "/api/stammvereine", spielerTok(t))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET: status %d, want 200", res.StatusCode)
	}
	lr := decodeList(t, res)
	if len(lr.Items) != 8 {
		t.Fatalf("aktive Vereine = %d, want 8 (Seed)", len(lr.Items))
	}
	for _, v := range lr.Items {
		if !v.Aktiv {
			t.Errorf("Verein %q sollte aktiv sein", v.Name)
		}
	}
}

func TestStammvereine_ListUnauthorized(t *testing.T) {
	srv := newSrv(t)
	if res := testutil.Get(t, srv, "/api/stammvereine", ""); res.StatusCode != http.StatusUnauthorized {
		t.Errorf("GET ohne Token: status %d, want 401", res.StatusCode)
	}
}

func TestStammvereine_CreateAndDuplicate(t *testing.T) {
	srv := newSrv(t)
	tok := vorstandTok(t)
	res := testutil.Post(t, srv, "/api/stammvereine", tok, map[string]any{"name": "SV Beispiel 1900"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST: status %d, want 201", res.StatusCode)
	}
	// Doppelter Name → 409
	if res := testutil.Post(t, srv, "/api/stammvereine", tok,
		map[string]any{"name": "SV Beispiel 1900"}); res.StatusCode != http.StatusConflict {
		t.Errorf("doppelter Name: status %d, want 409", res.StatusCode)
	}
}

func TestStammvereine_CreateForbidden(t *testing.T) {
	srv := newSrv(t)
	if res := testutil.Post(t, srv, "/api/stammvereine", spielerTok(t),
		map[string]any{"name": "SV Verboten"}); res.StatusCode != http.StatusForbidden {
		t.Errorf("POST als spieler: status %d, want 403", res.StatusCode)
	}
}

func TestStammvereine_SoftDeleteErhältReferenz(t *testing.T) {
	srv := newSrv(t)
	tok := vorstandTok(t)
	// Anlegen
	res := testutil.Post(t, srv, "/api/stammvereine", tok, map[string]any{"name": "SV Temporär"})
	var created stammvereine.Verein
	json.NewDecoder(res.Body).Decode(&created)

	// Soft-Delete
	if res := testutil.Do(t, srv, http.MethodDelete, "/api/stammvereine/"+strconv.Itoa(created.ID), tok, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE: status %d, want 204", res.StatusCode)
	}

	// Aus Standard-Liste verschwunden …
	for _, v := range decodeList(t, testutil.Get(t, srv, "/api/stammvereine", tok)).Items {
		if v.ID == created.ID {
			t.Fatalf("deaktivierter Verein erscheint noch in Standard-Liste")
		}
	}
	// … aber mit include_inactive (Vorstand) noch sichtbar und aktiv=false.
	found := false
	for _, v := range decodeList(t, testutil.Get(t, srv, "/api/stammvereine?include_inactive=1", tok)).Items {
		if v.ID == created.ID {
			found = true
			if v.Aktiv {
				t.Errorf("deaktivierter Verein sollte aktiv=false haben")
			}
		}
	}
	if !found {
		t.Errorf("deaktivierter Verein fehlt in include_inactive-Liste")
	}
}

func TestStammvereine_IncludeInactiveNurFürVorstand(t *testing.T) {
	srv := newSrv(t)
	vTok := vorstandTok(t)
	res := testutil.Post(t, srv, "/api/stammvereine", vTok, map[string]any{"name": "SV Inaktiv"})
	var created stammvereine.Verein
	json.NewDecoder(res.Body).Decode(&created)
	testutil.Do(t, srv, http.MethodDelete, "/api/stammvereine/"+strconv.Itoa(created.ID), vTok, nil)

	// Normaler Nutzer darf include_inactive nicht nutzen → deaktivierter Verein bleibt verborgen.
	for _, v := range decodeList(t, testutil.Get(t, srv, "/api/stammvereine?include_inactive=1", spielerTok(t))).Items {
		if v.ID == created.ID {
			t.Errorf("Spieler sollte deaktivierte Vereine nicht sehen")
		}
	}
}

func TestStammvereine_UpdateForbidden(t *testing.T) {
	srv := newSrv(t)
	if res := testutil.Do(t, srv, http.MethodPut, "/api/stammvereine/1", spielerTok(t),
		map[string]any{"name": "Hack"}); res.StatusCode != http.StatusForbidden {
		t.Errorf("PUT als spieler: status %d, want 403", res.StatusCode)
	}
}

func TestStammvereine_DeleteForbidden(t *testing.T) {
	srv := newSrv(t)
	if res := testutil.Do(t, srv, http.MethodDelete, "/api/stammvereine/1", spielerTok(t), nil); res.StatusCode != http.StatusForbidden {
		t.Errorf("DELETE als spieler: status %d, want 403", res.StatusCode)
	}
}
