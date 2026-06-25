package migration_test

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/crypto"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/migration"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// clientMagic spiegelt crypto.clientFileMagic ("TWENC1\n") — ein clientseitig verschlüsselter
// Mandat-Blob muss diesen Header tragen, sonst lehnt der Upload ihn ab.
var clientMagic = []byte("TWENC1\n")

func enc(t *testing.T, plain string) string {
	t.Helper()
	v, err := crypto.Encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return v
}

// seed legt einen v1:-Altbestand an: ein Mitglied mit Bankdaten + Mandat-PDF und Vereins-SEPA.
func seed(t *testing.T, database *sql.DB, uploadDir string) int {
	t.Helper()
	if _, err := database.Exec(
		`INSERT INTO clubs (name, glaeubiger_id, iban, bic, kontoinhaber) VALUES (?, ?, ?, ?, ?)`,
		"Team Stuttgart", enc(t, "DE98ZZZ09999999999"), enc(t, "DE89370400440532013000"), enc(t, "COBADEFFXXX"), enc(t, "Team Stuttgart e.V."),
	); err != nil {
		t.Fatalf("seed club: %v", err)
	}
	m := testutil.CreateMember(t, database, 0)

	// Mandat-PDF (server-verschlüsselt) ablegen.
	rel := filepath.Join("sepa-mandats", "legacy.bin")
	blob, err := crypto.EncryptBytes([]byte("%PDF-1.4 fake mandate"))
	if err != nil {
		t.Fatalf("encrypt bytes: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(uploadDir, "sepa-mandats"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uploadDir, rel), blob, 0644); err != nil {
		t.Fatalf("write blob: %v", err)
	}
	if _, err := database.Exec(
		`UPDATE members SET iban=?, account_holder=?, sepa_mandat_path=? WHERE id=?`,
		enc(t, "DE89370400440532013000"), enc(t, "Max Mustermann"), rel, m,
	); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	return m
}

// TestMigration_Forbidden: ohne Finance-Funktion → 403 (Gate).
func TestMigration_Forbidden(t *testing.T) {
	database := testutil.NewDB(t)
	h := migration.NewHandler(database, hub.NewHub(), t.TempDir())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "kassierer"))
			r.Get("/api/admin/migrate-legacy/status", h.Status)
		})
	})
	spieler := testutil.Token(t, 2, "standard", []string{"spieler"})
	if res := testutil.Get(t, srv, "/api/admin/migrate-legacy/status", spieler); res.StatusCode != http.StatusForbidden {
		t.Errorf("Spieler: status %d, want 403", res.StatusCode)
	}
}

// TestMigration_RequiresBridge: ohne Brücken-Schlüssel liefern data/upload 404 ("nur wenn Bridge").
func TestMigration_RequiresBridge(t *testing.T) {
	database := testutil.NewDB(t)
	h := migration.NewHandler(database, hub.NewHub(), t.TempDir())
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/admin/migrate-legacy/data", h.Data)
		r.Post("/api/admin/migrate-legacy/upload", h.Upload)
	})
	tok := testutil.Token(t, 1, "admin", []string{"kassierer"})

	crypto.ClearKey()
	t.Cleanup(restoreTestKey)

	if res := testutil.Get(t, srv, "/api/admin/migrate-legacy/data", tok); res.StatusCode != http.StatusNotFound {
		t.Errorf("data ohne Bridge: status %d, want 404", res.StatusCode)
	}
	if res := testutil.Post(t, srv, "/api/admin/migrate-legacy/upload", tok, map[string]any{}); res.StatusCode != http.StatusNotFound {
		t.Errorf("upload ohne Bridge: status %d, want 404", res.StatusCode)
	}
}

// TestMigration_HappyPath: data liefert entschlüsselten v1:-Klartext; upload schreibt die
// Envelopes UND nullt die Legacy-Spalten; danach ist status.complete und data leer (idempotent).
func TestMigration_HappyPath(t *testing.T) {
	database := testutil.NewDB(t)
	uploadDir := t.TempDir()
	m := seed(t, database, uploadDir)
	h := migration.NewHandler(database, hub.NewHub(), uploadDir)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Get("/api/admin/migrate-legacy/status", h.Status)
		r.Get("/api/admin/migrate-legacy/data", h.Data)
		r.Post("/api/admin/migrate-legacy/upload", h.Upload)
	})
	tok := testutil.Token(t, 1, "admin", []string{"kassierer"})

	// data liefert den entschlüsselten Altbestand.
	res := testutil.Get(t, srv, "/api/admin/migrate-legacy/data", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("data: status %d, want 200", res.StatusCode)
	}
	var data struct {
		Members []struct {
			MemberID      int    `json:"member_id"`
			IBAN          string `json:"iban"`
			AccountHolder string `json:"account_holder"`
		} `json:"members"`
		Club *struct {
			GlaeubigerID string `json:"glaeubiger_id"`
			IBAN         string `json:"iban"`
		} `json:"club"`
		Mandates []struct {
			MemberID  int    `json:"member_id"`
			PDFBase64 string `json:"pdf_base64"`
		} `json:"mandates"`
	}
	json.NewDecoder(res.Body).Decode(&data)
	if len(data.Members) != 1 || data.Members[0].IBAN != "DE89370400440532013000" || data.Members[0].AccountHolder != "Max Mustermann" {
		t.Fatalf("member-Klartext falsch: %+v", data.Members)
	}
	if data.Club == nil || data.Club.GlaeubigerID != "DE98ZZZ09999999999" {
		t.Fatalf("club-Klartext falsch: %+v", data.Club)
	}
	if len(data.Mandates) != 1 {
		t.Fatalf("mandate fehlt: %+v", data.Mandates)
	}
	pdf, _ := base64.StdEncoding.DecodeString(data.Mandates[0].PDFBase64)
	if string(pdf) != "%PDF-1.4 fake mandate" {
		t.Fatalf("mandat-PDF-Klartext falsch: %q", pdf)
	}

	// upload der clientseitig erzeugten Envelopes (hier mit Platzhalter-Ciphertext).
	mandatBlob := append(append([]byte{}, clientMagic...), []byte("ciphertext-bytes")...)
	body := map[string]any{
		"members": []map[string]any{{"member_id": m, "bank_ciphertext": "CT_member", "bank_dek_enc": "WRAP_member"}},
		"club":    map[string]any{"sepa_ciphertext": "CT_club", "sepa_dek_enc": "WRAP_club"},
		"mandates": []map[string]any{{
			"member_id":   m,
			"blob_base64": base64.StdEncoding.EncodeToString(mandatBlob),
			"dek_enc":     "WRAP_mandat",
		}},
	}
	if res := testutil.Post(t, srv, "/api/admin/migrate-legacy/upload", tok, body); res.StatusCode != http.StatusNoContent {
		t.Fatalf("upload: status %d, want 204", res.StatusCode)
	}

	// Envelope geschrieben + Legacy genullt (Mitglied).
	var ct, wrap string
	database.QueryRow(`SELECT ciphertext, dek_enc_vorstand FROM member_sensitive WHERE member_id=?`, m).Scan(&ct, &wrap)
	if ct != "CT_member" || wrap != "WRAP_member" {
		t.Errorf("member_sensitive falsch: ct=%q wrap=%q", ct, wrap)
	}
	var iban, holder sql.NullString
	database.QueryRow(`SELECT iban, account_holder FROM members WHERE id=?`, m).Scan(&iban, &holder)
	if iban.Valid || holder.Valid {
		t.Errorf("Legacy-Member-Spalten nicht genullt: iban=%v holder=%v", iban, holder)
	}

	// Envelope geschrieben + Legacy genullt (Verein).
	var sepaCT string
	var cg sql.NullString
	database.QueryRow(`SELECT sepa_ciphertext, glaeubiger_id FROM clubs LIMIT 1`).Scan(&sepaCT, &cg)
	if sepaCT != "CT_club" || cg.Valid {
		t.Errorf("club-Envelope/Legacy falsch: sepa_ciphertext=%q glaeubiger_id=%v", sepaCT, cg)
	}

	// Mandat: dek_enc gesetzt, Datei trägt Client-Magic.
	var mandPath, mandDek string
	database.QueryRow(`SELECT sepa_mandat_path, sepa_mandat_dek_enc FROM members WHERE id=?`, m).Scan(&mandPath, &mandDek)
	if mandDek != "WRAP_mandat" {
		t.Errorf("sepa_mandat_dek_enc=%q, want WRAP_mandat", mandDek)
	}
	stored, _ := os.ReadFile(filepath.Join(uploadDir, mandPath))
	if !crypto.IsClientEncryptedBytes(stored) {
		t.Errorf("Mandat-Datei trägt keinen Client-Magic-Header")
	}

	// status.complete + data jetzt leer (idempotent).
	res = testutil.Get(t, srv, "/api/admin/migrate-legacy/status", tok)
	var st struct {
		Complete        bool `json:"complete"`
		PendingMembers  int  `json:"pending_members"`
		PendingClub     bool `json:"pending_club"`
		PendingMandates int  `json:"pending_mandates"`
	}
	json.NewDecoder(res.Body).Decode(&st)
	if !st.Complete || st.PendingMembers != 0 || st.PendingClub || st.PendingMandates != 0 {
		t.Errorf("status nach Migration nicht complete: %+v", st)
	}
	res = testutil.Get(t, srv, "/api/admin/migrate-legacy/data", tok)
	json.NewDecoder(res.Body).Decode(&data)
	if len(data.Members) != 0 || data.Club != nil || len(data.Mandates) != 0 {
		t.Errorf("data nach Migration nicht leer: %+v", data)
	}
}

// restoreTestKey stellt den deterministischen Test-Schlüssel wieder her (siehe testutil.init).
func restoreTestKey() {
	key := make([]byte, crypto.KeySize)
	for i := range key {
		key[i] = byte(i)
	}
	_ = crypto.Init(key)
}
