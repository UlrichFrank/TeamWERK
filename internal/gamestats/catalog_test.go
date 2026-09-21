package gamestats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/bwhv"
)

// katalogServer zählt die Abrufe je Kommando — daran hängt die Aussage, dass
// ein Lauf den Katalog nur einmal lädt.
func katalogServer(t *testing.T, zaehler *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*zaehler++
		name := "catalog_bw.json"
		if r.URL.Query().Get("o") == "251" {
			name = "catalog_srm.json"
		}
		b, err := os.ReadFile(filepath.Join("..", "bwhv", "testdata", name))
		if err != nil {
			t.Fatalf("Fixture: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLoadCatalog_FindetVerbandsUndBezirksStaffeln(t *testing.T) {
	var n int
	srv := katalogServer(t, &n)
	cat, err := loadCatalog(context.Background(), bwhv.NewClientWithBase(srv.URL), 216)
	if err != nil {
		t.Fatalf("loadCatalog: %v", err)
	}
	if cat.Period == "" {
		t.Error("Spielzeit nicht ermittelt")
	}
	// Verbandsebene
	if e, ok := cat.lookup("mB-RL-BW"); !ok || e.ClassID != "161291" || e.SubOrgID != 0 {
		t.Errorf("mB-RL-BW = %+v (gefunden %v), erwartet ClassID 161291 auf Verbandsebene", e, ok)
	}
	// Bezirk: nur über den o-Parameter erreichbar
	if e, ok := cat.lookup("gD-BOL-SRM"); !ok || e.SubOrgID != 251 {
		t.Errorf("gD-BOL-SRM = %+v (gefunden %v), erwartet SubOrgID 251", e, ok)
	}
}

// Der Code kommt teils aus Freitext-Eingabe — Groß-/Kleinschreibung und
// Leerraum dürfen die Auflösung nicht kippen.
func TestLoadCatalog_CodeVergleichIstTolerant(t *testing.T) {
	var n int
	srv := katalogServer(t, &n)
	cat, _ := loadCatalog(context.Background(), bwhv.NewClientWithBase(srv.URL), 216)
	for _, variante := range []string{"mB-RL-BW", "mb-rl-bw", "  mB-RL-BW  "} {
		if _, ok := cat.lookup(variante); !ok {
			t.Errorf("Variante %q nicht gefunden", variante)
		}
	}
}

func TestLoadCatalog_UnbekannterCode(t *testing.T) {
	var n int
	srv := katalogServer(t, &n)
	cat, _ := loadCatalog(context.Background(), bwhv.NewClientWithBase(srv.URL), 216)
	if _, ok := cat.lookup("xY-GIBTS-NICHT"); ok {
		t.Error("erfundener Code wurde gefunden")
	}
}

// Der eigentliche Grund für den Cache: ohne ihn braucht die Auflösung von neun
// Staffelcodes rund hundert Abrufe. Der Katalog wird je Lauf EINMAL geladen,
// unabhängig davon, wie viele Codes daraus aufgelöst werden.
func TestLoadCatalog_WirdNurEinmalGeladen(t *testing.T) {
	var abrufe int
	srv := katalogServer(t, &abrufe)
	cat, err := loadCatalog(context.Background(), bwhv.NewClientWithBase(srv.URL), 216)
	if err != nil {
		t.Fatal(err)
	}
	nachLaden := abrufe

	for _, code := range []string{
		"mB-RL-BW", "mA-RL-BW", "mC-OL-3-BW", "wC-OL-2-BW",
		"gD-BOL-SRM", "mA-BOL-SRM", "mC-BOL-SRM", "wB-BOL-1-SRM",
	} {
		if _, ok := cat.lookup(code); !ok {
			t.Errorf("Code %q nicht im Katalog", code)
		}
	}
	if abrufe != nachLaden {
		t.Errorf("Auflösung löste %d zusätzliche Abrufe aus — der Katalog muss ohne Netz auskommen",
			abrufe-nachLaden)
	}
	// Perioden + Orgs + ein Katalog je Organisation: zweistellig, nicht dreistellig.
	if abrufe > 15 {
		t.Errorf("%d Abrufe für einen Katalog-Lauf — erwartet wenige (Perioden, Orgs, je Org ein Katalog)", abrufe)
	}
}

// Ein nicht erreichbarer Bezirk darf den Lauf nicht mitreißen: die Staffeln der
// übrigen Organisationen bleiben auflösbar.
func TestLoadCatalog_EinzelnerBezirkFaelltAus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("o") != "" {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		b, _ := os.ReadFile(filepath.Join("..", "bwhv", "testdata", "catalog_bw.json"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)

	cat, err := loadCatalog(context.Background(), bwhv.NewClientWithBase(srv.URL), 216)
	if err != nil {
		t.Fatalf("ein ausgefallener Bezirk darf den Lauf nicht kippen: %v", err)
	}
	if _, ok := cat.lookup("mB-RL-BW"); !ok {
		t.Error("Verbands-Staffel fehlt, obwohl nur der Bezirk ausfiel")
	}
	if _, ok := cat.lookup("gD-BOL-SRM"); ok {
		t.Error("Bezirks-Staffel gefunden, obwohl der Bezirk nicht erreichbar war")
	}
}

func TestLoadCatalog_LeererKatalogIstEinFehler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := os.ReadFile(filepath.Join("..", "bwhv", "testdata", "catalog_bw.json"))
		s := string(b)
		// Klassenliste leeren, Menü (Perioden/Orgs) behalten.
		i := strings.Index(s, `"classes"`)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(s[:i] + `"classes": []}}]`))
	}))
	t.Cleanup(srv.Close)

	if _, err := loadCatalog(context.Background(), bwhv.NewClientWithBase(srv.URL), 216); err == nil {
		t.Error("erwartet: Fehler bei leerem Katalog statt stiller Leerlauf")
	}
}
