package beitragslauf_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func TestExport_HappyPath(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())
	res := testutil.Post(t, srv, "/api/fee-run/export", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/xml" {
		t.Errorf("Content-Type = %q", ct)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "pain.008.001.08") {
		t.Errorf("kein pain.008-Namespace im Body")
	}
}

func TestExport_EinPmtInfBlockRCUR(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id1 := insertMember(t, db, "A", defaultMember())
	m2 := defaultMember()
	m2.memberNumber = "1099"
	id2 := insertMember(t, db, "B", m2)
	res := testutil.Post(t, srv, "/api/fee-run/export", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id1, id2}})
	body, _ := io.ReadAll(res.Body)
	str := string(body)
	if n := strings.Count(str, "<PmtInf>"); n != 1 {
		t.Errorf("PmtInf-Blöcke = %d, want 1", n)
	}
	if strings.Contains(str, "FRST") || strings.Count(str, "<SeqTp>RCUR</SeqTp>") != 1 {
		t.Errorf("SeqTp nicht ausschließlich RCUR")
	}
	if strings.Count(str, "<DrctDbtTxInf>") != 2 {
		t.Errorf("DrctDbtTxInf-Einträge != 2")
	}
}

func TestExport_VerwendungszweckFormat(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())
	res := testutil.Post(t, srv, "/api/fee-run/export", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}})
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "<Ustrd>Jahresbeitrag Saison 2027/28 – Mitgliedsnr. 1042</Ustrd>") {
		t.Errorf("Verwendungszweck-Format falsch:\n%s", body)
	}
}

func TestExport_FehlendeStammdaten400(t *testing.T) {
	srv, db, _ := setupSrv(t)
	db.Exec(`UPDATE clubs SET glaeubiger_id=NULL`)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())
	res := testutil.Post(t, srv, "/api/fee-run/export", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, want 400", res.StatusCode)
	}
}

func TestExport_ExcludedMember400(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.sepaMandat = 0 // ausgeschlossen
	id := insertMember(t, db, "Ohne", m)
	res := testutil.Post(t, srv, "/api/fee-run/export", tok(t),
		map[string]any{"saison_id": s, "member_ids": []int{id}})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, want 400", res.StatusCode)
	}
}

func TestExport_KassiererErlaubt(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())
	res := testutil.Post(t, srv, "/api/fee-run/export",
		testutil.Token(t, 5, "standard", []string{"kassierer"}),
		map[string]any{"saison_id": s, "member_ids": []int{id}})
	if res.StatusCode != http.StatusOK {
		t.Errorf("kassierer status %d, want 200", res.StatusCode)
	}
}

func TestExport_Forbidden(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	res := testutil.Post(t, srv, "/api/fee-run/export",
		testutil.Token(t, 9, "standard", []string{"spieler"}),
		map[string]any{"saison_id": s, "member_ids": []int{1}})
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.StatusCode)
	}
}

func TestConfirm_HaengtProtokollAn(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())
	body := map[string]any{"saison_id": s, "results": []map[string]any{
		{"member_id": id, "betrag_cent": 9600, "success": true},
	}}
	for i := 0; i < 2; i++ {
		if res := testutil.Post(t, srv, "/api/fee-run/confirm", tok(t), body); res.StatusCode != http.StatusOK {
			t.Fatalf("confirm %d: status %d", i, res.StatusCode)
		}
	}
	res := testutil.Get(t, srv, "/api/fee-run/protocol?saison_id="+itoa(s), tok(t))
	txt, _ := io.ReadAll(res.Body)
	if n := strings.Count(string(txt), "=== Lauf bestätigt"); n != 2 {
		t.Errorf("Protokoll-Blöcke = %d, want 2 (append-only):\n%s", n, txt)
	}
}

func TestConfirm_ErfolgUndFehlerGetrennt(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	ok := insertMember(t, db, "Ok", defaultMember())
	m2 := defaultMember()
	m2.memberNumber = "1200"
	fail := insertMember(t, db, "Fail", m2)
	body := map[string]any{"saison_id": s, "results": []map[string]any{
		{"member_id": ok, "betrag_cent": 9600, "success": true},
		{"member_id": fail, "betrag_cent": 22600, "success": false},
	}}
	testutil.Post(t, srv, "/api/fee-run/confirm", tok(t), body)
	res := testutil.Get(t, srv, "/api/fee-run/protocol?saison_id="+itoa(s), tok(t))
	txt, _ := io.ReadAll(res.Body)
	s2 := string(txt)
	if !strings.Contains(s2, "Erfolgreich (1)") || !strings.Contains(s2, "Nicht erfolgreich (1)") {
		t.Errorf("Erfolg/Fehler nicht getrennt:\n%s", s2)
	}
}

func TestConfirm_Forbidden(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	res := testutil.Post(t, srv, "/api/fee-run/confirm",
		testutil.Token(t, 9, "standard", []string{"spieler"}),
		map[string]any{"saison_id": s, "results": []map[string]any{}})
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.StatusCode)
	}
}

func TestProtocol_LeerWennKeinLauf(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	res := testutil.Get(t, srv, "/api/fee-run/protocol?saison_id="+itoa(s), tok(t))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	txt, _ := io.ReadAll(res.Body)
	if len(txt) != 0 {
		t.Errorf("erwartete leeren Body, got %q", txt)
	}
}

func TestProtocol_Forbidden(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	res := testutil.Get(t, srv, "/api/fee-run/protocol?saison_id="+itoa(s),
		testutil.Token(t, 9, "standard", []string{"spieler"}))
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.StatusCode)
	}
}
