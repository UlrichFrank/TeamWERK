package members_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

// TestServesBankRoutesWithoutEncryptionKey deckt das Spec-Szenario „Server startet nach
// Migration ohne Schlüssel" ab: Mit NICHT gesetztem FIELD_ENCRYPTION_KEY (Zero-Knowledge,
// Modell B — der Server hält keinen Entschlüsselungsschlüssel mehr) bedient die volle
// Produktions-Router-Stack die Bank-Route und liefert ausschließlich den Envelope
// (Ciphertext + gewrappter DEK), ohne serverseitig zu entschlüsseln.
func TestServesBankRoutesWithoutEncryptionKey(t *testing.T) {
	t.Setenv("FIELD_ENCRYPTION_KEY", "") // explizit: kein Brücken-/At-Rest-Schlüssel

	db := testutil.NewDB(t)
	uid := testutil.CreateUser(t, db, "standard")
	mid := testutil.CreateMember(t, db, uid)
	if _, err := db.Exec(
		`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?, 'CT', 'WRAP')`, mid,
	); err != nil {
		t.Fatalf("seed envelope: %v", err)
	}

	srv := prodserver.New(t, db)
	tok := testutil.Token(t, uid, "standard", []string{"vorstand"})

	res := testutil.Get(t, srv, "/api/members/"+strconv.Itoa(mid), tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET member ohne Schlüssel: status %d, want 200", res.StatusCode)
	}
	var body struct {
		BankCiphertext *string `json:"bank_ciphertext"`
		BankDekEnc     *string `json:"bank_dek_enc"`
	}
	json.NewDecoder(res.Body).Decode(&body)
	if body.BankCiphertext == nil || *body.BankCiphertext != "CT" || body.BankDekEnc == nil || *body.BankDekEnc != "WRAP" {
		t.Errorf("Envelope nicht ausgeliefert: %+v", body)
	}
}
