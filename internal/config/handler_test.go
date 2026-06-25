package config_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Modell B: Vereins-SEPA-Stammdaten als clientseitiger Envelope; der Server speichert nur
// Ciphertext + Wrap und lehnt Klartext-SEPA-Felder ab.
func TestClub_SepaEnvelope_GetSet(t *testing.T) {
	database := testutil.NewDB(t)
	if _, err := database.Exec(`INSERT INTO clubs (name) VALUES ('Team Stuttgart')`); err != nil {
		t.Fatalf("seed club: %v", err)
	}
	h := config.NewHandler(database, hub.NewHub())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/club", h.GetClub)
		r.Put("/api/club", h.UpdateClub)
	})
	tok := testutil.Token(t, 1, "admin", []string{"vorstand"})

	const ct = "ZW52ZWxvcGU="
	const dek = "d3JhcA=="

	// SEPA-Envelope + Stammdaten setzen
	body := map[string]any{
		"name":            "Team Stuttgart",
		"address":         "Musterweg 1",
		"sepa_ciphertext": ct,
		"sepa_dek_enc":    dek,
	}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/club", tok, body); res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT club: status %d", res.StatusCode)
	}

	// DB hält den Envelope, kein Klartext.
	var dbCt, dbDek string
	database.QueryRow(`SELECT sepa_ciphertext, sepa_dek_enc FROM clubs LIMIT 1`).Scan(&dbCt, &dbDek)
	if dbCt != ct || dbDek != dek {
		t.Errorf("Envelope nicht gespeichert: ct=%q dek=%q", dbCt, dbDek)
	}

	// Zurücklesen liefert den Envelope (nicht entschlüsselt)
	res := testutil.Get(t, srv, "/api/club", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET club: status %d", res.StatusCode)
	}
	var got map[string]any
	json.NewDecoder(res.Body).Decode(&got)
	if got["sepa_ciphertext"] != ct || got["sepa_dek_enc"] != dek {
		t.Errorf("GET club Envelope falsch: %v / %v", got["sepa_ciphertext"], got["sepa_dek_enc"])
	}
	if got["name"] != "Team Stuttgart" || got["address"] != "Musterweg 1" {
		t.Errorf("Stammdaten falsch: %v / %v", got["name"], got["address"])
	}

	// Klartext-SEPA-Feld → 400
	bad := map[string]any{"name": "x", "iban": "DE89370400440532013000"}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/club", tok, bad); res.StatusCode != http.StatusBadRequest {
		t.Errorf("Klartext-IBAN: status %d, want 400", res.StatusCode)
	}
	bad2 := map[string]any{"name": "x", "glaeubiger_id": "DE98ZZZ09999999999"}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/club", tok, bad2); res.StatusCode != http.StatusBadRequest {
		t.Errorf("Klartext-Gläubiger-ID: status %d, want 400", res.StatusCode)
	}
}
