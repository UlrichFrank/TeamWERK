package bwhv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureServer liefert die eingecheckten Antworten abhängig von den
// Query-Parametern und hält fest, welche URLs abgefragt wurden.
func fixtureServer(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.String())
		q := r.URL.Query()
		switch {
		case strings.HasSuffix(r.URL.Path, "sboPublicReports.php"):
			b, err := os.ReadFile(filepath.Join("testdata", "spielbericht_905272.pdf"))
			if err != nil {
				t.Fatalf("Fixture nicht lesbar: %v", err)
			}
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write(b)
		case q.Get("cmd") == "po" && q.Get("o") == "251":
			serveFixture(t, w, "catalog_srm.json")
		case q.Get("cmd") == "po":
			serveFixture(t, w, "catalog_bw.json")
		case q.Get("cmd") == "ps":
			serveFixture(t, w, "schedule_mb_rl_bw.json")
		default:
			serveFixture(t, w, "error_permission_denied.json")
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &seen
}

func serveFixture(t *testing.T, w http.ResponseWriter, name string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("Fixture %s nicht lesbar: %v", name, err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}

func newTestClient(t *testing.T, base string) *Client {
	t.Helper()
	return NewClientWithBase(base) // ohne Höflichkeitspause, siehe NewClientWithBase
}

func TestFetchCatalog_VerbandsebeneLiefertStaffelcodes(t *testing.T) {
	srv, _ := fixtureServer(t)
	classes, err := newTestClient(t, srv.URL).FetchCatalog(context.Background(), 216, 0, "142")
	if err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}
	byCode := map[string]Class{}
	for _, c := range classes {
		byCode[c.Sname] = c
	}
	c, ok := byCode["mB-RL-BW"]
	if !ok {
		t.Fatalf("mB-RL-BW fehlt im Katalog, vorhanden: %v", keys(byCode))
	}
	if c.ID != "161291" {
		t.Errorf("gClassID = %q, erwartet 161291", c.ID)
	}
	if !strings.Contains(c.Lname, "B-Jugend") {
		t.Errorf("Langname = %q, erwartet etwas mit B-Jugend", c.Lname)
	}
}

// Die og/o-Falle: ein Bezirk wird über o selektiert, og bleibt der Verband.
// Ein Zugriff mit og=<bezirk> wird serverseitig abgelehnt (design.md §1.2).
func TestFetchCatalog_BezirkWirdUeberSubOrgAufgeloest(t *testing.T) {
	srv, seen := fixtureServer(t)
	classes, err := newTestClient(t, srv.URL).FetchCatalog(context.Background(), 216, 251, "142")
	if err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}
	var got []string
	for _, c := range classes {
		got = append(got, c.Sname)
	}
	for _, want := range []string{"mA-BOL-SRM", "mC-BOL-SRM", "gD-BOL-SRM", "wB-BOL-1-SRM"} {
		if !contains(got, want) {
			t.Errorf("Staffel %s fehlt, vorhanden: %v", want, got)
		}
	}
	q := mustQuery(t, (*seen)[0])
	if q.Get("og") != "216" {
		t.Errorf("og = %q, muss die Verbands-Org bleiben (og=<bezirk> liefert HTTP 401)", q.Get("og"))
	}
	if q.Get("o") != "251" {
		t.Errorf("o = %q, erwartet 251", q.Get("o"))
	}
}

func TestFetchCatalog_VerbandsebeneSendetKeinO(t *testing.T) {
	srv, seen := fixtureServer(t)
	if _, err := newTestClient(t, srv.URL).FetchCatalog(context.Background(), 216, 0, "142"); err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}
	if q := mustQuery(t, (*seen)[0]); q.Has("o") {
		t.Errorf("o wurde gesendet (%q), auf Verbandsebene darf es fehlen", q.Get("o"))
	}
}

// Eine Fehlerantwort kommt als Objekt statt als Array. Sie muss ein klarer
// Fehler werden und nicht als leeres Ergebnis durchrutschen.
func TestFetchCatalog_PermissionDeniedIstEinFehler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveFixture(t, w, "error_permission_denied.json")
	}))
	t.Cleanup(srv.Close)
	_, err := newTestClient(t, srv.URL).FetchCatalog(context.Background(), 251, 0, "142")
	if err == nil {
		t.Fatal("erwartet: Fehler bei permission denied, bekam nil")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("Fehlertext = %q, sollte die Begründung des Dienstes nennen", err)
	}
}

func TestFetchSchedule_LiefertVollsaisonUndTabelle(t *testing.T) {
	srv, seen := fixtureServer(t)
	sch, err := newTestClient(t, srv.URL).FetchSchedule(context.Background(), 216, 0, "142", "161291")
	if err != nil {
		t.Fatalf("FetchSchedule: %v", err)
	}
	if len(sch.Games) != 8 {
		t.Fatalf("Begegnungen = %d, erwartet 8", len(sch.Games))
	}
	if len(sch.Table) == 0 {
		t.Error("Tabelle ist leer — sie fällt beim selben Abruf ab und braucht kein PDF")
	}
	if !strings.Contains(sch.ReportURL, "sboPublicReports.php") {
		t.Errorf("ReportURL = %q, erwartet das repURL-Präfix aus head", sch.ReportURL)
	}
	if q := mustQuery(t, (*seen)[0]); q.Get("ca") != "1" {
		t.Errorf("ca = %q, erwartet 1 (sonst kommen nur künftige Spiele)", q.Get("ca"))
	}
}

// sGID ist das Bereitschaftssignal: gespielte Begegnungen tragen eine ID,
// künftige liefern die ZAHL 0 — deshalb muss das Feld beide Formen lesen.
func TestFetchSchedule_SGIDNurBeiGespieltenBegegnungen(t *testing.T) {
	srv, _ := fixtureServer(t)
	sch, err := newTestClient(t, srv.URL).FetchSchedule(context.Background(), 216, 0, "142", "161291")
	if err != nil {
		t.Fatalf("FetchSchedule: %v", err)
	}
	var mit, ohne int
	for _, g := range sch.Games {
		if g.SGID != "" {
			mit++
			if !g.Played() {
				t.Errorf("Spiel %s trägt sGID, aber kein Ergebnis", g.GameNo)
			}
		} else {
			ohne++
			if g.Played() {
				t.Errorf("Spiel %s hat ein Ergebnis, aber keine sGID", g.GameNo)
			}
		}
	}
	if mit != 3 || ohne != 5 {
		t.Errorf("mit sGID = %d, ohne = %d; erwartet 3 und 5", mit, ohne)
	}
}

func TestFetchSchedule_FelderEinerGespieltenBegegnung(t *testing.T) {
	srv, _ := fixtureServer(t)
	sch, err := newTestClient(t, srv.URL).FetchSchedule(context.Background(), 216, 0, "142", "161291")
	if err != nil {
		t.Fatalf("FetchSchedule: %v", err)
	}
	var g *Game
	for i := range sch.Games {
		if sch.Games[i].GameNo == "905272" {
			g = &sch.Games[i]
		}
	}
	if g == nil {
		t.Fatal("Begegnung 905272 fehlt")
	}
	if g.SGID != "3504061" {
		t.Errorf("sGID = %q, erwartet 3504061", g.SGID)
	}
	if g.Date != "2026-09-20" {
		t.Errorf("Datum = %q, erwartet 2026-09-20 (aus 20.09.26 normalisiert)", g.Date)
	}
	if g.HallNumber != "21005" {
		t.Errorf("Hallennummer = %q, erwartet 21005 (= venues.hall_number)", g.HallNumber)
	}
	if *g.HomeGoals != 29 || *g.GuestGoals != 25 {
		t.Errorf("Endstand = %d:%d, erwartet 29:25", *g.HomeGoals, *g.GuestGoals)
	}
	if *g.HomeGoalsHT != 14 || *g.GuestGoalsHT != 13 {
		t.Errorf("Halbzeit = %d:%d, erwartet 14:13", *g.HomeGoalsHT, *g.GuestGoalsHT)
	}
}

// Ein nicht gespieltes Spiel trägt " " (Leerzeichen) in den Ergebnisfeldern.
// Ohne TrimSpace liefe es als 0:0 durch und sähe aus wie ein torloses Remis.
func TestFetchSchedule_KuenftigeBegegnungHatKeinErgebnis(t *testing.T) {
	srv, _ := fixtureServer(t)
	sch, err := newTestClient(t, srv.URL).FetchSchedule(context.Background(), 216, 0, "142", "161291")
	if err != nil {
		t.Fatalf("FetchSchedule: %v", err)
	}
	for _, g := range sch.Games {
		if g.SGID != "" {
			continue
		}
		if g.HomeGoals != nil || g.GuestGoals != nil {
			t.Errorf("Spiel %s ohne sGID trägt ein Ergebnis %v:%v", g.GameNo, g.HomeGoals, g.GuestGoals)
		}
	}
}

func TestFetchReport_LiefertPDF(t *testing.T) {
	srv, _ := fixtureServer(t)
	c := newTestClient(t, srv.URL)
	b, err := c.FetchReport(context.Background(), srv.URL+"/misc/sboPublicReports.php?sGID=", "3504061")
	if err != nil {
		t.Fatalf("FetchReport: %v", err)
	}
	if !strings.HasPrefix(string(b), "%PDF") {
		t.Errorf("Antwort trägt keine PDF-Signatur")
	}
}

func TestFetchReport_LeereSGIDWirdNichtAbgerufen(t *testing.T) {
	srv, seen := fixtureServer(t)
	c := newTestClient(t, srv.URL)
	for _, sgid := range []string{"", "0"} {
		if _, err := c.FetchReport(context.Background(), srv.URL+"/misc/sboPublicReports.php?sGID=", sgid); err == nil {
			t.Errorf("sGID %q: erwartet Fehler, bekam nil", sgid)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("es wurde abgerufen (%v), obwohl keine sGID vorlag", *seen)
	}
}

// Eine HTML-Fehlerseite mit HTTP 200 darf nicht als Bericht durchgehen und
// später im Parser als unverständliches Dokument auflaufen.
func TestFetchReport_NichtPDFWirdAbgelehnt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>Wartungsarbeiten</body></html>"))
	}))
	t.Cleanup(srv.Close)
	_, err := newTestClient(t, srv.URL).FetchReport(context.Background(), srv.URL+"/x?sGID=", "3504061")
	if err == nil {
		t.Fatal("erwartet: Fehler bei Nicht-PDF-Antwort")
	}
}

func TestClient_SendetUserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		serveFixture(t, w, "catalog_bw.json")
	}))
	t.Cleanup(srv.Close)
	if _, err := newTestClient(t, srv.URL).FetchCatalog(context.Background(), 216, 0, ""); err != nil {
		t.Fatalf("FetchCatalog: %v", err)
	}
	if !strings.Contains(ua, "TeamWERK") {
		t.Errorf("User-Agent = %q, muss TeamWERK nennen (design.md §3)", ua)
	}
}

func keys(m map[string]Class) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func mustQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("URL %q nicht lesbar: %v", raw, err)
	}
	return u.Query()
}
