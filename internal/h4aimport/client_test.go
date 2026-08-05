package h4aimport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient hängt den Client an einen httptest-TLS-Server (kein echtes H4A).
func newTestClient(ts *httptest.Server) *Client {
	c := NewClient()
	c.http = ts.Client() // vertraut dem Test-Zertifikat
	c.base = ts.URL      // https://127.0.0.1:port → requireHTTPS erfüllt
	return c
}

func TestLogin_Success(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><a href="?logout">ABMELDEN</a></html>`))
	}))
	defer ts.Close()

	if err := newTestClient(ts).Login(context.Background(), "v_109", "geheim"); err != nil {
		t.Fatalf("Login sollte erfolgreich sein: %v", err)
	}
}

func TestLogin_FailureHidesPassword(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<form name="login">...</form>`)) // kein ABMELDEN
	}))
	defer ts.Close()

	const pw = "supersecret123"
	err := newTestClient(ts).Login(context.Background(), "v_109", pw)
	if err == nil {
		t.Fatal("Login sollte fehlschlagen")
	}
	if strings.Contains(err.Error(), pw) {
		t.Fatalf("Fehlermeldung enthält das Passwort: %v", err)
	}
}

func TestRequireHTTPS_RejectsPlainHTTP(t *testing.T) {
	c := NewClient()
	c.base = "http://meinh4a.handball4all.de"
	if err := c.Login(context.Background(), "u", "p"); err == nil {
		t.Fatal("erwartet Ablehnung von http:// base")
	}
}

func TestFetchGamesHTML_ExtractsContainer(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			t.Errorf("X-Requested-With-Header fehlt")
		}
		w.Header().Set("Content-Type", "application/json")
		// Ein kurzes as-Kommando (<50 Zeichen, ignoriert) + der echte Container.
		w.Write([]byte(`{"xjxobj":[` +
			`{"cmd":"as","id":"noise","data":"kurz"},` +
			`{"cmd":"as","id":"gametable_container","data":"<table class=\"ge_gameday\">viel viel viel viel viel Inhalt hier drin</table>"}` +
			`]}`))
	}))
	defer ts.Close()

	got, err := newTestClient(ts).FetchGamesHTML(context.Background(), "142")
	if err != nil {
		t.Fatalf("FetchGamesHTML: %v", err)
	}
	if !strings.Contains(got, "ge_gameday") {
		t.Errorf("erwartetes Container-HTML nicht extrahiert: %q", got)
	}
}

func TestFetchPeriods_ParsesOptions(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><body>
			<select id="ge_periods" name="ge_periods">
			<option value="141">Feldrunde 2025</option>
			<option value="142" selected>Hallenrunde 2026/2027</option>
			</select></body></html>`))
	}))
	defer ts.Close()

	periods, err := newTestClient(ts).FetchPeriods(context.Background())
	if err != nil {
		t.Fatalf("FetchPeriods: %v", err)
	}
	if len(periods) != 2 {
		t.Fatalf("erwartet 2 Perioden, bekommen %d: %+v", len(periods), periods)
	}
	if periods[1].ID != "142" || periods[1].Name != "Hallenrunde 2026/2027" {
		t.Errorf("Periode falsch geparst: %+v", periods[1])
	}
}
