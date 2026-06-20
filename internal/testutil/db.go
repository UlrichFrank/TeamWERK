package testutil

import (
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/db"
	_ "modernc.org/sqlite"
)

var dbCounter atomic.Uint64

// NewDB opens a fresh in-memory SQLite database with all migrations applied.
// Each test gets its own named shared-cache database so that multiple goroutines
// (e.g. HTTP handlers in httptest servers) can share the migrated schema without
// needing SetMaxOpenConns(1), which would serialize concurrent-claim tests.
// The connection is closed automatically when the test ends.
func NewDB(t *testing.T) *sql.DB {
	t.Helper()
	name := fmt.Sprintf("testdb_%d", dbCounter.Add(1))
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys=on", name)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("testutil.NewDB open: %v", err)
	}
	if err := db.Migrate(database, db.MigrationsFS); err != nil {
		database.Close()
		t.Fatalf("testutil.NewDB migrate: %v", err)
	}
	if err := seedBaseData(database); err != nil {
		database.Close()
		t.Fatalf("testutil.NewDB seed: %v", err)
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
