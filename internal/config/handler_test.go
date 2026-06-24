package config_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/crypto"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func TestClub_SepaFelder_GetSet(t *testing.T) {
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

	// Gültige SEPA-Stammdaten setzen
	body := map[string]any{
		"name":          "Team Stuttgart",
		"glaeubiger_id": "DE98ZZZ09999999999",
		"iban":          "DE89370400440532013000",
		"bic":           "GENODEF1S02",
		"kontoinhaber":  "Team Stuttgart e.V.",
	}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/club", tok, body); res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT club: status %d", res.StatusCode)
	}

	// At-rest verschlüsselt: die DB-Spalten halten Ciphertext, nicht den Klartext.
	var dbIBAN, dbBIC string
	database.QueryRow(`SELECT iban, bic FROM clubs LIMIT 1`).Scan(&dbIBAN, &dbBIC)
	if !crypto.IsEncryptedString(dbIBAN) || dbIBAN == "DE89370400440532013000" {
		t.Errorf("clubs.iban nicht verschlüsselt: %q", dbIBAN)
	}
	if !crypto.IsEncryptedString(dbBIC) {
		t.Errorf("clubs.bic nicht verschlüsselt: %q", dbBIC)
	}

	// Zurücklesen
	res := testutil.Get(t, srv, "/api/club", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET club: status %d", res.StatusCode)
	}
	var got map[string]any
	json.NewDecoder(res.Body).Decode(&got)
	if got["glaeubiger_id"] != "DE98ZZZ09999999999" {
		t.Errorf("glaeubiger_id = %v", got["glaeubiger_id"])
	}
	if got["iban"] != "DE89370400440532013000" {
		t.Errorf("iban = %v", got["iban"])
	}
	if got["bic"] != "GENODEF1S02" {
		t.Errorf("bic = %v", got["bic"])
	}
	if got["kontoinhaber"] != "Team Stuttgart e.V." {
		t.Errorf("kontoinhaber = %v", got["kontoinhaber"])
	}

	// Ungültige Gläubiger-ID → 400
	bad := map[string]any{"name": "x", "glaeubiger_id": "INVALID"}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/club", tok, bad); res.StatusCode != http.StatusBadRequest {
		t.Errorf("ungültige Gläubiger-ID: status %d, want 400", res.StatusCode)
	}

	// Ungültige IBAN → 400
	bad2 := map[string]any{"name": "x", "iban": "DE88370400440532013000"}
	if res := testutil.Do(t, srv, http.MethodPut, "/api/club", tok, bad2); res.StatusCode != http.StatusBadRequest {
		t.Errorf("ungültige IBAN: status %d, want 400", res.StatusCode)
	}
}
