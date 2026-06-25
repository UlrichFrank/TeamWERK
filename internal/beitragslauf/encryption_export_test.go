package beitragslauf_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Modell B: export-data liefert ausschließlich Ciphertext + Wraps (Mitglieds-Bankdaten und
// Vereins-SEPA) sowie nicht-geheime Felder — niemals eine Klartext-IBAN. Das pain.008-XML
// entsteht clientseitig.
func TestExportData_LiefertNurEnvelopes(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember()) // legt member_sensitive 'CT'/'DEK' an

	res := testutil.Post(t, srv, "/api/fee-run/export-data", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("export-data status %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	// Keine Klartext-IBAN im Response.
	if strings.Contains(string(body), validIBAN) {
		t.Errorf("export-data enthält Klartext-IBAN (darf nie passieren):\n%s", body)
	}

	var resp struct {
		ClubSepa struct {
			Ciphertext string `json:"ciphertext"`
			DekEnc     string `json:"dek_enc"`
		} `json:"club_sepa"`
		Items []struct {
			BankCiphertext string `json:"bank_ciphertext"`
			BankDekEnc     string `json:"bank_dek_enc"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ClubSepa.Ciphertext != "CLUBCT" || resp.ClubSepa.DekEnc != "CLUBDEK" {
		t.Errorf("club_sepa-Envelope fehlt: %+v", resp.ClubSepa)
	}
	if len(resp.Items) != 1 || resp.Items[0].BankCiphertext != "CT" || resp.Items[0].BankDekEnc != "DEK" {
		t.Errorf("Mitglieds-Envelope fehlt: %+v", resp.Items)
	}
}

// Ohne eingerichtete Vereins-SEPA (kein Envelope) → 400.
func TestExportData_OhneVereinsSepa400(t *testing.T) {
	srv, db, _ := setupSrv(t)
	db.Exec(`UPDATE clubs SET sepa_ciphertext=NULL, sepa_dek_enc=NULL`)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())
	res := testutil.Post(t, srv, "/api/fee-run/export-data", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("ohne Vereins-SEPA: status %d, want 400", res.StatusCode)
	}
}
