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

// Ohne 'faelligkeit' im Body → Default 01.07. der Saison zurückgeliefert.
func TestExportData_FaelligkeitDefault(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())

	res := testutil.Post(t, srv, "/api/fee-run/export-data", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var resp struct {
		Faelligkeit string `json:"faelligkeit"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()
	if resp.Faelligkeit != "2027-07-01" {
		t.Errorf("Default-Fälligkeit: got %q, want 2027-07-01", resp.Faelligkeit)
	}
}

// 'faelligkeit' überschreibt den Default, wenn gültig und in der Zukunft.
func TestExportData_FaelligkeitOverride(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())

	// Ein Datum weit in der Zukunft, damit der Test unabhängig von time.Now stabil bleibt.
	future := "2099-08-15"
	res := testutil.Post(t, srv, "/api/fee-run/export-data", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}, "faelligkeit": future})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	var resp struct {
		Faelligkeit string `json:"faelligkeit"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()
	if resp.Faelligkeit != future {
		t.Errorf("Fälligkeit-Override: got %q, want %q", resp.Faelligkeit, future)
	}
}

// Ungültiges Datumsformat → 400.
func TestExportData_FaelligkeitUngueltig400(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())

	res := testutil.Post(t, srv, "/api/fee-run/export-data", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}, "faelligkeit": "01.07.2027"})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("ungültiges Format: status %d, want 400", res.StatusCode)
	}
}

// Fälligkeit in der Vergangenheit → 400 (SEPA-XSD lehnt Vergangenheit ab).
func TestExportData_FaelligkeitVergangenheit400(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())

	res := testutil.Post(t, srv, "/api/fee-run/export-data", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}, "faelligkeit": "2000-01-01"})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("Vergangenheit: status %d, want 400", res.StatusCode)
	}
}
