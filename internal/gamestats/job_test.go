package gamestats

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/bwhv"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// BWHV_ORG_ID=0 schaltet den Abruf hart ab. Das Ablageverzeichnis taugt dafür
// nicht: sein Default greift immer, der Wert ist nie leer.
func TestSchedulerJob_OrgIDNullSchaltetAb(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "26/27")
	if _, err := db.Exec(`UPDATE seasons SET is_active = 1 WHERE id = ?`, seasonID); err != nil {
		t.Fatal(err)
	}
	// Ablageverzeichnis gesetzt, Org aus: es darf trotzdem nichts laufen.
	SchedulerJob(db, &appconfig.Config{BwhvReportDir: t.TempDir(), BwhvOrgID: 0})()
}

func TestSchedulerJob_NilConfigIstUnschaedlich(t *testing.T) {
	db := testutil.NewDB(t)
	SchedulerJob(db, nil)()
}

// Ohne aktive Saison gibt es nichts abzurufen — der Job endet vor jedem
// Netzzugriff.
func TestSchedulerJob_OhneAktiveSaisonKeinAbruf(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := &appconfig.Config{BwhvReportDir: t.TempDir(), BwhvOrgID: 216}
	SchedulerJob(db, cfg)()
}

// Eine aktive Saison ohne Staffeln erzeugt ebenfalls keinen Abruf: DueStaffeln
// liefert eine leere Liste, und der Job endet vor jedem Netzzugriff.
func TestSchedulerJob_AktiveSaisonOhneStaffelnKeinAbruf(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "26/27")
	if _, err := db.Exec(`UPDATE seasons SET is_active = 1 WHERE id = ?`, seasonID); err != nil {
		t.Fatal(err)
	}
	cfg := &appconfig.Config{BwhvReportDir: t.TempDir(), BwhvOrgID: 216}
	SchedulerJob(db, cfg)()
}

// bwhvServer bedient Katalog (cmd=po) und Spielplan (cmd=ps) aus den
// Fixtures; Berichte gibt es nicht (404 — der Bericht bleibt pending).
func bwhvServer(t *testing.T, abrufe *atomic.Int32) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		abrufe.Add(1)
		if !strings.HasSuffix(r.URL.Path, "if_g_json.php") {
			http.NotFound(w, r)
			return
		}
		name := "catalog_bw.json"
		switch {
		case r.URL.Query().Get("cmd") == "ps":
			name = "schedule_mb_rl_bw.json"
		case r.URL.Query().Get("o") == "251":
			name = "catalog_srm.json"
		}
		b, err := os.ReadFile(filepath.Join("..", "bwhv", "testdata", name))
		if err != nil {
			t.Errorf("Fixture: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// faelligeStaffel legt eine aktive Saison mit einer Staffel an, die am
// 19.09.2026 um 20:00 fällig ist (Anwurf 16:00, kein Bericht).
func faelligeStaffel(t *testing.T) (*Store, int) {
	t.Helper()
	db, s, seasonID, staffelID := newStaffel(t)
	if _, err := db.Exec(`UPDATE seasons SET is_active = 1 WHERE id = ?`, seasonID); err != nil {
		t.Fatal(err)
	}
	seedGame(t, s, staffelID, "900001", "2026-09-19", "16:00", "")
	return s, staffelID
}

func polledAt(t *testing.T, s *Store, staffelID int) string {
	t.Helper()
	var v sql.NullString
	if err := s.db.QueryRow(`SELECT polled_at FROM bwhv_staffeln WHERE id = ?`, staffelID).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v.String
}

// Regression: der Scheduler läuft als kurzlebiger Cron-Prozess. Startete der
// Job den Abruf im Hintergrund, endete der Prozess, bevor der Abruf fertig war
// — auf Prod lief so kein einziger automatischer Abruf. Der Lauf muss
// abgeschlossen sein, wenn der Job zurückkehrt.
func TestSchedulerJob_LaufIstBeimZurueckkehrenAbgeschlossen(t *testing.T) {
	s, staffelID := faelligeStaffel(t)
	var abrufe atomic.Int32
	srv := bwhvServer(t, &abrufe)
	cfg := &appconfig.Config{BwhvReportDir: t.TempDir(), BwhvOrgID: 216}

	runPollTick(s.db, cfg, bwhv.NewClientWithBase(srv.URL), at(t, "2026-09-19", "20:00"))

	if polledAt(t, s, staffelID) == "" {
		t.Fatal("polled_at nicht gesetzt — der Spielplan wurde nicht vor dem Zurückkehren gespeichert")
	}
	if n := countRows(t, s.db, "bwhv_games"); n <= 1 {
		t.Errorf("bwhv_games = %d, erwartet den Spielplan aus der Fixture", n)
	}
}

// Hält ein noch laufender Vorgänger die Sperre, überspringt der Tick — kein
// einziger Abruf, kein doppelter Lauf.
func TestSchedulerJob_GesperrtUeberspringt(t *testing.T) {
	s, staffelID := faelligeStaffel(t)
	var abrufe atomic.Int32
	srv := bwhvServer(t, &abrufe)
	cfg := &appconfig.Config{BwhvReportDir: t.TempDir(), BwhvOrgID: 216}

	release, ok, err := tryLock(filepath.Join(cfg.BwhvReportDir, pollLockName))
	if err != nil || !ok {
		t.Fatalf("Sperre für den Test nicht erhalten: ok=%v err=%v", ok, err)
	}
	defer release()

	runPollTick(s.db, cfg, bwhv.NewClientWithBase(srv.URL), at(t, "2026-09-19", "20:00"))

	if n := abrufe.Load(); n != 0 {
		t.Errorf("%d Abrufe trotz gehaltener Sperre, erwartet 0", n)
	}
	if v := polledAt(t, s, staffelID); v != "" {
		t.Errorf("polled_at = %q, erwartet leer", v)
	}
}

func TestTryLock_ZweiteSperreScheitertBisZurFreigabe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", pollLockName)
	release, ok, err := tryLock(path)
	if err != nil || !ok {
		t.Fatalf("erste Sperre: ok=%v err=%v", ok, err)
	}
	if _, ok2, err := tryLock(path); err != nil || ok2 {
		t.Fatalf("zweite Sperre: ok=%v err=%v, erwartet ok=false ohne Fehler", ok2, err)
	}
	release()
	release2, ok3, err := tryLock(path)
	if err != nil || !ok3 {
		t.Fatalf("nach Freigabe: ok=%v err=%v", ok3, err)
	}
	release2()
}
