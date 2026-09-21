package gamestats

import (
	"testing"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// Ohne konfiguriertes Ablageverzeichnis tut der Job nichts. Das ist der Schalter,
// mit dem eine Instanz ohne BWHV-Anbindung den Abruf abgeschaltet lässt.
func TestSchedulerJob_OhneAblageverzeichnisPassiertNichts(t *testing.T) {
	db := testutil.NewDB(t)
	job := SchedulerJob(db, &appconfig.Config{})
	job() // darf nicht panicken und nichts abrufen
}

func TestSchedulerJob_NilConfigIstUnschaedlich(t *testing.T) {
	db := testutil.NewDB(t)
	SchedulerJob(db, nil)()
}

// Ohne aktive Saison gibt es nichts abzurufen — der Job endet vor jedem
// Netzzugriff. testutil.NewDB liefert eine Datei-DB; eine In-Memory-DB trüge
// die Goroutine des Jobs nicht (docs/agent/07-testing.md).
func TestSchedulerJob_OhneAktiveSaisonKeinAbruf(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := &appconfig.Config{BwhvReportDir: t.TempDir(), BwhvOrgID: 216}
	SchedulerJob(db, cfg)()
}

// Eine aktive Saison ohne Staffeln erzeugt ebenfalls keinen Abruf: DueStaffeln
// liefert eine leere Liste, und der Job startet gar keine Goroutine.
func TestSchedulerJob_AktiveSaisonOhneStaffelnKeinAbruf(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "26/27")
	if _, err := db.Exec(`UPDATE seasons SET is_active = 1 WHERE id = ?`, seasonID); err != nil {
		t.Fatal(err)
	}
	cfg := &appconfig.Config{BwhvReportDir: t.TempDir(), BwhvOrgID: 216}
	SchedulerJob(db, cfg)()
}
