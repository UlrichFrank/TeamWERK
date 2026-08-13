package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	// Importiert das db-Package, damit init() den Wrapping-Driver
	// "sqlite-busy-counting" registriert. Tests gehen so durch denselben Pfad
	// wie Produktion, inklusive teamwerk_sqlite_busy_total-Counter.
	"github.com/teamstuttgart/teamwerk/internal/db"
)

var dbCounter atomic.Uint64

var (
	goldenOnce  sync.Once
	goldenBytes []byte
	goldenErr   error
)

// goldenSchema migriert einmal pro Testbinary eine vollständige SQLite-Datei
// (alle Migrationen + seedBaseData) und hält ihre Bytes im Speicher vor.
// NewDB kopiert diese Bytes statt bei jedem Test alle Migrationen erneut
// abzuspielen: bei ~500 Testfunktionen allein in internal/auth, internal/games
// und internal/members sprengte das wiederholte db.Migrate() unter `-race`
// (~10× langsamer als ohne, siehe Makefile test-race) den 10-Minuten-Timeout
// von `go test` (deploy.yml gate, gescheitert am 2026-07-17).
func goldenSchema() ([]byte, error) {
	goldenOnce.Do(func() {
		dir, err := os.MkdirTemp("", "teamwerk-golden")
		if err != nil {
			goldenErr = fmt.Errorf("golden tempdir: %w", err)
			return
		}
		defer os.RemoveAll(dir)

		path := filepath.Join(dir, "golden.db")
		database, err := sql.Open("sqlite-busy-counting", fmt.Sprintf("file:%s?_pragma=foreign_keys=on", path))
		if err != nil {
			goldenErr = fmt.Errorf("golden open: %w", err)
			return
		}
		// Ephemere Einmal-Datei — Durability ist irrelevant, nur Tempo zählt.
		if _, err := database.Exec(`PRAGMA synchronous=OFF`); err != nil {
			database.Close()
			goldenErr = fmt.Errorf("golden pragma: %w", err)
			return
		}
		if err := db.Migrate(database, db.MigrationsFS); err != nil {
			database.Close()
			goldenErr = fmt.Errorf("golden migrate: %w", err)
			return
		}
		if err := seedBaseData(database); err != nil {
			database.Close()
			goldenErr = fmt.Errorf("golden seed: %w", err)
			return
		}
		if err := database.Close(); err != nil {
			goldenErr = fmt.Errorf("golden close: %w", err)
			return
		}
		goldenBytes, err = os.ReadFile(path)
		if err != nil {
			goldenErr = fmt.Errorf("golden read: %w", err)
		}
	})
	return goldenBytes, goldenErr
}

// NewDB liefert eine aus der golden schema kopierte SQLite-Datei mit allen
// Migrationen + Basisdaten. Jeder Test bekommt seine eigene, in t.TempDir()
// liegende Kopie — eine echte Datei statt des früheren ":memory:"-Shared-Cache-
// Tricks, dadurch teilen sich mehrere Goroutinen (z.B. HTTP-Handler in
// httptest-Servern) die DB weiterhin ohne SetMaxOpenConns(1) (das
// Concurrent-Claim-Tests serialisieren würde) — SQLite-Dateien sind dafür
// naturgemäß ausgelegt, ganz ohne cache=shared-Sonderfall.
// Die Verbindung wird beim Testende automatisch geschlossen.
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	schema, err := goldenSchema()
	if err != nil {
		t.Fatalf("testutil.NewDB golden schema: %v", err)
	}
	path := filepath.Join(t.TempDir(), fmt.Sprintf("testdb_%d.db", dbCounter.Add(1)))
	if err := os.WriteFile(path, schema, 0o600); err != nil {
		t.Fatalf("testutil.NewDB write: %v", err)
	}
	// busy_timeout wie in db.Open (Produktion): ohne ihn scheitert ein zweiter
	// gleichzeitiger Schreiber SOFORT mit SQLITE_BUSY, statt kurz auf die Sperre zu
	// warten. Genau daran kippte TestClaimDutySlot_NoConcurrentOverclaim sporadisch —
	// der geprüfte Zielzustand („genau einer gewinnt") war gar nicht erreichbar, beide
	// Anfragen liefen in 409/500.
	//
	// journal_mode bleibt bewusst der Default (delete) statt WAL wie in Produktion:
	// WAL legt -wal/-shm neben die Datei, und in Tests mit Hintergrund-Goroutinen
	// (videos-Transcode-Worker) kollidieren diese Dateien mit dem RemoveAll von
	// t.TempDir() → "directory not empty". Für die Sperr-Semantik, um die es hier geht,
	// ist der busy_timeout der wirksame Teil.
	database, err := sql.Open("sqlite-busy-counting",
		fmt.Sprintf("file:%s?_pragma=foreign_keys=on&_pragma=busy_timeout(5000)", path))
	if err != nil {
		t.Fatalf("testutil.NewDB open: %v", err)
	}
	// Ephemere Datei je Test — Durability ist irrelevant, nur Tempo zählt.
	if _, err := database.Exec(`PRAGMA synchronous=OFF`); err != nil {
		database.Close()
		t.Fatalf("testutil.NewDB pragma: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// seedBaseData fills Stammvereine und Beitragssätze, die früher Teil der
// (jetzt kollabierten) Migrationen 043/046/047/048 waren. Damit Tests gegen
// die deterministischen Seed-IDs (z.B. "TV Cannstatt 1846" = 8) weiter laufen,
// ohne dass das Produktiv-Schema Seeds enthalten muss.
func seedBaseData(database *sql.DB) error {
	if _, err := database.Exec(`INSERT OR IGNORE INTO stammvereine (id, name, sort_order) VALUES
		(1,  'SKG Gablenberg 1884',                 1),
		(2,  'SKG Stuttgart Max-Eyth-See 1898',     2),
		(3,  'SportKultur Stuttgart',               3),
		(4,  'Spvgg 1897 Cannstatt',                4),
		(5,  'TB Gaisburg 1886',                    5),
		(6,  'TB Untertürkheim 1888',               6),
		(7,  'TSV Stuttgart-Münster 1875/99',       7),
		(8,  'TV Cannstatt 1846',                   8),
		(9,  'HSG Cannstatt/Münster/Max-Eyth-See',  9),
		(10, 'HSG Oberer Neckar',                  10),
		(11, 'Hbi Weilimdorf/Feuerbach',           11),
		(12, 'HSG Gablenberg-Gaisburg',            12),
		(13, 'Sportvg Feuerbach',                  13),
		(14, 'HSV Zuffenhausen',                   14),
		(15, 'TuS Stuttgart',                      15),
		(16, 'TV Obertürkheim',                    16),
		(17, 'TSV Korntal',                        17),
		(18, 'SV Stuttgarter Kickers',             18),
		(19, 'TV Fellbach',                        19),
		(20, 'TV Deizisau',                        20),
		(21, 'SG Asperg',                          21),
		(22, 'SG Hegensberg-Liebersbronn',         22);`); err != nil {
		return fmt.Errorf("seed stammvereine: %w", err)
	}
	if _, err := database.Exec(`INSERT OR IGNORE INTO beitrags_saetze (kategorie, betrag_eur, valid_from) VALUES
		('aktiv_ohne', 22600, '2026-07-01'),
		('aktiv_mit',   9600, '2026-07-01'),
		('passiv',      6000, '2026-07-01'),
		('passiv',      6000, '2027-01-01');`); err != nil {
		return fmt.Errorf("seed beitrags_saetze: %w", err)
	}
	return nil
}
