package scheduler

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// diaryRetentionConfig liefert eine Config mit temporärem TrainingDiaryDir,
// damit der Job echte Dateien löschen kann.
func diaryRetentionConfig(t *testing.T) *appconfig.Config {
	t.Helper()
	cfg := testutil.TestConfig()
	cfg.TrainingDiaryDir = t.TempDir()
	return cfg
}

// diaryEntryWithProof legt einen Eintrag samt Nachweis-Datei an und liefert
// Eintrags-ID und Dateinamen.
func diaryEntryWithProof(t *testing.T, db *sql.DB, dir string, memberID, seasonID int) (int, string) {
	t.Helper()
	id := testutil.CreateTrainingDiaryEntry(t, db, memberID, seasonID, "2026-05-01", 45, 6)
	diskName := "proof-" + itoa(id) + ".jpg"
	if err := os.WriteFile(filepath.Join(dir, diskName), []byte("dummy"), 0644); err != nil {
		t.Fatalf("write proof file: %v", err)
	}
	testutil.SetTrainingDiaryProof(t, db, id, diskName, "image/jpeg")
	return id, diskName
}

func proofState(t *testing.T, db *sql.DB, id int) (diskName sql.NullString, purgedAt sql.NullString) {
	t.Helper()
	if err := db.QueryRow(
		`SELECT proof_disk_name, proof_purged_at FROM training_diary_entries WHERE id = ?`,
		id).Scan(&diskName, &purgedAt); err != nil {
		t.Fatalf("proofState: %v", err)
	}
	return diskName, purgedAt
}

// Saisonende älter als 90 Tage → Datei weg, Marker gesetzt, Fachdaten intakt.
func TestTrainingDiaryRetention_PurgesAfterSeasonEnd(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := diaryRetentionConfig(t)
	s := New(db, cfg, nil)

	seasonID := testutil.CreateSeason(t, db, "25/26")
	setSeasonEndDate(t, db, seasonID, -91)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	id, diskName := diaryEntryWithProof(t, db, cfg.TrainingDiaryDir, memberID, seasonID)

	s.runTrainingDiaryRetention()

	if _, err := os.Stat(filepath.Join(cfg.TrainingDiaryDir, diskName)); !os.IsNotExist(err) {
		t.Errorf("Nachweisdatei existiert noch")
	}
	disk, purged := proofState(t, db, id)
	if disk.Valid && disk.String != "" {
		t.Errorf("proof_disk_name = %q, want NULL", disk.String)
	}
	if !purged.Valid || purged.String == "" {
		t.Errorf("proof_purged_at nicht gesetzt")
	}

	// Der Eintrag selbst bleibt vollständig erhalten.
	var trainedOn, kind string
	var duration, rpe int
	if err := db.QueryRow(
		`SELECT trained_on, kind, duration_min, rpe FROM training_diary_entries WHERE id = ?`,
		id).Scan(&trainedOn, &kind, &duration, &rpe); err != nil {
		t.Fatalf("Eintrag verschwunden: %v", err)
	}
	if duration != 45 || rpe != 6 || kind != "kraft" {
		t.Errorf("Fachdaten verändert: %s/%s/%d/%d", trainedOn, kind, duration, rpe)
	}
}

// Innerhalb der Frist bleibt alles unangetastet.
func TestTrainingDiaryRetention_KeepsWithinWindow(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := diaryRetentionConfig(t)
	s := New(db, cfg, nil)

	seasonID := testutil.CreateSeason(t, db, "25/26")
	setSeasonEndDate(t, db, seasonID, -89)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	id, diskName := diaryEntryWithProof(t, db, cfg.TrainingDiaryDir, memberID, seasonID)

	s.runTrainingDiaryRetention()

	if _, err := os.Stat(filepath.Join(cfg.TrainingDiaryDir, diskName)); err != nil {
		t.Errorf("Datei wurde zu früh gelöscht: %v", err)
	}
	if _, purged := proofState(t, db, id); purged.Valid && purged.String != "" {
		t.Errorf("proof_purged_at wurde zu früh gesetzt")
	}
}

// season_id IS NULL: kein Anker, also niemals automatisch löschen.
func TestTrainingDiaryRetention_NullSeasonNeverPurged(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := diaryRetentionConfig(t)
	s := New(db, cfg, nil)

	// Eine längst beendete Saison existiert, der Eintrag hängt aber an keiner.
	seasonID := testutil.CreateSeason(t, db, "25/26")
	setSeasonEndDate(t, db, seasonID, -500)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	id, diskName := diaryEntryWithProof(t, db, cfg.TrainingDiaryDir, memberID, 0)

	s.runTrainingDiaryRetention()

	if _, err := os.Stat(filepath.Join(cfg.TrainingDiaryDir, diskName)); err != nil {
		t.Errorf("Datei ohne Saison-Anker wurde gelöscht: %v", err)
	}
	if _, purged := proofState(t, db, id); purged.Valid && purged.String != "" {
		t.Errorf("proof_purged_at gesetzt, obwohl season_id NULL ist")
	}
}

// Zweiter Lauf verändert nichts und wirft nicht.
func TestTrainingDiaryRetention_Idempotent(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := diaryRetentionConfig(t)
	s := New(db, cfg, nil)

	seasonID := testutil.CreateSeason(t, db, "25/26")
	setSeasonEndDate(t, db, seasonID, -120)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	id, _ := diaryEntryWithProof(t, db, cfg.TrainingDiaryDir, memberID, seasonID)

	s.runTrainingDiaryRetention()
	_, firstPurge := proofState(t, db, id)

	s.runTrainingDiaryRetention()
	_, secondPurge := proofState(t, db, id)

	if firstPurge.String != secondPurge.String {
		t.Errorf("proof_purged_at wurde im zweiten Lauf überschrieben: %q → %q",
			firstPurge.String, secondPurge.String)
	}
}

// Eine bereits von Hand entfernte Datei darf den Lauf nicht aufhalten — der
// Marker muss trotzdem gesetzt werden, sonst läuft die Zeile ewig mit.
func TestTrainingDiaryRetention_MissingFileStillMarks(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := diaryRetentionConfig(t)
	s := New(db, cfg, nil)

	seasonID := testutil.CreateSeason(t, db, "25/26")
	setSeasonEndDate(t, db, seasonID, -100)
	userID := testutil.CreateUser(t, db, "standard")
	memberID := testutil.CreateMember(t, db, userID)
	id, diskName := diaryEntryWithProof(t, db, cfg.TrainingDiaryDir, memberID, seasonID)
	if err := os.Remove(filepath.Join(cfg.TrainingDiaryDir, diskName)); err != nil {
		t.Fatalf("remove file: %v", err)
	}

	s.runTrainingDiaryRetention()

	disk, purged := proofState(t, db, id)
	if disk.Valid && disk.String != "" {
		t.Errorf("proof_disk_name = %q, want NULL", disk.String)
	}
	if !purged.Valid || purged.String == "" {
		t.Errorf("proof_purged_at nicht gesetzt")
	}
}
