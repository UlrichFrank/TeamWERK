package members_test

import (
	"database/sql"
	"net/http"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/testutil/prodserver"
)

func TestMembers_KassiererDarfLesen(t *testing.T) {
	db := testutil.NewDB(t)
	testutil.CreateMember(t, db, 0)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 1, "standard", []string{"kassierer"})
	if res := testutil.Get(t, srv, "/api/members", tok); res.StatusCode != http.StatusOK {
		t.Errorf("kassierer GET /api/members: status %d, want 200", res.StatusCode)
	}
}

func TestMembers_SpielerVerboten(t *testing.T) {
	db := testutil.NewDB(t)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 9, "standard", []string{"spieler"})
	if res := testutil.Get(t, srv, "/api/members", tok); res.StatusCode != http.StatusForbidden {
		t.Errorf("spieler GET /api/members: status %d, want 403", res.StatusCode)
	}
}

func TestBankdaten_KassiererUpdatetNurBankfelder(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	db.Exec(`UPDATE members SET status='aktiv', beitragsfrei=1, first_name='Vorname', last_name='Nachname' WHERE id=?`, id)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 5, "standard", []string{"kassierer"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+itoaTest(id)+"/bank-details", tok, map[string]any{
		"iban": "DE89370400440532013000", "sepa_mandat": true, "sepa_mandat_date": "2026-05-01",
		"account_holder": "Vorname Nachname", "street": "Neue Str. 5", "zip": "70000", "city": "Stuttgart",
	})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT bankdaten: status %d", res.StatusCode)
	}

	var iban, status, first string
	var beitragsfrei int
	db.QueryRow(`SELECT iban, status, first_name, beitragsfrei FROM members WHERE id=?`, id).
		Scan(&iban, &status, &first, &beitragsfrei)
	if iban != "DE89370400440532013000" {
		t.Errorf("iban nicht aktualisiert: %q", iban)
	}
	if status != "aktiv" || first != "Vorname" || beitragsfrei != 1 {
		t.Errorf("Nicht-Bankfelder verändert: status=%q first=%q beitragsfrei=%d", status, first, beitragsfrei)
	}
}

func TestBankdaten_UngueltigeIBAN400(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 5, "standard", []string{"kassierer"})
	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+itoaTest(id)+"/bank-details", tok,
		map[string]any{"iban": "DE88370400440532013000"})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, want 400", res.StatusCode)
	}
}

func TestBankdaten_Forbidden(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 9, "standard", []string{"spieler"})
	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+itoaTest(id)+"/bank-details", tok,
		map[string]any{"iban": "DE89370400440532013000"})
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.StatusCode)
	}
}

func TestBankdaten_SpielerForbidden(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 9, "standard", []string{"spieler"})
	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+itoaTest(id)+"/bank-details", tok,
		map[string]any{"beitragsfrei": true, "beitragsfrei_grund": "kein aktiver Sportler mehr"})
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.StatusCode)
	}
}

func TestBankdaten_KassiererPflegtBeitragsfreiGrund(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	db.Exec(`UPDATE members SET status='aktiv', first_name='Vorname', last_name='Nachname' WHERE id=?`, id)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 5, "standard", []string{"kassierer"})

	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+itoaTest(id)+"/bank-details", tok,
		map[string]any{
			"iban":               "DE89370400440532013000",
			"beitragsfrei":       true,
			"beitragsfrei_grund": "kein aktiver Sportler mehr",
		})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT bankdaten: status %d", res.StatusCode)
	}

	var iban, status, first string
	var beitragsfrei int
	var grund sql.NullString
	db.QueryRow(
		`SELECT iban, status, first_name, beitragsfrei, beitragsfrei_grund FROM members WHERE id=?`, id).
		Scan(&iban, &status, &first, &beitragsfrei, &grund)
	if iban != "DE89370400440532013000" {
		t.Errorf("iban: got %q", iban)
	}
	if beitragsfrei != 1 || !grund.Valid || grund.String != "kein aktiver Sportler mehr" {
		t.Errorf("beitragsfrei/grund: got %d / %v %q, want 1 / true %q",
			beitragsfrei, grund.Valid, grund.String, "kein aktiver Sportler mehr")
	}
	// Stammdaten unverändert
	if status != "aktiv" || first != "Vorname" {
		t.Errorf("Stammdaten verändert: status=%q first=%q", status, first)
	}
}

func TestBankdaten_BeitragsfreiFalseClearsGrund(t *testing.T) {
	db := testutil.NewDB(t)
	id := testutil.CreateMember(t, db, 0)
	db.Exec(`UPDATE members SET beitragsfrei=1, beitragsfrei_grund='Zweitspielrecht' WHERE id=?`, id)
	srv := prodserver.New(t, db)
	tok := testutil.Token(t, 5, "standard", []string{"kassierer"})

	// Trotz mitgesendetem Grund-Wert MUSS beitragsfrei=false den Grund leeren.
	res := testutil.Do(t, srv, http.MethodPut, "/api/members/"+itoaTest(id)+"/bank-details", tok,
		map[string]any{"beitragsfrei": false, "beitragsfrei_grund": "wird ignoriert"})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("PUT bankdaten: status %d", res.StatusCode)
	}

	var beitragsfrei int
	var grund sql.NullString
	db.QueryRow(
		`SELECT beitragsfrei, beitragsfrei_grund FROM members WHERE id=?`, id).
		Scan(&beitragsfrei, &grund)
	if beitragsfrei != 0 {
		t.Errorf("beitragsfrei: got %d, want 0", beitragsfrei)
	}
	if grund.Valid {
		t.Errorf("beitragsfrei_grund: got %q, want NULL", grund.String)
	}
}

func itoaTest(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
