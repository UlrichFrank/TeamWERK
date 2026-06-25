package members_test

import (
	"encoding/json"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/members"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Modell B: Ein bankdaten-Draft trägt den clientseitigen Envelope; Annehmen übernimmt ihn
// unverändert nach member_sensitive (kein Server-Crypto), nicht nach members.iban.
func TestBankdatenDraft_EnvelopeWirdNachMemberSensitiveUebernommen(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	h := members.NewHandler(db, hub.NewHub())

	envelope, _ := json.Marshal(map[string]string{
		"bank_ciphertext": testCiphertext,
		"bank_dek_enc":    testDekEnc,
	})

	draft, err := h.CreateOrUpdateDraft(memberID, userID, members.ChangeRequest{
		FieldName: "bankdaten",
		NewValue:  json.RawMessage(envelope),
	})
	if err != nil {
		t.Fatalf("CreateOrUpdateDraft: %v", err)
	}

	// Der Server hat den Envelope NICHT verändert (kein Re-Encrypt) und keinen Altwert geleakt.
	var storedNew, storedOld string
	db.QueryRow(`SELECT new_value, COALESCE(old_value,'null') FROM member_change_drafts WHERE id=?`, draft.ID).
		Scan(&storedNew, &storedOld)
	if storedOld != "null" {
		t.Errorf("old_value für bankdaten sollte null sein, war %q", storedOld)
	}

	if err := h.AcceptDraft(draft.ID); err != nil {
		t.Fatalf("AcceptDraft: %v", err)
	}

	var ct, dek string
	if err := db.QueryRow(`SELECT ciphertext, dek_enc_vorstand FROM member_sensitive WHERE member_id=?`, memberID).
		Scan(&ct, &dek); err != nil {
		t.Fatalf("member_sensitive nach Accept: %v", err)
	}
	if ct != testCiphertext || dek != testDekEnc {
		t.Errorf("Envelope nicht übernommen: ct=%q dek=%q", ct, dek)
	}
	// members.iban bleibt leer.
	var iban string
	db.QueryRow(`SELECT COALESCE(iban,'') FROM members WHERE id=?`, memberID).Scan(&iban)
	if iban != "" {
		t.Errorf("members.iban beschrieben (%q) — Bankdaten gehören in member_sensitive", iban)
	}
}
