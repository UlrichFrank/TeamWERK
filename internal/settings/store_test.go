package settings_test

import (
	"context"
	"testing"

	"github.com/teamstuttgart/teamwerk/internal/settings"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

func TestStore_InitialValue_DefaultsOff(t *testing.T) {
	db := testutil.NewDB(t)
	s := settings.NewStoreForTest(db, 0)
	if s.MaintenanceMode() {
		t.Fatal("frische DB: maintenance_mode sollte off sein")
	}
}

func TestStore_SetMaintenanceMode_PersistsAndUpdatesCache(t *testing.T) {
	db := testutil.NewDB(t)
	admin := testutil.CreateUser(t, db, "admin")
	s := settings.NewStoreForTest(db, 0)

	if err := s.SetMaintenanceMode(context.Background(), true, admin); err != nil {
		t.Fatalf("set on: %v", err)
	}
	if !s.MaintenanceMode() {
		t.Fatal("Cache sollte nach SetMaintenanceMode(true) on sein")
	}

	// DB-Row prüfen
	var value string
	var updatedBy int
	if err := db.QueryRow(
		`SELECT value, updated_by FROM system_settings WHERE key='maintenance_mode'`,
	).Scan(&value, &updatedBy); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if value != "on" {
		t.Errorf("DB value: erwartet 'on', bekam %q", value)
	}
	if updatedBy != admin {
		t.Errorf("updated_by: erwartet %d, bekam %d", admin, updatedBy)
	}

	// Wieder aus
	if err := s.SetMaintenanceMode(context.Background(), false, admin); err != nil {
		t.Fatalf("set off: %v", err)
	}
	if s.MaintenanceMode() {
		t.Fatal("Cache sollte nach SetMaintenanceMode(false) off sein")
	}
}

func TestStore_Reload_PicksUpExternalChanges(t *testing.T) {
	db := testutil.NewDB(t)
	s := settings.NewStoreForTest(db, 0)

	// Externer UPDATE (z. B. via CLI-Subcommand) — Cache weiß noch nichts davon.
	if _, err := db.Exec(
		`UPDATE system_settings SET value='on' WHERE key='maintenance_mode'`,
	); err != nil {
		t.Fatalf("external update: %v", err)
	}
	if s.MaintenanceMode() {
		t.Fatal("vor Reload: Cache sollte noch off zeigen")
	}

	if err := s.Reload(context.Background()); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !s.MaintenanceMode() {
		t.Fatal("nach Reload: Cache sollte on zeigen")
	}
}

func TestStore_Snapshot_ReturnsMetadata(t *testing.T) {
	db := testutil.NewDB(t)
	admin := testutil.CreateUser(t, db, "admin")
	s := settings.NewStoreForTest(db, 0)

	if err := s.SetMaintenanceMode(context.Background(), true, admin); err != nil {
		t.Fatalf("set on: %v", err)
	}

	snap, err := s.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if !snap.Enabled {
		t.Error("snapshot.Enabled: erwartet true")
	}
	if !snap.UpdatedByID.Valid || snap.UpdatedByID.Int64 != int64(admin) {
		t.Errorf("snapshot.UpdatedByID: erwartet %d, bekam %+v", admin, snap.UpdatedByID)
	}
	if !snap.UpdatedByName.Valid || snap.UpdatedByName.String == "" {
		t.Errorf("snapshot.UpdatedByName: erwartet nicht-leer, bekam %+v", snap.UpdatedByName)
	}
	if !snap.UpdatedAt.Valid || snap.UpdatedAt.String == "" {
		t.Errorf("snapshot.UpdatedAt: erwartet gültigen Zeitstempel, bekam %+v", snap.UpdatedAt)
	}
}
