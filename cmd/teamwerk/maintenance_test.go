package main

import (
	"path/filepath"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/db"
)

// dbPathForTest legt eine frische SQLite-Datei im Temp-Verzeichnis an und
// führt alle Migrationen aus, damit maintenanceToggle das Update auf einer
// gültigen Schema-Version machen kann.
func dbPathForTest(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "teamwerk.db")
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Migrate(database, db.MigrationsFS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.Close()
	return path
}

func readMaintenanceValue(t *testing.T, path string) string {
	t.Helper()
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer database.Close()
	var value string
	if err := database.QueryRow(
		`SELECT value FROM system_settings WHERE key='maintenance_mode'`,
	).Scan(&value); err != nil {
		t.Fatalf("query: %v", err)
	}
	return value
}

func TestCLI_MaintenanceOn(t *testing.T) {
	path := dbPathForTest(t)
	if got := readMaintenanceValue(t, path); got != "off" {
		t.Fatalf("Vor Toggle: erwartet 'off', bekam %q", got)
	}

	if err := maintenanceToggle([]string{"on", "--db", path}); err != nil {
		t.Fatalf("maintenanceToggle(on): %v", err)
	}
	if got := readMaintenanceValue(t, path); got != "on" {
		t.Errorf("Nach on-Toggle: erwartet 'on', bekam %q", got)
	}
}

func TestCLI_MaintenanceOff(t *testing.T) {
	path := dbPathForTest(t)
	if err := maintenanceToggle([]string{"on", "--db", path}); err != nil {
		t.Fatalf("preset on: %v", err)
	}

	if err := maintenanceToggle([]string{"off", "--db", path}); err != nil {
		t.Fatalf("maintenanceToggle(off): %v", err)
	}
	if got := readMaintenanceValue(t, path); got != "off" {
		t.Errorf("Nach off-Toggle: erwartet 'off', bekam %q", got)
	}
}

func TestCLI_MaintenanceInvalidArg(t *testing.T) {
	if err := maintenanceToggle([]string{"garbage"}); err == nil {
		t.Error("erwartet Fehler bei ungültigem Argument")
	}
	if err := maintenanceToggle(nil); err == nil {
		t.Error("erwartet Fehler bei fehlendem Argument")
	}
}
