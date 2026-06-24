package beitragslauf_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/crypto"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// TestExport_EntschluesseltVerschluesselteFelder (7.5): bei at-rest
// verschlüsselten Mitglieds- und Vereinsfeldern erzeugt der Export trotzdem
// korrektes SEPA-XML mit den Klartext-Werten.
func TestExport_EntschluesseltVerschluesselteFelder(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())

	enc := func(v string) string {
		e, err := crypto.Encrypt(v)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		return e
	}
	// Mitglieds- und Vereins-SEPA-Felder verschlüsseln (wie nach encrypt-pii).
	if _, err := db.Exec(`UPDATE members SET iban=?, account_holder=? WHERE id=?`,
		enc(validIBAN), enc("Max Test"), id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE clubs SET glaeubiger_id=?, iban=?, bic=?, kontoinhaber=?`,
		enc("DE98ZZZ09999999999"), enc(validIBAN), enc("GENODEF1S02"), enc("Team Stuttgart e.V.")); err != nil {
		t.Fatal(err)
	}

	res := testutil.Post(t, srv, "/api/fee-run/export", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("export status %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	str := string(body)

	if !strings.Contains(str, validIBAN) {
		t.Errorf("XML enthält die entschlüsselte Mitglieds-IBAN nicht:\n%s", str)
	}
	if !strings.Contains(str, "DE98ZZZ09999999999") {
		t.Errorf("XML enthält die entschlüsselte Gläubiger-ID nicht")
	}
	if strings.Contains(str, "v1:") {
		t.Errorf("XML enthält rohen Ciphertext (v1:) — Entschlüsselung fehlt")
	}
}
