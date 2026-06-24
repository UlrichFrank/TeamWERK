package members_test

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/crypto"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

const testPlainIBAN = "DE89370400440532013000"

type memberBankResp struct {
	IBAN          string `json:"iban"`
	AccountHolder string `json:"account_holder"`
}

func encBank(t *testing.T, s string) string {
	t.Helper()
	enc, err := crypto.Encrypt(s)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return enc
}

// TestGet_VorstandSiehtEntschluesselteIBAN (7.2): Berechtigte erhalten Klartext.
func TestGet_VorstandSiehtEntschluesselteIBAN(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	db.Exec(`UPDATE members SET iban=?, account_holder=? WHERE id=?`,
		encBank(t, testPlainIBAN), encBank(t, "Max Mustermann"), id)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 1, "standard", []string{"vorstand"})

	res := testutil.Get(t, srv, "/api/members/"+strconv.Itoa(id), tok)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET member: status %d", res.StatusCode)
	}
	var m memberBankResp
	json.NewDecoder(res.Body).Decode(&m)
	res.Body.Close()
	if m.IBAN != testPlainIBAN {
		t.Errorf("iban = %q, want entschlüsselt %q", m.IBAN, testPlainIBAN)
	}
	if m.AccountHolder != "Max Mustermann" {
		t.Errorf("account_holder = %q, want entschlüsselt", m.AccountHolder)
	}
}

// TestGet_TrainerForbidden (7.2): Trainer erreicht die Route gar nicht.
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

// TestBankdaten_TrainerForbidden (7.1): Trainer darf bank-details nicht schreiben.
func TestBankdaten_TrainerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 3, "standard", []string{"trainer"})
	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+strconv.Itoa(id)+"/bank-details", tok,
		map[string]any{"iban": testPlainIBAN})
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("Trainer bank-details: status %d, want 403", res.StatusCode)
	}
}

// TestProfileMe_EigentuemerSiehtEigeneIBAN (7.3).
func TestProfileMe_EigentuemerSiehtEigeneIBAN(t *testing.T) {
	db := testutil.NewDB(t)
	userID := testutil.CreateUser(t, db, "standard")
	id := testutil.CreateMember(t, db, userID)
	db.Exec(`UPDATE members SET iban=?, account_holder=? WHERE id=?`,
		encBank(t, testPlainIBAN), encBank(t, "Eigentümer"), id)
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
	if resp.OwnMember.IBAN != testPlainIBAN {
		t.Errorf("own iban = %q, want %q", resp.OwnMember.IBAN, testPlainIBAN)
	}
}

// TestProfileKind_ElternteilSiehtIBAN + FremdesKind403 (7.4).
func TestProfileKind_ElternteilSiehtIBAN(t *testing.T) {
	db := testutil.NewDB(t)
	parentUserID := testutil.CreateUser(t, db, "standard")
	childID := testutil.CreateMember(t, db, 0)
	db.Exec(`INSERT INTO family_links (parent_user_id, member_id) VALUES (?,?)`, parentUserID, childID)
	db.Exec(`UPDATE members SET iban=?, account_holder=? WHERE id=?`,
		encBank(t, testPlainIBAN), encBank(t, "Kind"), childID)
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
	if resp.Member.IBAN != testPlainIBAN {
		t.Errorf("child iban = %q, want %q", resp.Member.IBAN, testPlainIBAN)
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

// TestInvariant_BankdatenNieKlartextInDB (7.8): nach einem Schreibzugriff über
// die API steht in der Spalte niemals der Klartext.
func TestInvariant_BankdatenNieKlartextInDB(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 1, "standard", []string{"kassierer"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+strconv.Itoa(id)+"/bank-details", tok,
		map[string]any{"iban": testPlainIBAN, "account_holder": "Max Mustermann"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT bank-details: status %d", res.StatusCode)
	}

	var iban, holder string
	db.QueryRow(`SELECT iban, account_holder FROM members WHERE id=?`, id).Scan(&iban, &holder)
	if iban == testPlainIBAN || !crypto.IsEncryptedString(iban) {
		t.Errorf("iban-Spalte enthält Klartext oder ist unverschlüsselt: %q", iban)
	}
	if holder == "Max Mustermann" || !crypto.IsEncryptedString(holder) {
		t.Errorf("account_holder-Spalte enthält Klartext: %q", holder)
	}
}
