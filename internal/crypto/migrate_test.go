package crypto_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/crypto"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// seedPII legt Klartext-Bestand in den vier Speichern an und gibt das
// uploadDir mit einer unverschlüsselten SEPA-PDF zurück.
func seedPII(t *testing.T, db *sql.DB) (uploadDir, pdfRel string) {
	t.Helper()
	uploadDir = t.TempDir()
	pdfRel = "sepa-mandats/mandat.pdf"
	pdf := []byte("%PDF-1.4 Mandat Klartext")
	if err := os.MkdirAll(filepath.Join(uploadDir, "sepa-mandats"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploadDir, pdfRel), pdf, 0o644); err != nil {
		t.Fatal(err)
	}

	mid := testutil.CreateMember(t, db, 0)
	if _, err := db.Exec(`UPDATE members SET iban=?, account_holder=?, sepa_mandat_path=? WHERE id=?`,
		"DE89370400440532013000", "Max Mustermann", pdfRel, mid); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO clubs (name, glaeubiger_id, iban, bic, kontoinhaber)
		VALUES ('Verein','DE98ZZZ09999999999','DE89370400440532013000','COBADEFFXXX','Verein e.V.')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO member_change_drafts (member_id, field_name, old_value, new_value)
		VALUES (?, 'bankdaten', '{}', '{"iban":"DE89370400440532013000","account_holder":"Max"}')`, mid); err != nil {
		t.Fatal(err)
	}
	return uploadDir, pdfRel
}

func TestEncryptPII_EncryptsAndIsIdempotent(t *testing.T) {
	db := testutil.NewDB(t)
	uploadDir, pdfRel := seedPII(t, db)

	rep, err := crypto.EncryptPII(db, uploadDir)
	if err != nil {
		t.Fatalf("EncryptPII: %v", err)
	}
	if rep.MemberRows != 1 || rep.ClubRows != 1 || rep.Drafts != 1 || rep.Files != 1 {
		t.Fatalf("unerwarteter Report: %+v", rep)
	}

	assertEncrypted := func(label, got string) {
		if !crypto.IsEncryptedString(got) {
			t.Errorf("%s nicht verschlüsselt: %q", label, got)
		}
	}
	var iban, holder, cIban, draft string
	db.QueryRow(`SELECT iban, account_holder FROM members WHERE iban IS NOT NULL`).Scan(&iban, &holder)
	db.QueryRow(`SELECT iban FROM clubs LIMIT 1`).Scan(&cIban)
	db.QueryRow(`SELECT new_value FROM member_change_drafts WHERE field_name='bankdaten'`).Scan(&draft)
	assertEncrypted("members.iban", iban)
	assertEncrypted("members.account_holder", holder)
	assertEncrypted("clubs.iban", cIban)
	assertEncrypted("drafts.new_value", draft)

	fileData, _ := os.ReadFile(filepath.Join(uploadDir, pdfRel))
	if !crypto.IsEncryptedBytes(fileData) {
		t.Error("SEPA-PDF nicht verschlüsselt")
	}

	// Zweiter Lauf ist idempotent: nichts wird erneut transformiert.
	rep2, err := crypto.EncryptPII(db, uploadDir)
	if err != nil {
		t.Fatalf("EncryptPII (2): %v", err)
	}
	if rep2 != (crypto.PIIReport{}) {
		t.Errorf("zweiter Lauf nicht idempotent: %+v", rep2)
	}
}

func TestDecryptPII_RestoresPlaintext(t *testing.T) {
	db := testutil.NewDB(t)
	uploadDir, pdfRel := seedPII(t, db)

	if _, err := crypto.EncryptPII(db, uploadDir); err != nil {
		t.Fatalf("EncryptPII: %v", err)
	}
	if _, err := crypto.DecryptPII(db, uploadDir); err != nil {
		t.Fatalf("DecryptPII: %v", err)
	}

	var iban, cIban, draft string
	db.QueryRow(`SELECT iban FROM members WHERE iban IS NOT NULL`).Scan(&iban)
	db.QueryRow(`SELECT iban FROM clubs LIMIT 1`).Scan(&cIban)
	db.QueryRow(`SELECT new_value FROM member_change_drafts WHERE field_name='bankdaten'`).Scan(&draft)
	if iban != "DE89370400440532013000" {
		t.Errorf("members.iban roundtrip: %q", iban)
	}
	if cIban != "DE89370400440532013000" {
		t.Errorf("clubs.iban roundtrip: %q", cIban)
	}
	if draft != `{"iban":"DE89370400440532013000","account_holder":"Max"}` {
		t.Errorf("draft roundtrip: %q", draft)
	}
	fileData, _ := os.ReadFile(filepath.Join(uploadDir, pdfRel))
	if string(fileData) != "%PDF-1.4 Mandat Klartext" {
		t.Errorf("PDF roundtrip: %q", fileData)
	}
}
