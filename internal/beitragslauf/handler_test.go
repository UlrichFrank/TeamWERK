package beitragslauf_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/teamstuttgart/teamwerk/internal/auth"
	"github.com/teamstuttgart/teamwerk/internal/beitragslauf"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

const validIBAN = "DE89370400440532013000"

type memberSpec struct {
	status       string
	memberNumber string
	iban         string
	sepaMandat   int
	mandatPath   string
	mandatDate   string
	street       string
	zip          string
	city         string
	homeClub     string
	homeClubID   *int
	beitragsfrei int
}

func defaultMember() memberSpec {
	return memberSpec{
		status: "aktiv", memberNumber: "1042", iban: validIBAN, sepaMandat: 1,
		mandatPath: "/uploads/m.pdf", mandatDate: "2026-05-01",
		street: "Hauptstr. 12", zip: "70182", city: "Stuttgart",
	}
}

func insertMember(t *testing.T, db *sql.DB, first string, m memberSpec) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO members
		(first_name, last_name, status, member_number, sepa_mandat, sepa_mandat_path,
		 sepa_mandat_date, street, zip, city, home_club, home_club_id, beitragsfrei)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		first, "Test", m.status, m.memberNumber, m.sepaMandat, m.mandatPath,
		m.mandatDate, m.street, m.zip, m.city, m.homeClub, m.homeClubID, m.beitragsfrei)
	if err != nil {
		t.Fatalf("insertMember: %v", err)
	}
	id, _ := res.LastInsertId()
	// Modell B: Bankdaten liegen im member_sensitive-Envelope. Eine "vorhandene IBAN" im
	// Spec bedeutet einen vorhandenen Envelope (HasBank=true); der Server entschlüsselt nicht.
	if m.iban != "" {
		if _, err := db.Exec(`INSERT INTO member_sensitive (member_id, ciphertext, dek_enc_vorstand) VALUES (?, 'CT', 'DEK')`, id); err != nil {
			t.Fatalf("insertMember sensitive: %v", err)
		}
	}
	return int(id)
}

// season 2027/28 → Stichtag 2027-07-01, alle 3 Kategorien gültig
func insertSeason2027(t *testing.T, db *sql.DB) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO seasons (name, start_date, end_date, is_active) VALUES ('2027/28','2027-09-01','2028-06-30',1)`)
	if err != nil {
		t.Fatalf("insertSeason: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// season 2026/27 → Stichtag 2026-07-01; deckt den Datums-Bug ab, bei dem der
// Passiv-Satz erst ab 2027-01-01 galt (Migration 046 ergänzt 2026-07-01).
func insertSeason2026(t *testing.T, db *sql.DB) int {
	t.Helper()
	res, err := db.Exec(`INSERT INTO seasons (name, start_date, end_date, is_active) VALUES ('2026/27','2026-07-01','2027-06-30',1)`)
	if err != nil {
		t.Fatalf("insertSeason: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func setupSrv(t *testing.T) (*httptest.Server, *sql.DB, string) {
	t.Helper()
	db := testutil.NewDB(t)
	// Modell B: Vereins-SEPA als Envelope (Server entschlüsselt nicht).
	db.Exec(`INSERT INTO clubs (name, sepa_ciphertext, sepa_dek_enc)
		VALUES ('Team Stuttgart','CLUBCT','CLUBDEK')`)
	dir := t.TempDir()
	h := beitragslauf.NewHandler(db, hub.NewHub(), dir)
	srv := testutil.NewServer(t, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireClubFunction("vorstand", "kassierer"))
			r.Get("/api/fee-run/preview", h.Preview)
			r.Post("/api/fee-run/export-data", h.ExportData)
			r.Post("/api/fee-run/confirm", h.Confirm)
			r.Get("/api/fee-run/protocol", h.Protocol)
		})
	})
	return srv, db, dir
}

func tok(t *testing.T) string { return testutil.Token(t, 1, "standard", []string{"vorstand"}) }

type previewItem struct {
	MemberID   int      `json:"member_id"`
	Kategorie  string   `json:"kategorie"`
	BetragCent int      `json:"betrag_cent"`
	Half       bool     `json:"half"`
	HalfReason string   `json:"half_reason"`
	Included   bool     `json:"included"`
	Warnings   []string `json:"warnings"`
	Exclusions []string `json:"exclusions"`
}

type previewResp struct {
	Items []previewItem `json:"items"`
}

func getPreview(t *testing.T, srv *httptest.Server, saisonID int) previewResp {
	t.Helper()
	res := testutil.Get(t, srv, "/api/fee-run/preview?saison_id="+itoa(saisonID), tok(t))
	if res.StatusCode != http.StatusOK {
		t.Fatalf("preview status %d", res.StatusCode)
	}
	var pr previewResp
	json.NewDecoder(res.Body).Decode(&pr)
	return pr
}

func itoa(n int) string {
	b := []byte{}
	if n == 0 {
		return "0"
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func itemFor(pr previewResp, id int) (previewItem, bool) {
	for _, it := range pr.Items {
		if it.MemberID == id {
			return it, true
		}
	}
	return pr.Items[0], false
}

func TestPreview_AktivMitStammverein(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	tvCannstatt := 8 // Seed-ID "TV Cannstatt 1846"
	m.homeClubID = &tvCannstatt
	id := insertMember(t, db, "Max", m)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if !it.Included || it.Kategorie != "aktiv_mit" || it.BetragCent != 9600 {
		t.Errorf("got %+v", it)
	}
}

func TestPreview_AktivOhneStammverein(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Anna", defaultMember())
	it, _ := itemFor(getPreview(t, srv, s), id)
	if !it.Included || it.Kategorie != "aktiv_ohne" || it.BetragCent != 22600 {
		t.Errorf("got %+v", it)
	}
}

func TestPreview_PassivVollerBeitrag(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.status = "passiv"
	id := insertMember(t, db, "Paul", m)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if !it.Included || it.Kategorie != "passiv" || it.BetragCent != 6000 {
		t.Errorf("got %+v", it)
	}
}

// Regressionstest für den Datums-Bug: In Saison 2026/27 (Stichtag 2026-07-01)
// wurde ein passives Mitglied fälschlich mit kein_beitragssatz ausgeschlossen,
// weil der Passiv-Satz erst ab 2027-01-01 galt. Migration 046 behebt das.
func TestPreview_PassivSaison2026(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2026(t, db)
	m := defaultMember()
	m.status = "passiv"
	id := insertMember(t, db, "Petra", m)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if !it.Included || it.Kategorie != "passiv" || it.BetragCent != 6000 {
		t.Errorf("got %+v, want included passiv 6000", it)
	}
	if contains(it.Exclusions, "kein_beitragssatz") {
		t.Errorf("passives Mitglied darf nicht mit kein_beitragssatz ausgeschlossen werden: %+v", it.Exclusions)
	}
}

func TestPreview_AusschlussOhneMandat(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.sepaMandat = 0
	id := insertMember(t, db, "Ohne", m)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if it.Included || !contains(it.Exclusions, "kein_sepa_mandat") {
		t.Errorf("got %+v", it)
	}
	// Betrag bleibt sichtbar, damit das Frontend "nicht abbuchbar" ausweisen kann.
	if it.BetragCent != 22600 {
		t.Errorf("BetragCent=%d, want 22600 (aktiv_ohne) trotz Ausschluss", it.BetragCent)
	}
}

func TestPreview_BeitragsfreiOhneBetrag(t *testing.T) {
	// Gegenstück: beitragsfrei darf keinen Betrag tragen (zählt nicht zur
	// Gesamtsumme).
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.beitragsfrei = 1
	id := insertMember(t, db, "Frei", m)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if it.Included || it.BetragCent != 0 {
		t.Errorf("beitragsfrei: included=%v betrag=%d, want false/0", it.Included, it.BetragCent)
	}
}

func TestPreview_AusschlussOhneIBAN(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.iban = ""
	id := insertMember(t, db, "NoIban", m)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if it.Included || !contains(it.Exclusions, "iban_fehlt") {
		t.Errorf("got %+v", it)
	}
}

// Hinweis: Die IBAN-Gültigkeitsprüfung (iban_ungueltig) ist unter Modell B clientseitig
// (der Server kann die verschlüsselte IBAN nicht prüfen). Abgedeckt durch
// web/src/lib/sepa.test.ts (isValidIBAN, Parität zu internal/sepa/iban.go). Der Server
// schließt nur "keine Bankdaten" aus (iban_fehlt) — siehe TestPreview_AusschlussOhneBankdaten.
func TestPreview_AusschlussOhneBankdaten(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.iban = "" // kein member_sensitive-Envelope → HasBank=false
	id := insertMember(t, db, "NoBank", m)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if it.Included || !contains(it.Exclusions, "iban_fehlt") {
		t.Errorf("got %+v", it)
	}
}

// Ohne home_club_id-Zuordnung gilt das Mitglied deterministisch als aktiv_ohne;
// der frühere Freitext-/Fuzzy-Abgleich und die Warnung home_club_unklar entfallen.
func TestPreview_KeineHomeClubUnklarWarnung(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.homeClub = "FC Bayern" // Freitext bleibt als Audit-Spur, beeinflusst die Kategorie aber nicht
	id := insertMember(t, db, "Fcb", m)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if !it.Included || it.Kategorie != "aktiv_ohne" {
		t.Errorf("got %+v, want included aktiv_ohne", it)
	}
	if contains(it.Warnings, "home_club_unklar") {
		t.Errorf("home_club_unklar-Warnung sollte nicht mehr auftreten: %+v", it.Warnings)
	}
}

func TestPreview_BeitragsfreiAusgeschlossen(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.beitragsfrei = 1
	id := insertMember(t, db, "Frei", m)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if it.Included || !contains(it.Exclusions, "beitragsfrei") {
		t.Errorf("got %+v", it)
	}
}

// Unterjähriger Eintritt (join_date im Saisonfenster) → halber Jahresbeitrag.
func TestPreview_NeumitgliedZahltHalbenBeitrag(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db) // Fenster 2027-09-01 .. 2028-06-30
	m := defaultMember()
	id := insertMember(t, db, "Neu", m)
	db.Exec(`UPDATE members SET join_date='2027-09-15' WHERE id=?`, id)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if !it.Included || it.BetragCent != 11300 || !it.Half || it.HalfReason != "eintritt" {
		t.Errorf("Neumitglied zahlt nicht halben Beitrag: %+v", it)
	}
}

// Ganzjähriges Bestandsmitglied (join_date vor Saisonstart) → voller Beitrag.
func TestPreview_GanzjaehrigZahltVoll(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Alt", defaultMember())
	db.Exec(`UPDATE members SET join_date='2020-01-01' WHERE id=?`, id)
	it, _ := itemFor(getPreview(t, srv, s), id)
	if !it.Included || it.BetragCent != 22600 || it.Half {
		t.Errorf("Bestandsmitglied zahlt nicht vollen Beitrag: %+v", it)
	}
}

// Unterjähriger Austritt (ausgetreten + exit_date im Fenster) → einbezogen, halb.
func TestPreview_UnterjaehrigerAustrittEinbezogen(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.status = "ausgetreten"
	id := insertMember(t, db, "Geht", m)
	db.Exec(`UPDATE members SET join_date='2020-01-01', exit_date='2027-11-01' WHERE id=?`, id)
	it, ok := itemFor(getPreview(t, srv, s), id)
	if !ok || !it.Included || it.BetragCent != 11300 || !it.Half || it.HalfReason != "austritt" {
		t.Errorf("unterjähriger Austritt nicht halb einbezogen: %+v (ok=%v)", it, ok)
	}
}

// Früher ausgetreten (exit_date vor Saison) → nicht im Preview.
func TestPreview_FruehererAustrittNichtImPreview(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	aktivID := insertMember(t, db, "Aktiv", defaultMember())
	m := defaultMember()
	m.status = "ausgetreten"
	m.memberNumber = "9001"
	gone := insertMember(t, db, "Weg", m)
	db.Exec(`UPDATE members SET exit_date='2025-01-01' WHERE id=?`, gone)
	pr := getPreview(t, srv, s)
	if len(pr.Items) != 1 || pr.Items[0].MemberID != aktivID {
		t.Fatalf("erwarte nur aktives Mitglied, got %d: %+v", len(pr.Items), pr.Items)
	}
}

// Erstes Abrechnungsjahr (is_inaugural) → alle Eingeschlossenen zahlen halb.
func TestPreview_ErstjahrAlleHalb(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	db.Exec(`UPDATE seasons SET is_inaugural=1 WHERE id=?`, s)
	id := insertMember(t, db, "Egal", defaultMember())
	db.Exec(`UPDATE members SET join_date='2020-01-01' WHERE id=?`, id) // ganzjährig, trotzdem halb
	it, _ := itemFor(getPreview(t, srv, s), id)
	if !it.Included || it.BetragCent != 11300 || !it.Half || it.HalfReason != "erstjahr" {
		t.Errorf("Erstjahr halbiert nicht: %+v", it)
	}
}

// Mitglieder mit Status ausgetreten/honorar/anwaerter sind fachlich nie Teil
// des Beitragslaufs und werden deshalb gar nicht erst geladen — weder in der
// Preview-Tabelle noch in den Summen.
func TestPreview_StatusOhneBeitragNichtImPreview(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	aktivID := insertMember(t, db, "Aktiv", defaultMember())
	for i, status := range []string{"ausgetreten", "honorar", "anwaerter"} {
		m := defaultMember()
		m.status = status
		m.memberNumber = "200" + itoa(i)
		insertMember(t, db, status, m)
	}
	pr := getPreview(t, srv, s)
	if len(pr.Items) != 1 || pr.Items[0].MemberID != aktivID {
		t.Fatalf("erwarte exakt 1 Item (aktiv), got %d: %+v", len(pr.Items), pr.Items)
	}
}

func TestPreview_Forbidden(t *testing.T) {
	srv, db, _ := setupSrv(t)
	insertSeason2027(t, db)
	res := testutil.Get(t, srv, "/api/fee-run/preview?saison_id=1", testutil.Token(t, 9, "standard", []string{"spieler"}))
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status %d, want 403", res.StatusCode)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
