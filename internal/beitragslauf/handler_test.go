package beitragslauf_test

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

type previewSummary struct {
	IncludedCount   int `json:"included_count"`
	ExcludedCount   int `json:"excluded_count"`
	WarnedCount     int `json:"warned_count"`
	TotalCent       int `json:"total_cent"`
	ExcludedCent    int `json:"excluded_cent"`
	GesamtsummeCent int `json:"gesamtsumme_cent"`
}

type previewResp struct {
	Items   []previewItem  `json:"items"`
	Summary previewSummary `json:"summary"`
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

type confirmResp struct {
	SaisonLabel          string `json:"saison_label"`
	Erfolgreich          int    `json:"erfolgreich"`
	NichtErfolgreich     int    `json:"nicht_erfolgreich"`
	SummeErfolgreichCent int    `json:"summe_erfolgreich_cent"`
}

func readProtokoll(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "beitragslauf_2027-28.txt"))
	if err != nil {
		t.Fatalf("Protokolldatei lesen: %v", err)
	}
	return string(data)
}

// Confirm schreibt Protokoll und liefert die Aggregat-Zahlen 1:1 aus dem Body.
func TestConfirm_HappyPath_SchreibtProtokoll(t *testing.T) {
	srv, db, dir := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())

	res := testutil.Post(t, srv, "/api/fee-run/confirm", tok(t), map[string]any{
		"saison_id": s,
		"results":   []map[string]any{{"member_id": id, "betrag_cent": 9600, "success": true}},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("confirm status %d, want 200", res.StatusCode)
	}
	var cr confirmResp
	json.NewDecoder(res.Body).Decode(&cr)
	res.Body.Close()
	if cr.SaisonLabel != "2027/28" || cr.Erfolgreich != 1 || cr.NichtErfolgreich != 0 || cr.SummeErfolgreichCent != 9600 {
		t.Errorf("confirm-JSON unerwartet: %+v", cr)
	}

	content := readProtokoll(t, dir)
	for _, want := range []string{"=== Lauf bestätigt", "durch test@test.local (User #1)", "Erfolgreich (1)", "Mitgl.-Nr 1042", "96,00"} {
		if !strings.Contains(content, want) {
			t.Errorf("Protokoll enthält %q nicht:\n%s", want, content)
		}
	}
}

// Das Protokoll darf keine IBAN führen (ProtokollResult hat kein IBAN-Feld).
func TestConfirm_ProtokollOhneIBAN(t *testing.T) {
	srv, db, dir := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())

	res := testutil.Post(t, srv, "/api/fee-run/confirm", tok(t), map[string]any{
		"saison_id": s,
		"results":   []map[string]any{{"member_id": id, "betrag_cent": 9600, "success": true}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("confirm status %d, want 200", res.StatusCode)
	}
	content := readProtokoll(t, dir)
	// Struktur-Tripwire (kein echter Security-Test): die eigentliche Garantie ist
	// Modell B (Server hält keinen Entschlüsselungsschlüssel) + `ProtokollResult` ohne
	// IBAN-Feld (protokoll.go). Dieser Runtime-Check kann unter erreichbarem Handler-
	// Verhalten nicht rot werden — er dokumentiert die Absicht und fängt nur eine grobe
	// künftige Regression (jemand hängt Bankdaten an das Protokoll).
	if strings.Contains(content, validIBAN) || strings.Contains(content, "DE89") {
		t.Errorf("Protokoll enthält IBAN (darf nie passieren):\n%s", content)
	}
	// Die eigentlichen Zähne dieses Tests: das Protokoll-Format enthält genau die
	// erlaubten Felder (Mitgliedsnummer + Betrag).
	if !strings.Contains(content, "Mitgl.-Nr 1042") || !strings.Contains(content, "96,00") {
		t.Errorf("Protokoll fehlen erwartete Felder:\n%s", content)
	}
}

// Erfolgs- und Fehlblock werden getrennt gezählt und gerendert.
func TestConfirm_MixedSuccessFailure(t *testing.T) {
	srv, db, dir := setupSrv(t)
	s := insertSeason2027(t, db)
	id1 := insertMember(t, db, "Max", defaultMember())
	m2 := defaultMember()
	m2.memberNumber = "1043"
	id2 := insertMember(t, db, "Moritz", m2)

	res := testutil.Post(t, srv, "/api/fee-run/confirm", tok(t), map[string]any{
		"saison_id": s,
		"results": []map[string]any{
			{"member_id": id1, "betrag_cent": 9600, "success": true},
			{"member_id": id2, "betrag_cent": 9600, "success": false},
		},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("confirm status %d, want 200", res.StatusCode)
	}
	var cr confirmResp
	json.NewDecoder(res.Body).Decode(&cr)
	res.Body.Close()
	if cr.Erfolgreich != 1 || cr.NichtErfolgreich != 1 {
		t.Errorf("confirm-JSON: %+v, want erfolgreich=1 nicht_erfolgreich=1", cr)
	}
	// Money-kritisch: der fehlgeschlagene Einzug darf NICHT in die bestätigte Summe fließen.
	if cr.SummeErfolgreichCent != 9600 {
		t.Errorf("summe_erfolgreich_cent=%d, want 9600 (Fehlschlag darf nicht mitzählen)", cr.SummeErfolgreichCent)
	}
	content := readProtokoll(t, dir)
	if !strings.Contains(content, "Erfolgreich (1)") || !strings.Contains(content, "Nicht erfolgreich (1)") {
		t.Errorf("Protokoll fehlen Erfolgs-/Fehlblock:\n%s", content)
	}
}

// Zwei Läufe hängen zwei Header-Blöcke an dieselbe Datei (append-only).
func TestConfirm_AppendOnly_ZweiLaeufe(t *testing.T) {
	srv, db, dir := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())

	for _, success := range []bool{true, false} {
		res := testutil.Post(t, srv, "/api/fee-run/confirm", tok(t), map[string]any{
			"saison_id": s,
			"results":   []map[string]any{{"member_id": id, "betrag_cent": 9600, "success": success}},
		})
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("confirm status %d, want 200", res.StatusCode)
		}
	}
	content := readProtokoll(t, dir)
	if n := strings.Count(content, "=== Lauf bestätigt"); n != 2 {
		t.Errorf("erwarte 2 Lauf-Blöcke, got %d:\n%s", n, content)
	}
}

func TestConfirm_UnbekannteSaison404(t *testing.T) {
	srv, db, dir := setupSrv(t)
	id := insertMember(t, db, "Max", defaultMember())
	res := testutil.Post(t, srv, "/api/fee-run/confirm", tok(t), map[string]any{
		"saison_id": 99999,
		"results":   []map[string]any{{"member_id": id, "betrag_cent": 9600, "success": true}},
	})
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status %d, want 404", res.StatusCode)
	}
	// Invariante (spec): bei unbekannter Saison darf KEIN Protokoll entstehen.
	if files, _ := filepath.Glob(filepath.Join(dir, "beitragslauf_*.txt")); len(files) != 0 {
		t.Errorf("kein Protokoll darf bei unbekannter Saison entstehen, got %v", files)
	}
}

func TestConfirm_UngueltigerBody400(t *testing.T) {
	srv, db, _ := setupSrv(t)
	insertSeason2027(t, db)
	// saison_id als String → Decode-Fehler im int-Feld.
	res := testutil.Post(t, srv, "/api/fee-run/confirm", tok(t), map[string]any{
		"saison_id": "keine-zahl",
		"results":   []map[string]any{},
	})
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status %d, want 400", res.StatusCode)
	}
	if !strings.Contains(string(body), "ungültiger Body") {
		t.Errorf("Body enthält nicht 'ungültiger Body': %s", body)
	}
}

// Nach Confirm liefert Protocol den Klartext-Report zurück (ohne IBAN).
func TestProtocol_RueckleseNachConfirm(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	id := insertMember(t, db, "Max", defaultMember())
	cRes := testutil.Post(t, srv, "/api/fee-run/confirm", tok(t), map[string]any{
		"saison_id": s,
		"results":   []map[string]any{{"member_id": id, "betrag_cent": 9600, "success": true}},
	})
	cRes.Body.Close()
	if cRes.StatusCode != http.StatusOK {
		t.Fatalf("confirm status %d", cRes.StatusCode)
	}

	res := testutil.Get(t, srv, "/api/fee-run/protocol?saison_id="+itoa(s), tok(t))
	body, _ := io.ReadAll(res.Body)
	ct := res.Header.Get("Content-Type")
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("protocol status %d, want 200", res.StatusCode)
	}
	if !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type %q, want text/plain*", ct)
	}
	if !strings.Contains(string(body), "Mitgl.-Nr 1042") {
		t.Errorf("Protocol-Body fehlt Mitgl.-Nr:\n%s", body)
	}
	if strings.Contains(string(body), validIBAN) {
		t.Errorf("Protocol-Body enthält IBAN:\n%s", body)
	}
}

func TestProtocol_UnbekannteSaison404(t *testing.T) {
	srv, _, _ := setupSrv(t)
	res := testutil.Get(t, srv, "/api/fee-run/protocol?saison_id=99999", tok(t))
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status %d, want 404", res.StatusCode)
	}
}

// Saison ohne bislang bestätigten Lauf → 200 mit leerem Body.
func TestProtocol_OhneDateiLeer200(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	res := testutil.Get(t, srv, "/api/fee-run/protocol?saison_id="+itoa(s), tok(t))
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	if len(body) != 0 {
		t.Errorf("erwarte leeren Body, got %d Bytes: %q", len(body), body)
	}
}

// Unterjähriger Austritt mit Stammverein → aktiv_mit, halbiert (austritt), 4800.
func TestPreview_UnterjaehrigerAustrittMitStammvereinAktivMit(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	m := defaultMember()
	m.status = "ausgetreten"
	tv := 8 // TV Cannstatt 1846
	m.homeClubID = &tv
	id := insertMember(t, db, "Geht", m)
	db.Exec(`UPDATE members SET join_date='2020-01-01', exit_date='2027-11-01' WHERE id=?`, id)
	it, ok := itemFor(getPreview(t, srv, s), id)
	if !ok || !it.Included || it.Kategorie != "aktiv_mit" || !it.Half || it.HalfReason != "austritt" || it.BetragCent != 4800 {
		t.Errorf("got %+v (ok=%v), want included aktiv_mit half austritt 4800", it, ok)
	}
}

// Summary aggregiert einbezogene (total_cent) und ausgeschlossene (excluded_cent) Beträge.
func TestPreview_SummaryTotals(t *testing.T) {
	srv, db, _ := setupSrv(t)
	s := insertSeason2027(t, db)
	insertMember(t, db, "Aktiv", defaultMember()) // aktiv_ohne, included, 22600

	ausgeschlossen := defaultMember()
	ausgeschlossen.sepaMandat = 0 // ausgeschlossen (kein_sepa_mandat), Betrag 22600 sichtbar
	ausgeschlossen.memberNumber = "1043"
	insertMember(t, db, "Ohne", ausgeschlossen)

	pr := getPreview(t, srv, s)
	if pr.Summary.IncludedCount != 1 {
		t.Errorf("included_count=%d, want 1", pr.Summary.IncludedCount)
	}
	if pr.Summary.TotalCent != 22600 {
		t.Errorf("total_cent=%d, want 22600", pr.Summary.TotalCent)
	}
	if pr.Summary.ExcludedCount != 1 || pr.Summary.ExcludedCent != 22600 {
		t.Errorf("excluded_count=%d excluded_cent=%d, want 1/22600", pr.Summary.ExcludedCount, pr.Summary.ExcludedCent)
	}
	if pr.Summary.GesamtsummeCent != 45200 {
		t.Errorf("gesamtsumme_cent=%d, want 45200", pr.Summary.GesamtsummeCent)
	}
}
