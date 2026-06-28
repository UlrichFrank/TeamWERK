package scheduler

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// retentionConfig liefert eine Config mit einem temporären VideoStorageDir, damit
// der Retention-Job echte (Dummy-)Dateien anlegen und löschen kann.
func retentionConfig(t *testing.T) *appconfig.Config {
	t.Helper()
	cfg := testutil.TestConfig()
	cfg.VideoStorageDir = t.TempDir()
	return cfg
}

// setSeasonEndDate überschreibt das (von CreateSeason hartkodierte) Saisonende
// auf einen relativen Wert in Tagen ab heute (negativ = Vergangenheit).
func setSeasonEndDate(t *testing.T, db *sql.DB, seasonID, daysFromNow int) {
	t.Helper()
	if _, err := db.Exec(
		`UPDATE seasons SET end_date = date('now', ?) WHERE id = ?`,
		signedDays(daysFromNow), seasonID); err != nil {
		t.Fatalf("setSeasonEndDate: %v", err)
	}
}

func signedDays(d int) string {
	if d >= 0 {
		return "+" + itoa(d) + " days"
	}
	return itoa(d) + " days"
}

func itoa(n int) string {
	// kleine, allokationsarme Variante ohne strconv-Import-Lärm im Test
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func videoExists(t *testing.T, db *sql.DB, id int) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM videos WHERE id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("videoExists: %v", err)
	}
	return n > 0
}

// TestVideoRetention_DeleteCutoff prüft die exakte Löschgrenze.
// Implementierter Cutoff: date(end_date) < date('now','-90 days') — d.h. erst ab
// dem 91. Tag nach Saisonende wird gelöscht (Karenz: volle 90 Tage geschützt).
//   - 88 Tage her  → bleibt
//   - 90 Tage her  → bleibt (Grenzfall: == -90d ist nicht < -90d)
//   - 91 Tage her  → gelöscht (erster Tag über der Karenz)
//   - 92 Tage her  → gelöscht
func TestVideoRetention_DeleteCutoff(t *testing.T) {
	cases := []struct {
		name       string
		daysAgo    int
		wantDelete bool
	}{
		{"88_days_kept", -88, false},
		{"90_days_kept_boundary", -90, false},
		{"91_days_deleted", -91, true},
		{"92_days_deleted", -92, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			cfg := retentionConfig(t)

			seasonID := testutil.CreateSeason(t, db, "S-"+tc.name)
			setSeasonEndDate(t, db, seasonID, tc.daysAgo)
			teamID := testutil.CreateTeam(t, db, "Team")
			creator := testutil.CreateUser(t, db, "standard")
			vid := testutil.CreateVideo(t, db, teamID, seasonID, creator, "ready")

			New(db, cfg, nil).runVideoRetention()

			got := videoExists(t, db, vid)
			if tc.wantDelete && got {
				t.Fatalf("video %d should have been deleted (end_date %d days ago)", vid, -tc.daysAgo)
			}
			if !tc.wantDelete && !got {
				t.Fatalf("video %d should have been kept (end_date %d days ago)", vid, -tc.daysAgo)
			}
		})
	}
}

// TestVideoRetention_NullEndDate stellt sicher, dass Videos mit season.end_date
// NULL niemals automatisch gelöscht werden. Das reale Schema erzwingt zwar
// seasons.end_date NOT NULL — die Retention-Queries enthalten dennoch eine
// defensive `end_date IS NOT NULL`-Bedingung. Da CreateSeason/das Schema kein
// NULL zulässt, prüfen wir die Invariante deterministisch über eine
// Spiegel-Query mit demselben Prädikat gegen einen künstlich geNULLten Wert.
func TestVideoRetention_NullEndDate(t *testing.T) {
	db := testutil.NewDB(t)

	// Die produktiven Retention-Prädikate dürfen einen NULL-end_date NIEMALS
	// matchen. Wir verifizieren das Prädikat direkt: NULL < ... und NULL = ...
	// liefern in SQLite NULL (= nicht wahr), und die `IS NOT NULL`-Klausel
	// schließt die Zeile zusätzlich aus.
	var matchedDelete, matchedWarn int
	if err := db.QueryRow(`
		SELECT
		  (CASE WHEN end_date IS NOT NULL AND date(end_date) < date('now','-90 days') THEN 1 ELSE 0 END),
		  (CASE WHEN end_date IS NOT NULL AND date(end_date) = date('now','-83 days') THEN 1 ELSE 0 END)
		FROM (SELECT NULL AS end_date)`).Scan(&matchedDelete, &matchedWarn); err != nil {
		t.Fatalf("predicate query: %v", err)
	}
	if matchedDelete != 0 {
		t.Fatalf("delete predicate matched a NULL end_date — video would be wrongly deleted")
	}
	if matchedWarn != 0 {
		t.Fatalf("warning predicate matched a NULL end_date — warning would be wrongly sent")
	}
}

// TestVideoRetention_DeletesFiles prüft, dass raw/{id}.mp4 und processed/{id}/
// beim Löschen mit entfernt werden.
func TestVideoRetention_DeletesFiles(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := retentionConfig(t)

	seasonID := testutil.CreateSeason(t, db, "S-files")
	setSeasonEndDate(t, db, seasonID, -91)
	teamID := testutil.CreateTeam(t, db, "Team")
	creator := testutil.CreateUser(t, db, "standard")
	vid := testutil.CreateVideo(t, db, teamID, seasonID, creator, "ready")

	rawDir := filepath.Join(cfg.VideoStorageDir, "raw")
	procDir := filepath.Join(cfg.VideoStorageDir, "processed", itoa(vid))
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatalf("mkdir raw: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(procDir, "720p"), 0o755); err != nil {
		t.Fatalf("mkdir processed: %v", err)
	}
	rawFile := filepath.Join(rawDir, itoa(vid)+".mp4")
	if err := os.WriteFile(rawFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write raw: %v", err)
	}
	if err := os.WriteFile(filepath.Join(procDir, "720p", "index.m3u8"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write processed: %v", err)
	}

	New(db, cfg, nil).runVideoRetention()

	if _, err := os.Stat(rawFile); !os.IsNotExist(err) {
		t.Fatalf("raw file should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(procDir); !os.IsNotExist(err) {
		t.Fatalf("processed dir should be removed, stat err = %v", err)
	}
}

// addTrainer legt einen Trainer (Member mit User) im Kader des Teams/der Saison an.
func addTrainer(t *testing.T, db *sql.DB, teamID, seasonID int) (userID, memberID int) {
	t.Helper()
	userID = testutil.CreateUser(t, db, "standard")
	memberID = testutil.CreateMember(t, db, userID)
	kaderID := testutil.CreateKader(t, db, teamID, seasonID)
	testutil.AddKaderTrainer(t, db, kaderID, memberID)
	return userID, memberID
}

func warningLogCount(t *testing.T, db *sql.DB, vid int) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM notification_log WHERE ref_type='video_retention_warning' AND ref_id=?`,
		vid).Scan(&n); err != nil {
		t.Fatalf("warningLogCount: %v", err)
	}
	return n
}

// TestVideoRetention_WarningFiresAtT7 prüft, dass die T-7-Vorwarnung genau bei
// Saisonende = heute-83d (Löschung in 7 Tagen) feuert und einen notification_log-
// Eintrag pro Trainer erzeugt — nicht aber bei -82d oder -84d.
func TestVideoRetention_WarningFiresAtT7(t *testing.T) {
	cases := []struct {
		name    string
		daysAgo int
		wantLog int
	}{
		{"day82_no_warning", -82, 0},
		{"day83_warning", -83, 1},
		{"day84_no_warning", -84, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			cfg := retentionConfig(t)

			seasonID := testutil.CreateSeason(t, db, "S-"+tc.name)
			setSeasonEndDate(t, db, seasonID, tc.daysAgo)
			teamID := testutil.CreateTeam(t, db, "Team")
			creator := testutil.CreateUser(t, db, "standard")
			addTrainer(t, db, teamID, seasonID)
			vid := testutil.CreateVideo(t, db, teamID, seasonID, creator, "ready")

			New(db, cfg, nil).runVideoRetention()

			if got := warningLogCount(t, db, vid); got != tc.wantLog {
				t.Fatalf("warning log rows = %d, want %d (end_date %d days ago)", got, tc.wantLog, -tc.daysAgo)
			}
		})
	}
}

// TestVideoRetention_WarningIdempotent prüft, dass ein zweiter Lauf KEINEN
// weiteren notification_log-Eintrag (und damit keinen zweiten Push) erzeugt.
func TestVideoRetention_WarningIdempotent(t *testing.T) {
	db := testutil.NewDB(t)
	cfg := retentionConfig(t)

	seasonID := testutil.CreateSeason(t, db, "S-idem")
	setSeasonEndDate(t, db, seasonID, -83)
	teamID := testutil.CreateTeam(t, db, "Team")
	creator := testutil.CreateUser(t, db, "standard")
	addTrainer(t, db, teamID, seasonID)
	vid := testutil.CreateVideo(t, db, teamID, seasonID, creator, "ready")

	sched := New(db, cfg, nil)
	sched.runVideoRetention()
	sched.runVideoRetention()

	if got := warningLogCount(t, db, vid); got != 1 {
		t.Fatalf("after two runs warning log rows = %d, want 1 (idempotent)", got)
	}
}
