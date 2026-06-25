package members_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

const testCiphertext = "ZW52ZWxvcGVDaXBoZXJ0ZXh0" // beliebiger Envelope-Blob (Server entschlüsselt nie)
const testDekEnc = "d3JhcHBlZERFSw=="

type memberBankResp struct {
	IBAN           string `json:"iban"`
	AccountHolder  string `json:"account_holder"`
	BankCiphertext string `json:"bank_ciphertext"`
	BankDekEnc     string `json:"bank_dek_enc"`
}

// (7.2) Berechtigte (Finance-Gruppe) erhalten den Envelope (Ciphertext + Wrap), NICHT
// Klartext — entschlüsselt wird ausschließlich clientseitig.
func TestGet_VorstandErhältEnvelope(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?,?,?)`,
		id, testCiphertext, testDekEnc)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 1, "standard", []string{"vorstand"})

	res := testutil.Get(t, srv, "/api/members/"+strconv.Itoa(id), tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET member: status %d", res.StatusCode)
	}
	var m memberBankResp
	json.NewDecoder(res.Body).Decode(&m)
	res.Body.Close()
	if m.BankCiphertext != testCiphertext || m.BankDekEnc != testDekEnc {
		t.Errorf("Envelope fehlt: ciphertext=%q dek=%q", m.BankCiphertext, m.BankDekEnc)
	}
	if m.IBAN != "" {
		t.Errorf("Server lieferte Klartext-IBAN %q (darf nie passieren)", m.IBAN)
	}
}

// (7.2) Trainer erreicht die Route gar nicht.
func TestGet_TrainerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 2, "standard", []string{"trainer"})
	res := testutil.Get(t, srv, "/api/members/"+strconv.Itoa(id), tok)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("Trainer GET member: status %d, want 403", res.StatusCode)
	}
}

// (7.1) Trainer darf bank-details nicht schreiben.
func TestBankdaten_TrainerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 3, "standard", []string{"trainer"})
	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+strconv.Itoa(id)+"/bank-details", tok,
		map[string]any{"bank_ciphertext": testCiphertext, "bank_dek_enc": testDekEnc})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("Trainer bank-details: status %d, want 403", res.StatusCode)
	}
}

// (G2) Eigentümer liest seine Bankdaten NICHT mehr zurück.
func TestProfileMe_EigentuemerLiestKeineBankdaten(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	id := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?,?,?)`,
		id, testCiphertext, testDekEnc)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, userID, "standard", []string{"spieler"})

	res := testutil.Get(t, srv, "/api/profile/me", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET profile/me: status %d", res.StatusCode)
	}
	var resp struct {
		OwnMember memberBankResp `json:"own_member"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()
	if resp.OwnMember.IBAN != "" || resp.OwnMember.BankCiphertext != "" {
		t.Errorf("Eigentümer erhielt Bankdaten: iban=%q ciphertext=%q (G2: darf nicht)",
			resp.OwnMember.IBAN, resp.OwnMember.BankCiphertext)
	}
}

// (G2) Elternteil liest die Bankdaten des Kindes NICHT mehr zurück.
func TestProfileKind_ElternteilLiestKeineBankdaten(t *testing.T) {
	db := testutil.NewDB(t)
	parentUserID := testutil.CreateUser(t, db, "standard")
	childID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, parentUserID, childID)
	db.Exec(`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?,?,?)`,
		childID, testCiphertext, testDekEnc)
	srv := prodserver.New(t, db)
	tok := testutil.TokenWithIsParent(t, parentUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/profile/kind/"+strconv.Itoa(childID), tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET profile/kind: status %d", res.StatusCode)
	}
	var resp struct {
		Member memberBankResp `json:"member"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()
	if resp.Member.IBAN != "" || resp.Member.BankCiphertext != "" {
		t.Errorf("Elternteil erhielt Bankdaten: iban=%q ciphertext=%q (G2: darf nicht)",
			resp.Member.IBAN, resp.Member.BankCiphertext)
	}
}

func TestProfileKind_FremdesKind403(t *testing.T) {
	db := testutil.NewDB(t)
	strangerUserID := testutil.CreateUser(t, db, "standard")
	childID := testutil.CreateMember(t, db, 0) // nicht mit stranger verlinkt
	srv := prodserver.New(t, db)
	tok := testutil.TokenWithIsParent(t, strangerUserID, "standard", nil, true)

	res := testutil.Get(t, srv, "/api/profile/kind/"+strconv.Itoa(childID), tok)
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("fremdes Kind: status %d, want 403", res.StatusCode)
	}
}

// has_bank_data: true wenn member_sensitive-Row vorhanden.
func TestGetProfile_HasBankDataFlag(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?,?,?)`,
		memberID, testCiphertext, testDekEnc)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv, "/api/profile/me", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET profile/me: status %d", res.StatusCode)
	}
	var resp struct {
		OwnMember struct {
			HasBankData bool `json:"has_bank_data"`
		} `json:"own_member"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()
	if !resp.OwnMember.HasBankData {
		t.Error("has_bank_data sollte true sein wenn member_sensitive-Row vorhanden")
	}
}

// has_bank_data: false wenn keine member_sensitive-Row.
func TestGetProfile_NoBankData(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	testutil.CreateMember(t, db, userID)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv, "/api/profile/me", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET profile/me: status %d", res.StatusCode)
	}
	var resp struct {
		OwnMember struct {
			HasBankData bool `json:"has_bank_data"`
		} `json:"own_member"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()
	if resp.OwnMember.HasBankData {
		t.Error("has_bank_data sollte false sein wenn keine member_sensitive-Row vorhanden")
	}
}

// sepa_mandat und sepa_mandat_date werden im Profil korrekt ausgeliefert.
func TestGetProfile_SepaMandatFields(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	db.Exec(`UPDATE members SET sepa_mandat=1, sepa_mandat_date='2024-03-01' WHERE id=?`, memberID)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, userID, "standard", nil)

	res := testutil.Get(t, srv, "/api/profile/me", tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET profile/me: status %d", res.StatusCode)
	}
	var resp struct {
		OwnMember struct {
			SepaMandat     bool   `json:"sepa_mandat"`
			SepaMandatDate string `json:"sepa_mandat_date"`
		} `json:"own_member"`
	}
	json.NewDecoder(res.Body).Decode(&resp)
	res.Body.Close()
	if !resp.OwnMember.SepaMandat {
		t.Error("sepa_mandat sollte true sein")
	}
	if resp.OwnMember.SepaMandatDate == "" {
		t.Error("sepa_mandat_date sollte befüllt sein")
	}
}

// (Invariante) Schreiben legt den Envelope in member_sensitive ab; der Server sieht nie
// Klartext und Klartext-Felder werden mit 400 abgewiesen.
func TestBankdaten_EnvelopeGespeichert_KlartextAbgelehnt(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 1, "standard", []string{"kassierer"})

	// Envelope schreiben → 204
	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+strconv.Itoa(id)+"/bank-details", tok,
		map[string]any{"bank_ciphertext": testCiphertext, "bank_dek_enc": testDekEnc})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT bank-details (Envelope): status %d", res.StatusCode)
	}
	var ct, dek string
	db.QueryRow(`SELECT ciphertext, dek_enc_vorstand FROM member_sensitive WHERE member_id=?`, id).Scan(&ct, &dek)
	if ct != testCiphertext || dek != testDekEnc {
		t.Errorf("Envelope nicht gespeichert: ct=%q dek=%q", ct, dek)
	}
	// members.iban bleibt leer (kein serverseitiges Schreiben mehr)
	var iban string
	db.QueryRow(`SELECT COALESCE(iban,'') FROM members WHERE id=?`, id).Scan(&iban)
	if iban != "" {
		t.Errorf("members.iban beschrieben (%q) — sollte leer bleiben", iban)
	}

	// Klartext-IBAN → 400
	res = testutil.Do(t, srv, http.MethodPut, "/api/members/"+strconv.Itoa(id)+"/bank-details", tok,
		map[string]any{"iban": "DE89370400440532013000"})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("Klartext-IBAN: status %d, want 400", res.StatusCode)
	}
}
