package scheduler

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/testutil"
	"github.com/teamstuttgart/teamwerk/internal/timez"
)

// sundayFourAM liegt im Wartungsfenster (Sonntag 04:00 Uhr Europe/Berlin).
// 2026-01-04 ist ein Sonntag.
func sundayFourAM() time.Time {
	return time.Date(2026, 1, 4, 4, 0, 0, 0, timez.Berlin())
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// Im Wartungsfenster MUSS der Checkpoint genau einmal laufen — ein zweiter
// Aufruf im selben Fenster (gleicher Tag) darf nicht erneut checkpointen
// (specs/db-maintenance/spec.md: "Der Job MUSS idempotent sein").
func TestWalCheckpoint_RunsOnceInWindow(t *testing.T) {
	db := testutil.NewDB(t)
	s := New(db, testutil.TestConfig(), nil)
	buf := captureLogs(t)

	now := sundayFourAM()
	s.walCheckpointAt(now)
	s.walCheckpointAt(now)

	if count := strings.Count(buf.String(), "scheduler wal checkpoint done"); count != 1 {
		t.Fatalf("\"scheduler wal checkpoint done\" count = %d, want 1 (idempotent innerhalb des Fensters)\nlog:\n%s", count, buf.String())
	}

	var marker string
	if err := db.QueryRow(`SELECT value FROM system_settings WHERE key = ?`, walCheckpointSettingKey).Scan(&marker); err != nil {
		t.Fatalf("marker nicht geschrieben: %v", err)
	}
	if want := now.Format("2006-01-02"); marker != want {
		t.Fatalf("marker = %q, want %q", marker, want)
	}
}

// Außerhalb des Wartungsfensters (anderer Wochentag oder andere Stunde) darf
// der Job weder checkpointen noch einen Marker schreiben.
func TestWalCheckpoint_OutsideWindow_NoOp(t *testing.T) {
	cases := []struct {
		name string
		at   time.Time
	}{
		{"montag statt sonntag", time.Date(2026, 1, 5, 4, 0, 0, 0, timez.Berlin())},
		{"sonntag aber falsche stunde", time.Date(2026, 1, 4, 12, 0, 0, 0, timez.Berlin())},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewDB(t)
			s := New(db, testutil.TestConfig(), nil)
			buf := captureLogs(t)

			s.walCheckpointAt(tc.at)

			if strings.Contains(buf.String(), "scheduler wal checkpoint done") {
				t.Fatal("Checkpoint außerhalb des Wartungsfensters ausgeführt")
			}
			var count int
			if err := db.QueryRow(
				`SELECT COUNT(*) FROM system_settings WHERE key = ?`, walCheckpointSettingKey,
			).Scan(&count); err != nil {
				t.Fatalf("count query: %v", err)
			}
			if count != 0 {
				t.Fatal("Marker außerhalb des Wartungsfensters geschrieben")
			}
		})
	}
}

// Im nächsten Wartungsfenster (eine Woche später) MUSS der Checkpoint erneut
// laufen — die Idempotenz gilt nur innerhalb desselben Kalendertags.
func TestWalCheckpoint_NextWindow_RunsAgain(t *testing.T) {
	db := testutil.NewDB(t)
	s := New(db, testutil.TestConfig(), nil)
	buf := captureLogs(t)

	first := sundayFourAM()
	second := first.AddDate(0, 0, 7)

	s.walCheckpointAt(first)
	s.walCheckpointAt(second)

	if count := strings.Count(buf.String(), "scheduler wal checkpoint done"); count != 2 {
		t.Fatalf("\"scheduler wal checkpoint done\" count = %d, want 2 (ein Lauf je Wartungsfenster)\nlog:\n%s", count, buf.String())
	}

	var marker string
	if err := db.QueryRow(`SELECT value FROM system_settings WHERE key = ?`, walCheckpointSettingKey).Scan(&marker); err != nil {
		t.Fatalf("marker nicht geschrieben: %v", err)
	}
	if want := second.Format("2006-01-02"); marker != want {
		t.Fatalf("marker = %q, want %q", marker, want)
	}
}
